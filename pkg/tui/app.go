package tui

import (
	"context"
	"fmt"
	"strings"

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

	// Add headers
	headers := []string{"Instance ID", "Name", "State", "Type", "IP Address"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)
		table.SetCell(0, i, cell)
	}

	// Update footer
	footer.SetText("[yellow]↑↓: Navigate[white] [gray]|[white] [yellow]Enter: Details[white] [gray]|[white] [yellow]m: Metrics[white] [gray]|[white] [yellow]ESC: Back[white] [gray]|[white] [yellow]q: Quit[white]")

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

	// Show loading
	table.SetCell(1, 0, tview.NewTableCell("Loading instances...").
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
				table.SetCell(1, 0, tview.NewTableCell("No EC2 instances found").
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

				table.SetCell(row, 0, tview.NewTableCell(instance.ID).SetTextColor(tcell.ColorWhite))
				table.SetCell(row, 1, tview.NewTableCell(instance.Name).SetTextColor(tcell.ColorBlue))
				table.SetCell(row, 2, tview.NewTableCell(instance.State).SetTextColor(stateColor))
				table.SetCell(row, 3, tview.NewTableCell(string(instance.Type)).SetTextColor(tcell.ColorWhite))
				table.SetCell(row, 4, tview.NewTableCell(instance.PublicIP).SetTextColor(tcell.ColorWhite))
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
	footer.SetText("[yellow]↑↓: Navigate[white] [gray]|[white] [yellow]Enter: Open[white] [gray]|[white] [yellow]Backspace: Up[white] [gray]|[white] [yellow]d: Download[white] [gray]|[white] [yellow]ESC: Back[white]")

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
