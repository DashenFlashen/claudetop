package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"claudetop/internal/session"
)

var (
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("250"))

	statusBarBold = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("255")).
			Bold(true)

	needsInputStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("196"))
)

// renderStatusBar returns the top status line string.
func renderStatusBar(sessions []*session.Session, inboxCount int, width int, statusMsg string) string {
	total := len(sessions)
	needsInput := 0
	for _, s := range sessions {
		if s.Status == session.StatusNeedsInput || s.Status == session.StatusPermission {
			needsInput++
		}
	}

	left := statusBarBold.Render("claudetop")
	left += statusBarStyle.Render(fmt.Sprintf("  %d sessions", total))
	if needsInput > 0 {
		left += "  " + needsInputStyle.Render(fmt.Sprintf("● %d needs input", needsInput))
	}

	inboxBadgeStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("214"))

	if inboxCount > 0 {
		left += "  " + inboxBadgeStyle.Render(fmt.Sprintf("[INBOX: %d]", inboxCount))
	}

	if statusMsg != "" {
		left += "  " + lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("196")).
			Render("⚠ "+statusMsg)
	}

	clock := statusBarStyle.Render(time.Now().Format("15:04"))

	leftLen := lipgloss.Width(left)
	clockLen := lipgloss.Width(clock)
	padding := width - leftLen - clockLen
	if padding < 1 {
		padding = 1
	}

	pad := statusBarStyle.Render(strings.Repeat(" ", padding))
	return statusBarStyle.Width(width).Render(left + pad + clock)
}
