package tui

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/javanhut/genesys/pkg/provider"
	"github.com/javanhut/genesys/pkg/provider/aws"
	"github.com/rivo/tview"
)

// SSHConfig holds configuration for SSH connection
type SSHConfig struct {
	Host    string
	User    string
	KeyPath string
	Port    int
}

// SSHConnectionDialog shows a dialog to configure and initiate SSH connection
type SSHConnectionDialog struct {
	*tview.Flex
	appCtx    *AppContext
	instance  *provider.Instance
	form      *tview.Form
	keyPath   string
	username  string
	port      string
	onConnect func(config *SSHConfig)
	onCancel  func()
}

// NewSSHConnectionDialog creates a new SSH connection dialog
func NewSSHConnectionDialog(appCtx *AppContext, instance *provider.Instance, onConnect func(*SSHConfig), onCancel func()) *SSHConnectionDialog {
	dialog := &SSHConnectionDialog{
		Flex:      tview.NewFlex(),
		appCtx:    appCtx,
		instance:  instance,
		form:      tview.NewForm(),
		username:  guessDefaultUser(instance),
		port:      "22",
		onConnect: onConnect,
		onCancel:  onCancel,
	}

	// Try to find default key path
	dialog.keyPath = findDefaultKeyPath(instance)

	dialog.setupForm()
	return dialog
}

func (d *SSHConnectionDialog) setupForm() {
	d.form.SetBorder(true)
	d.form.SetTitle(fmt.Sprintf(" SSH Connect to %s ", d.instance.Name))
	d.form.SetTitleColor(tcell.ColorYellow)
	d.form.SetBorderColor(tcell.ColorBlue)

	// Instance info (read-only display)
	hostInfo := d.instance.PublicIP
	if hostInfo == "" {
		hostInfo = d.instance.PrivateIP + " (private)"
	}

	// Get region from ProviderData
	region := "unknown"
	if d.instance.ProviderData != nil {
		if r, ok := d.instance.ProviderData["Region"].(string); ok {
			region = r
		}
	}

	d.form.AddTextView("Host:", hostInfo, 40, 1, true, false)
	d.form.AddTextView("Region:", region, 20, 1, true, false)

	// Key file input
	d.form.AddInputField("Key File:", d.keyPath, 50, nil, func(text string) {
		d.keyPath = text
	})

	// Username input
	d.form.AddInputField("Username:", d.username, 20, nil, func(text string) {
		d.username = text
	})

	// Port input
	d.form.AddInputField("Port:", d.port, 10, func(textToCheck string, lastChar rune) bool {
		// Only allow digits
		return lastChar >= '0' && lastChar <= '9'
	}, func(text string) {
		d.port = text
	})

	// Buttons
	d.form.AddButton("Connect", func() {
		if err := d.validateAndConnect(); err != nil {
			d.showError(err.Error())
			return
		}
	})

	d.form.AddButton("New Key", func() {
		// Get region for key pair creation
		keyRegion := "us-east-1"
		if d.instance.ProviderData != nil {
			if r, ok := d.instance.ProviderData["Region"].(string); ok {
				keyRegion = r
			}
		}

		ShowCreateKeyPairDialog(d.appCtx, keyRegion, func(keyPath string) {
			// Update the key file field with the new key path
			d.keyPath = keyPath
			// Recreate the form to show updated value
			d.form.Clear(true)
			d.setupForm()
		})
	})

	d.form.AddButton("SSH Rules", func() {
		// Show dialog to manage SSH security group rules
		ShowAddSSHRuleDialog(d.appCtx, d.instance, nil)
	})

	d.form.AddButton("Cancel", func() {
		if d.onCancel != nil {
			d.onCancel()
		}
	})

	d.form.SetButtonsAlign(tview.AlignCenter)
	d.form.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	d.form.SetButtonBackgroundColor(tcell.ColorDarkCyan)

	// Handle escape key
	d.form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			if d.onCancel != nil {
				d.onCancel()
			}
			return nil
		}
		return event
	})

	// Center the form in a flex layout
	d.SetDirection(tview.FlexRow)
	d.AddItem(nil, 0, 1, false)

	horizontal := tview.NewFlex().SetDirection(tview.FlexColumn)
	horizontal.AddItem(nil, 0, 1, false)
	horizontal.AddItem(d.form, 65, 0, true)
	horizontal.AddItem(nil, 0, 1, false)

	d.AddItem(horizontal, 18, 0, true)
	d.AddItem(nil, 0, 1, false)
}

