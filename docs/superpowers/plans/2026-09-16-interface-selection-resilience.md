
# Interface Selection Resilience Implementation Plan

> For agentic workers: REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Prevent NetQuota from silently switching accounting to an unrelated network interface while preferring the active default route for an unconfigured monitor.

**Architecture:** Enrich internal/network.Interface with platform identity, IPv6, and default-route metadata. Keep selection policy in internal/network, make internal/app distinguish unavailable selections from provider failures, and make the tray require explicit confirmation before changing the tracked interface.

**Tech Stack:** Go 1.26, gopsutil/v4, net.Interface, Linux /proc route tables, Windows GetIpForwardTable2, Fyne v2, embedded JSON translation catalogs.

**Spec:** docs/superpowers/specs/2026-09-16-interface-selection-resilience-design.md

## Global Constraints

- Preserve existing configuration JSON fields and decode older files without migration failure.
- Never fall back from a non-empty saved selection to an unrelated interface.
- Treat IPv4 and IPv6 as valid interface addresses; IPv4 is not required.
- Only an explicit Settings save of a different interface may reset the usage baseline.
- Add every user-facing key to English, Traditional Chinese, and Japanese catalogs.
- Do not add packet capture, remote services, or unrelated UI redesign.
- Use Conventional Commits with no Co-authored-by trailer.

---

### Task 1: Define identity and selection policy with failing tests

**Files:**

- Modify: internal/model/model.go
- Modify: internal/network/network.go
- Modify: internal/network/network_test.go

**Interfaces:**

- model.InterfaceSelection gains Index int and IPv6 string JSON fields.
- network.Interface gains Index int, IPv6 string, and DefaultRoute bool.
- network.Select(selection, interfaces) retains its signature and returns a typed ErrSelectedInterfaceUnavailable for a non-empty selection that cannot be resolved.
- Add network.SelectionForInterface(network.Interface) model.InterfaceSelection and an identity comparison helper for the monitor.

- [ ] Step 1: Write the failing selection tests.

Add tests for default-route preference, IPv6-only selection, and no fallback
for an unavailable saved selection. Also add a stable-index test showing that
a renamed interface with the same index resolves, while a same-name interface
with changed hardware identity does not.

Example test shape:

~~~
func TestSelectDoesNotFallbackForUnavailableSavedSelection(t *testing.T) {
    _, err := Select(model.InterfaceSelection{
        Name: "Wi-Fi", HardwareAddress: "aa:bb:cc:dd:ee:ff",
    }, []Interface{{Name: "Ethernet", IPv4: "192.0.2.20"}})
    if !errors.Is(err, ErrSelectedInterfaceUnavailable) {
        t.Fatalf("Select error = %v, want ErrSelectedInterfaceUnavailable", err)
    }
}
~~~

- [ ] Step 2: Run the focused tests and verify RED.

Run:

~~~
go test ./internal/network -run 'TestSelectPrefersDefaultRoute|TestSelectSupportsIPv6Only|TestSelectDoesNotFallback|TestSelect.*Identity' -count=1
~~~

Expected: compilation or assertion failures because the new fields, error, and
route-aware policy do not exist yet.

- [ ] Step 3: Implement the minimal model and policy.

Resolve a persisted identity by matching non-zero index first, then
case-insensitive hardware address, then a legacy name-only selection when no
stronger identity was persisted. For an empty selection choose default-route,
then addressed non-loopback, then non-loopback, then the first interface.
Return ErrSelectedInterfaceUnavailable instead of selecting a different
interface when a saved identity cannot be resolved.

- [ ] Step 4: Run the focused tests and network package.

Run:

~~~
gofmt -w internal/model/model.go internal/network/network.go internal/network/network_test.go
go test ./internal/network -count=1
~~~

Expected: all network tests pass, including the existing IPv4 and fallback
tests.

- [ ] Step 5: Commit the policy unit.

