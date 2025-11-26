package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/javanhut/genesys/pkg/provider"
	"github.com/rivo/tview"
)

// LaunchDashboard launches the main TUI dashboard
func LaunchDashboard(ctx context.Context, p provider.Provider) error {
	appCtx := NewAppContext(ctx, p)

	// Create header
	header := createHeader(p.Name(), p.Region())

	// Create footer
	footer := createFooter([]string{"↑↓: Navigate", "Enter: Select", "ESC: Back", "q: Quit"})

	// Create dashboard
	dashboard := createDashboard(appCtx, header, footer)

	// Create main layout
	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(dashboard, 0, 1, true).
		AddItem(footer, 1, 0, false)

	// Add pages
	appCtx.Pages.AddPage("dashboard", mainLayout, true, true)

	// Set root and run
	appCtx.App.SetRoot(appCtx.Pages, true)

	// Initialize navigation
	appCtx.PushPage("dashboard")

	if err := appCtx.App.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

// LaunchManageUI launches the TUI in management mode for a specific resource
func LaunchManageUI(ctx context.Context, p provider.Provider, resourceType, resourceID string) error {
	appCtx := NewAppContext(ctx, p)
	appCtx.CurrentResourceType = resourceType
	appCtx.CurrentResourceID = resourceID

	header := createHeader("Manage", p.Region())
	footer := createFooter([]string{"↑↓: Navigate", "d: Download", "ESC: Back", "q: Quit"})

	var mainView tview.Primitive

	switch resourceType {
	case "s3", "storage":
		mainView = createS3BrowserInteractive(appCtx, resourceID, footer)
	default:
		return fmt.Errorf("TUI management not yet implemented for resource type: %s", resourceType)
	}

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(mainView, 0, 1, true).
		AddItem(footer, 1, 0, false)

	appCtx.Pages.AddPage("main", mainLayout, true, true)

	appCtx.App.SetRoot(appCtx.Pages, true)
	appCtx.PushPage("main")

	if err := appCtx.App.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

// LaunchMonitorUI launches the TUI in monitoring mode
func LaunchMonitorUI(ctx context.Context, p provider.Provider) error {
	return LaunchDashboard(ctx, p)
}

func createHeader(title, region string) *tview.TextView {
	header := tview.NewTextView()
	header.SetDynamicColors(true)
	header.SetTextAlign(tview.AlignLeft)
	text := fmt.Sprintf("[white::b]Genesys TUI[white] - [blue::b]AWS[white] ([yellow]%s[white])                  [gray]Press ? for help[white]", region)
	header.SetText(text)
	return header
}

func createFooter(shortcuts []string) *tview.TextView {
	footer := tview.NewTextView()
	footer.SetDynamicColors(true)
	footer.SetTextAlign(tview.AlignLeft)
	text := ""
	for i, sc := range shortcuts {
		if i > 0 {
			text += " [gray]|[white] "
		}
		text += "[yellow]" + sc + "[white]"
	}
	footer.SetText(text)
	return footer
}

func createDashboard(appCtx *AppContext, header, footer *tview.TextView) *tview.Flex {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Dashboard ")
	list.SetBorderColor(tcell.ColorBlue)
	list.ShowSecondaryText(true)
	list.SetHighlightFullLine(true)
	list.SetSelectedBackgroundColor(tcell.ColorDarkCyan)
	list.SetSelectedTextColor(tcell.ColorWhite)

	// Add menu items
	list.AddItem("EC2 Instances", "Manage compute instances", '2', func() {
		showEC2List(appCtx, header, footer)
	})

	list.AddItem("S3 Buckets", "Browse storage buckets", '3', func() {
		showS3List(appCtx, header, footer)
	})

	list.AddItem("Lambda Functions", "Manage serverless functions", '4', func() {
		showLambdaList(appCtx, header, footer)
	})

	list.AddItem("DynamoDB Tables", "Manage NoSQL database tables", '5', func() {
		showDynamoDBList(appCtx, header, footer)
	})

	list.AddItem("Quit", "Exit the TUI", 'q', func() {
		appCtx.Stop()
	})

	// Handle keyboard shortcuts
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			appCtx.Stop()
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				appCtx.Stop()
				return nil
			}
		}
		return event
	})

	flex := tview.NewFlex()
	flex.AddItem(list, 0, 1, true)
	return flex
}

func showEC2List(appCtx *AppContext, header, footer *tview.TextView) {
	table := tview.NewTable()
	table.SetBorders(false)
	table.SetSelectable(true, false)
	table.SetFixed(1, 0)
	table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkCyan).
		Foreground(tcell.ColorWhite))

	// Add headers - include Region column for multi-region discovery
	headers := []string{"Instance ID", "Name", "Region", "State", "Type", "IP Address"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)
		table.SetCell(0, i, cell)
	}

	// Update footer - add SSH connect and SSH rules options
	footer.SetText("[yellow]↑↓: Navigate[white] [gray]|[white] [yellow]c: SSH Connect[white] [gray]|[white] [yellow]s: SSH Rules[white] [gray]|[white] [yellow]Enter: Details[white] [gray]|[white] [yellow]m: Metrics[white] [gray]|[white] [yellow]r: Refresh[white] [gray]|[white] [yellow]ESC: Back[white] [gray]|[white] [yellow]q: Quit[white]")

	// Handle keyboard shortcuts
	var instances []*provider.Instance
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			showDashboard(appCtx, header, footer)
			return nil
		case tcell.KeyEnter:
			row, _ := table.GetSelection()
			if row > 0 && row <= len(instances) {
				showEC2Detail(appCtx, header, footer, instances[row-1])
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				appCtx.Stop()
				return nil
			case 'r':
				loadEC2Instances(appCtx, table, &instances)
				return nil
			case 'c':
				// SSH Connect
				row, _ := table.GetSelection()
				if row > 0 && row <= len(instances) {
					ShowSSHDialog(appCtx, instances[row-1])
				}
				return nil
			case 's':
				// SSH Security Group Rules
				row, _ := table.GetSelection()
				if row > 0 && row <= len(instances) {
					ShowAddSSHRuleDialog(appCtx, instances[row-1], nil)
				}
				return nil
			}
		}
		return event
	})

	// Create layout
	flex := tview.NewFlex()
	flex.SetBorder(true)
	flex.SetTitle(" EC2 Instances ")
	flex.SetBorderColor(tcell.ColorBlue)
	flex.AddItem(table, 0, 1, true)

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(flex, 0, 1, true).
		AddItem(footer, 1, 0, false)

	appCtx.Pages.AddPage("ec2-list", mainLayout, true, false)
	appCtx.Pages.SwitchToPage("ec2-list")
	appCtx.App.SetFocus(table)

	// Load instances
	loadEC2Instances(appCtx, table, &instances)
}

func loadEC2Instances(appCtx *AppContext, table *tview.Table, instancesPtr *[]*provider.Instance) {
	// Clear existing rows except header
	for i := table.GetRowCount() - 1; i > 0; i-- {
		table.RemoveRow(i)
	}

	// Show loading - scanning all regions takes time
	table.SetCell(1, 0, tview.NewTableCell("Scanning all AWS regions for instances...").
		SetTextColor(tcell.ColorGray))

	// Load instances asynchronously
	go func() {
		instances, err := appCtx.Provider.Compute().DiscoverInstances(appCtx.Ctx)

		appCtx.App.QueueUpdateDraw(func() {
			// Clear loading message
			table.RemoveRow(1)

			if err != nil {
				table.SetCell(1, 0, tview.NewTableCell(fmt.Sprintf("Error: %v", err)).
					SetTextColor(tcell.ColorRed))
				return
			}

			if len(instances) == 0 {
				table.SetCell(1, 0, tview.NewTableCell("No EC2 instances found in any region").
					SetTextColor(tcell.ColorGray))
				return
			}

			// Store instances for navigation
			*instancesPtr = instances

			// Add instance rows
			for i, instance := range instances {
				row := i + 1

				stateColor := tcell.ColorGreen
				if instance.State != "running" {
					stateColor = tcell.ColorRed
				}

				// Get region from ProviderData
				region := ""
				if instance.ProviderData != nil {
					if r, ok := instance.ProviderData["Region"].(string); ok {
						region = r
					}
				}

				// Display public IP if available, otherwise private IP
				displayIP := instance.PublicIP
				if displayIP == "" {
					displayIP = instance.PrivateIP
					if displayIP != "" {
						displayIP = displayIP + " (private)"
					}
				}

				table.SetCell(row, 0, tview.NewTableCell(instance.ID).SetTextColor(tcell.ColorWhite))
				table.SetCell(row, 1, tview.NewTableCell(instance.Name).SetTextColor(tcell.ColorBlue))
				table.SetCell(row, 2, tview.NewTableCell(region).SetTextColor(tcell.ColorYellow))
				table.SetCell(row, 3, tview.NewTableCell(instance.State).SetTextColor(stateColor))
				table.SetCell(row, 4, tview.NewTableCell(string(instance.Type)).SetTextColor(tcell.ColorWhite))
				table.SetCell(row, 5, tview.NewTableCell(displayIP).SetTextColor(tcell.ColorWhite))
			}
		})
	}()
}

