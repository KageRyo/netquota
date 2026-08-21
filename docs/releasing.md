# Releasing NetQuota

NetQuota is an open-source project. Releases are built from GitHub Actions and must be inspected as a draft before they are published.

The project privacy notice is documented in [PRIVACY.md](../PRIVACY.md). Keep it aligned with any future network or telemetry behavior.

## Release flow

From WSL, start the manual workflow:

```sh
gh workflow run draft-release.yml --ref main -f tag=v0.1.0-alpha.3
```

The workflow runs checks on `windows-latest` and `ubuntu-latest`, embeds the generated Windows ICO into the GUI executable with `go-winres`, builds the GUI and console binaries for Windows, builds the Linux amd64 portable archive, generates a `SHA256SUMS` manifest, and creates or updates a GitHub draft release with all four release assets. The maintainer should install the draft Windows installer and run the Linux archive on real machines before publishing the release.

`SHA256SUMS` is required by the in-app updater. Do not publish a release with a missing or stale manifest: it must contain the hashes of `netquota-windows-amd64.zip`, `netquota-windows-amd64-setup.exe`, and `netquota-linux-amd64.tar.gz`.

The Windows GUI executable, installer, Start Menu shortcut, optional desktop shortcut, and uninstaller all use the generated `assets/windows/icon.ico`. The ICO is regenerated from `assets/icon.svg` by `make icons`; CI fails if the committed derived files are stale.

The Linux artifact is `netquota-linux-amd64.tar.gz`. It contains the `netquota` GUI/CLI binary, project documentation, and `icon-256.png`; it requires the Linux desktop libraries listed in the README and is intentionally a portable tarball rather than a distro-specific package.

The installer shows the repository's `LICENSE` file and requires the user to accept it before continuing. This is an installer consent step; the project remains available under the MIT License.

## Windows code signing

The release workflow has an optional Authenticode signing path. It signs these three Windows PE files when the signing secrets are present:

```text
netquota.exe
netquota-console.exe
netquota-windows-amd64-setup.exe
```

It uses SHA-256 file and timestamp digests and an RFC 3161 timestamp. Without the secrets, the workflow succeeds but explicitly produces unsigned artifacts. Do not describe those artifacts as signed, and expect Windows reputation warnings to be possible.

For a private certificate, add these repository secrets:

- `WINDOWS_SIGNING_CERTIFICATE_BASE64`: the base64-encoded PFX certificate, including its private key
- `WINDOWS_SIGNING_CERTIFICATE_PASSWORD`: the PFX password

Optionally add the repository variable `WINDOWS_SIGNING_TIMESTAMP_URL`. If it is omitted, the workflow uses `http://timestamp.digicert.com`. Never commit a PFX file, certificate password, or private key to the repository.

On a Windows machine, create the base64 value without putting the PFX in Git:

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes("C:\secure\netquota-signing.pfx"))
```

After a signed build, verify the files with the Windows SDK's SignTool:

```powershell
signtool verify /pa /all netquota.exe
signtool verify /pa /all netquota-console.exe
signtool verify /pa /all netquota-windows-amd64-setup.exe
```

For an open-source project, [SignPath Foundation](https://signpath.org/) is another option to investigate. It offers free signing for eligible open-source projects and keeps the certificate private key in its HSM; acceptance is subject to its current requirements and manual release-approval process. If NetQuota uses that service, update this document with the project-specific signing policy and integrate its approved workflow rather than exporting a private key into GitHub Actions.

## Publication gate

Do not publish a draft until the Windows and Linux real-machine checks are complete. A release is ready only after:

- the installer displays the MIT License and blocks continuation until it is accepted;
- the GUI starts without a console window;
- the GUI executable, installer, and shortcuts display the NetQuota icon;
- the Linux archive starts after its documented desktop dependencies are installed;
- the tray menu shows total, download, and upload usage;
- the installer and portable package contain the same release version;
- the application can be uninstalled cleanly; and
- signing has been verified, or the release is explicitly labeled as unsigned.
