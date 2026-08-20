# NetQuota design

## Responsibility boundaries

The application is intentionally small and local:

~~~text
gopsutil interface counters
            │
            ▼
     network.Provider
            │
            ▼
       usage.Tracker ──────▶ storage.Store
            │
            ▼
       quota.Calculate
        │          │
        ▼          ▼
     CLI/tray   notify.Notifier
~~~

- internal/network reads and resolves interfaces and cumulative receive/send counters.
- internal/usage turns cumulative counters into daily deltas, handles counter resets, and rolls over at the local date boundary.
- internal/quota evaluates total, download, and upload limits independently and produces threshold events.
- internal/storage writes versioned JSON through a temporary file and rename.
- internal/app coordinates sampling, state persistence, and notifications without knowing about Fyne widgets.
- internal/tray is a presentation layer for the monitor and settings.
- cmd/netquota selects tray, headless, and inspection modes.

## State invariants

1. The first sample for a date is a counter baseline; historical counter bytes are never added to that date.
2. A counter decrease contributes zero bytes for that direction and becomes the next baseline.
3. Download and upload usage are accumulated separately. Total usage is their saturating sum.
4. A zero-byte limit is disabled. Enabled limits must have ascending thresholds from 1% through 100%.
5. Threshold keys include the dimension, configured limit, and percentage. A newly configured limit can therefore produce an event immediately when current usage is already beyond its threshold, without repeating an unchanged event.
6. State is saved after every successful sample, including the alert marks for that sample.

## Persistence

Configuration and daily state are separate JSON files. Writes use:

~~~text
marshal → create temporary file in the target directory → flush → rename
~~~

The files are user-local and are not part of the source tree. The state includes the local date, daily download/upload totals, last cumulative counters, and the alert keys already delivered for that date.
