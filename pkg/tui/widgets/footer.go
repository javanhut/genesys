package widgets

import (
	"fmt"

	"github.com/rivo/tview"
)

// Footer creates a footer widget with keyboard shortcuts
type Footer struct {
	*tview.TextView
	shortcuts []Shortcut
}

// Shortcut represents a keyboard shortcut
type Shortcut struct {
	Key         string
	Description string
}

// NewFooter creates a new footer widget
func NewFooter(shortcuts []Shortcut) *Footer {
	f := &Footer{
		TextView:  tview.NewTextView(),
		shortcuts: shortcuts,
	}

	f.SetDynamicColors(true)
	f.SetTextAlign(tview.AlignLeft)
	f.SetBorder(false)
	f.updateText()

	return f
}

// SetShortcuts updates the shortcuts displayed
func (f *Footer) SetShortcuts(shortcuts []Shortcut) {
	f.shortcuts = shortcuts
	f.updateText()
}

// updateText updates the footer text
func (f *Footer) updateText() {
	text := ""
	for i, sc := range f.shortcuts {
		if i > 0 {
			text += " [gray]|[white] "
		}
		text += fmt.Sprintf("[yellow]%s[white]: %s", sc.Key, sc.Description)
	}
	f.SetText(text)
}

// DefaultShortcuts returns the default global shortcuts
func DefaultShortcuts() []Shortcut {
	return []Shortcut{
		{Key: "↑↓", Description: "Navigate"},
		{Key: "Enter", Description: "Select"},
		{Key: "ESC", Description: "Back"},
		{Key: "q", Description: "Quit"},
	}
}
