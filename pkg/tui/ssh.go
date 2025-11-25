package tui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/javanhut/genesys/pkg/provider"
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
	horizontal.AddItem(d.form, 60, 0, true)
	horizontal.AddItem(nil, 0, 1, false)

	d.AddItem(horizontal, 17, 0, true)
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
	args := []string{
		"-i", config.KeyPath,
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ServerAliveInterval=60",
		"-p", strconv.Itoa(config.Port),
		fmt.Sprintf("%s@%s", config.User, config.Host),
	}

	// Suspend the TUI and run SSH
	appCtx.App.Suspend(func() {
		// Clear screen
		fmt.Print("\033[H\033[2J")
		fmt.Printf("Connecting to %s@%s...\n\n", config.User, config.Host)

		cmd := exec.Command("ssh", args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			fmt.Printf("\nSSH session ended with error: %v\n", err)
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
