# Billing cycle quotas

## Goal

Add selectable accounting cycles without changing the meaning of existing
daily configurations. Users can choose daily, calendar-monthly, or a custom
monthly reset day, and the dashboard/settings flow must show the active cycle
and next reset.

## Decisions

- `Config.BillingCycle` is persisted as a versioned field with `daily`,
  `monthly`, and `custom` kinds. Custom reset days are integers from 1 through
  31.
- Configuration version 1 migrates to the current version with the daily cycle
  selected. Existing quota values, thresholds, language, interface identity,
  notifications, and startup preference are retained.
- State stores a canonical `PeriodKey` based on the local calendar date at
  which the active period began, plus a cycle identity. The legacy `Date`
  field remains readable during migration and is kept synchronized for older
  diagnostics.
- Daily periods begin at local midnight. Monthly periods begin on the first
  local day of the month. A custom day is clamped to the last day of shorter
  months; for example, day 31 begins February's period on February 28 or 29,
  and the next period begins on March 31.
- All boundary calculations use calendar arithmetic in the current local
  timezone, never a 24-hour duration. This keeps DST transitions correct.
- The next sample after sleep or a missed boundary starts a new period and
  establishes a fresh counter baseline. Traffic during an unseen boundary is
  not retroactively assigned to either period.
- A timezone change is evaluated on the next sample using the current local
  timezone. If it changes the resolved period key, usage and alert marks are
  reset and the sample is a baseline; no historical bytes are reallocated.
- Changing the billing-cycle setting is an explicit accounting reset, like
  changing the monitored interface. The next sample is a baseline for the new
  cycle and state is persisted immediately after the setting change.
- Threshold marks are cleared once per new period. A period-boundary baseline
  never emits a quota alert, even when the first observed counter is already
  above a threshold.
- The dashboard displays the localized cycle name and the next reset date/time;
  Settings exposes the cycle selector and custom reset-day input. The selected
  cycle is also included in release documentation and all supported catalogs.

## Boundaries

The implementation remains local and offline. It does not attempt to infer
traffic from cumulative counters while the application was asleep, and it does
not change interface selection, quota arithmetic, or notification delivery
outside the period-reset behavior described above.

Fyne's supported semantic accessibility surface remains covered by the existing
automated checks; keyboard traversal, localized rendering, and the manual
screen-reader/high-contrast checklist continue to be release gates for the
expanded Settings form.