func showEC2Detail(appCtx *AppContext, header, footer *tview.TextView, instance *provider.Instance) {
	// Create detail view
	detailView := tview.NewTextView()
	detailView.SetDynamicColors(true)
	detailView.SetBorder(true)
	detailView.SetTitle(fmt.Sprintf(" EC2: %s ", instance.Name))
	detailView.SetBorderColor(tcell.ColorBlue)

	// Update footer
	footer.SetText("[yellow]m: Metrics[white] [gray]|[white] [yellow]l: Logs[white] [gray]|[white] [yellow]ESC: Back[white] [gray]|[white] [yellow]q: Quit[white]")

	// Build detail text
	var details strings.Builder
	details.WriteString(fmt.Sprintf("[yellow]Instance ID:[white] %s\n", instance.ID))
	details.WriteString(fmt.Sprintf("[yellow]Name:[white] %s\n", instance.Name))
	details.WriteString(fmt.Sprintf("[yellow]State:[white] %s\n", instance.State))
	details.WriteString(fmt.Sprintf("[yellow]Type:[white] %s\n", instance.Type))
	details.WriteString(fmt.Sprintf("[yellow]Public IP:[white] %s\n", instance.PublicIP))
	details.WriteString(fmt.Sprintf("[yellow]Private IP:[white] %s\n", instance.PrivateIP))
	details.WriteString(fmt.Sprintf("[yellow]Created:[white] %s\n\n", instance.CreatedAt.Format("2006-01-02 15:04:05")))

	if len(instance.Tags) > 0 {
		details.WriteString("[yellow]Tags:[white]\n")
		for key, value := range instance.Tags {
			details.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
		details.WriteString("\n")
	}

	details.WriteString("[gray]Loading metrics...[white]\n")
	detailView.SetText(details.String())

	// Handle keyboard shortcuts
	detailView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			showEC2List(appCtx, header, footer)
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				appCtx.Stop()
				return nil
			case 'm':
				showEC2Metrics(appCtx, header, footer, instance)
				return nil
			}
		}
		return event
	})

	// Create layout
	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(detailView, 0, 1, true).
		AddItem(footer, 1, 0, false)

	appCtx.Pages.AddPage("ec2-detail", mainLayout, true, false)
	appCtx.Pages.SwitchToPage("ec2-detail")
	appCtx.App.SetFocus(detailView)

	// Load metrics asynchronously
	go func() {
		metrics, err := appCtx.Provider.Monitoring().GetEC2Metrics(appCtx.Ctx, instance.ID, "1h")

		appCtx.App.QueueUpdateDraw(func() {
			var metricsText strings.Builder
			metricsText.WriteString(details.String())
			metricsText.WriteString("[yellow]Recent Metrics (1h):[white]\n")

			if err != nil {
				metricsText.WriteString(fmt.Sprintf("[red]Error loading metrics: %v[white]\n", err))
			} else if metrics != nil {
				if len(metrics.CPUUtilization) > 0 {
					latest := metrics.CPUUtilization[len(metrics.CPUUtilization)-1]
					metricsText.WriteString(fmt.Sprintf("  CPU: [green]%.1f%%[white]\n", latest.Value))
				}
				if len(metrics.NetworkIn) > 0 {
					latest := metrics.NetworkIn[len(metrics.NetworkIn)-1]
					metricsText.WriteString(fmt.Sprintf("  Network In: %.2f MB\n", latest.Value/1024/1024))
				}
				if len(metrics.NetworkOut) > 0 {
					latest := metrics.NetworkOut[len(metrics.NetworkOut)-1]
					metricsText.WriteString(fmt.Sprintf("  Network Out: %.2f MB\n", latest.Value/1024/1024))
				}
			}

			detailView.SetText(metricsText.String())
		})
	}()
}

func showEC2Metrics(appCtx *AppContext, header, footer *tview.TextView, instance *provider.Instance) {
	// Create metrics view
	metricsView := tview.NewTextView()
	metricsView.SetDynamicColors(true)
	metricsView.SetBorder(true)
	metricsView.SetTitle(fmt.Sprintf(" Metrics: %s ", instance.Name))
	metricsView.SetBorderColor(tcell.ColorBlue)
	metricsView.SetText("[gray]Loading metrics...[white]")

	// Update footer
	footer.SetText("[yellow]ESC: Back[white] [gray]|[white] [yellow]q: Quit[white]")

	// Handle keyboard shortcuts
	metricsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			showEC2Detail(appCtx, header, footer, instance)
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				appCtx.Stop()
				return nil
			}
		}
		return event
	})

	// Create layout
	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(metricsView, 0, 1, true).
		AddItem(footer, 1, 0, false)

	appCtx.Pages.AddPage("ec2-metrics", mainLayout, true, false)
	appCtx.Pages.SwitchToPage("ec2-metrics")
	appCtx.App.SetFocus(metricsView)

	// Load metrics
	go func() {
		metrics, err := appCtx.Provider.Monitoring().GetEC2Metrics(appCtx.Ctx, instance.ID, "6h")

		appCtx.App.QueueUpdateDraw(func() {
			var text strings.Builder
			text.WriteString(fmt.Sprintf("[yellow]EC2 Metrics for %s (Last 6 hours)[white]\n\n", instance.Name))

			if err != nil {
				text.WriteString(fmt.Sprintf("[red]Error: %v[white]\n", err))
			} else if metrics != nil {
				// CPU
				if len(metrics.CPUUtilization) > 0 {
					text.WriteString("[yellow]CPU Utilization:[white]\n")
					var sum, min, max float64
					min = metrics.CPUUtilization[0].Value
					for _, dp := range metrics.CPUUtilization {
						sum += dp.Value
						if dp.Value < min {
							min = dp.Value
						}
						if dp.Value > max {
							max = dp.Value
						}
					}
					avg := sum / float64(len(metrics.CPUUtilization))
					text.WriteString(fmt.Sprintf("  Current: [green]%.1f%%[white]\n", metrics.CPUUtilization[len(metrics.CPUUtilization)-1].Value))
					text.WriteString(fmt.Sprintf("  Average: %.1f%%\n", avg))
					text.WriteString(fmt.Sprintf("  Min: %.1f%% | Max: %.1f%%\n\n", min, max))
				}

				// Network
				if len(metrics.NetworkIn) > 0 {
					text.WriteString("[yellow]Network In:[white]\n")
					latest := metrics.NetworkIn[len(metrics.NetworkIn)-1]
					text.WriteString(fmt.Sprintf("  Latest: %.2f MB\n\n", latest.Value/1024/1024))
				}

				if len(metrics.NetworkOut) > 0 {
					text.WriteString("[yellow]Network Out:[white]\n")
					latest := metrics.NetworkOut[len(metrics.NetworkOut)-1]
					text.WriteString(fmt.Sprintf("  Latest: %.2f MB\n\n", latest.Value/1024/1024))
				}
			}

			metricsView.SetText(text.String())
		})
	}()
}

func showS3List(appCtx *AppContext, header, footer *tview.TextView) {
	table := tview.NewTable()
	table.SetBorders(false)
	table.SetSelectable(true, false)
	table.SetFixed(1, 0)
	table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkCyan).
		Foreground(tcell.ColorWhite))

	// Add headers
	headers := []string{"Bucket Name", "Region", "Created"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)
		table.SetCell(0, i, cell)
	}

	// Update footer
	footer.SetText("[yellow]↑↓: Navigate[white] [gray]|[white] [yellow]Enter: Browse[white] [gray]|[white] [yellow]ESC: Back[white] [gray]|[white] [yellow]q: Quit[white]")

	// Handle keyboard shortcuts
	var buckets []*provider.Bucket
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			showDashboard(appCtx, header, footer)
			return nil
		case tcell.KeyEnter:
			row, _ := table.GetSelection()
			if row > 0 && row <= len(buckets) {
				showS3Browser(appCtx, header, footer, buckets[row-1].Name)
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				appCtx.Stop()
				return nil
			case 'r':
				loadS3Buckets(appCtx, table, &buckets)
				return nil
			}
		}
		return event
	})

	// Create layout
	flex := tview.NewFlex()
	flex.SetBorder(true)
	flex.SetTitle(" S3 Buckets ")
	flex.SetBorderColor(tcell.ColorBlue)
	flex.AddItem(table, 0, 1, true)

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(flex, 0, 1, true).
		AddItem(footer, 1, 0, false)

	appCtx.Pages.AddPage("s3-list", mainLayout, true, false)
	appCtx.Pages.SwitchToPage("s3-list")
	appCtx.App.SetFocus(table)

	// Load buckets
	loadS3Buckets(appCtx, table, &buckets)
}

func loadS3Buckets(appCtx *AppContext, table *tview.Table, bucketsPtr *[]*provider.Bucket) {
	// Clear existing rows except header
	for i := table.GetRowCount() - 1; i > 0; i-- {
		table.RemoveRow(i)
	}

	// Show loading
	table.SetCell(1, 0, tview.NewTableCell("Loading buckets...").
		SetTextColor(tcell.ColorGray))

	// Load buckets asynchronously
	go func() {
		buckets, err := appCtx.Provider.Storage().DiscoverBuckets(appCtx.Ctx)

		appCtx.App.QueueUpdateDraw(func() {
			// Clear loading message
			table.RemoveRow(1)

			if err != nil {
				table.SetCell(1, 0, tview.NewTableCell(fmt.Sprintf("Error: %v", err)).
					SetTextColor(tcell.ColorRed))
				return
			}

			if len(buckets) == 0 {
				table.SetCell(1, 0, tview.NewTableCell("No S3 buckets found").
					SetTextColor(tcell.ColorGray))
				return
			}

			// Store buckets for navigation
			*bucketsPtr = buckets

			// Add bucket rows
			for i, bucket := range buckets {
				row := i + 1

				table.SetCell(row, 0, tview.NewTableCell(bucket.Name).SetTextColor(tcell.ColorWhite))
				table.SetCell(row, 1, tview.NewTableCell(bucket.Region).SetTextColor(tcell.ColorBlue))
				table.SetCell(row, 2, tview.NewTableCell(bucket.CreatedAt.Format("2006-01-02")).SetTextColor(tcell.ColorGray))
			}
		})
	}()
}

func showS3Browser(appCtx *AppContext, header, footer *tview.TextView, bucketName string) {
	showS3BrowserWithPrefix(appCtx, header, footer, bucketName, "")
}