func (d *SSHConnectionDialog) validateAndConnect() error {
	// Validate host
	host := d.instance.PublicIP
	if host == "" {
		host = d.instance.PrivateIP
	}
	if host == "" {
		return fmt.Errorf("instance has no IP address")
	}

	// Validate key path
	keyPath := expandPath(d.keyPath)
	if keyPath == "" {
		return fmt.Errorf("key file path is required")
	}

	// Check if key file exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return fmt.Errorf("key file not found: %s", keyPath)
	}

	// Validate username
	if d.username == "" {
		return fmt.Errorf("username is required")
	}

	// Validate port
	port, err := strconv.Atoi(d.port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid port number")
	}

	config := &SSHConfig{
		Host:    host,
		User:    d.username,
		KeyPath: keyPath,
		Port:    port,
	}

	if d.onConnect != nil {
		d.onConnect(config)
	}

	return nil
}

func (d *SSHConnectionDialog) showError(message string) {
	modal := tview.NewModal().
		SetText("Error: " + message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			d.appCtx.Pages.RemovePage("ssh-error")
		})

	d.appCtx.Pages.AddPage("ssh-error", modal, true, true)
}

// ExecuteSSH runs the SSH command, suspending the TUI
func ExecuteSSH(appCtx *AppContext, config *SSHConfig) error {
	// Build SSH arguments
	// -t forces TTY allocation which is required for interactive sessions
	// -v enables verbose mode to help diagnose connection issues
	args := []string{
		"-v", // Verbose mode for debugging
		"-t", // Force pseudo-terminal allocation
		"-i", config.KeyPath,
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ServerAliveInterval=60",
		"-o", "ConnectTimeout=15",
		"-p", strconv.Itoa(config.Port),
		fmt.Sprintf("%s@%s", config.User, config.Host),
	}

	// Suspend the TUI and run SSH
	appCtx.App.Suspend(func() {
		// Clear screen
		fmt.Print("\033[H\033[2J")
		fmt.Printf("Connecting to %s@%s...\n", config.User, config.Host)
		fmt.Printf("Key: %s\n", config.KeyPath)
		fmt.Printf("Port: %d\n\n", config.Port)

		cmd := exec.Command("ssh", args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			fmt.Printf("\nSSH session ended with error: %v\n", err)
			fmt.Println("\nTroubleshooting tips:")
			fmt.Println("  1. Check if the instance is running")
			fmt.Println("  2. Verify security group allows SSH (port 22) from your IP")
			fmt.Println("     -> Use 's' key in EC2 list to add SSH rule")
			fmt.Println("  3. Confirm the key file matches the instance's key pair")
			fmt.Println("  4. Check key file permissions: chmod 600 " + config.KeyPath)
		} else {
			fmt.Println("\nSSH session ended.")
		}

		fmt.Print("\nPress Enter to return to TUI...")
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
	})

	return nil
}

