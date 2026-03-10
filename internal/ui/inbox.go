package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"claudetop/internal/state"
)

var (
	inboxOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("220")).
				Background(lipgloss.Color("235")).
				Padding(1, 2)

	inboxTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true)

	inboxItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	inboxCursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")).
				Bold(true)

	inboxHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	inboxEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// activeInboxItems returns only non-parked items.
func activeInboxItems(items []*state.InboxItem) []*state.InboxItem {
	var active []*state.InboxItem
	for _, item := range items {
		if !item.Parked {
			active = append(active, item)
		}
	}
	return active
}

// renderInbox renders the inbox view overlay centered in the terminal.
func renderInbox(items []*state.InboxItem, cursor int, width, height int) string {
	active := activeInboxItems(items)

	// Compute inner content width (overlay will be ~80% of terminal width, min 50)
	overlayWidth := width * 4 / 5
	if overlayWidth < 50 {
		overlayWidth = 50
	}
	if overlayWidth > 100 {
		overlayWidth = 100
	}
	innerWidth := overlayWidth - 6 // border(2) + padding(4)

	var lines []string
	title := fmt.Sprintf("INBOX (%d items)", len(active))
	lines = append(lines, inboxTitleStyle.Width(innerWidth).Render(title))
	lines = append(lines, "")

	if len(active) == 0 {
		lines = append(lines, inboxEmptyStyle.Render("(inbox empty)"))
		lines = append(lines, "")
		lines = append(lines, inboxHintStyle.Render("c: capture   Esc: close"))
	} else {
		for i, item := range active {
			prefix := "  "
			lineStyle := inboxItemStyle
			if i == cursor {
				prefix = "▶ "
				lineStyle = inboxCursorStyle
			}
			// Truncate content to fit
			content := item.Content
			maxContent := innerWidth - 5 // room for "N  " prefix
			if len([]rune(content)) > maxContent {
				runes := []rune(content)
				content = string(runes[:maxContent-1]) + "…"
			}
			line := fmt.Sprintf("%s%d  %s", prefix, i+1, content)
			lines = append(lines, lineStyle.Width(innerWidth).Render(line))
		}
		lines = append(lines, "")
		lines = append(lines, inboxHintStyle.Render("j/k: navigate   s: start session   d: dismiss   p: park   Esc: close"))
	}

	content := inboxOverlayStyle.Width(overlayWidth).Render(strings.Join(lines, "\n"))

	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	leftPad := (width - contentWidth) / 2
	topPad := (height - contentHeight) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	if topPad < 0 {
		topPad = 0
	}

	var result []string
	for i := 0; i < topPad; i++ {
		result = append(result, strings.Repeat(" ", width))
	}
	for _, line := range strings.Split(content, "\n") {
		result = append(result, strings.Repeat(" ", leftPad)+line)
	}
	return strings.Join(result, "\n")
}