func showS3BrowserWithPrefix(appCtx *AppContext, header, footer *tview.TextView, bucketName, prefix string) {
	table := tview.NewTable()
	table.SetBorders(false)
	table.SetSelectable(true, false)
	table.SetFixed(1, 0)
	table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkCyan).
		Foreground(tcell.ColorWhite))

	// Add headers
	headers := []string{"Name", "Size", "Modified"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)
		table.SetCell(0, i, cell)
	}

	// Update footer
	footer.SetText("[yellow]↑↓: Navigate[white] [gray]|[white] [yellow]Enter: Open[white] [gray]|[white] [yellow]Backspace: Up[white] [gray]|[white] [yellow]d: Download[white] [gray]|[white] [yellow]u: Upload[white] [gray]|[white] [yellow]c: Cross-Region Copy[white] [gray]|[white] [yellow]ESC: Back[white]")

	// Store objects for navigation
	var objects []*provider.S3ObjectInfo

	// Handle keyboard shortcuts
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			showS3List(appCtx, header, footer)
			return nil
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if prefix != "" {
				// Navigate up
				parts := strings.Split(strings.TrimSuffix(prefix, "/"), "/")
				if len(parts) > 1 {
					newPrefix := strings.Join(parts[:len(parts)-1], "/") + "/"
					showS3BrowserWithPrefix(appCtx, header, footer, bucketName, newPrefix)
				} else {
					showS3BrowserWithPrefix(appCtx, header, footer, bucketName, "")
				}
			}
			return nil
		case tcell.KeyEnter:
			row, _ := table.GetSelection()
			if row > 0 && row <= len(objects) {
				obj := objects[row-1]
				if obj.IsPrefix {
					// Navigate into folder
					showS3BrowserWithPrefix(appCtx, header, footer, bucketName, obj.Key)
				}
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				appCtx.Stop()
				return nil
			case 'd':
				row, _ := table.GetSelection()
				if row > 0 && row <= len(objects) {
					obj := objects[row-1]
					if !obj.IsPrefix {
						downloadS3File(appCtx, bucketName, obj.Key)
					}
				}
				return nil
			case 'r':
				loadS3Objects(appCtx, table, bucketName, prefix, &objects)
				return nil
			case 'u':
				// Enter upload mode with split-pane view
				showS3UploadView(appCtx, header, footer, bucketName, prefix)
				return nil
			case 'c':
				// Enter cross-region copy mode
				showS3CrossRegionCopyView(appCtx, header, footer, bucketName, prefix, objects)
				return nil
			}
		}
		return event
	})

	// Create layout
	flex := tview.NewFlex()
	flex.SetBorder(true)
	title := fmt.Sprintf(" S3: %s ", bucketName)
	if prefix != "" {
		title = fmt.Sprintf(" S3: %s / %s ", bucketName, prefix)
	}
	flex.SetTitle(title)
	flex.SetBorderColor(tcell.ColorBlue)
	flex.AddItem(table, 0, 1, true)

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(flex, 0, 1, true).
		AddItem(footer, 1, 0, false)

	appCtx.Pages.AddPage("s3-browser", mainLayout, true, false)
	appCtx.Pages.SwitchToPage("s3-browser")
	appCtx.App.SetFocus(table)

	// Load objects
	loadS3Objects(appCtx, table, bucketName, prefix, &objects)
}

func loadS3Objects(appCtx *AppContext, table *tview.Table, bucketName, prefix string, objectsPtr *[]*provider.S3ObjectInfo) {
	// Clear existing rows except header
	for i := table.GetRowCount() - 1; i > 0; i-- {
		table.RemoveRow(i)
	}

	// Show loading
	table.SetCell(1, 0, tview.NewTableCell("Loading objects...").
		SetTextColor(tcell.ColorGray))

	// Load objects asynchronously
	go func() {
		objects, err := appCtx.Provider.Storage().ListObjects(appCtx.Ctx, bucketName, prefix, 1000)

		appCtx.App.QueueUpdateDraw(func() {
			// Clear loading message
			table.RemoveRow(1)

			if err != nil {
				table.SetCell(1, 0, tview.NewTableCell(fmt.Sprintf("Error: %v", err)).
					SetTextColor(tcell.ColorRed))
				return
			}

			if len(objects) == 0 {
				table.SetCell(1, 0, tview.NewTableCell("No objects found").
					SetTextColor(tcell.ColorGray))
				return
			}

			// Store objects for navigation
			*objectsPtr = objects

			// Add objects
			row := 1
			for _, obj := range objects {
				if obj.IsPrefix {
					name := strings.TrimPrefix(obj.Key, prefix)
					table.SetCell(row, 0, tview.NewTableCell("📁 "+name).SetTextColor(tcell.ColorBlue))
					table.SetCell(row, 1, tview.NewTableCell("-").SetTextColor(tcell.ColorGray))
					table.SetCell(row, 2, tview.NewTableCell("-").SetTextColor(tcell.ColorGray))
					row++
				}
			}

			for _, obj := range objects {
				if !obj.IsPrefix {
					name := strings.TrimPrefix(obj.Key, prefix)
					size := formatBytes(obj.Size)
					modified := obj.LastModified.Format("2006-01-02 15:04")

					table.SetCell(row, 0, tview.NewTableCell("📄 "+name).SetTextColor(tcell.ColorWhite))
					table.SetCell(row, 1, tview.NewTableCell(size).SetTextColor(tcell.ColorWhite))
					table.SetCell(row, 2, tview.NewTableCell(modified).SetTextColor(tcell.ColorGray))
					row++
				}
			}
		})
	}()
}

func downloadS3File(appCtx *AppContext, bucketName, key string) {
	// Create modal to show progress
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Downloading %s...", key)).
		AddButtons([]string{"Cancel"})

	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		appCtx.Pages.RemovePage("download-modal")
	})

	appCtx.Pages.AddPage("download-modal", modal, true, true)

	// Download in background
	go func() {
		localPath := "./" + key
		progress := make(chan *provider.TransferProgress, 10)
		done := make(chan error, 1)

		go func() {
			done <- appCtx.Provider.Storage().DownloadFile(appCtx.Ctx, bucketName, key, localPath, progress)
		}()

		for {
			select {
			case p := <-progress:
				appCtx.App.QueueUpdateDraw(func() {
					if p.Status == "complete" {
						modal.SetText(fmt.Sprintf("Downloaded %s successfully!", key))
					} else if p.Status == "failed" {
						modal.SetText(fmt.Sprintf("Download failed: %v", p.Error))
					} else {
						modal.SetText(fmt.Sprintf("Downloading %s... %.1f%%", key, p.PercentComplete))
					}
				})
			case err := <-done:
				appCtx.App.QueueUpdateDraw(func() {
					if err != nil {
						modal.SetText(fmt.Sprintf("Error: %v", err))
					}
				})
				return
			}
		}
	}()
}