// guessDefaultUser attempts to determine the default SSH user based on instance metadata
func guessDefaultUser(instance *provider.Instance) string {
	if instance.ProviderData == nil {
		return "ec2-user"
	}

	// Check ImageId for common patterns
	if imageID, ok := instance.ProviderData["ImageId"].(string); ok {
		imageIDLower := strings.ToLower(imageID)
		if strings.Contains(imageIDLower, "ubuntu") {
			return "ubuntu"
		}
	}

	// Check image name/description if available
	if imageName, ok := instance.ProviderData["ImageName"].(string); ok {
		imageNameLower := strings.ToLower(imageName)

		if strings.Contains(imageNameLower, "ubuntu") {
			return "ubuntu"
		}
		if strings.Contains(imageNameLower, "debian") {
			return "admin"
		}
		if strings.Contains(imageNameLower, "centos") {
			return "centos"
		}
		if strings.Contains(imageNameLower, "rhel") || strings.Contains(imageNameLower, "red hat") {
			return "ec2-user"
		}
		if strings.Contains(imageNameLower, "fedora") {
			return "fedora"
		}
		if strings.Contains(imageNameLower, "suse") {
			return "ec2-user"
		}
		if strings.Contains(imageNameLower, "bitnami") {
			return "bitnami"
		}
	}

	// Check platform for Windows
	if platform, ok := instance.ProviderData["Platform"].(string); ok {
		if strings.ToLower(platform) == "windows" {
			return "Administrator"
		}
	}

	// Check tags for hints
	if instance.Tags != nil {
		for key, value := range instance.Tags {
			keyLower := strings.ToLower(key)
			valueLower := strings.ToLower(value)

			if keyLower == "os" || keyLower == "operating_system" {
				if strings.Contains(valueLower, "ubuntu") {
					return "ubuntu"
				}
				if strings.Contains(valueLower, "debian") {
					return "admin"
				}
				if strings.Contains(valueLower, "centos") {
					return "centos"
				}
			}
		}
	}

	// Default to ec2-user (Amazon Linux)
	return "ec2-user"
}

// findDefaultKeyPath attempts to find a default SSH key path
func findDefaultKeyPath(instance *provider.Instance) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	sshDir := filepath.Join(homeDir, ".ssh")

	// First, check if instance has a KeyName and look for matching file
	if instance.ProviderData != nil {
		if keyName, ok := instance.ProviderData["KeyName"].(string); ok && keyName != "" {
			// Try common extensions
			extensions := []string{".pem", "", ".key"}
			for _, ext := range extensions {
				keyPath := filepath.Join(sshDir, keyName+ext)
				if _, err := os.Stat(keyPath); err == nil {
					return keyPath
				}
			}
		}
	}

	// Look for any .pem files in ~/.ssh
	pemFiles, _ := filepath.Glob(filepath.Join(sshDir, "*.pem"))
	if len(pemFiles) == 1 {
		// If there's exactly one PEM file, suggest it
		return pemFiles[0]
	}

	// Return empty string to let user specify
	return ""
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// ShowSSHDialog displays the SSH connection dialog for an instance
func ShowSSHDialog(appCtx *AppContext, instance *provider.Instance) {
	// Check if instance has an IP address
	if instance.PublicIP == "" && instance.PrivateIP == "" {
		modal := tview.NewModal().
			SetText("Cannot connect: Instance has no IP address.\nThe instance may be stopped or not yet initialized.").
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				appCtx.Pages.RemovePage("ssh-error")
			})
		appCtx.Pages.AddPage("ssh-error", modal, true, true)
		return
	}

	// Check if instance is running
	if instance.State != "running" {
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Cannot connect: Instance is %s.\nStart the instance first to connect via SSH.", instance.State)).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				appCtx.Pages.RemovePage("ssh-error")
			})
		appCtx.Pages.AddPage("ssh-error", modal, true, true)
		return
	}

	dialog := NewSSHConnectionDialog(
		appCtx,
		instance,
		func(config *SSHConfig) {
			// Remove dialog
			appCtx.Pages.RemovePage("ssh-dialog")
			// Execute SSH
			ExecuteSSH(appCtx, config)
		},
		func() {
			// Cancel - just remove the dialog
			appCtx.Pages.RemovePage("ssh-dialog")
		},
	)

	appCtx.Pages.AddPage("ssh-dialog", dialog, true, true)
}

