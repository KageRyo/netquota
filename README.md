# NetQuota

[![CI](https://github.com/KageRyo/netquota/actions/workflows/ci.yml/badge.svg)](https://github.com/KageRyo/netquota/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/KageRyo/netquota.svg)](https://pkg.go.dev/github.com/KageRyo/netquota)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

NetQuota is a lightweight cross-platform system-tray application that tracks daily network usage on a selected interface and warns before a configurable traffic quota is reached.

It measures the operating system's cumulative interface counters, keeps the daily download and upload totals locally, and exposes three independently configurable limits:

- total traffic (`download + upload`)
- download traffic
- upload traffic

Each enabled limit has its own notification percentages. A limit of `0` disables that dimension.

## Features

- Windows and Linux desktop tray application built with Go and Fyne
- Select an interface by name, hardware address, and IPv4 identity
- Daily download, upload, and total usage
- Adjustable total, download, and upload quotas
- Independent alert percentages for every enabled quota
- Desktop notifications with one notification per threshold per day
- Local-time day rollover, including after sleep or hibernation
- Counter reset detection after reboot or interface reconnect
- Atomic JSON persistence for configuration and daily state
- Optional start-on-login setup for Windows and Linux
- CLI inspection mode for troubleshooting without opening the tray

## Quick start

Go 1.26 or newer is required. On Linux, Fyne needs the desktop development libraries used by the CI workflow:

```sh
sudo apt-get update
sudo apt-get install -y libgl1-mesa-dev libegl1-mesa-dev libgles2-mesa-dev libwayland-dev libxkbcommon-dev libxkbcommon-x11-dev xorg-dev
```

Run the tray application from a checkout:

```sh
go run ./cmd/netquota
```

Useful CLI modes:

```sh
go run ./cmd/netquota --list-interfaces
go run ./cmd/netquota --once
go run ./cmd/netquota --headless
go run ./cmd/netquota --install-startup
go run ./cmd/netquota --uninstall-startup
```

Build a local binary:

```sh
mkdir -p bin
go build -trimpath -o bin/netquota ./cmd/netquota
```

The tray window's Settings page lets you select the interface, change each quota in GiB, edit each threshold list, enable notifications, and configure start-on-login.

## Configuration

NetQuota creates `config.json` and `state.json` below the platform's per-user configuration directory. The exact location is reported by the platform; it is never stored in the repository. Custom paths can be supplied for diagnostics:

```sh
go run ./cmd/netquota --config ./config.json --state ./state.json --once
```

The generated configuration follows this shape:

```json
{
  "version": 1,
  "interface": {
    "name": "Ethernet",
    "hardware_address": "00:11:22:33:44:55",
    "ipv4": "192.0.2.10"
  },
  "quotas": {
    "total": {
      "bytes": 107374182400,
      "alert_percentages": [70, 85, 95, 100]
    },
    "download": {
      "bytes": 0,
      "alert_percentages": []
    },
    "upload": {
      "bytes": 0,
      "alert_percentages": []
    }
  },
  "poll_interval_seconds": 2,
  "notifications": {
    "enabled": true
  },
  "start_on_login": false
}
```

JSON quota values use bytes. The Settings page uses binary GiB (`1 GiB = 1024³ bytes`). Alert percentages must be whole numbers from 1 to 100, in ascending order. A quota can be enabled with a different threshold list from the other two quotas.

## How accounting works

On every sample NetQuota reads the selected interface's cumulative receive and send counters through [gopsutil](https://github.com/shirou/gopsutil):

```text
current counter - previous counter = sample delta
daily download/upload total + sample delta = current usage
```

When no valid baseline exists for the current day, the first sample establishes one. Traffic that occurred before that baseline cannot be reconstructed. If an operating-system counter becomes smaller than the previous value, NetQuota treats it as a reset and does not add a negative delta. When the local calendar date changes, the next sample starts a new daily baseline, so sleep and hibernation do not depend on a timer firing at midnight.

## Important limitation

NetQuota reports the local operating system's interface counters. It is an estimate of the traffic visible to this machine, not an authoritative counter from a school gateway, ISP, switch, or other network administrator. Keep a safety margin below any externally enforced quota.

NetQuota does not inspect packets, require administrator/root privileges, measure traffic per process, shape bandwidth, block network access, or upload usage data to a server.

## Development

The project keeps accounting logic independent from the tray UI:

```text
network provider → usage tracker → quota evaluator
                         ├──────→ JSON state
                         ├──────→ desktop notification
                         └──────→ CLI / system tray
```

Run the same checks used by CI:

```sh
gofmt -w .
go vet ./...
go test ./...
go build ./cmd/netquota
```

New behavior should include focused unit tests. The test suite covers baselines, counter resets, local-date rollover, overflow handling, independent limits, alert de-duplication, persistence, monitor integration, and Settings parsing.

See [docs/design.md](docs/design.md) for the component boundaries and data model. Contributions are welcome; [CONTRIBUTING.md](CONTRIBUTING.md) describes the workflow.

## Versioning and license

The current development release is **NetQuota v0.1.0**. Releases use semantic version numbers such as `v0.1.0`.

NetQuota is released under the [MIT License](LICENSE).