func showS3UploadView(appCtx *AppContext, header, footer *tview.TextView, bucketName, s3Prefix string) {
	// State variables
	var localPath string
	var showHiddenFiles bool
	var localEntries []localFileEntry
	var s3Objects []*provider.S3ObjectInfo
	focusLocal := true

	// Initialize local path
	cwd, err := os.Getwd()
	if err != nil {
		cwd, _ = os.UserHomeDir()
	}
	localPath = cwd

	// Create local file browser table (left pane)
	localTable := tview.NewTable()
	localTable.SetBorders(false)
	localTable.SetSelectable(true, false)
	localTable.SetFixed(1, 0)
	localTable.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkGreen).
		Foreground(tcell.ColorWhite))

	// Create S3 browser table (right pane)
	s3Table := tview.NewTable()
	s3Table.SetBorders(false)
	s3Table.SetSelectable(true, false)
	s3Table.SetFixed(1, 0)
	s3Table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkBlue).
		Foreground(tcell.ColorWhite))

	// Create status bar
	statusBar := tview.NewTextView()
	statusBar.SetDynamicColors(true)
	statusBar.SetTextAlign(tview.AlignCenter)

	// Create pane containers
	localPane := tview.NewFlex()
	localPane.SetDirection(tview.FlexRow)
	localPane.SetBorder(true)
	localPane.SetBorderColor(tcell.ColorGreen)
	localPane.AddItem(localTable, 0, 1, true)

	s3Pane := tview.NewFlex()
	s3Pane.SetDirection(tview.FlexRow)
	s3Pane.SetBorder(true)
	s3Pane.SetBorderColor(tcell.ColorGray)
	s3Pane.AddItem(s3Table, 0, 1, false)

	// Helper functions
	updateLocalTitle := func() {
		path := localPath
		if len(path) > 25 {
			path = "..." + path[len(path)-22:]
		}
		localPane.SetTitle(fmt.Sprintf(" Local: %s ", path))
	}

	updateS3Title := func() {
		title := fmt.Sprintf(" S3: %s", bucketName)
		if s3Prefix != "" {
			p := s3Prefix
			if len(p) > 15 {
				p = "..." + p[len(p)-12:]
			}
			title += "/" + strings.TrimSuffix(p, "/")
		}
		title += " "
		s3Pane.SetTitle(title)
	}

	updateStatusBar := func() {
		var focusIndicator string
		if focusLocal {
			focusIndicator = "[green]LOCAL[white]"
		} else {
			focusIndicator = "[blue]S3[white]"
		}
		statusBar.SetText(fmt.Sprintf("Focus: %s | [yellow]Tab[white]: Switch | [yellow]Enter[white]: Navigate/Upload | [yellow]Backspace[white]: Up | [yellow]ESC[white]: Exit", focusIndicator))
	}

	toggleFocus := func() {
		focusLocal = !focusLocal
		updateStatusBar()
		if focusLocal {
			appCtx.App.SetFocus(localTable)
			localPane.SetBorderColor(tcell.ColorGreen)
			s3Pane.SetBorderColor(tcell.ColorGray)
		} else {
			appCtx.App.SetFocus(s3Table)
			localPane.SetBorderColor(tcell.ColorGray)
			s3Pane.SetBorderColor(tcell.ColorBlue)
		}
	}

	// Load local directory function
	var loadLocalDir func()
	loadLocalDir = func() {
		// Clear existing rows
		for i := localTable.GetRowCount() - 1; i > 0; i-- {
			localTable.RemoveRow(i)
		}

		// Set headers
		localHeaders := []string{"Name", "Size", "Modified"}
		for i, h := range localHeaders {
			cell := tview.NewTableCell(h).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignLeft).
				SetSelectable(false).
				SetAttributes(tcell.AttrBold)
			localTable.SetCell(0, i, cell)
		}

		// Read directory
		dirEntries, err := os.ReadDir(localPath)
		if err != nil {
			localTable.SetCell(1, 0, tview.NewTableCell(fmt.Sprintf("Error: %v", err)).
				SetTextColor(tcell.ColorRed))
			return
		}

		localEntries = make([]localFileEntry, 0, len(dirEntries))
		for _, de := range dirEntries {
			name := de.Name()
			if !showHiddenFiles && strings.HasPrefix(name, ".") {
				continue
			}
			info, err := de.Info()
			if err != nil {
				continue
			}
			localEntries = append(localEntries, localFileEntry{
				Name:    name,
				Path:    filepath.Join(localPath, name),
				IsDir:   de.IsDir(),
				Size:    info.Size(),
				ModTime: info.ModTime().Format("2006-01-02 15:04"),
			})
		}

		// Sort: directories first
		sort.Slice(localEntries, func(i, j int) bool {
			if localEntries[i].IsDir != localEntries[j].IsDir {
				return localEntries[i].IsDir
			}
			return strings.ToLower(localEntries[i].Name) < strings.ToLower(localEntries[j].Name)
		})

		if len(localEntries) == 0 {
			localTable.SetCell(1, 0, tview.NewTableCell("(empty)").SetTextColor(tcell.ColorGray))
			return
		}

		row := 1
		for _, entry := range localEntries {
			var nameCell *tview.TableCell
			var sizeStr string
			if entry.IsDir {
				nameCell = tview.NewTableCell("[DIR] " + entry.Name).SetTextColor(tcell.ColorBlue)
				sizeStr = "-"
			} else {
				nameCell = tview.NewTableCell("      " + entry.Name).SetTextColor(tcell.ColorWhite)
				sizeStr = formatBytes(entry.Size)
			}
			localTable.SetCell(row, 0, nameCell)
			localTable.SetCell(row, 1, tview.NewTableCell(sizeStr).SetTextColor(tcell.ColorWhite))
			localTable.SetCell(row, 2, tview.NewTableCell(entry.ModTime).SetTextColor(tcell.ColorGray))
			row++
		}
		localTable.Select(1, 0)
	}

	// Load S3 objects function
	var loadS3Dir func()
	loadS3Dir = func() {
		// Clear existing rows
		for i := s3Table.GetRowCount() - 1; i > 0; i-- {
			s3Table.RemoveRow(i)
		}

		// Set headers
		s3Headers := []string{"Name", "Size", "Modified"}
		for i, h := range s3Headers {
			cell := tview.NewTableCell(h).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignLeft).
				SetSelectable(false).
				SetAttributes(tcell.AttrBold)
			s3Table.SetCell(0, i, cell)
		}

		s3Table.SetCell(1, 0, tview.NewTableCell("Loading...").SetTextColor(tcell.ColorGray))

		go func() {
			objects, err := appCtx.Provider.Storage().ListObjects(appCtx.Ctx, bucketName, s3Prefix, 1000)
			appCtx.App.QueueUpdateDraw(func() {
				s3Table.RemoveRow(1)
				if err != nil {
					s3Table.SetCell(1, 0, tview.NewTableCell(fmt.Sprintf("Error: %v", err)).SetTextColor(tcell.ColorRed))
					return
				}
				s3Objects = objects
				if len(objects) == 0 {
					s3Table.SetCell(1, 0, tview.NewTableCell("(empty)").SetTextColor(tcell.ColorGray))
					return
				}
				row := 1
				for _, obj := range objects {
					if obj.IsPrefix {
						name := strings.TrimPrefix(obj.Key, s3Prefix)
						s3Table.SetCell(row, 0, tview.NewTableCell("[DIR] "+name).SetTextColor(tcell.ColorBlue))
						s3Table.SetCell(row, 1, tview.NewTableCell("-").SetTextColor(tcell.ColorGray))
						s3Table.SetCell(row, 2, tview.NewTableCell("-").SetTextColor(tcell.ColorGray))
						row++
					}
				}
				for _, obj := range objects {
					if !obj.IsPrefix {
						name := strings.TrimPrefix(obj.Key, s3Prefix)
						size := formatBytes(obj.Size)
						modified := obj.LastModified.Format("2006-01-02 15:04")
						s3Table.SetCell(row, 0, tview.NewTableCell("      "+name).SetTextColor(tcell.ColorWhite))
						s3Table.SetCell(row, 1, tview.NewTableCell(size).SetTextColor(tcell.ColorWhite))
						s3Table.SetCell(row, 2, tview.NewTableCell(modified).SetTextColor(tcell.ColorGray))
						row++
					}
				}
			})
		}()
	}

	// Upload file function
	uploadFile := func(localFilePath string) {
		fileName := filepath.Base(localFilePath)
		s3Key := s3Prefix + fileName

		modal := tview.NewModal().
			SetText(fmt.Sprintf("Uploading %s...", fileName)).
			AddButtons([]string{"OK"})
		modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			appCtx.Pages.RemovePage("upload-modal")
		})
		appCtx.Pages.AddPage("upload-modal", modal, true, true)

		go func() {
			progress := make(chan *provider.TransferProgress, 10)
			done := make(chan error, 1)
			go func() {
				done <- appCtx.Provider.Storage().UploadFile(appCtx.Ctx, bucketName, s3Key, localFilePath, progress)
			}()

			for {
				select {
				case p := <-progress:
					appCtx.App.QueueUpdateDraw(func() {
						switch p.Status {
						case "complete":
							modal.SetText(fmt.Sprintf("Uploaded %s successfully!\n\nDestination: s3://%s/%s", fileName, bucketName, s3Key))
							loadS3Dir()
						case "failed":
							modal.SetText(fmt.Sprintf("Upload failed: %v", p.Error))
						default:
							modal.SetText(fmt.Sprintf("Uploading %s...\n\n%.1f%% complete", fileName, p.PercentComplete))
						}
					})
				case err := <-done:
					appCtx.App.QueueUpdateDraw(func() {
						if err != nil {
							modal.SetText(fmt.Sprintf("Error: %v", err))
						}
					})
					return
				}
			}
		}()
	}

	// Local table keyboard handling
	localTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab, tcell.KeyBacktab:
			toggleFocus()
			return nil
		case tcell.KeyEsc:
			showS3BrowserWithPrefix(appCtx, header, footer, bucketName, s3Prefix)
			return nil
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			parent := filepath.Dir(localPath)
			if parent != localPath {
				localPath = parent
				updateLocalTitle()
				loadLocalDir()
			}
			return nil
		case tcell.KeyEnter:
			row, _ := localTable.GetSelection()
			if row > 0 && row <= len(localEntries) {
				entry := localEntries[row-1]
				if entry.IsDir {
					localPath = entry.Path
					updateLocalTitle()
					loadLocalDir()
				} else {
					// Upload file
					uploadFile(entry.Path)
				}
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				appCtx.Stop()
				return nil
			case 'h':
				showHiddenFiles = !showHiddenFiles
				loadLocalDir()
				return nil
			case '~':
				home, err := os.UserHomeDir()
				if err == nil {
					localPath = home
					updateLocalTitle()
					loadLocalDir()
				}
				return nil
			case 'r':
				loadLocalDir()
				return nil
			}
		}
		return event
	})

	// S3 table keyboard handling
	s3Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab, tcell.KeyBacktab:
			toggleFocus()
			return nil
		case tcell.KeyEsc:
			showS3BrowserWithPrefix(appCtx, header, footer, bucketName, s3Prefix)
			return nil
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if s3Prefix != "" {
				parts := strings.Split(strings.TrimSuffix(s3Prefix, "/"), "/")
				if len(parts) > 1 {
					s3Prefix = strings.Join(parts[:len(parts)-1], "/") + "/"
				} else {
					s3Prefix = ""
				}
				updateS3Title()
				loadS3Dir()
			}
			return nil
		case tcell.KeyEnter:
			row, _ := s3Table.GetSelection()
			if row > 0 && row <= len(s3Objects) {
				obj := s3Objects[row-1]
				if obj.IsPrefix {
					s3Prefix = obj.Key
					updateS3Title()
					loadS3Dir()
				}
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				appCtx.Stop()
				return nil
			case 'r':
				loadS3Dir()
				return nil
			}
		}
		return event
	})

	// Initialize titles and status
	updateLocalTitle()
	updateS3Title()
	updateStatusBar()

	// Load initial data
	loadLocalDir()
	loadS3Dir()

	// Update footer
	footer.SetText("[yellow]Tab: Switch[white] [gray]|[white] [yellow]Enter: Open/Upload[white] [gray]|[white] [yellow]Backspace: Up[white] [gray]|[white] [yellow]h: Hidden[white] [gray]|[white] [yellow]~: Home[white] [gray]|[white] [yellow]ESC: Exit[white]")

	// Create horizontal split with local and S3 panes
	browserFlex := tview.NewFlex()
	browserFlex.SetDirection(tview.FlexColumn)
	browserFlex.AddItem(localPane, 0, 1, true)
	browserFlex.AddItem(s3Pane, 0, 1, false)

	// Create main layout
	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(browserFlex, 0, 1, true).
		AddItem(statusBar, 1, 0, false).
		AddItem(footer, 1, 0, false)

	appCtx.Pages.AddPage("s3-upload", mainLayout, true, false)
	appCtx.PushPage("s3-upload")
	appCtx.Pages.SwitchToPage("s3-upload")
	appCtx.App.SetFocus(localTable)
}

// localFileEntry represents a local file or directory for the upload view
type localFileEntry struct {
	Name    string
	Path    string
	IsDir   bool
	Size    int64
	ModTime string
}

