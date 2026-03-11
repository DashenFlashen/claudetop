package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"claudetop/internal/git"
	"claudetop/internal/session"
	"claudetop/internal/state"
)

var briefingSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

var (
	briefingBg = lipgloss.Color("232")

	briefingHeaderStyle = lipgloss.NewStyle().
				Background(briefingBg).
				Foreground(lipgloss.Color("255")).
				Bold(true).
				Padding(0, 2)

	briefingSectionStyle = lipgloss.NewStyle().
				Background(briefingBg).
				Foreground(lipgloss.Color("39")).
				Bold(true).
				Padding(0, 2)

	briefingDividerStyle = lipgloss.NewStyle().
				Background(briefingBg).
				Foreground(lipgloss.Color("238"))

	briefingTextStyle = lipgloss.NewStyle().
				Background(briefingBg).
				Foreground(lipgloss.Color("250")).
				Padding(0, 2)

	briefingDimStyle = lipgloss.NewStyle().
				Background(briefingBg).
				Foreground(lipgloss.Color("240")).
				Padding(0, 2)

	briefingHintStyle = lipgloss.NewStyle().
				Background(briefingBg).
				Foreground(lipgloss.Color("240")).
				Padding(0, 2)

	briefingPriorityLabelStyle = lipgloss.NewStyle().
					Background(briefingBg).
					Foreground(lipgloss.Color("220")).
					Bold(true).
					Padding(0, 2)

	briefingInputStyle = lipgloss.NewStyle().
				Background(briefingBg).
				Padding(0, 2)

	briefingDividerLineStyle = lipgloss.NewStyle().
					Background(briefingBg).
					Foreground(lipgloss.Color("236"))
)

