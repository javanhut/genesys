package views

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/javanhut/genesys/pkg/provider"
	"github.com/javanhut/genesys/pkg/tui"
	"github.com/javanhut/genesys/pkg/tui/widgets"
	"github.com/rivo/tview"
)

// EC2ListView shows a list of EC2 instances
type EC2ListView struct {
	*tview.Flex
	appCtx    *tui.AppContext
	table     *tview.Table
	instances []*provider.Instance // Store instances for reference
}

// NewEC2ListView creates a new EC2 list view
func NewEC2ListView(appCtx *tui.AppContext) *EC2ListView {
	ev := &EC2ListView{
		Flex:   tview.NewFlex(),
		appCtx: appCtx,
		table:  tview.NewTable(),
	}

	ev.SetDirection(tview.FlexRow)
	ev.SetBorder(true)
	ev.SetTitle(" EC2 Instances ")
	ev.SetBorderColor(tcell.ColorBlue)

	ev.setupTable()
	ev.loadInstances()

	ev.AddItem(ev.table, 0, 1, true)

	return ev
}

func (ev *EC2ListView) setupTable() {
	ev.table.SetBorders(false)
	ev.table.SetSelectable(true, false)
	ev.table.SetFixed(1, 0)
	ev.table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkCyan).
		Foreground(tcell.ColorWhite))

	// Add headers
	headers := []string{"Instance ID", "Name", "Region", "State", "Type", "IP Address"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)
		ev.table.SetCell(0, i, cell)
	}

	// Handle keyboard shortcuts
	ev.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			ev.appCtx.NavigateBack()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				ev.appCtx.Stop()
				return nil
			case 'r':
				ev.loadInstances()
				return nil
			case 'm':
				row, _ := ev.table.GetSelection()
				if row > 0 {
					// TODO: Navigate to metrics view
				}
				return nil
			case 'c':
				// SSH Connect
				row, _ := ev.table.GetSelection()
				if row > 0 && row-1 < len(ev.instances) {
					instance := ev.instances[row-1]
					tui.ShowSSHDialog(ev.appCtx, instance)
				}
				return nil
			case 's':
				// Security Group SSH Rules
				row, _ := ev.table.GetSelection()
				if row > 0 && row-1 < len(ev.instances) {
					instance := ev.instances[row-1]
					tui.ShowAddSSHRuleDialog(ev.appCtx, instance, nil)
				}
				return nil
			}
		}
		return event
	})
}

func (ev *EC2ListView) loadInstances() {
	// Clear existing rows
	for i := ev.table.GetRowCount() - 1; i > 0; i-- {
		ev.table.RemoveRow(i)
	}

	// Clear stored instances
	ev.instances = nil

	// Show loading message - scanning all regions takes time
	ev.table.SetCell(1, 0, tview.NewTableCell("Scanning all AWS regions for instances...").
		SetTextColor(tcell.ColorGray))

	// Load instances asynchronously
	go func() {
		instances, err := ev.appCtx.Provider.Compute().DiscoverInstances(ev.appCtx.Ctx)

		ev.appCtx.App.QueueUpdateDraw(func() {
			// Clear loading message
			ev.table.RemoveRow(1)

			if err != nil {
				ev.table.SetCell(1, 0, tview.NewTableCell(fmt.Sprintf("Error: %v", err)).
					SetTextColor(tcell.ColorRed))
				return
			}

			if len(instances) == 0 {
				ev.table.SetCell(1, 0, tview.NewTableCell("No EC2 instances found").
					SetTextColor(tcell.ColorGray))
				return
			}

			// Store instances for reference (used by SSH connect)
			ev.instances = instances

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

				ev.table.SetCell(row, 0, tview.NewTableCell(instance.ID).SetTextColor(tcell.ColorWhite))
				ev.table.SetCell(row, 1, tview.NewTableCell(instance.Name).SetTextColor(tcell.ColorBlue))
				ev.table.SetCell(row, 2, tview.NewTableCell(region).SetTextColor(tcell.ColorYellow))
				ev.table.SetCell(row, 3, tview.NewTableCell(instance.State).SetTextColor(stateColor))
				ev.table.SetCell(row, 4, tview.NewTableCell(string(instance.Type)).SetTextColor(tcell.ColorWhite))
				ev.table.SetCell(row, 5, tview.NewTableCell(displayIP).SetTextColor(tcell.ColorWhite))
			}
		})
	}()
}

// GetFooter returns the footer for this view
func (ev *EC2ListView) GetFooter() *widgets.Footer {
	return widgets.NewFooter([]widgets.Shortcut{
		{Key: "↑↓", Description: "Navigate"},
		{Key: "c", Description: "SSH Connect"},
		{Key: "s", Description: "SSH Rules"},
		{Key: "m", Description: "Metrics"},
		{Key: "r", Description: "Refresh"},
		{Key: "ESC", Description: "Back"},
		{Key: "q", Description: "Quit"},
	})
}
