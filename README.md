<div align="center">
  <img src="assets/icon.svg" alt="NetQuota icon" width="128" height="128">
  <h1>NetQuota</h1>
  <p>A private, local daily network-usage monitor for your system tray.</p>
  <p>
    <a href="https://github.com/KageRyo/netquota/releases/latest">Download the latest release</a>
    ·
    <a href="#install">Install</a>
    ·
    <a href="https://github.com/KageRyo/netquota/issues">Report an issue</a>
  </p>
</div>

[![CI](https://github.com/KageRyo/netquota/actions/workflows/ci.yml/badge.svg)](https://github.com/KageRyo/netquota/actions/workflows/ci.yml)
[![Latest release](https://img.shields.io/github/v/release/KageRyo/netquota?display_name=tag&sort=semver)](https://github.com/KageRyo/netquota/releases)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

NetQuota tracks the traffic reported by one network interface, keeps daily
download and upload totals on your device, and warns you before a quota is
reached. It is designed for people who want a lightweight desktop reminder—not
a network-management service, packet inspector, or cloud account.

## Install

### Windows (x64)

1. Download [the Windows installer](https://github.com/KageRyo/netquota/releases/latest/download/netquota-windows-amd64-setup.exe).
2. Run it and follow the setup wizard. The installer is available in English,
   正體中文, and 日本語.
3. Open NetQuota from the Start menu. It will appear in the system tray.

The installer creates an uninstaller and can optionally create a desktop
shortcut. A portable ZIP is also available on the
[release page](https://github.com/KageRyo/netquota/releases/latest).

### Linux (amd64)

1. Download [the Linux portable archive](https://github.com/KageRyo/netquota/releases/latest/download/netquota-linux-amd64.tar.gz).
2. Extract it and run the application:

   ```sh
   tar -xzf netquota-linux-amd64.tar.gz
   cd netquota-linux-amd64
   ./netquota
   ```

NetQuota is a portable archive, not a `.deb`, AppImage, or distribution
repository package. It needs a normal Linux desktop environment with the
OpenGL/EGL, Wayland or X11 runtime libraries that Fyne uses.

### Verify a download

Each release includes a `SHA256SUMS` manifest for the Windows installer,
Windows portable ZIP, and Linux archive. Download it from the same release if
you need to verify an artifact before installing it.

Releases also include `RELEASE-METADATA.json` and signed GitHub Artifact
Attestations. With the GitHub CLI, you can verify the build provenance of a
downloaded artifact:

```sh
gh attestation verify netquota-linux-amd64.tar.gz --repo KageRyo/netquota
```

## First run

1. Open **Settings** from the tray menu.
2. Select the network interface to monitor.
3. Set total, download, and upload quotas in GiB. Set a limit to `0` to
   disable it.
4. Choose alert percentages and, if you want, enable start-on-login.

The first successful sample creates the day's baseline. NetQuota cannot infer
traffic that happened before that baseline, so give yourself a little margin
below an externally enforced quota.

## Highlights

- Daily total, download, and upload accounting
- Independent quotas and notification thresholds for each direction
- Local-time daily rollover, including after sleep or hibernation
- Counter-reset detection after a reboot or interface reconnect
- Optional start-on-login on Windows and Linux
- User-confirmed updates with downloaded-artifact checksum verification
- English, 正體中文, and 日本語 interfaces with bundled CJK fonts
- CLI modes for diagnostics and headless monitoring

## Languages and privacy

Choose `English`, `正體中文`, or `日本語` in Settings at any time. The application
language is saved separately from the Windows installer language.

Usage data and settings stay in per-user local configuration files. NetQuota
does not inspect packets, upload your usage data, require administrator/root
privileges, throttle traffic, or block connections. It does check GitHub for
new published releases; see the [privacy policy](PRIVACY.md) for details.

## What NetQuota measures

NetQuota reads the operating system's cumulative receive and send counters for
the selected interface. It is useful for personal awareness, but it is not an
authoritative record from an ISP, school, gateway, or network administrator.

It tracks one selected interface at a time. Traffic through a different
adapter, VPN, or device is not included unless that interface is selected.

## Command line

The graphical app is the normal way to use NetQuota. The same binary also
provides a few commands for troubleshooting and automation:

```sh
netquota --version
netquota --list-interfaces
netquota --once
netquota --headless
```

Use `--config` and `--state` to provide custom paths when diagnosing a setup.
Run `netquota --help` to see every option.

## Build from source

Go 1.26 or newer is required. On Debian/Ubuntu-based Linux systems, install
the desktop development libraries Fyne needs before building:

```sh
sudo apt-get update
sudo apt-get install -y libgl1-mesa-dev libegl1-mesa-dev libgles2-mesa-dev libwayland-dev libxkbcommon-dev libxkbcommon-x11-dev xorg-dev
```

Then run or build NetQuota:

```sh
go run ./cmd/netquota
go build -trimpath -o bin/netquota ./cmd/netquota
```

For the full contributor workflow, required checks, and pull-request process,
see [CONTRIBUTING.md](CONTRIBUTING.md).

## Project information

- [Report a bug or request a feature](https://github.com/KageRyo/netquota/issues)
- [Roadmap milestones](https://github.com/KageRyo/netquota/milestones)
- [Security policy](SECURITY.md)
- [Architecture notes](docs/design.md)
- [Maintainer release guide](docs/releasing.md)

## License

NetQuota is open source under the [MIT License](LICENSE).
