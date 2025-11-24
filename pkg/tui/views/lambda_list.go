package views

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/javanhut/genesys/pkg/tui"
	"github.com/javanhut/genesys/pkg/tui/widgets"
	"github.com/rivo/tview"
)

// LambdaListView shows a list of Lambda functions
type LambdaListView struct {
	*tview.Flex
	appCtx *tui.AppContext
	table  *tview.Table
}

// NewLambdaListView creates a new Lambda list view
func NewLambdaListView(appCtx *tui.AppContext) *LambdaListView {
	lv := &LambdaListView{
		Flex:   tview.NewFlex(),
		appCtx: appCtx,
		table:  tview.NewTable(),
	}

	lv.SetDirection(tview.FlexRow)
	lv.SetBorder(true)
	lv.SetTitle(" Lambda Functions ")
	lv.SetBorderColor(tcell.ColorBlue)

	lv.setupTable()
	lv.loadFunctions()

	lv.AddItem(lv.table, 0, 1, true)

	return lv
}

func (lv *LambdaListView) setupTable() {
	lv.table.SetBorders(false)
	lv.table.SetSelectable(true, false)
	lv.table.SetFixed(1, 0)
	lv.table.SetSelectedStyle(tcell.StyleDefault.
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
		lv.table.SetCell(0, i, cell)
	}

	// Handle keyboard shortcuts
	lv.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			lv.appCtx.NavigateBack()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				lv.appCtx.Stop()
				return nil
			case 'r':
				lv.loadFunctions()
				return nil
			case 'l':
				row, _ := lv.table.GetSelection()
				if row > 0 {
					// TODO: View logs
				}
				return nil
			}
		}
		return event
	})
}

func (lv *LambdaListView) loadFunctions() {
	// Clear existing rows
	for i := lv.table.GetRowCount() - 1; i > 0; i-- {
		lv.table.RemoveRow(i)
	}

	// Show loading
	lv.table.SetCell(1, 0, tview.NewTableCell("Loading functions...").
		SetTextColor(tcell.ColorGray))

	// Load functions asynchronously
	go func() {
		functions, err := lv.appCtx.Provider.Serverless().DiscoverFunctions(lv.appCtx.Ctx)

		lv.appCtx.App.QueueUpdateDraw(func() {
			// Clear loading message
			lv.table.RemoveRow(1)

			if err != nil {
				lv.table.SetCell(1, 0, tview.NewTableCell(fmt.Sprintf("Error: %v", err)).
					SetTextColor(tcell.ColorRed))
				return
			}

			if len(functions) == 0 {
				lv.table.SetCell(1, 0, tview.NewTableCell("No Lambda functions found").
					SetTextColor(tcell.ColorGray))
				return
			}

			// Add function rows
			for i, function := range functions {
				row := i + 1

				memory := fmt.Sprintf("%d MB", function.Memory)
				timeout := fmt.Sprintf("%d s", function.Timeout)

				lv.table.SetCell(row, 0, tview.NewTableCell(function.Name).SetTextColor(tcell.ColorWhite))
				lv.table.SetCell(row, 1, tview.NewTableCell(function.Runtime).SetTextColor(tcell.ColorBlue))
				lv.table.SetCell(row, 2, tview.NewTableCell(memory).SetTextColor(tcell.ColorWhite))
				lv.table.SetCell(row, 3, tview.NewTableCell(timeout).SetTextColor(tcell.ColorWhite))
			}
		})
	}()
}

// GetFooter returns the footer for this view
func (lv *LambdaListView) GetFooter() *widgets.Footer {
	return widgets.NewFooter([]widgets.Shortcut{
		{Key: "↑↓", Description: "Navigate"},
		{Key: "l", Description: "Logs"},
		{Key: "r", Description: "Refresh"},
		{Key: "ESC", Description: "Back"},
		{Key: "q", Description: "Quit"},
	})
}
