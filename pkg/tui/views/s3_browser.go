package views

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/javanhut/genesys/pkg/tui"
	"github.com/javanhut/genesys/pkg/tui/widgets"
	"github.com/rivo/tview"
)

// S3BrowserView shows a file browser for an S3 bucket
type S3BrowserView struct {
	*tview.Flex
	appCtx        *tui.AppContext
	table         *tview.Table
	bucketName    string
	currentPrefix string
}

// NewS3BrowserView creates a new S3 browser view
func NewS3BrowserView(appCtx *tui.AppContext, bucketName string) *S3BrowserView {
	sv := &S3BrowserView{
		Flex:          tview.NewFlex(),
		appCtx:        appCtx,
		table:         tview.NewTable(),
		bucketName:    bucketName,
		currentPrefix: "",
	}

	sv.SetDirection(tview.FlexRow)
	sv.SetBorder(true)
	sv.updateTitle()
	sv.SetBorderColor(tcell.ColorBlue)

	sv.setupTable()
	sv.loadObjects()

	sv.AddItem(sv.table, 0, 1, true)

	return sv
}

func (sv *S3BrowserView) updateTitle() {
	title := fmt.Sprintf(" S3: %s", sv.bucketName)
	if sv.currentPrefix != "" {
		title += fmt.Sprintf(" / %s", sv.currentPrefix)
	}
	title += " "
	sv.SetTitle(title)
}

func (sv *S3BrowserView) setupTable() {
	sv.table.SetBorders(false)
	sv.table.SetSelectable(true, false)
	sv.table.SetFixed(1, 0)
	sv.table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDarkCyan).
		Foreground(tcell.ColorWhite))

	// Add headers
	headers := []string{"Name", "Size", "Modified"}
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
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			sv.navigateUp()
			return nil
		case tcell.KeyEnter:
			row, _ := sv.table.GetSelection()
			if row > 0 {
				// Get the key from the first column
				key := sv.table.GetCell(row, 0).Text
				if strings.HasPrefix(key, "📁 ") {
					// Navigate into folder
					folder := strings.TrimPrefix(key, "📁 ")
					sv.currentPrefix += folder
					sv.updateTitle()
					sv.loadObjects()
				}
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				sv.appCtx.Stop()
				return nil
			case 'r':
				sv.loadObjects()
				return nil
			}
		}
		return event
	})
}

func (sv *S3BrowserView) navigateUp() {
	if sv.currentPrefix == "" {
		return
	}

	// Remove the last folder from the prefix
	parts := strings.Split(strings.TrimSuffix(sv.currentPrefix, "/"), "/")
	if len(parts) > 1 {
		sv.currentPrefix = strings.Join(parts[:len(parts)-1], "/") + "/"
	} else {
		sv.currentPrefix = ""
	}

	sv.updateTitle()
	sv.loadObjects()
}

func (sv *S3BrowserView) loadObjects() {
	// Clear existing rows
	for i := sv.table.GetRowCount() - 1; i > 0; i-- {
		sv.table.RemoveRow(i)
	}

	// Show loading
	sv.table.SetCell(1, 0, tview.NewTableCell("Loading objects...").
		SetTextColor(tcell.ColorGray))

	// Load objects asynchronously
	go func() {
		objects, err := sv.appCtx.Provider.Storage().ListObjects(sv.appCtx.Ctx, sv.bucketName, sv.currentPrefix, 1000)

		sv.appCtx.App.QueueUpdateDraw(func() {
			// Clear loading message
			sv.table.RemoveRow(1)

			if err != nil {
				sv.table.SetCell(1, 0, tview.NewTableCell(fmt.Sprintf("Error: %v", err)).
					SetTextColor(tcell.ColorRed))
				return
			}

			if len(objects) == 0 {
				sv.table.SetCell(1, 0, tview.NewTableCell("No objects found").
					SetTextColor(tcell.ColorGray))
				return
			}

			// Separate folders and files
			row := 1

			// Add folders first
			for _, obj := range objects {
				if obj.IsPrefix {
					name := strings.TrimPrefix(obj.Key, sv.currentPrefix)
					sv.table.SetCell(row, 0, tview.NewTableCell("📁 "+name).SetTextColor(tcell.ColorBlue))
					sv.table.SetCell(row, 1, tview.NewTableCell("-").SetTextColor(tcell.ColorGray))
					sv.table.SetCell(row, 2, tview.NewTableCell("-").SetTextColor(tcell.ColorGray))
					row++
				}
			}

			// Add files
			for _, obj := range objects {
				if !obj.IsPrefix {
					name := strings.TrimPrefix(obj.Key, sv.currentPrefix)
					size := formatBytes(obj.Size)
					modified := obj.LastModified.Format("2006-01-02 15:04")

					sv.table.SetCell(row, 0, tview.NewTableCell("📄 "+name).SetTextColor(tcell.ColorWhite))
					sv.table.SetCell(row, 1, tview.NewTableCell(size).SetTextColor(tcell.ColorWhite))
					sv.table.SetCell(row, 2, tview.NewTableCell(modified).SetTextColor(tcell.ColorGray))
					row++
				}
			}
		})
	}()
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

// GetFooter returns the footer for this view
func (sv *S3BrowserView) GetFooter() *widgets.Footer {
	return widgets.NewFooter([]widgets.Shortcut{
		{Key: "↑↓", Description: "Navigate"},
		{Key: "Enter", Description: "Open"},
		{Key: "Backspace", Description: "Up"},
		{Key: "r", Description: "Refresh"},
		{Key: "ESC", Description: "Back"},
	})
}
