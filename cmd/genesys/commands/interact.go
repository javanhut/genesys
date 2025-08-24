package commands

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

// NewInteractCommand creates the interactive command
func NewInteractCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "interact",
		Short: "Start interactive mode",
		Long:  `Start an interactive wizard to help you deploy infrastructure`,
		RunE:  runInteract,
	}
}

func runInteract(cmd *cobra.Command, args []string) error {
	fmt.Println("Welcome to Genesys Interactive Mode!")
	fmt.Println("====================================\n")

	// Select outcome
	var outcome string
	outcomePrompt := &survey.Select{
		Message: "What would you like to deploy?",
		Options: []string{
			"Static Website",
			"Database",
			"API Function",
			"Network Infrastructure",
			"Storage Bucket",
			"Web Application",
		},
	}
	if err := survey.AskOne(outcomePrompt, &outcome); err != nil {
		return err
	}

	// Based on outcome, gather specific parameters
	switch outcome {
	case "Static Website":
		return interactStaticSite()
	case "Database":
		return interactDatabase()
	case "API Function":
		return interactFunction()
	case "Storage Bucket":
		return interactBucket()
	default:
		fmt.Printf("Outcome '%s' not yet implemented\n", outcome)
	}

	return nil
}

func interactStaticSite() error {
	var params struct {
		Domain    string
		EnableCDN bool
		EnableSSL bool
	}

	questions := []*survey.Question{
		{
			Name:     "Domain",
			Prompt:   &survey.Input{Message: "Domain name (optional):"},
			Validate: survey.Required,
		},
		{
			Name:   "EnableCDN",
			Prompt: &survey.Confirm{Message: "Enable CDN for global distribution?", Default: true},
		},
		{
			Name:   "EnableSSL",
			Prompt: &survey.Confirm{Message: "Enable HTTPS/SSL?", Default: true},
		},
	}

	if err := survey.Ask(questions, &params); err != nil {
		return err
	}

	// Generate and show plan
	fmt.Println("\n📋 Plan for Static Website Deployment")
	fmt.Println("=====================================")
	fmt.Println("\nWhat will happen:")
	fmt.Println("1. Create S3 bucket for hosting website files")
	fmt.Println("2. Configure bucket for static website hosting")
	if params.EnableCDN {
		fmt.Println("3. Set up CloudFront CDN for fast global delivery")
	}
	if params.EnableSSL {
		fmt.Println("4. Configure SSL certificate for HTTPS")
	}
	if params.Domain != "" {
		fmt.Printf("5. Configure custom domain: %s\n", params.Domain)
	}

	fmt.Println("\nEstimated cost: ~$5-10/month")
	fmt.Println("Time to deploy: ~3-5 minutes")

	// Confirm apply
	var applyConfirm bool
	prompt := &survey.Confirm{
		Message: "Would you like to apply these changes?",
		Default: false,
	}
	if err := survey.AskOne(prompt, &applyConfirm); err != nil {
		return err
	}

	if applyConfirm {
		fmt.Println("\n🚀 Deploying static website...")
		fmt.Println("Note: Executor not yet implemented - this is a preview")
	} else {
		fmt.Println("\n📝 Plan saved. You can apply it later.")
	}

	return nil
}

func interactDatabase() error {
	var params struct {
		Engine   string
		Size     string
		MultiAZ  bool
		Backups  bool
	}

	questions := []*survey.Question{
		{
			Name: "Engine",
			Prompt: &survey.Select{
				Message: "Database engine:",
				Options: []string{"PostgreSQL", "MySQL", "MariaDB"},
			},
		},
		{
			Name: "Size",
			Prompt: &survey.Select{
				Message: "Database size:",
				Options: []string{"Small (1-2 vCPU, 1-4 GB RAM)", "Medium (2-4 vCPU, 4-16 GB RAM)", "Large (4-8 vCPU, 16-64 GB RAM)"},
			},
		},
		{
			Name:   "MultiAZ",
			Prompt: &survey.Confirm{Message: "Enable Multi-AZ for high availability?", Default: false},
		},
		{
			Name:   "Backups",
			Prompt: &survey.Confirm{Message: "Enable automated backups?", Default: true},
		},
	}

	if err := survey.Ask(questions, &params); err != nil {
		return err
	}

	fmt.Println("\n📋 Plan for Database Deployment")
	fmt.Println("================================")
	fmt.Printf("\nDatabase Engine: %s\n", params.Engine)
	fmt.Printf("Size: %s\n", params.Size)
	fmt.Printf("High Availability: %v\n", params.MultiAZ)
	fmt.Printf("Automated Backups: %v\n", params.Backups)

	return nil
}

func interactFunction() error {
	var params struct {
		Runtime string
		Trigger string
		Memory  int
	}

	questions := []*survey.Question{
		{
			Name: "Runtime",
			Prompt: &survey.Select{
				Message: "Function runtime:",
				Options: []string{"Python 3.11", "Node.js 18", "Go 1.21", "Java 17"},
			},
		},
		{
			Name: "Trigger",
			Prompt: &survey.Select{
				Message: "Trigger type:",
				Options: []string{"HTTP/API", "Schedule", "Queue", "Storage"},
			},
		},
		{
			Name: "Memory",
			Prompt: &survey.Select{
				Message: "Memory allocation:",
				Options: []string{"128 MB", "256 MB", "512 MB", "1024 MB"},
			},
		},
	}

	if err := survey.Ask(questions, &params); err != nil {
		return err
	}

	fmt.Println("\n📋 Plan for Function Deployment")
	fmt.Println("================================")
	fmt.Printf("\nRuntime: %s\n", params.Runtime)
	fmt.Printf("Trigger: %s\n", params.Trigger)
	fmt.Printf("Memory: %d MB\n", params.Memory)

	return nil
}

func interactBucket() error {
	var params struct {
		Name       string
		Versioning bool
		Encryption bool
		PublicAccess bool
	}

	questions := []*survey.Question{
		{
			Name:     "Name",
			Prompt:   &survey.Input{Message: "Bucket name:"},
			Validate: survey.Required,
		},
		{
			Name:   "Versioning",
			Prompt: &survey.Confirm{Message: "Enable versioning?", Default: true},
		},
		{
			Name:   "Encryption",
			Prompt: &survey.Confirm{Message: "Enable encryption?", Default: true},
		},
		{
			Name:   "PublicAccess",
			Prompt: &survey.Confirm{Message: "Allow public access?", Default: false},
		},
	}

	if err := survey.Ask(questions, &params); err != nil {
		return err
	}

	fmt.Println("\n📋 Plan for Storage Bucket")
	fmt.Println("==========================")
	fmt.Printf("\nBucket name: %s\n", params.Name)
	fmt.Printf("Versioning: %v\n", params.Versioning)
	fmt.Printf("Encryption: %v\n", params.Encryption)
	fmt.Printf("Public Access: %v\n", params.PublicAccess)

	return nil
}