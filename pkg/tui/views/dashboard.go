package views

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/javanhut/genesys/pkg/tui"
	"github.com/javanhut/genesys/pkg/tui/widgets"
	"github.com/rivo/tview"
)

// DashboardView represents the main dashboard view
type DashboardView struct {
	*tview.Flex
	appCtx *tui.AppContext
	list   *tview.List
}

// NewDashboardView creates a new dashboard view
func NewDashboardView(appCtx *tui.AppContext) *DashboardView {
	dv := &DashboardView{
		Flex:   tview.NewFlex(),
		appCtx: appCtx,
		list:   tview.NewList(),
	}

	dv.SetDirection(tview.FlexRow)
	dv.SetBorder(true)
	dv.SetTitle(" Dashboard ")
	dv.SetBorderColor(tcell.ColorBlue)

	dv.setupList()
	dv.loadResources()

	// Create layout
	dv.AddItem(dv.list, 0, 1, true)

	return dv
}

func (dv *DashboardView) setupList() {
	dv.list.SetBorder(false)
	dv.list.ShowSecondaryText(true)
	dv.list.SetHighlightFullLine(true)
	dv.list.SetSelectedBackgroundColor(tcell.ColorDarkCyan)
	dv.list.SetSelectedTextColor(tcell.ColorWhite)

	// Add menu items
	dv.list.AddItem("View All Resources", "Browse all discovered resources", '1', func() {
		dv.appCtx.NavigateTo("resources")
	})

	dv.list.AddItem("EC2 Instances", "Manage compute instances", '2', func() {
		dv.appCtx.NavigateTo("ec2-list")
	})

	dv.list.AddItem("S3 Buckets", "Browse storage buckets", '3', func() {
		dv.appCtx.NavigateTo("s3-list")
	})

	dv.list.AddItem("Lambda Functions", "Manage serverless functions", '4', func() {
		dv.appCtx.NavigateTo("lambda-list")
	})

	dv.list.AddItem("Monitor Resources", "View metrics and monitoring", '5', func() {
		dv.appCtx.NavigateTo("monitor")
	})

	dv.list.AddItem("Help", "View keyboard shortcuts and help", '?', func() {
		dv.showHelp()
	})

	dv.list.AddItem("Quit", "Exit the TUI", 'q', func() {
		dv.appCtx.Stop()
	})

	// Handle keyboard shortcuts
	dv.list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			dv.appCtx.Stop()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				dv.appCtx.Stop()
				return nil
			case 'r':
				dv.loadResources()
				return nil
			case '?':
				dv.showHelp()
				return nil
			}
		}
		return event
	})
}

func (dv *DashboardView) loadResources() {
	// Load resource counts asynchronously
	go func() {
		// Get resource counts
		compute, _ := dv.appCtx.Provider.Compute().DiscoverInstances(dv.appCtx.Ctx)
		storage, _ := dv.appCtx.Provider.Storage().DiscoverBuckets(dv.appCtx.Ctx)
		serverless, _ := dv.appCtx.Provider.Serverless().DiscoverFunctions(dv.appCtx.Ctx)

		// Update dashboard with counts
		dv.appCtx.App.QueueUpdateDraw(func() {
			dv.updateCounts(len(compute), len(storage), len(serverless))
		})
	}()
}

func (dv *DashboardView) updateCounts(compute, storage, serverless int) {
	// Update secondary text with resource counts
	dv.list.SetItemText(1, "EC2 Instances", fmt.Sprintf("%d instances available", compute))
	dv.list.SetItemText(2, "S3 Buckets", fmt.Sprintf("%d buckets available", storage))
	dv.list.SetItemText(3, "Lambda Functions", fmt.Sprintf("%d functions available", serverless))
}

func (dv *DashboardView) showHelp() {
	modal := tview.NewModal().
		SetText(`Keyboard Shortcuts:

Global:
  ↑/↓ or j/k - Navigate lists
  Enter      - Select item
  ESC        - Go back
  q          - Quit
  r          - Refresh
  ?          - Show this help

Resource Actions:
  m - View metrics
  l - View logs
  d - Details/Describe

Navigation:
  Tab - Switch panels
  / - Search/Filter`).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			dv.appCtx.Pages.RemovePage("help")
		})

	modal.SetBackgroundColor(tcell.ColorDefault)
	modal.SetBorderColor(tcell.ColorBlue)

	dv.appCtx.Pages.AddPage("help", modal, true, true)
}

// GetFooter returns the footer for this view
func (dv *DashboardView) GetFooter() *widgets.Footer {
	return widgets.NewFooter([]widgets.Shortcut{
		{Key: "1-5", Description: "Quick Select"},
		{Key: "Enter", Description: "Select"},
		{Key: "r", Description: "Refresh"},
		{Key: "?", Description: "Help"},
		{Key: "q", Description: "Quit"},
	})
}
