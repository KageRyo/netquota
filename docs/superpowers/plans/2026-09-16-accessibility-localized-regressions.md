
# Accessibility and Localized UI Regression Implementation Plan

> For agentic workers: REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Establish automated keyboard, focus, localization, font, layout, and theme regression coverage for the Fyne tray UI, plus a desktop manual accessibility checklist.

**Architecture:** Extract the Settings form into a deterministic view builder, expose a small keyboard hint/Escape path, and test the resulting widget tree through Fyne's software driver. Keep toolkit-supported accessibility assertions separate from manual screen-reader/high-contrast verification.

**Tech Stack:** Go 1.26, Fyne v2.8.1 software test driver, Fyne light/dark themes, embedded CJK fonts, JSON translation catalogs.

**Spec:** docs/superpowers/specs/2026-09-16-accessibility-localized-regressions-design.md

## Global Constraints

- Settings must be completable with keyboard focus traversal and Enter/Space actions.
- Every user-facing key must exist in English, Traditional Chinese, and Japanese.
- Automated tests must not claim screen-reader or OS high-contrast support that Fyne does not expose.
- Preserve the existing dashboard structure and accounting behavior.
- Keep Fyne GUI verification in the normal Linux/Windows CI matrix.
- Use Conventional Commits with no Co-authored-by trailer and no codex/agent branch prefix.

---

### Task 1: Extract a deterministic Settings view builder

**Files:**

- Modify: internal/tray/tray.go
- Modify: internal/tray/tray_test.go

**Interfaces:**

- Add a settingsView struct containing the Form, all input widgets, and the interface map used by readSettings.
- Add newSettingsView(translator, config, interfaces) that creates controls in the documented order without saving or changing monitor state.
- showSettings uses settingsView and remains responsible only for loading interfaces, save confirmation, startup configuration, and window navigation.

- [ ] Step 1: Write a failing form-order test.

Build a settingsView from a config and two interfaces, then assert the form item
labels are exactly:

~~~
Language
Network interface
Total quota (GiB)
Total alerts (%)
Download quota (GiB)
Download alerts (%)
Upload quota (GiB)
Upload alerts (%)
Notifications
Startup
~~~

Also assert every form item widget implements fyne.Focusable and the view has
non-nil Save and Cancel callbacks.

- [ ] Step 2: Run the focused test and verify RED.

Run:

~~~
go test ./internal/tray -run TestSettingsView -count=1
~~~

Expected: the settingsView type or builder is missing.

- [ ] Step 3: Implement the builder.

Move the existing widget creation and form.Append calls into newSettingsView.
Keep the current defaults, selected values, threshold parsing fields, and
unavailable-interface preservation. Add a translated keyboard hint as a
non-input label before the first form row; do not include the hint in the
focusable form item list. Set SubmitText and CancelText from the translator.

- [ ] Step 4: Add keyboard navigation hooks.

Set the form's OnSubmit and OnCancel from showSettings after the view is
created. Install a window canvas key handler while Settings is visible:
Escape invokes the same navigation as Cancel, and other keys are forwarded
to the previous handler when one exists. Restore the previous handler when
returning to the dashboard.

- [ ] Step 5: Run the focused tray tests.

Run:

~~~
gofmt -w internal/tray/tray.go internal/tray/tray_test.go
go test ./internal/tray -run TestSettingsView -count=1
~~~

Expected: the form-order and focusability tests pass in an environment with
Fyne's Linux desktop headers.

- [ ] Step 6: Commit the view seam.

~~~
git add internal/tray/tray.go internal/tray/tray_test.go
git commit -m "refactor: expose settings form for UI tests"
~~~

### Task 2: Verify keyboard-only interactions

**Files:**

- Modify: internal/tray/tray_test.go
- Modify: internal/tray/tray.go

**Interfaces:**

- Add test helpers that mount settingsView.form in a Fyne software window and call Canvas.FocusNext/FocusPrevious.
- Add keyboard action tests for form navigation, checkbox toggling, select activation, Save, Cancel, and Escape.

- [ ] Step 1: Write the failing interaction tests.

Mount the view in a software-test window at the production settings size.
Assert focus moves through the controls in form order, Shift+Tab moves
backwards, Space toggles both checkboxes, and the form buttons can be invoked
without a pointer. Test Escape calls the cancel callback and does not call the
save callback. Use real widget event methods rather than testing only callback
assignment.

- [ ] Step 2: Run the tests and verify RED.

Run:

~~~
go test ./internal/tray -run 'Test(SettingsKeyboard|SettingsFocus|SettingsEscape)' -count=1
~~~

Expected: at least Escape and the complete focus path fail before the keyboard
handler and test seam are present.

- [ ] Step 3: Implement the smallest keyboard behavior.

Use Fyne's existing focus and TypedKey behavior. Do not add a parallel
keyboard event system. Ensure the Cancel and Save buttons are included in the
canvas content after the form fields and that Escape uses the same cancel
function as the visible Cancel button.

- [ ] Step 4: Run the interaction tests and the existing tray tests.

Run:

~~~
go test ./internal/tray -count=1
~~~