func showS3CrossRegionCopyView(appCtx *AppContext, header, footer *tview.TextView, bucketName, prefix string, objects []*provider.S3ObjectInfo) {
	// Get source bucket region
	srcRegion, err := appCtx.Provider.Storage().GetBucketRegion(appCtx.Ctx, bucketName)
	if err != nil {
		srcRegion = appCtx.Provider.Region()
	}

	// State variables
	selectedObjects := make(map[string]bool)
	var dstRegion string
	var copyInProgress bool
	focusIndex := 0 // 0: srcTable, 1: regionList, 2: dstBucketInput

	// Initialize selected objects (select all by default)
	for _, obj := range objects {
		if !obj.IsPrefix {
			selectedObjects[obj.Key] = true
		}
	}

	// Create source objects table (left pane)
	srcTable := tview.NewTable()
	srcTable.SetBorders(false)
	srcTable.SetSelectable(true, false)
	srcTable.SetFixed(1, 0)
	srcTable.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkGreen).
		Foreground(tcell.ColorWhite))

	// Set headers for source table
	srcHeaders := []string{"[x]", "Name", "Size"}
	for i, h := range srcHeaders {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)
		srcTable.SetCell(0, i, cell)
	}

	// Populate source objects
	row := 1
	for _, obj := range objects {
		if obj.IsPrefix {
			continue
		}
		checkbox := "[x]"
		srcTable.SetCell(row, 0, tview.NewTableCell(checkbox).SetTextColor(tcell.ColorGreen))
		name := obj.Key
		if prefix != "" {
			name = strings.TrimPrefix(obj.Key, prefix)
		}
		srcTable.SetCell(row, 1, tview.NewTableCell(name).SetTextColor(tcell.ColorWhite))
		srcTable.SetCell(row, 2, tview.NewTableCell(formatBytes(obj.Size)).SetTextColor(tcell.ColorGray))
		row++
	}

	if row == 1 {
		srcTable.SetCell(1, 1, tview.NewTableCell("No objects to copy").SetTextColor(tcell.ColorGray))
	}

	// Create region selection list
	regionList := tview.NewList()
	regionList.ShowSecondaryText(false)
	regionList.SetHighlightFullLine(true)
	regionList.SetSelectedBackgroundColor(tcell.ColorDarkBlue)

	// Populate regions (excluding source region)
	for _, region := range AWSRegions {
		if region.Code != srcRegion {
			regionList.AddItem(fmt.Sprintf("%s (%s)", region.Code, region.Name), "", 0, nil)
		}
	}

	// Set initial selection
	if regionList.GetItemCount() > 0 {
		regionList.SetCurrentItem(0)
		mainText, _ := regionList.GetItemText(0)
		parts := strings.Split(mainText, " ")
		if len(parts) > 0 {
			dstRegion = parts[0]
		}
	}

	// Create destination bucket input
	dstBucketInput := tview.NewInputField()
	dstBucketInput.SetLabel("Dest Bucket: ")
	dstBucketInput.SetFieldWidth(30)
	dstBucketInput.SetFieldBackgroundColor(tcell.ColorDarkBlue)
	dstBucketInput.SetText(bucketName + "-" + dstRegion)

	// Progress text
	progressText := tview.NewTextView()
	progressText.SetDynamicColors(true)
	progressText.SetBorder(true)
	progressText.SetTitle(" Progress ")
	progressText.SetBorderColor(tcell.ColorGray)
	progressText.SetText("[gray]Press 'c' to start copying selected objects to the destination bucket[white]")

	// Status bar
	statusBar := tview.NewTextView()
	statusBar.SetDynamicColors(true)
	statusBar.SetTextAlign(tview.AlignCenter)

	// Update status bar function
	updateStatusBar := func() {
		var selectedCount int
		var selectedSize int64
		for _, obj := range objects {
			if obj.IsPrefix {
				continue
			}
			if selectedObjects[obj.Key] {
				selectedCount++
				selectedSize += obj.Size
			}
		}

		focusName := "Source Objects"
		switch focusIndex {
		case 1:
			focusName = "Region Selection"
		case 2:
			focusName = "Bucket Name"
		}

		dstBucket := dstBucketInput.GetText()
		statusBar.SetText(fmt.Sprintf(
			"[yellow]Focus:[white] %s | [yellow]Selected:[white] %d objects (%s) | [yellow]Dest:[white] %s/%s | [yellow]Tab:[white] Switch | [yellow]Space:[white] Toggle | [yellow]c:[white] Copy",
			focusName, selectedCount, formatBytes(selectedSize), dstRegion, dstBucket,
		))
	}

	// Toggle selection helper
	toggleSelection := func(tableRow int) {
		objIndex := 0
		for _, obj := range objects {
			if obj.IsPrefix {
				continue
			}
			objIndex++
			if objIndex == tableRow {
				selectedObjects[obj.Key] = !selectedObjects[obj.Key]
				checkbox := "[ ]"
				color := tcell.ColorGray
				if selectedObjects[obj.Key] {
					checkbox = "[x]"
					color = tcell.ColorGreen
				}
				srcTable.GetCell(tableRow, 0).SetText(checkbox).SetTextColor(color)
				updateStatusBar()
				break
			}
		}
	}

	// Select all helper
	selectAll := func(selected bool) {
		r := 1
		for _, obj := range objects {
			if obj.IsPrefix {
				continue
			}
			selectedObjects[obj.Key] = selected
			checkbox := "[ ]"
			color := tcell.ColorGray
			if selected {
				checkbox = "[x]"
				color = tcell.ColorGreen
			}
			srcTable.GetCell(r, 0).SetText(checkbox).SetTextColor(color)
			r++
		}
		updateStatusBar()
	}

	// Start copy function
	startCopy := func() {
		if copyInProgress {
			return
		}

		dstBucket := dstBucketInput.GetText()
		if dstBucket == "" {
			progressText.SetText("[red]Error: Please enter a destination bucket name[white]")
			return
		}

		if dstRegion == "" {
			progressText.SetText("[red]Error: Please select a destination region[white]")
			return
		}

		// Get selected objects
		var selectedObjs []*provider.S3ObjectInfo
		for _, obj := range objects {
			if obj.IsPrefix {
				continue
			}
			if selectedObjects[obj.Key] {
				selectedObjs = append(selectedObjs, obj)
			}
		}

		if len(selectedObjs) == 0 {
			progressText.SetText("[red]Error: No objects selected for copying[white]")
			return
		}

		copyInProgress = true

		progressText.SetText(fmt.Sprintf(
			"[yellow]Starting cross-region copy...[white]\n"+
				"Source: s3://%s (%s)\n"+
				"Destination: s3://%s (%s)\n"+
				"Objects: %d\n\n"+
				"[gray]Please wait...[white]",
			bucketName, srcRegion,
			dstBucket, dstRegion,
			len(selectedObjs),
		))

		// Perform copy in background
		go func() {
			startTime := time.Now()
			var totalSize int64
			for _, obj := range selectedObjs {
				totalSize += obj.Size
			}

			var copiedObjects, copiedBytes int64
			var failedKeys []string

			for i, obj := range selectedObjs {
				// Update progress
				appCtx.App.QueueUpdateDraw(func() {
					elapsed := time.Since(startTime)
					pct := float64(i) / float64(len(selectedObjs)) * 100

					var progressBar string
					barWidth := 30
					filled := int(pct / 100 * float64(barWidth))
					for j := 0; j < barWidth; j++ {
						if j < filled {
							progressBar += "="
						} else if j == filled {
							progressBar += ">"
						} else {
							progressBar += " "
						}
					}

					displayKey := obj.Key
					if len(displayKey) > 35 {
						displayKey = "..." + displayKey[len(displayKey)-32:]
					}

					progressText.SetText(fmt.Sprintf(
						"[yellow]Status: COPYING[white]\n\n"+
							"Source: [blue]s3://%s[white] (%s)\n"+
							"Dest:   [blue]s3://%s[white] (%s)\n\n"+
							"Progress: [%s] %.1f%%\n"+
							"Objects:  %d / %d\n"+
							"Data:     %s / %s\n"+
							"Current:  [gray]%s[white]\n"+
							"Elapsed:  %s",
						bucketName, srcRegion,
						dstBucket, dstRegion,
						progressBar, pct,
						copiedObjects, len(selectedObjs),
						formatBytes(copiedBytes), formatBytes(totalSize),
						displayKey,
						elapsed.Round(time.Second),
					))
				})

				// Determine destination key
				dstKey := obj.Key
				if prefix != "" {
					dstKey = strings.TrimPrefix(obj.Key, prefix)
				}

				// Copy the object
				err := appCtx.Provider.Storage().CopyObjectCrossRegion(
					appCtx.Ctx,
					bucketName,
					obj.Key,
					dstRegion,
					dstBucket,
					dstKey,
				)

				if err != nil {
					failedKeys = append(failedKeys, obj.Key)
				} else {
					copiedObjects++
					copiedBytes += obj.Size
				}
			}

			// Final update
			appCtx.App.QueueUpdateDraw(func() {
				elapsed := time.Since(startTime)
				bytesPerSecond := float64(copiedBytes) / elapsed.Seconds()

				status := "[green]COMPLETE[white]"
				if len(failedKeys) > 0 {
					if copiedObjects == 0 {
						status = "[red]FAILED[white]"
					} else {
						status = "[yellow]PARTIAL[white]"
					}
				}

				var text strings.Builder
				text.WriteString(fmt.Sprintf("Status: %s\n\n", status))
				text.WriteString(fmt.Sprintf("Source: [blue]s3://%s[white] (%s)\n", bucketName, srcRegion))
				text.WriteString(fmt.Sprintf("Dest:   [blue]s3://%s[white] (%s)\n\n", dstBucket, dstRegion))
				text.WriteString(fmt.Sprintf("Objects:  %d / %d copied", copiedObjects, len(selectedObjs)))
				if len(failedKeys) > 0 {
					text.WriteString(fmt.Sprintf(" ([red]%d failed[white])", len(failedKeys)))
				}
				text.WriteString("\n")
				text.WriteString(fmt.Sprintf("Data:     %s transferred\n", formatBytes(copiedBytes)))
				text.WriteString(fmt.Sprintf("Speed:    %s/s\n", formatBytes(int64(bytesPerSecond))))
				text.WriteString(fmt.Sprintf("Elapsed:  %s\n\n", elapsed.Round(time.Second)))
				text.WriteString("[gray]Press ESC to go back[white]")

				progressText.SetText(text.String())
				copyInProgress = false
			})
		}()
	}

	// Region list change handler
	regionList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		parts := strings.Split(mainText, " ")
		if len(parts) > 0 {
			dstRegion = parts[0]
			// Update default bucket name
			dstBucketInput.SetText(bucketName + "-" + dstRegion)
			updateStatusBar()
		}
	})

	// Source table input handler
	srcTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			focusIndex = 1
			appCtx.App.SetFocus(regionList)
			updateStatusBar()
			return nil
		case tcell.KeyEsc:
			if !copyInProgress {
				appCtx.Pages.RemovePage("s3-copy")
				showS3BrowserWithPrefix(appCtx, header, footer, bucketName, prefix)
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case ' ':
				r, _ := srcTable.GetSelection()
				if r > 0 {
					toggleSelection(r)
				}
				return nil
			case 'a':
				selectAll(true)
				return nil
			case 'n':
				selectAll(false)
				return nil
			case 'c':
				startCopy()
				return nil
			case 'q':
				if !copyInProgress {
					appCtx.Stop()
				}
				return nil
			}
		}
		return event
	})

	// Region list input handler
	regionList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			focusIndex = 2
			appCtx.App.SetFocus(dstBucketInput)
			updateStatusBar()
			return nil
		case tcell.KeyBacktab:
			focusIndex = 0
			appCtx.App.SetFocus(srcTable)
			updateStatusBar()
			return nil
		case tcell.KeyEsc:
			if !copyInProgress {
				appCtx.Pages.RemovePage("s3-copy")
				showS3BrowserWithPrefix(appCtx, header, footer, bucketName, prefix)
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'c':
				startCopy()
				return nil
			case 'q':
				if !copyInProgress {
					appCtx.Stop()
				}
				return nil
			}
		}
		return event
	})

	// Bucket input handler
	dstBucketInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			focusIndex = 0
			appCtx.App.SetFocus(srcTable)
			updateStatusBar()
			return nil
		case tcell.KeyBacktab:
			focusIndex = 1
			appCtx.App.SetFocus(regionList)
			updateStatusBar()
			return nil
		case tcell.KeyEsc:
			if !copyInProgress {
				appCtx.Pages.RemovePage("s3-copy")
				showS3BrowserWithPrefix(appCtx, header, footer, bucketName, prefix)
			}
			return nil
		case tcell.KeyEnter:
			startCopy()
			return nil
		}
		return event
	})

	// Create panes
	srcPane := tview.NewFlex()
	srcPane.SetDirection(tview.FlexRow)
	srcPane.SetBorder(true)
	srcPane.SetTitle(fmt.Sprintf(" Source: %s (%s) ", bucketName, srcRegion))
	srcPane.SetBorderColor(tcell.ColorGreen)
	srcPane.AddItem(srcTable, 0, 1, true)

	dstPane := tview.NewFlex()
	dstPane.SetDirection(tview.FlexRow)
	dstPane.SetBorder(true)
	dstPane.SetTitle(" Destination ")
	dstPane.SetBorderColor(tcell.ColorBlue)

	regionLabel := tview.NewTextView()
	regionLabel.SetText("[yellow]Select Region:[white]")
	regionLabel.SetDynamicColors(true)

	bucketLabel := tview.NewTextView()
	bucketLabel.SetText("[yellow]Bucket Name:[white]")
	bucketLabel.SetDynamicColors(true)

	dstPane.AddItem(regionLabel, 1, 0, false)
	dstPane.AddItem(regionList, 0, 1, false)
	dstPane.AddItem(bucketLabel, 1, 0, false)
	dstPane.AddItem(dstBucketInput, 1, 0, false)

	// Main content
	mainContent := tview.NewFlex()
	mainContent.SetDirection(tview.FlexColumn)
	mainContent.AddItem(srcPane, 0, 1, true)
	mainContent.AddItem(dstPane, 0, 1, false)

	// Full layout
	copyLayout := tview.NewFlex()
	copyLayout.SetDirection(tview.FlexRow)
	copyLayout.AddItem(mainContent, 0, 1, true)
	copyLayout.AddItem(progressText, 12, 0, false)
	copyLayout.AddItem(statusBar, 1, 0, false)

	// Update footer
	footer.SetText("[yellow]Tab: Switch[white] [gray]|[white] [yellow]Space: Toggle[white] [gray]|[white] [yellow]a/n: All/None[white] [gray]|[white] [yellow]c: Start Copy[white] [gray]|[white] [yellow]ESC: Back[white]")

	// Initialize status bar
	updateStatusBar()

	// Create main layout
	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(copyLayout, 0, 1, true).
		AddItem(footer, 1, 0, false)

	appCtx.Pages.AddPage("s3-copy", mainLayout, true, false)
	appCtx.PushPage("s3-copy")
	appCtx.Pages.SwitchToPage("s3-copy")
	appCtx.App.SetFocus(srcTable)
}