// renderBriefing renders the full-screen morning briefing.
func renderBriefing(
	commits []git.CommitSummary,
	commitsLoading bool,
	inboxItems []*state.InboxItem,
	parkedSessions []*session.Session,
	standupOutput string,
	standupRunning bool,
	prioritiesInput textinput.Model,
	prioritiesFocused bool,
	scrollOffset int,
	width, height, tick int,
) string {
	bg := lipgloss.NewStyle().Background(briefingBg).Width(width)

	// Build all content lines (scrollable region)
	var lines []string

	// Header
	lines = append(lines, "")
	day := time.Now().Format("Monday 2 January")
	lines = append(lines, briefingHeaderStyle.Width(width).Render("GOOD MORNING  ·  "+day))
	lines = append(lines, "")

	// YESTERDAY section
	lines = append(lines, briefingSectionStyle.Width(width).Render("YESTERDAY"))
	lines = append(lines, briefingDividerStyle.Width(width).Render(strings.Repeat("─", width-4)))
	if commitsLoading {
		lines = append(lines, briefingDimStyle.Width(width).Render(briefingSpinnerFrames[tick%len(briefingSpinnerFrames)]+"  Scanning repos..."))
	} else if len(commits) == 0 {
		lines = append(lines, briefingDimStyle.Width(width).Render("(no commits yesterday)"))
	} else {
		maxCommits := 10
		shown := commits
		if len(shown) > maxCommits {
			shown = shown[:maxCommits]
		}
		for _, c := range shown {
			line := fmt.Sprintf("%-20s · %s", c.Repo, c.Message)
			if len([]rune(line)) > width-6 {
				runes := []rune(line)
				line = string(runes[:width-9]) + "..."
			}
			lines = append(lines, briefingTextStyle.Width(width).Render(line))
		}
		if len(commits) > maxCommits {
			lines = append(lines, briefingDimStyle.Width(width).Render(fmt.Sprintf("+ %d more", len(commits)-maxCommits)))
		}
	}
	lines = append(lines, "")

	// INBOX section (only if non-empty)
	active := activeInboxItems(inboxItems)
	if len(active) > 0 {
		lines = append(lines, briefingSectionStyle.Width(width).Render(fmt.Sprintf("INBOX  %d items", len(active))))
		lines = append(lines, briefingDividerStyle.Width(width).Render(strings.Repeat("─", width-4)))
		maxShow := 2
		for i, item := range active {
			if i >= maxShow {
				break
			}
			content := item.Content
			if len([]rune(content)) > width-10 {
				runes := []rune(content)
				content = string(runes[:width-13]) + "…"
			}
			src := ""
			if item.Source != "manual" {
				src = "  — " + item.Source
			}
			lines = append(lines, briefingTextStyle.Width(width).Render("· "+content+src))
		}
		if len(active) > maxShow {
			lines = append(lines, briefingDimStyle.Width(width).Render(fmt.Sprintf("+ %d more  ·  b to open inbox", len(active)-maxShow)))
		} else {
			lines = append(lines, briefingDimStyle.Width(width).Render("b to open inbox"))
		}
		lines = append(lines, "")
	}

	// PARKED section (only if non-empty)
	if len(parkedSessions) > 0 {
		lines = append(lines, briefingSectionStyle.Width(width).Render("PARKED"))
		lines = append(lines, briefingDividerStyle.Width(width).Render(strings.Repeat("─", width-4)))
		for _, s := range parkedSessions {
			note := ""
			if s.ParkNote != "" {
				note = "  \u201c" + s.ParkNote + "\u201d"
			}
			lines = append(lines, briefingTextStyle.Width(width).Render("· "+s.DisplayName()+note))
		}
		lines = append(lines, "")
	}

	// STANDUP DRAFT section
	standupHint := "s to generate"
	if standupRunning {
		standupHint = "generating..."
	} else if standupOutput != "" {
		standupHint = "s to regenerate"
	}
	header := fmt.Sprintf("%-*s%s", width/2, "STANDUP DRAFT", standupHint)
	lines = append(lines, briefingSectionStyle.Width(width).Render(header))
	lines = append(lines, briefingDividerStyle.Width(width).Render(strings.Repeat("─", width-4)))
	if standupRunning {
		lines = append(lines, briefingDimStyle.Width(width).Render(briefingSpinnerFrames[tick%len(briefingSpinnerFrames)]+"  Generating standup draft..."))
	} else if standupOutput == "" {
		lines = append(lines, briefingDimStyle.Width(width).Render("(not yet generated)"))
	} else {
		for _, l := range strings.Split(strings.TrimSpace(standupOutput), "\n") {
			lines = append(lines, briefingTextStyle.Width(width).Render(l))
		}
	}
	lines = append(lines, "")

	// Apply scroll offset and clip to content area height
	contentAreaHeight := height - 4 // reserve 4 lines for bottom section
	if contentAreaHeight < 1 {
		contentAreaHeight = 1
	}
	if scrollOffset > len(lines)-1 {
		scrollOffset = len(lines) - 1
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}
	visible := lines[scrollOffset:]
	if len(visible) > contentAreaHeight {
		visible = visible[:contentAreaHeight]
	}
	// Pad remaining lines with blank styled lines
	for len(visible) < contentAreaHeight {
		visible = append(visible, bg.Render(""))
	}

	// Bottom section (fixed, always visible)
	divider := briefingDividerLineStyle.Width(width).Render(strings.Repeat("─", width))
	label := briefingPriorityLabelStyle.Width(width).Render("TODAY'S PRIORITIES")
	inputPrefix := lipgloss.NewStyle().Background(briefingBg).Foreground(lipgloss.Color("240")).Render("> ")
	if prioritiesFocused {
		inputPrefix = lipgloss.NewStyle().Background(briefingBg).Foreground(lipgloss.Color("220")).Bold(true).Render("> ")
	}
	inputLine := briefingInputStyle.Width(width).Render(inputPrefix + prioritiesInput.View())
	hint := "Tab: focus · Enter: save · Esc: skip · j/k: scroll · s: standup · b: inbox"
	hintLine := briefingHintStyle.Width(width).Render(hint)

	return strings.Join(append(visible, divider, label, inputLine, hintLine), "\n")
}

// newBriefingPrioritiesInput returns a configured textinput for the briefing priorities field.
func newBriefingPrioritiesInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "e.g. Resolve annotation timeout, move CSV export forward"
	ti.CharLimit = 300
	ti.Width = 70
	return ti
}