// ShowCreateKeyPairDialog shows a dialog to create a new key pair
func ShowCreateKeyPairDialog(appCtx *AppContext, region string, onSuccess func(keyPath string)) {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Create New Key Pair ")
	form.SetTitleColor(tcell.ColorYellow)
	form.SetBorderColor(tcell.ColorBlue)

	// Generate default key name with timestamp
	defaultName := fmt.Sprintf("genesys-key-%d", time.Now().Unix())
	keyName := defaultName

	form.AddInputField("Key Name:", defaultName, 40, nil, func(text string) {
		keyName = text
	})

	form.AddTextView("Region:", region, 20, 1, true, false)
	form.AddTextView("Save to:", "~/.ssh/<keyname>.pem", 40, 1, true, false)

	form.AddButton("Create", func() {
		if keyName == "" {
			showErrorModal(appCtx, "Key name is required")
			return
		}

		// Show creating message
		appCtx.Pages.RemovePage("create-keypair")
		showInfoModal(appCtx, "Creating key pair...", "creating-info")

		go func() {
			// Create the key pair
			ctx := context.Background()
			computeService := appCtx.Provider.Compute().(*aws.ComputeService)
			keyPair, err := computeService.CreateKeyPairInRegion(ctx, keyName, region)

			appCtx.App.QueueUpdateDraw(func() {
				appCtx.Pages.RemovePage("creating-info")

				if err != nil {
					showErrorModal(appCtx, fmt.Sprintf("Failed to create key pair: %v", err))
					return
				}

				// Save the private key to ~/.ssh/
				keyPath, err := saveKeyPair(keyPair)
				if err != nil {
					showErrorModal(appCtx, fmt.Sprintf("Key pair created but failed to save: %v", err))
					return
				}

				// Show success and call callback
				modal := tview.NewModal().
					SetText(fmt.Sprintf("Key pair created successfully!\n\nSaved to: %s\n\nKey Fingerprint:\n%s", keyPath, keyPair.KeyFingerprint)).
					AddButtons([]string{"OK"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						appCtx.Pages.RemovePage("keypair-success")
						if onSuccess != nil {
							onSuccess(keyPath)
						}
					})
				appCtx.Pages.AddPage("keypair-success", modal, true, true)
			})
		}()
	})

	form.AddButton("Cancel", func() {
		appCtx.Pages.RemovePage("create-keypair")
	})

	form.SetButtonsAlign(tview.AlignCenter)
	form.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	form.SetButtonBackgroundColor(tcell.ColorDarkCyan)

	// Handle escape key
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			appCtx.Pages.RemovePage("create-keypair")
			return nil
		}
		return event
	})

	// Center the form
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.AddItem(nil, 0, 1, false)

	horizontal := tview.NewFlex().SetDirection(tview.FlexColumn)
	horizontal.AddItem(nil, 0, 1, false)
	horizontal.AddItem(form, 50, 0, true)
	horizontal.AddItem(nil, 0, 1, false)

	flex.AddItem(horizontal, 12, 0, true)
	flex.AddItem(nil, 0, 1, false)

	appCtx.Pages.AddPage("create-keypair", flex, true, true)
}

// saveKeyPair saves the private key material to ~/.ssh/ with proper permissions
func saveKeyPair(keyPair *aws.KeyPair) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	sshDir := filepath.Join(homeDir, ".ssh")

	// Create .ssh directory if it doesn't exist
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	keyPath := filepath.Join(sshDir, keyPair.KeyName+".pem")

	// Check if file already exists
	if _, err := os.Stat(keyPath); err == nil {
		return "", fmt.Errorf("key file already exists: %s", keyPath)
	}

	// Write the private key with restrictive permissions (0600)
	if err := os.WriteFile(keyPath, []byte(keyPair.KeyMaterial), 0600); err != nil {
		return "", fmt.Errorf("failed to write key file: %w", err)
	}

	return keyPath, nil
}

// showErrorModal displays an error message
func showErrorModal(appCtx *AppContext, message string) {
	modal := tview.NewModal().
		SetText("Error: " + message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			appCtx.Pages.RemovePage("error-modal")
		})
	appCtx.Pages.AddPage("error-modal", modal, true, true)
}

// showInfoModal displays an info message
func showInfoModal(appCtx *AppContext, message string, pageName string) {
	modal := tview.NewModal().
		SetText(message).
		SetBackgroundColor(tcell.ColorDarkBlue)
	appCtx.Pages.AddPage(pageName, modal, true, true)
}

// ListKeyPairsInRegion lists key pairs in a specific region (helper for TUI)
func ListKeyPairsInRegion(appCtx *AppContext, region string) ([]*aws.KeyPair, error) {
	ctx := context.Background()
	computeService := appCtx.Provider.Compute().(*aws.ComputeService)
	return computeService.ListKeyPairsInRegion(ctx, region)
}