~~~
git add internal/model/model.go internal/network/network.go internal/network/network_test.go
git commit -m "fix: make interface selection identity aware"
~~~

### Task 2: Discover default routes on supported platforms

**Files:**

- Modify: internal/network/network.go
- Create: internal/network/default_route_linux.go
- Create: internal/network/default_route_windows.go
- Create: internal/network/default_route_other.go
- Create: internal/network/default_route_linux_test.go

**Interfaces:**

- Each build target provides defaultRouteInterfaceIndexes(context.Context) (map[int]struct{}, error).
- GopsutilProvider.Interfaces combines gopsutil flags/addresses with net.Interfaces indexes and platform route results.

- [ ] Step 1: Write Linux parser tests before the parser.

Test synthetic /proc/net/route data with a destination and mask of zero,
ignore non-default routes, and test an IPv6 default route record from
/proc/net/ipv6_route. Include a malformed-row error test.

- [ ] Step 2: Run parser tests and verify RED.

Run:

~~~
go test ./internal/network -run 'TestParseLinux.*Route' -count=1
~~~

Expected: missing parser symbols or failing assertions.

- [ ] Step 3: Implement route discovery.

On Linux, read both route files and select default routes only. On Windows,
call windows.GetIpForwardTable2 for IPv4 and IPv6, select rows whose
destination prefix length is zero, and collect InterfaceIndex values; free
the returned MIB table in all paths. The fallback build returns an empty set.
Check context cancellation around filesystem/API calls.

- [ ] Step 4: Enrich the provider.

Use standard-library interface data keyed by name for Index, derive first
usable IPv4 and IPv6 addresses from gopsutil addresses, and mark DefaultRoute
when the index is in the discovered set. Keep the result sorted by name.

- [ ] Step 5: Run Linux tests and cross-compile Windows.

Run:

~~~
go test ./internal/network -count=1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c ./internal/network -o /tmp/netquota-network.test.exe
rm -f /tmp/netquota-network.test.exe
~~~

Expected: Linux tests pass and the Windows package compiles without executing
Windows-only code locally.

- [ ] Step 6: Commit route discovery.

~~~
git add internal/network
git commit -m "feat: prefer active default network routes"
~~~

### Task 3: Keep monitor state safe across disappearance and reappearance

**Files:**

- Modify: internal/app/monitor.go
- Modify: internal/app/monitor_test.go

**Interfaces:**

- Monitor.Sample returns the typed unavailable error without calling StateSaver when the current saved/runtime identity cannot be resolved.
- When Config.Interface is empty, the monitor keeps its first automatic interface identity for subsequent samples.

- [ ] Step 1: Add monitor regression tests.

Add tests that seed a saved Wi-Fi selection while only Ethernet is present,
move the default-route marker from Wi-Fi to VPN while Wi-Fi remains present,
and remove/re-add an active adapter with a lower cumulative counter. Assert no
state save on disappearance, no unrelated sample, identity retention across
route changes, and zero delta plus DownloadReset on counter decrease.

- [ ] Step 2: Run the new tests and verify RED.

Run:

~~~
go test ./internal/app -run 'TestMonitor(DoesNotFallback|RetainsAutomatic|Reappearing)' -count=1
~~~

Expected: the unavailable case currently samples the unrelated interface or
fails to expose the typed error.

- [ ] Step 3: Implement the monitor identity guard.

When configuration has no explicit selection and m.selected is populated,
resolve the runtime selection returned by SelectionForInterface(m.selected)
instead of recomputing a new default route. Compare identity through the
network helper. Do not reset or save state on an unavailable selection.
Retain reset behavior only after SetConfig receives a different explicit
selection.

- [ ] Step 4: Run app and non-GUI suites.

Run:

~~~
gofmt -w internal/app/monitor.go internal/app/monitor_test.go
go test ./internal/app ./internal/config ./internal/format ./internal/i18n ./internal/network ./internal/storage ./internal/usage ./internal/quota -count=1
~~~