Expected: all existing update, language, font, and tray-menu tests plus the new
keyboard tests pass.

- [ ] Step 5: Commit keyboard coverage.

~~~
git add internal/tray/tray.go internal/tray/tray_test.go
git commit -m "test: cover keyboard settings interactions"
~~~

### Task 3: Add supported accessibility and theme assertions

**Files:**

- Modify: internal/tray/tray_test.go
- Modify: internal/tray/language_theme.go
- Create: internal/tray/accessibility_test.go

**Interfaces:**

- Add a recursive test helper that collects fyne.Accessible objects from a view tree.
- Add a theme rendering helper that applies light/dark themes and restores the original theme after each test.

- [ ] Step 1: Write failing supported-semantics and focus-visibility tests.

Assert dashboard action buttons and labels implement fyne.Accessible with
button/text roles and localized labels. Assert the focused widget's theme
focus color differs from the background for both light and dark variants.
Render the dashboard and Settings form at 430 by 500 and assert primary
buttons have visible bounds and visible pixels for English, Traditional
Chinese, and Japanese.

- [ ] Step 2: Run the focused tests and verify RED.

Run:

~~~
go test ./internal/tray -run 'Test(Accessible|Theme|DashboardLayout|SettingsLayout)' -count=1
~~~

Expected: missing accessibility collection/layout assertions fail before the
new test helpers and any required layout adjustments.

- [ ] Step 3: Implement only required layout/theme seams.

Keep the current padded dashboard and scrollable Settings form. Add explicit
minimum sizes or wrapping only where a test demonstrates clipping at the
production window size. Preserve the localized font wrapper and ensure theme
changes do not replace the selected CJK font.

- [ ] Step 4: Run the full tray package.

Run:

~~~
go test ./internal/tray -count=1
~~~

Expected: all software-canvas rendering, focus, accessibility, and existing
tests pass.

- [ ] Step 5: Commit supported accessibility coverage.

~~~
git add internal/tray
git commit -m "test: add localized UI accessibility regressions"
~~~

### Task 4: Add manual desktop checklist and release gate

**Files:**

- Create: docs/accessibility-checklist.md
- Modify: docs/releasing.md
- Modify: CONTRIBUTING.md

- [ ] Step 1: Write the checklist with exact scenarios.

Include separate sections for automated checks and manual checks. Manual
scenarios must cover:

1. Keyboard-only Settings completion from dashboard to interface selection,
   each quota entry, both checkboxes, Save, Cancel, and Escape.
2. Focus visibility and order in English, Traditional Chinese, and Japanese.
3. Screen-reader announcement of dashboard labels, buttons, form labels,
   selects, entries, checks, confirmation dialogs, and error status.
4. Windows high-contrast mode and Linux desktop accessibility/theme settings.
5. Minimum window size, long translated labels, CJK glyphs, and clipped or
   overlapping primary controls.
6. Rebaseline confirmation default action and Escape/Cancel behavior.

Record that Fyne 2.8.1 exposes Accessible on labels/buttons but not all form
controls, so screen-reader observations remain a required manual gate.

- [ ] Step 2: Add the checklist to release documentation.

Require the checklist to be executed on a real Linux or Windows desktop for
release candidates, and state that automated CI does not replace the manual
screen-reader/high-contrast checks.

- [ ] Step 3: Add contributor commands.

Document the focused tray test command, full Go test command, and the desktop
dependency prerequisite already used by CI.

- [ ] Step 4: Render/grep the Markdown and validate catalog completeness.

Run:

~~~
go test ./internal/i18n -count=1
git diff --check
~~~

Review the rendered Markdown headings and numbered scenarios for clarity.

- [ ] Step 5: Commit the documentation.

~~~
git add docs/accessibility-checklist.md docs/releasing.md CONTRIBUTING.md
git commit -m "docs: add accessibility release checklist"
~~~

### Task 5: Verify the complete #11 branch

**Files:**

- All files changed by Tasks 1-4.

- [ ] Step 1: Run formatting and static checks.

Run:

~~~
gofmt -l .
go vet ./...
~~~

Expected: no formatting output and zero vet errors.

- [ ] Step 2: Run all tests and build.

Run:

~~~
go test ./...
go build -trimpath ./cmd/netquota
~~~

Expected: both Linux and Windows CI jobs pass. If the local machine lacks the
desktop headers, record that exact environmental limitation and use the CI
results as the authoritative GUI gate.

- [ ] Step 3: Audit branch naming and commit trailers.

Run:

~~~
git branch --show-current
git log --format='%H%n%B' origin/main..HEAD | rg -n -i 'co-authored-by|codex|agent' && exit 1 || true
git diff --check origin/main...HEAD
~~~

Expected: branch is test/accessibility-localized-regressions and no forbidden
trailer or branch prefix appears.

- [ ] Step 4: Commit any final verification-only documentation.

Use:

~~~
git add docs/accessibility-checklist.md docs/releasing.md CONTRIBUTING.md
git commit -m "docs: finalize accessibility verification gate"
~~~

only when the preceding verification found a documentation correction.
