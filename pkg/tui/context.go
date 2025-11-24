package tui

import (
	"context"
	"sync"

	"github.com/javanhut/genesys/pkg/provider"
	"github.com/rivo/tview"
)

// AppContext holds shared state for the TUI application
type AppContext struct {
	App      *tview.Application
	Pages    *tview.Pages
	Provider provider.Provider
	Ctx      context.Context
	Cancel   context.CancelFunc

	// Navigation history
	history []string
	mu      sync.RWMutex

	// Current resource info
	CurrentResourceType string
	CurrentResourceID   string
}

// NewAppContext creates a new application context
func NewAppContext(ctx context.Context, p provider.Provider) *AppContext {
	appCtx, cancel := context.WithCancel(ctx)

	return &AppContext{
		App:      tview.NewApplication(),
		Pages:    tview.NewPages(),
		Provider: p,
		Ctx:      appCtx,
		Cancel:   cancel,
		history:  make([]string, 0),
	}
}

// PushPage adds a page to the navigation history
func (ac *AppContext) PushPage(name string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.history = append(ac.history, name)
}

// PopPage removes the last page from history and returns it
func (ac *AppContext) PopPage() string {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if len(ac.history) == 0 {
		return ""
	}

	last := ac.history[len(ac.history)-1]
	ac.history = ac.history[:len(ac.history)-1]
	return last
}

// PeekPage returns the current page without removing it
func (ac *AppContext) PeekPage() string {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	if len(ac.history) == 0 {
		return ""
	}

	return ac.history[len(ac.history)-1]
}

// NavigateBack goes back to the previous page
func (ac *AppContext) NavigateBack() {
	ac.PopPage()
	previous := ac.PeekPage()

	if previous == "" {
		previous = "dashboard"
	}

	ac.Pages.SwitchToPage(previous)
}

// NavigateTo navigates to a specific page
func (ac *AppContext) NavigateTo(pageName string) {
	ac.PushPage(pageName)
	ac.Pages.SwitchToPage(pageName)
}

// Stop stops the TUI application
func (ac *AppContext) Stop() {
	ac.Cancel()
	ac.App.Stop()
}
