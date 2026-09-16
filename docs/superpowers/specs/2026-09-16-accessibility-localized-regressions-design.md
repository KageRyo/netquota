
# Accessibility and Localized UI Regression Design

## Goal

Make the tray dashboard and Settings flow measurably keyboard-friendly and
protect English, Traditional Chinese, and Japanese rendering from regressions,
while separating Fyne-automatable checks from desktop-only accessibility checks.

## Chosen approach

Extract Settings construction into a deterministic view builder that returns
the form and its input controls. The form keeps a documented order:
language, network interface, total quota, total alerts, download quota,
download alerts, upload quota, upload alerts, notifications, startup, then
Cancel/Save. Tests will render the view with Fyne's software driver and move
focus with Canvas.FocusNext, activate controls with keyboard events, and assert
that the form exposes both submit and cancel actions.

Fyne 2.8.1 exposes fyne.Accessible for labels and buttons but does not expose
the same interface on Select, Entry, or Check. Automated tests will verify
labels/buttons and record the control limitation explicitly; the manual
checklist will require a real screen reader to confirm how form labels are
announced. The application will not claim broader screen-reader support than
the toolkit provides.

The view will include a localized keyboard hint explaining Tab/Shift+Tab,
Enter/Space, and Escape behavior. Escape will return from Settings to the
dashboard, while Save and Cancel remain visible, focusable, and actionable.
The hint and visible labels make the available keyboard path discoverable
without adding platform-specific accelerators.

## Rendering and theme verification

Software-canvas tests will render the dashboard and Settings form at the
production window size for all three languages. They will assert that primary
controls have non-zero bounds, fit within the content width, and render
non-empty pixels. The same checks will run with Fyne light and dark themes;
focus color must remain distinct from the background. CJK font tests will
continue to assert visible glyph pixels for Traditional Chinese and Japanese.

A manual checklist will cover Linux/Windows keyboard-only completion,
screen-reader announcements, system high-contrast mode, dark/light theme,
focus visibility, dialog default/cancel actions, and clipped text at the
minimum production window size.

## Non-goals

- Do not invent a custom accessibility API outside Fyne.
- Do not claim automated screen-reader or OS high-contrast coverage.
- Do not change quota/accounting behavior or redesign the dashboard layout.