func showLambdaList(appCtx *AppContext, header, footer *tview.TextView) {
	table := tview.NewTable()
	table.SetBorders(false)
	table.SetSelectable(true, false)
	table.SetFixed(1, 0)
	table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkCyan).
		Foreground(tcell.ColorWhite))

	// Add headers
	headers := []string{"Function Name", "Runtime", "Memory", "Timeout"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)
		table.SetCell(0, i, cell)
	}

	// Update footer
	footer.SetText("[yellow]↑↓: Navigate[white] [gray]|[white] [yellow]Enter: Details[white] [gray]|[white] [yellow]i: Invoke[white] [gray]|[white] [yellow]l: Logs[white] [gray]|[white] [yellow]ESC: Back[white]")

	// Handle keyboard shortcuts
	var functions []*provider.Function
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			showDashboard(appCtx, header, footer)
			return nil
		case tcell.KeyEnter:
			row, _ := table.GetSelection()
			if row > 0 && row <= len(functions) {
				showLambdaDetail(appCtx, header, footer, functions[row-1])
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				appCtx.Stop()
				return nil
			case 'i':
				row, _ := table.GetSelection()
				if row > 0 && row <= len(functions) {
					invokeLambda(appCtx, functions[row-1])
				}
				return nil
			case 'r':
				loadLambdaFunctions(appCtx, table, &functions)
				return nil
			}
		}
		return event
	})

	// Create layout
	flex := tview.NewFlex()
	flex.SetBorder(true)
	flex.SetTitle(" Lambda Functions ")
	flex.SetBorderColor(tcell.ColorBlue)
	flex.AddItem(table, 0, 1, true)

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(flex, 0, 1, true).
		AddItem(footer, 1, 0, false)

	appCtx.Pages.AddPage("lambda-list", mainLayout, true, false)
	appCtx.Pages.SwitchToPage("lambda-list")
	appCtx.App.SetFocus(table)

	// Load functions
	loadLambdaFunctions(appCtx, table, &functions)
}

func loadLambdaFunctions(appCtx *AppContext, table *tview.Table, functionsPtr *[]*provider.Function) {
	// Clear existing rows except header
	for i := table.GetRowCount() - 1; i > 0; i-- {
		table.RemoveRow(i)
	}

	// Show loading
	table.SetCell(1, 0, tview.NewTableCell("Loading functions...").
		SetTextColor(tcell.ColorGray))

	// Load functions asynchronously
	go func() {
		functions, err := appCtx.Provider.Serverless().DiscoverFunctions(appCtx.Ctx)

		appCtx.App.QueueUpdateDraw(func() {
			// Clear loading message
			table.RemoveRow(1)

			if err != nil {
				table.SetCell(1, 0, tview.NewTableCell(fmt.Sprintf("Error: %v", err)).
					SetTextColor(tcell.ColorRed))
				return
			}

			if len(functions) == 0 {
				table.SetCell(1, 0, tview.NewTableCell("No Lambda functions found").
					SetTextColor(tcell.ColorGray))
				return
			}

			// Store functions for navigation
			*functionsPtr = functions

			// Add function rows
			for i, function := range functions {
				row := i + 1

				memory := fmt.Sprintf("%d MB", function.Memory)
				timeout := fmt.Sprintf("%d s", function.Timeout)

				table.SetCell(row, 0, tview.NewTableCell(function.Name).SetTextColor(tcell.ColorWhite))
				table.SetCell(row, 1, tview.NewTableCell(function.Runtime).SetTextColor(tcell.ColorBlue))
				table.SetCell(row, 2, tview.NewTableCell(memory).SetTextColor(tcell.ColorWhite))
				table.SetCell(row, 3, tview.NewTableCell(timeout).SetTextColor(tcell.ColorWhite))
			}
		})
	}()
}

func showLambdaDetail(appCtx *AppContext, header, footer *tview.TextView, function *provider.Function) {
	// Create detail view
	detailView := tview.NewTextView()
	detailView.SetDynamicColors(true)
	detailView.SetBorder(true)
	detailView.SetTitle(fmt.Sprintf(" Lambda: %s ", function.Name))
	detailView.SetBorderColor(tcell.ColorBlue)

	// Update footer
	footer.SetText("[yellow]i: Invoke[white] [gray]|[white] [yellow]l: Logs[white] [gray]|[white] [yellow]m: Metrics[white] [gray]|[white] [yellow]ESC: Back[white] [gray]|[white] [yellow]q: Quit[white]")

	// Build detail text
	var details strings.Builder
	details.WriteString(fmt.Sprintf("[yellow]Function Name:[white] %s\n", function.Name))
	details.WriteString(fmt.Sprintf("[yellow]Runtime:[white] %s\n", function.Runtime))
	details.WriteString(fmt.Sprintf("[yellow]Memory:[white] %d MB\n", function.Memory))
	details.WriteString(fmt.Sprintf("[yellow]Timeout:[white] %d seconds\n", function.Timeout))
	details.WriteString(fmt.Sprintf("[yellow]Handler:[white] %s\n", function.Handler))
	details.WriteString(fmt.Sprintf("[yellow]Created:[white] %s\n\n", function.CreatedAt.Format("2006-01-02 15:04:05")))

	if len(function.Environment) > 0 {
		details.WriteString("[yellow]Environment Variables:[white]\n")
		for key, value := range function.Environment {
			details.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
		details.WriteString("\n")
	}

	detailView.SetText(details.String())

	// Handle keyboard shortcuts
	detailView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			showLambdaList(appCtx, header, footer)
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				appCtx.Stop()
				return nil
			case 'i':
				invokeLambda(appCtx, function)
				return nil
			case 'l':
				showLambdaLogs(appCtx, header, footer, function)
				return nil
			}
		}
		return event
	})

	// Create layout
	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(detailView, 0, 1, true).
		AddItem(footer, 1, 0, false)

	appCtx.Pages.AddPage("lambda-detail", mainLayout, true, false)
	appCtx.Pages.SwitchToPage("lambda-detail")
	appCtx.App.SetFocus(detailView)
}