// SecurityGroupInfo contains information about SSH access via security groups
type SecurityGroupInfo struct {
	GroupId    string
	GroupName  string
	HasSSHRule bool
	SSHCidrs   []string
	Region     string
}

// CheckSecurityGroupSSH checks if an instance's security groups allow SSH access
func CheckSecurityGroupSSH(appCtx *AppContext, instance *provider.Instance) ([]SecurityGroupInfo, error) {
	if instance.ProviderData == nil {
		return nil, fmt.Errorf("no provider data available for instance")
	}

	sgIds, ok := instance.ProviderData["SecurityGroupIds"].([]string)
	if !ok || len(sgIds) == 0 {
		return nil, fmt.Errorf("no security groups found for instance")
	}

	region := "us-east-1"
	if r, ok := instance.ProviderData["Region"].(string); ok {
		region = r
	}

	ctx := context.Background()
	networkService := appCtx.Provider.Network().(*aws.NetworkService)

	var sgInfos []SecurityGroupInfo
	for _, sgId := range sgIds {
		sgDetail, err := networkService.DescribeSecurityGroupInRegion(ctx, sgId, region)
		if err != nil {
			// Log error but continue with other security groups
			continue
		}

		info := SecurityGroupInfo{
			GroupId:    sgDetail.GroupId,
			GroupName:  sgDetail.GroupName,
			HasSSHRule: sgDetail.HasSSHRule(),
			Region:     region,
		}

		// Collect SSH CIDR blocks
		for _, rule := range sgDetail.IngressRules {
			if (rule.Protocol == "tcp" && rule.FromPort <= 22 && rule.ToPort >= 22) || rule.Protocol == "-1" {
				info.SSHCidrs = append(info.SSHCidrs, rule.CidrBlocks...)
			}
		}

		sgInfos = append(sgInfos, info)
	}

	return sgInfos, nil
}

// ShowAddSSHRuleDialog displays a dialog to add an SSH rule to a security group
func ShowAddSSHRuleDialog(appCtx *AppContext, instance *provider.Instance, onSuccess func()) {
	showAddSSHRuleDialogWithCIDR(appCtx, instance, "0.0.0.0/0", onSuccess)
}

