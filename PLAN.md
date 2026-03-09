# claudetop Iteration 1 — Build Plan

## Components

- [x] Task 1: Project scaffolding
- [x] Task 2: Config loading
- [x] Task 3: Session model
- [x] Task 4: tmux manager
- [x] Task 5: State store
- [x] Task 6: Status detector
- [x] Task 7: Status bar component
- [x] Task 8: Sidebar component
- [x] Task 9: Main viewport component
- [x] Task 10: Help overlay
- [x] Task 11: New session flow
- [x] Task 12: Main app model
- [x] Task 13: Main entry point
- [ ] Task 14: Integration testing (3+ concurrent sessions)
- [x] Task 15: Phase 4 review
- [x] Task 16: Phase 5 review
- [ ] Task 17: Phase 6 final + DONE.md

## Deviations from Spec

(record here as they occur)

- Stuck detection: if a session transitions to Done (silence, no question patterns) and then stops responding entirely, it will show Done rather than Stuck. This is a heuristic limitation. Stuck only fires if the current status is Working.
- Background session status: only the active session's pane content is polled. Background sessions retain their last-known status until they become the active session.

## Phase 4/5 Review fixes applied

- Dead sessions are now shown to user with [dead] badge and dimmed style instead of being silently dropped on reconnect.
- errMsg is now surfaced in the status bar as a warning (⚠) instead of silently dropped.
- Permission badge changed from ●! to ●[!] for clarity.