func invokeLambda(appCtx *AppContext, function *provider.Function) {
	// Create input form
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Invoke Lambda Function ")
	form.SetBorderColor(tcell.ColorBlue)

	payload := "{}"
	form.AddInputField("Payload (JSON):", payload, 50, nil, func(text string) {
		payload = text
	})

	form.AddButton("Invoke", func() {
		appCtx.Pages.RemovePage("lambda-invoke-form")

		// Create progress modal
		modal := tview.NewModal().
			SetText("Invoking function...").
			AddButtons([]string{"Cancel"})

		appCtx.Pages.AddPage("lambda-invoke-modal", modal, true, true)

		// Invoke in background
		go func() {
			result, err := appCtx.Provider.Serverless().InvokeFunction(appCtx.Ctx, function.Name, []byte(payload))

			appCtx.App.QueueUpdateDraw(func() {
				if err != nil {
					modal.SetText(fmt.Sprintf("Invocation failed:\n%v", err))
				} else {
					modal.SetText(fmt.Sprintf("Result:\n%s", string(result)))
				}

				modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					appCtx.Pages.RemovePage("lambda-invoke-modal")
				})
			})
		}()
	})

	form.AddButton("Cancel", func() {
		appCtx.Pages.RemovePage("lambda-invoke-form")
	})

	appCtx.Pages.AddPage("lambda-invoke-form", form, true, true)
}

func showLambdaLogs(appCtx *AppContext, header, footer *tview.TextView, function *provider.Function) {
	// Create logs view
	logsView := tview.NewTextView()
	logsView.SetDynamicColors(true)
	logsView.SetBorder(true)
	logsView.SetTitle(fmt.Sprintf(" Logs: %s ", function.Name))
	logsView.SetBorderColor(tcell.ColorBlue)
	logsView.SetText("[gray]Loading logs...[white]")
	logsView.SetScrollable(true)

	// Update footer
	footer.SetText("[yellow]ESC: Back[white] [gray]|[white] [yellow]q: Quit[white]")

	// Handle keyboard shortcuts
	logsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			showLambdaDetail(appCtx, header, footer, function)
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				appCtx.Stop()
				return nil
			}
		}
		return event
	})

	// Create layout
	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(logsView, 0, 1, true).
		AddItem(footer, 1, 0, false)

	appCtx.Pages.AddPage("lambda-logs", mainLayout, true, false)
	appCtx.Pages.SwitchToPage("lambda-logs")
	appCtx.App.SetFocus(logsView)

	// Load logs
	go func() {
		logs, err := appCtx.Provider.Logs().GetLambdaLogs(appCtx.Ctx, function.Name, 0, 0, 100)

		appCtx.App.QueueUpdateDraw(func() {
			var text strings.Builder
			text.WriteString(fmt.Sprintf("[yellow]Recent Logs for %s (Last 100 events)[white]\n\n", function.Name))

			if err != nil {
				text.WriteString(fmt.Sprintf("[red]Error: %v[white]\n", err))
			} else if len(logs) == 0 {
				text.WriteString("[gray]No logs found[white]\n")
			} else {
				for _, logEvent := range logs {
					text.WriteString(fmt.Sprintf("[gray]%s[white] %s\n", logEvent.Timestamp.Format("2006-01-02 15:04:05"), logEvent.Message))
				}
			}

			logsView.SetText(text.String())
		})
	}()
}

// DynamoDB TUI functions

func showDynamoDBList(appCtx *AppContext, header, footer *tview.TextView) {
	table := tview.NewTable()
	table.SetBorders(false)
	table.SetSelectable(true, false)
	table.SetFixed(1, 0)
	table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkCyan).
		Foreground(tcell.ColorWhite))

	// Add headers
	headers := []string{"Table Name", "Status", "Billing Mode", "Items", "Size", "Region"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)
		table.SetCell(0, i, cell)
	}

	// Update footer
	footer.SetText("[yellow]Up/Down: Navigate[white] [gray]|[white] [yellow]Enter: Details[white] [gray]|[white] [yellow]b: Browse Items[white] [gray]|[white] [yellow]d: Delete[white] [gray]|[white] [yellow]r: Refresh[white] [gray]|[white] [yellow]ESC: Back[white]")

	// Handle keyboard shortcuts
	var tables []*provider.DynamoDBTable
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			showDashboard(appCtx, header, footer)
			return nil
		case tcell.KeyEnter:
			row, _ := table.GetSelection()
			if row > 0 && row <= len(tables) {
				showDynamoDBDetail(appCtx, header, footer, tables[row-1])
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				appCtx.Stop()
				return nil
			case 'b':
				row, _ := table.GetSelection()
				if row > 0 && row <= len(tables) {
					showDynamoDBBrowser(appCtx, header, footer, tables[row-1])
				}
				return nil
			case 'd':
				row, _ := table.GetSelection()
				if row > 0 && row <= len(tables) {
					confirmDeleteDynamoDBTable(appCtx, header, footer, table, tables[row-1], &tables)
				}
				return nil
			case 'r':
				loadDynamoDBTables(appCtx, table, &tables)
				return nil
			}
		}
		return event
	})

	// Create layout
	flex := tview.NewFlex()
	flex.SetBorder(true)
	flex.SetTitle(" DynamoDB Tables ")
	flex.SetBorderColor(tcell.ColorBlue)
	flex.AddItem(table, 0, 1, true)

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(flex, 0, 1, true).
		AddItem(footer, 1, 0, false)

	appCtx.Pages.AddPage("dynamodb-list", mainLayout, true, false)
	appCtx.Pages.SwitchToPage("dynamodb-list")
	appCtx.App.SetFocus(table)

	// Load tables
	loadDynamoDBTables(appCtx, table, &tables)
}

func loadDynamoDBTables(appCtx *AppContext, table *tview.Table, tablesPtr *[]*provider.DynamoDBTable) {
	// Clear existing rows except header
	for i := table.GetRowCount() - 1; i > 0; i-- {
		table.RemoveRow(i)
	}

	// Show loading
	table.SetCell(1, 0, tview.NewTableCell("Loading tables...").
		SetTextColor(tcell.ColorGray))

	// Load tables asynchronously
	go func() {
		tables, err := appCtx.Provider.DynamoDB().ListTables(appCtx.Ctx)

		appCtx.App.QueueUpdateDraw(func() {
			// Clear loading message
			table.RemoveRow(1)

			if err != nil {
				table.SetCell(1, 0, tview.NewTableCell(fmt.Sprintf("Error: %v", err)).
					SetTextColor(tcell.ColorRed))
				return
			}

			*tablesPtr = tables

			if len(tables) == 0 {
				table.SetCell(1, 0, tview.NewTableCell("No DynamoDB tables found").
					SetTextColor(tcell.ColorGray))
				return
			}

			for i, t := range tables {
				row := i + 1

				// Format size
				size := formatDynamoDBBytes(t.TableSizeBytes)

				// Format billing mode
				billingMode := "On-Demand"
				if t.BillingMode == provider.BillingModeProvisioned {
					billingMode = "Provisioned"
				}

				// Status color
				statusColor := tcell.ColorGreen
				if t.Status != "ACTIVE" {
					statusColor = tcell.ColorYellow
				}

				table.SetCell(row, 0, tview.NewTableCell(t.Name).SetTextColor(tcell.ColorWhite))
				table.SetCell(row, 1, tview.NewTableCell(t.Status).SetTextColor(statusColor))
				table.SetCell(row, 2, tview.NewTableCell(billingMode).SetTextColor(tcell.ColorBlue))
				table.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%d", t.ItemCount)).SetTextColor(tcell.ColorWhite))
				table.SetCell(row, 4, tview.NewTableCell(size).SetTextColor(tcell.ColorWhite))
				table.SetCell(row, 5, tview.NewTableCell(t.Region).SetTextColor(tcell.ColorGray))
			}
		})
	}()
}

func formatDynamoDBBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func showDynamoDBDetail(appCtx *AppContext, header, footer *tview.TextView, dynamoTable *provider.DynamoDBTable) {
	detail := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	// Build detail content
	content := fmt.Sprintf(`[yellow]Table Name:[white] %s
[yellow]Status:[white] %s
[yellow]ARN:[white] %s
[yellow]Billing Mode:[white] %s
[yellow]Item Count:[white] %d
[yellow]Size:[white] %s
[yellow]Region:[white] %s
[yellow]Created:[white] %s

[yellow]Key Schema:[white]
`, dynamoTable.Name, dynamoTable.Status, dynamoTable.ARN, dynamoTable.BillingMode, dynamoTable.ItemCount,
		formatDynamoDBBytes(dynamoTable.TableSizeBytes), dynamoTable.Region, dynamoTable.CreatedAt.Format("2006-01-02 15:04:05"))

	for _, ks := range dynamoTable.KeySchema {
		content += fmt.Sprintf("  - %s (%s)\n", ks.AttributeName, ks.KeyType)
	}

	if dynamoTable.ProvisionedThroughput != nil {
		content += fmt.Sprintf(`
[yellow]Provisioned Throughput:[white]
  Read Capacity Units: %d
  Write Capacity Units: %d
`, dynamoTable.ProvisionedThroughput.ReadCapacityUnits, dynamoTable.ProvisionedThroughput.WriteCapacityUnits)
	}

	if dynamoTable.StreamEnabled {
		content += fmt.Sprintf(`
[yellow]Streams:[white] Enabled (%s)
`, dynamoTable.StreamViewType)
	}

	if len(dynamoTable.GlobalSecondaryIndexes) > 0 {
		content += "\n[yellow]Global Secondary Indexes:[white]\n"
		for _, gsi := range dynamoTable.GlobalSecondaryIndexes {
			content += fmt.Sprintf("  - %s (Status: %s)\n", gsi.IndexName, gsi.IndexStatus)
		}
	}

	detail.SetText(content)
	detail.SetBorder(true)
	detail.SetTitle(fmt.Sprintf(" Table: %s ", dynamoTable.Name))
	detail.SetBorderColor(tcell.ColorBlue)

	// Update footer
	footer.SetText("[yellow]Up/Down: Scroll[white] [gray]|[white] [yellow]b: Browse Items[white] [gray]|[white] [yellow]ESC: Back[white] [gray]|[white] [yellow]q: Quit[white]")

	// Handle input
	detail.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			appCtx.Pages.RemovePage("dynamodb-detail")
			showDynamoDBList(appCtx, header, footer)
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				appCtx.Stop()
				return nil
			case 'b':
				appCtx.Pages.RemovePage("dynamodb-detail")
				showDynamoDBBrowser(appCtx, header, footer, dynamoTable)
				return nil
			}
		}
		return event
	})

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(detail, 0, 1, true).
		AddItem(footer, 1, 0, false)

	appCtx.Pages.AddPage("dynamodb-detail", mainLayout, true, true)
	appCtx.App.SetFocus(detail)
}