// showAddSSHRuleDialogWithCIDR displays the dialog with a pre-filled CIDR
func showAddSSHRuleDialogWithCIDR(appCtx *AppContext, instance *provider.Instance, initialCIDR string, onSuccess func()) {
	// Remove any existing dialog first
	appCtx.Pages.RemovePage("add-ssh-rule")

	// Get security group info
	sgInfos, err := CheckSecurityGroupSSH(appCtx, instance)
	if err != nil {
		showErrorModal(appCtx, fmt.Sprintf("Failed to check security groups: %v", err))
		return
	}

	if len(sgInfos) == 0 {
		showErrorModal(appCtx, "No security groups found for this instance")
		return
	}

	// Check if any security group already has SSH access with actual CIDR blocks
	for _, sg := range sgInfos {
		if sg.HasSSHRule && len(sg.SSHCidrs) > 0 {
			showInfoModal(appCtx, fmt.Sprintf("Security group %s (%s) already allows SSH access.\nCIDR blocks: %v",
				sg.GroupName, sg.GroupId, sg.SSHCidrs), "ssh-already-open")
			return
		}
	}

	// If we get here, either no SSH rule exists or it has no CIDR blocks (broken rule)

	// Build form to add SSH rule
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Add SSH Rule to Security Group ")
	form.SetTitleColor(tcell.ColorYellow)
	form.SetBorderColor(tcell.ColorBlue)

	// Security group selection
	var sgOptions []string
	for _, sg := range sgInfos {
		sgOptions = append(sgOptions, fmt.Sprintf("%s (%s)", sg.GroupName, sg.GroupId))
	}
	selectedSgIndex := 0

	form.AddDropDown("Security Group:", sgOptions, 0, func(option string, optionIndex int) {
		selectedSgIndex = optionIndex
	})

	// CIDR input - use initial value
	cidrInput := initialCIDR
	form.AddInputField("Source CIDR:", cidrInput, 20, nil, func(text string) {
		cidrInput = text
	})

	// Warning about 0.0.0.0/0
	form.AddTextView("Warning:", "0.0.0.0/0 allows SSH from anywhere.\nConsider using your IP for better security.", 50, 2, true, false)

	// Get my IP button - fetches IP and reopens dialog with it filled in
	form.AddButton("Use My IP", func() {
		showInfoModal(appCtx, "Detecting your public IP...", "detecting-ip")
		go func() {
			ip, err := getMyPublicIP()
			appCtx.App.QueueUpdateDraw(func() {
				appCtx.Pages.RemovePage("detecting-ip")
				if err != nil {
					showErrorModal(appCtx, fmt.Sprintf("Failed to get your IP: %v", err))
					return
				}
				// Reopen the dialog with the IP filled in
				showAddSSHRuleDialogWithCIDR(appCtx, instance, ip+"/32", onSuccess)
			})
		}()
	})

	form.AddButton("Add Rule", func() {
		if selectedSgIndex >= len(sgInfos) {
			showErrorModal(appCtx, "Invalid security group selection")
			return
		}

		sg := sgInfos[selectedSgIndex]

		// Validate CIDR
		if cidrInput == "" {
			showErrorModal(appCtx, "CIDR is required")
			return
		}

		// Show adding message
		appCtx.Pages.RemovePage("add-ssh-rule")
		showInfoModal(appCtx, "Adding SSH rule...", "adding-rule")

		go func() {
			ctx := context.Background()
			networkService := appCtx.Provider.Network().(*aws.NetworkService)
			err := networkService.AddSSHRuleInRegion(ctx, sg.GroupId, sg.Region, cidrInput)

			appCtx.App.QueueUpdateDraw(func() {
				appCtx.Pages.RemovePage("adding-rule")

				if err != nil {
					// Check if it's a duplicate rule error
					if strings.Contains(err.Error(), "InvalidPermission.Duplicate") {
						showInfoModal(appCtx, "SSH rule already exists for this CIDR block.", "rule-exists")
					} else {
						showErrorModal(appCtx, fmt.Sprintf("Failed to add SSH rule: %v", err))
					}
					return
				}

				// Success
				modal := tview.NewModal().
					SetText(fmt.Sprintf("SSH rule added successfully!\n\nSecurity Group: %s\nPort: 22 (SSH)\nSource: %s", sg.GroupName, cidrInput)).
					AddButtons([]string{"OK"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						appCtx.Pages.RemovePage("ssh-rule-success")
						if onSuccess != nil {
							onSuccess()
						}
					})
				appCtx.Pages.AddPage("ssh-rule-success", modal, true, true)
			})
		}()
	})

	form.AddButton("Cancel", func() {
		appCtx.Pages.RemovePage("add-ssh-rule")
	})

	form.SetButtonsAlign(tview.AlignCenter)
	form.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	form.SetButtonBackgroundColor(tcell.ColorDarkCyan)

	// Handle escape key
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			appCtx.Pages.RemovePage("add-ssh-rule")
			return nil
		}
		return event
	})

	// Center the form
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.AddItem(nil, 0, 1, false)

	horizontal := tview.NewFlex().SetDirection(tview.FlexColumn)
	horizontal.AddItem(nil, 0, 1, false)
	horizontal.AddItem(form, 60, 0, true)
	horizontal.AddItem(nil, 0, 1, false)

	flex.AddItem(horizontal, 16, 0, true)
	flex.AddItem(nil, 0, 1, false)

	appCtx.Pages.AddPage("add-ssh-rule", flex, true, true)
}

// getMyPublicIP attempts to get the user's public IP address
func getMyPublicIP() (string, error) {
	// Try multiple services in case one is down
	services := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
	}

	for _, service := range services {
		resp, err := httpGet(service)
		if err != nil {
			continue
		}
		ip := strings.TrimSpace(resp)
		if ip != "" {
			return ip, nil
		}
	}

	return "", fmt.Errorf("failed to determine public IP")
}

// httpGet performs a simple HTTP GET request
func httpGet(url string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	body := make([]byte, 64) // IP addresses are small
	n, _ := resp.Body.Read(body)
	return string(body[:n]), nil
}
