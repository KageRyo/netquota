
# Accessibility and Localized UI Checklist

Use this checklist for a release candidate. Automated checks are evidence for
repeatable rendering and keyboard seams; they do not replace a real desktop
review.

## Automated checks

Run in an environment with the desktop development libraries used by CI:

~~~sh
gofmt -l .
go vet ./...
go test ./...
go build -trimpath ./cmd/netquota
~~~

The tray tests cover:

- Settings form order and focusability.
- Tab/Shift+Tab traversal, checkbox Space activation, and Escape handling.
- English, 正體中文, and 日本語 catalog completeness.
- Bundled CJK glyph visibility.
- Dashboard and Settings minimum widths at their production sizes.
- Fyne-supported Accessible labels/roles on text and buttons.
- Light/dark theme focus-color visibility.

Fyne 2.8.1 exposes fyne.Accessible on labels and buttons, but not on every
form control such as Select, Entry, and Check. Automated tests must not claim
that those controls are screen-reader-complete.

## Manual desktop checks

Run each item on a real Linux and Windows desktop when preparing a release.

### Keyboard-only flow

- [ ] Open the dashboard and reach Settings without a pointer.
- [ ] Move through language, interface, quota, threshold, notification, and
      startup controls with Tab.
- [ ] Move backwards with Shift+Tab and confirm the order is predictable.
- [ ] Change a Select using keyboard controls.
- [ ] Edit every quota and threshold Entry without a pointer.
- [ ] Toggle both Check controls with Space.
- [ ] Activate Save and Cancel with keyboard focus.
- [ ] Press Escape from Settings and confirm it returns to the dashboard.
- [ ] On interface changes, confirm the re-baseline dialog is reachable,
      understandable, and cancellable with Escape/Cancel.

### Screen reader

- [ ] Dashboard status, interface, sample time, metrics, and action buttons
      are announced in English.
- [ ] Repeat in 正體中文 and 日本語; confirm the selected language is spoken
      consistently.
- [ ] Settings labels are announced with their associated Select, Entry, and
      Check controls.
- [ ] Confirmation dialogs announce title, warning, confirm, and cancel
      actions.
- [ ] Monitoring errors and unavailable-interface recovery instructions are
      announced.
- [ ] Record any missing control semantics separately; Fyne's current
      Accessible API limitation is not silently treated as pass.

### Theme, contrast, and layout

- [ ] Test the operating system high-contrast setting on Windows.
- [ ] Test the desktop accessibility/theme settings on Linux.
- [ ] Verify light and dark themes keep focus visibly distinct.
- [ ] Resize to the production minimum dashboard size and confirm primary
      buttons, status, and metrics are not clipped or overlapped.
- [ ] Open Settings at its minimum usable width and inspect long translated
      labels and dialog buttons.
- [ ] Confirm 正體中文 and 日本語 show glyphs rather than missing-glyph boxes.
- [ ] Repeat the layout checks after changing language while Settings is open.

## Evidence

Record the OS, desktop environment, display scaling, selected language, and
any accessibility settings used. Attach screenshots or a short observation
for failures and link the issue or PR that addresses them.
