package views

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/javanhut/genesys/pkg/tui"
	"github.com/javanhut/genesys/pkg/tui/widgets"
	"github.com/rivo/tview"
)

// S3ListView shows a list of S3 buckets
type S3ListView struct {
	*tview.Flex
	appCtx *tui.AppContext
	table  *tview.Table
}

// NewS3ListView creates a new S3 list view
func NewS3ListView(appCtx *tui.AppContext) *S3ListView {
	sv := &S3ListView{
		Flex:   tview.NewFlex(),
		appCtx: appCtx,
		table:  tview.NewTable(),
	}

	sv.SetDirection(tview.FlexRow)
	sv.SetBorder(true)
	sv.SetTitle(" S3 Buckets ")
	sv.SetBorderColor(tcell.ColorBlue)

	sv.setupTable()
	sv.loadBuckets()

	sv.AddItem(sv.table, 0, 1, true)

	return sv
}

func (sv *S3ListView) setupTable() {
	sv.table.SetBorders(false)
	sv.table.SetSelectable(true, false)
	sv.table.SetFixed(1, 0)
	sv.table.SetSelectedStyle(tcell.StyleDefault.
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
		sv.table.SetCell(0, i, cell)
	}

	// Handle keyboard shortcuts
	sv.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			sv.appCtx.NavigateBack()
			return nil
		case tcell.KeyEnter:
			row, _ := sv.table.GetSelection()
			if row > 0 {
				bucketName := sv.table.GetCell(row, 0).Text
				sv.appCtx.CurrentResourceType = "s3"
				sv.appCtx.CurrentResourceID = bucketName
				sv.appCtx.NavigateTo("s3-browser")
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				sv.appCtx.Stop()
				return nil
			case 'r':
				sv.loadBuckets()
				return nil
			}
		}
		return event
	})
}

func (sv *S3ListView) loadBuckets() {
	// Clear existing rows
	for i := sv.table.GetRowCount() - 1; i > 0; i-- {
		sv.table.RemoveRow(i)
	}

	// Show loading
	sv.table.SetCell(1, 0, tview.NewTableCell("Loading buckets...").
		SetTextColor(tcell.ColorGray))

	// Load buckets asynchronously
	go func() {
		buckets, err := sv.appCtx.Provider.Storage().DiscoverBuckets(sv.appCtx.Ctx)

		sv.appCtx.App.QueueUpdateDraw(func() {
			// Clear loading message
			sv.table.RemoveRow(1)

			if err != nil {
				sv.table.SetCell(1, 0, tview.NewTableCell(fmt.Sprintf("Error: %v", err)).
					SetTextColor(tcell.ColorRed))
				return
			}

			if len(buckets) == 0 {
				sv.table.SetCell(1, 0, tview.NewTableCell("No S3 buckets found").
					SetTextColor(tcell.ColorGray))
				return
			}

			// Add bucket rows
			for i, bucket := range buckets {
				row := i + 1

				sv.table.SetCell(row, 0, tview.NewTableCell(bucket.Name).SetTextColor(tcell.ColorWhite))
				sv.table.SetCell(row, 1, tview.NewTableCell(bucket.Region).SetTextColor(tcell.ColorBlue))
				sv.table.SetCell(row, 2, tview.NewTableCell(bucket.CreatedAt.Format("2006-01-02")).SetTextColor(tcell.ColorGray))
			}
		})
	}()
}

// GetFooter returns the footer for this view
func (sv *S3ListView) GetFooter() *widgets.Footer {
	return widgets.NewFooter([]widgets.Shortcut{
		{Key: "↑↓", Description: "Navigate"},
		{Key: "Enter", Description: "Browse"},
		{Key: "r", Description: "Refresh"},
		{Key: "ESC", Description: "Back"},
		{Key: "q", Description: "Quit"},
	})
}