func confirmDeleteDynamoDBTable(appCtx *AppContext, header, footer *tview.TextView, listTable *tview.Table, dynamoTable *provider.DynamoDBTable, tablesPtr *[]*provider.DynamoDBTable) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Are you sure you want to delete table '%s'?\n\nThis action cannot be undone.", dynamoTable.Name)).
		AddButtons([]string{"Cancel", "Delete"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Delete" {
				deleteDynamoDBTable(appCtx, listTable, dynamoTable, tablesPtr)
			}
			appCtx.Pages.RemovePage("delete-dynamodb-confirm")
			appCtx.App.SetFocus(listTable)
		})

	appCtx.Pages.AddPage("delete-dynamodb-confirm", modal, true, true)
}

func deleteDynamoDBTable(appCtx *AppContext, listTable *tview.Table, dynamoTable *provider.DynamoDBTable, tablesPtr *[]*provider.DynamoDBTable) {
	// Show deleting status
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Deleting table '%s'...", dynamoTable.Name))

	appCtx.Pages.AddPage("delete-dynamodb-status", modal, true, true)

	go func() {
		err := appCtx.Provider.DynamoDB().DeleteTable(appCtx.Ctx, dynamoTable.Name)

		appCtx.App.QueueUpdateDraw(func() {
			appCtx.Pages.RemovePage("delete-dynamodb-status")

			if err != nil {
				errorModal := tview.NewModal().
					SetText(fmt.Sprintf("Failed to delete table: %v", err)).
					AddButtons([]string{"OK"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						appCtx.Pages.RemovePage("delete-dynamodb-error")
						appCtx.App.SetFocus(listTable)
					})
				appCtx.Pages.AddPage("delete-dynamodb-error", errorModal, true, true)
				return
			}

			// Refresh the table list
			loadDynamoDBTables(appCtx, listTable, tablesPtr)
			appCtx.App.SetFocus(listTable)
		})
	}()
}

func showDynamoDBBrowser(appCtx *AppContext, header, footer *tview.TextView, dynamoTable *provider.DynamoDBTable) {
	table := tview.NewTable()
	table.SetBorders(false)
	table.SetSelectable(true, false)
	table.SetFixed(1, 0)
	table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkCyan).
		Foreground(tcell.ColorWhite))

	// Update footer
	footer.SetText("[yellow]Up/Down: Navigate[white] [gray]|[white] [yellow]Enter: View Item[white] [gray]|[white] [yellow]n: Next Page[white] [gray]|[white] [yellow]r: Refresh[white] [gray]|[white] [yellow]ESC: Back[white]")

	var items []provider.DynamoDBItem
	var lastEvaluatedKey map[string]interface{}
	pageSize := int64(25)

	// Handle keyboard shortcuts
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			appCtx.Pages.RemovePage("dynamodb-browser")
			showDynamoDBList(appCtx, header, footer)
			return nil
		case tcell.KeyEnter:
			row, _ := table.GetSelection()
			if row > 0 && row <= len(items) {
				showDynamoDBItemDetail(appCtx, table, items[row-1])
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				appCtx.Stop()
				return nil
			case 'r':
				loadDynamoDBItems(appCtx, table, dynamoTable, nil, pageSize, &items, &lastEvaluatedKey)
				return nil
			case 'n':
				if lastEvaluatedKey != nil {
					loadDynamoDBItems(appCtx, table, dynamoTable, lastEvaluatedKey, pageSize, &items, &lastEvaluatedKey)
				}
				return nil
			}
		}
		return event
	})

	// Create layout
	flex := tview.NewFlex()
	flex.SetBorder(true)
	flex.SetTitle(fmt.Sprintf(" Browse: %s ", dynamoTable.Name))
	flex.SetBorderColor(tcell.ColorBlue)
	flex.AddItem(table, 0, 1, true)

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(flex, 0, 1, true).
		AddItem(footer, 1, 0, false)

	appCtx.Pages.AddPage("dynamodb-browser", mainLayout, true, false)
	appCtx.Pages.SwitchToPage("dynamodb-browser")
	appCtx.App.SetFocus(table)

	// Load items
	loadDynamoDBItems(appCtx, table, dynamoTable, nil, pageSize, &items, &lastEvaluatedKey)
}

func loadDynamoDBItems(appCtx *AppContext, table *tview.Table, dynamoTable *provider.DynamoDBTable, startKey map[string]interface{}, pageSize int64, itemsPtr *[]provider.DynamoDBItem, lastKeyPtr *map[string]interface{}) {
	// Clear existing rows
	for i := table.GetRowCount() - 1; i >= 0; i-- {
		table.RemoveRow(i)
	}

	// Show loading
	table.SetCell(0, 0, tview.NewTableCell("Loading items...").
		SetTextColor(tcell.ColorGray))

	// Load items asynchronously
	go func() {
		result, err := appCtx.Provider.DynamoDB().ScanTable(appCtx.Ctx, dynamoTable.Name, pageSize, startKey)

		appCtx.App.QueueUpdateDraw(func() {
			// Clear table
			for i := table.GetRowCount() - 1; i >= 0; i-- {
				table.RemoveRow(i)
			}

			if err != nil {
				table.SetCell(0, 0, tview.NewTableCell(fmt.Sprintf("Error: %v", err)).
					SetTextColor(tcell.ColorRed))
				return
			}

			*itemsPtr = result.Items
			*lastKeyPtr = result.LastEvaluatedKey

			if len(result.Items) == 0 {
				table.SetCell(0, 0, tview.NewTableCell("No items found").
					SetTextColor(tcell.ColorGray))
				return
			}

			// Determine columns from key schema and first item
			columns := getDynamoDBColumns(dynamoTable, result.Items)

			// Add headers
			for i, col := range columns {
				cell := tview.NewTableCell(col).
					SetTextColor(tcell.ColorYellow).
					SetAlign(tview.AlignLeft).
					SetSelectable(false).
					SetAttributes(tcell.AttrBold)
				table.SetCell(0, i, cell)
			}

			// Add item rows
			for i, item := range result.Items {
				row := i + 1
				for j, col := range columns {
					value := ""
					if v, ok := item.Attributes[col]; ok {
						value = formatDynamoDBValue(v)
					}
					// Truncate long values
					if len(value) > 40 {
						value = value[:37] + "..."
					}
					table.SetCell(row, j, tview.NewTableCell(value).SetTextColor(tcell.ColorWhite))
				}
			}
		})
	}()
}

func getDynamoDBColumns(dynamoTable *provider.DynamoDBTable, items []provider.DynamoDBItem) []string {
	var columns []string

	// Start with key schema columns
	for _, ks := range dynamoTable.KeySchema {
		columns = append(columns, ks.AttributeName)
	}

	// Add other columns from first item
	if len(items) > 0 {
		for key := range items[0].Attributes {
			// Skip if already in columns
			found := false
			for _, col := range columns {
				if col == key {
					found = true
					break
				}
			}
			if !found {
				columns = append(columns, key)
			}
		}
	}

	// Limit to reasonable number of columns
	if len(columns) > 8 {
		columns = columns[:8]
	}

	return columns
}

func formatDynamoDBValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case nil:
		return "null"
	default:
		// For complex types, try to format nicely
		return fmt.Sprintf("%v", val)
	}
}

func showDynamoDBItemDetail(appCtx *AppContext, browserTable *tview.Table, item provider.DynamoDBItem) {
	detail := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	// Build content as formatted key-value pairs
	var content string
	for key, value := range item.Attributes {
		content += fmt.Sprintf("[yellow]%s:[white] %v\n", key, value)
	}

	detail.SetText(content)
	detail.SetBorder(true)
	detail.SetTitle(" Item Detail ")
	detail.SetBorderColor(tcell.ColorBlue)

	// Handle input
	detail.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			appCtx.Pages.RemovePage("dynamodb-item-detail")
			appCtx.App.SetFocus(browserTable)
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				appCtx.Stop()
				return nil
			}
		}
		return event
	})

	appCtx.Pages.AddPage("dynamodb-item-detail", detail, true, true)
	appCtx.App.SetFocus(detail)
}

func showDashboard(appCtx *AppContext, header, footer *tview.TextView) {
	footer.SetText("[yellow]↑↓: Navigate[white] [gray]|[white] [yellow]Enter: Select[white] [gray]|[white] [yellow]ESC: Back[white] [gray]|[white] [yellow]q: Quit[white]")
	appCtx.Pages.SwitchToPage("dashboard")
}

func createS3BrowserInteractive(appCtx *AppContext, bucketName string, footer *tview.TextView) *tview.Flex {
	// This is just a placeholder for the manage command
	// In practice, it would call showS3Browser but we need the header
	table := tview.NewTable()
	table.SetBorders(false)
	table.SetSelectable(true, false)

	// Add headers
	headers := []string{"Name", "Size", "Modified"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)
		table.SetCell(0, i, cell)
	}

	// Load objects
	go func() {
		objects, err := appCtx.Provider.Storage().ListObjects(appCtx.Ctx, bucketName, "", 1000)
		appCtx.App.QueueUpdateDraw(func() {
			if err != nil {
				table.SetCell(1, 0, tview.NewTableCell(fmt.Sprintf("Error: %v", err)).
					SetTextColor(tcell.ColorRed))
				return
			}

			row := 1
			for _, obj := range objects {
				if !obj.IsPrefix {
					size := formatBytes(obj.Size)
					modified := obj.LastModified.Format("2006-01-02 15:04")

					table.SetCell(row, 0, tview.NewTableCell(obj.Key).SetTextColor(tcell.ColorWhite))
					table.SetCell(row, 1, tview.NewTableCell(size).SetTextColor(tcell.ColorWhite))
					table.SetCell(row, 2, tview.NewTableCell(modified).SetTextColor(tcell.ColorGray))
					row++
				}
			}
		})
	}()

	flex := tview.NewFlex()
	flex.SetBorder(true)
	flex.SetTitle(fmt.Sprintf(" S3: %s ", bucketName))
	flex.SetBorderColor(tcell.ColorBlue)
	flex.AddItem(table, 0, 1, true)

	return flex
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
