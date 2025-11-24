package widgets

import (
	"fmt"

	"github.com/rivo/tview"
)

// Header creates a header widget
type Header struct {
	*tview.TextView
	title    string
	provider string
	region   string
}

// NewHeader creates a new header widget
func NewHeader(title, provider, region string) *Header {
	h := &Header{
		TextView: tview.NewTextView(),
		title:    title,
		provider: provider,
		region:   region,
	}

	h.SetDynamicColors(true)
	h.SetTextAlign(tview.AlignLeft)
	h.SetBorder(false)
	h.updateText()

	return h
}

// SetTitle updates the header title
func (h *Header) SetTitle(title string) {
	h.title = title
	h.updateText()
}

// updateText updates the header text
func (h *Header) updateText() {
	providerInfo := ""
	if h.provider != "" && h.region != "" {
		providerInfo = fmt.Sprintf(" - [blue::b]%s[white] ([yellow]%s[white])", h.provider, h.region)
	} else if h.provider != "" {
		providerInfo = fmt.Sprintf(" - [blue::b]%s[white]", h.provider)
	}

	text := fmt.Sprintf("[white::b]Genesys TUI[white] %s                  [gray]Press ? for help[white]", providerInfo)
	h.SetText(text)
}
