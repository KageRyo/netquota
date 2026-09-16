# Billing cycle quotas implementation plan

> **Execution note:** implement this plan in the isolated
> `feat/billing-cycle-quotas` worktree. Follow test-driven development for each
> behavior change and run the focused test before moving to the next task.

## Architecture

Keep period resolution in `internal/usage`, where cumulative counter deltas
already become accounting usage. Add a small calendar resolver returning the
current period key, local start, and next reset. Extend versioned model/config/
state data, then let `internal/app` pass the configured cycle to the tracker
and expose period metadata to the tray. Settings parses the same model and the
dashboard renders the next reset from the current local clock.

## Tasks

### 1. Establish migration and cycle model contracts

Files: `internal/model/model.go`, `internal/config/config.go`, their tests.

- Add `BillingCycleKind`, `BillingCycle`, daily/monthly/custom constants, and
  `Config.BillingCycle` with the `billing_cycle` JSON field.
- Bump config/state versions and add state `PeriodKey` and cycle identity
  fields while retaining legacy `Date` for v1 reads.
- Make `WithDefaults` migrate v1/zero-version configurations to the current
  version and daily cycle without changing existing values.
- Validate cycle kind and custom reset-day range; normalize daily/monthly
  reset-day values consistently.
- Test default daily behavior, v1 migration, invalid kinds/days, and clone
  isolation.

### 2. Implement calendar period resolution first

Files: `internal/usage/cycle.go`, `internal/usage/cycle_test.go`.

- Resolve daily, monthly, and custom periods with injected locations.
- Cover month ends, February leap/non-leap clamping, exact reset instants,
  next-reset calculation, DST transitions, and timezone conversion.
- Keep the resolver independent from storage and GUI code.

### 3. Make tracking period-aware

Files: `internal/usage/tracker.go`, `internal/usage/tracker_test.go`.

- Preserve the existing `NewTracker` daily-compatible constructor and add a
  cycle-aware constructor for the monitor.
- Migrate legacy state fields in memory, detect cycle-key changes, and reset
  usage/counters/threshold marks at a new period.
- Add `NewPeriod`/period metadata while retaining `NewDay` compatibility.
- Test monthly/custom rollover, threshold mark clearing, counter baseline
  behavior after rollover, missed sampling, timezone changes, and counter
  decrease handling within a period.

### 4. Connect monitor persistence and notifications

Files: `internal/app/monitor.go`, `internal/app/monitor_test.go`,
`internal/storage/json.go`, `internal/storage/json_test.go`.

- Construct the tracker from `Config.BillingCycle` and refresh its local
  timezone before every sample.
- Reset accounting and persist state when an explicit cycle setting changes.
- Suppress alerts on any new-period baseline, not only daily rollover.
- Migrate v1 state on load and preserve state after successful samples.
- Test cycle changes, monthly alert de-duplication across samples, and state
  round trips/migration.

### 5. Add localized Settings controls

Files: `internal/tray/tray.go`, `internal/tray/tray_test.go`, translation
catalogs.

- Add a cycle selector and enabled custom reset-day entry to the deterministic
  Settings builder.
- Parse and validate the custom day with localized errors while preserving the
  existing settings reader compatibility path.
- Keep keyboard traversal, Escape handling, semantic button labels, and the
  existing layout/scroll behavior covered by the accessibility regression
  suite.
- Add English, Traditional Chinese, and Japanese labels/messages and extend
  catalog completeness tests.

### 6. Show cycle status and document operational behavior

Files: `internal/tray/tray.go`, `internal/tray/accessibility_test.go`,
`README.md`, `docs/design.md`, `docs/releasing.md`.

- Display the localized active cycle and next reset in the dashboard at the
  production width.
- Add localized rendering tests for all supported languages and cycle options.
- Document short-month clamping, DST/timezone changes, sleep/missed samples,
  explicit cycle changes, and the first-sample baseline rule.

### 7. Verify, commit, and deliver PR #10

- Run formatting, focused tests, all non-GUI package tests, `go vet`, and the
  repository build locally; let Linux/Windows CI execute the GUI build/tests.
- Check `git diff --check` and scan new commit messages for forbidden trailers
  or branch prefixes.
- Commit with descriptive Conventional Commits and no `Co-authored-by` line.
- Open a PR that closes issue #10, wait for all required checks, and squash
  merge after review/authorization.