- [ ] Step 5: Commit monitor behavior.

~~~
git add internal/app/monitor.go internal/app/monitor_test.go
git commit -m "fix: preserve monitor identity across route changes"
~~~

### Task 4: Add localized unavailable status and explicit re-baseline UI

**Files:**

- Modify: internal/tray/tray.go
- Modify: internal/tray/tray_test.go
- Modify: internal/i18n/translation/en.json
- Modify: internal/i18n/translation/zh-Hant.json
- Modify: internal/i18n/translation/ja.json
- Modify: cmd/netquota/main.go

**Interfaces:**

- interfaceText displays IPv6 when IPv4 is absent.
- ui.sample maps ErrSelectedInterfaceUnavailable to a localized message while exposing a reselect/settings action.
- Settings confirmation calls Monitor.SetConfig only after the user accepts a localized warning about baseline loss.

- [ ] Step 1: Add failing tests for IPv6 text and unavailable copy.

Test IPv6-only interface text, all three new catalog keys, and the UI error
path using the existing Fyne test seams.

- [ ] Step 2: Run focused tests and verify RED.

Run:

~~~
go test ./internal/i18n -run TestCatalogsAreComplete -count=1
go test ./internal/tray -run 'Test(InterfaceText|Unavailable|Rebaseline)' -count=1
~~~

Expected: new key/text assertions fail before the UI changes. If local desktop
headers are unavailable, retain the known Fyne prerequisite and use CI for the
tray package.

- [ ] Step 3: Implement localized dashboard and Settings behavior.

Add every key to every catalog. Show an unavailable status and a visible
Choose interface action that opens Settings. When the submitted interface
differs from the current one, display a confirmation dialog; only the positive
callback applies settings and triggers the existing saved-state reset. Include
an unavailable saved interface in the select options when it is absent from
the live list.

- [ ] Step 4: Update CLI interface output.

Print IPv6 when IPv4 is empty and include stable index/default-route markers in
--list-interfaces output without changing accounting behavior.

- [ ] Step 5: Run formatting, catalog, and available tests.

Run:

~~~
gofmt -w internal/tray/tray.go internal/tray/tray_test.go cmd/netquota/main.go
go test ./internal/i18n -count=1
go test ./internal/app ./internal/network ./internal/storage ./internal/usage ./internal/quota -count=1
~~~

- [ ] Step 6: Commit the UI unit.

~~~
git add internal/tray internal/i18n/translation cmd/netquota/main.go
git commit -m "feat: expose unavailable interface recovery"
~~~

### Task 5: Verify and document the complete #9 branch

**Files:**

- Modify: docs/design.md
- Modify: README.md
- Modify: docs/releasing.md

- [ ] Step 1: Document selection and recovery semantics.

State default-route preference, IPv6 support, no silent fallback for saved
selections, and the explicit re-baseline warning. Add manual desktop checks
for unavailable status and Settings confirmation.

- [ ] Step 2: Run available verification.

Run:

~~~
gofmt -l .
go vet ./internal/app ./internal/config ./internal/format ./internal/i18n ./internal/model ./internal/network ./internal/quota ./internal/storage ./internal/usage
go test ./...
go build -trimpath ./cmd/netquota
~~~

The GUI commands require the same Linux desktop headers installed by CI if
they are absent locally. Report exact missing packages and use Linux/Windows
CI as the platform build gate.

- [ ] Step 3: Audit branch constraints and commit documentation.

Run:

~~~
git log --format='%H%n%B' origin/main..HEAD | rg -n -i 'co-authored-by|codex|agent' && exit 1 || true
git status --short --branch
git diff --check origin/main...HEAD
~~~

If documentation changed, commit it with:

~~~
git add docs/design.md README.md docs/releasing.md
git commit -m "docs: describe resilient interface selection"
~~~
