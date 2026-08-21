# NetQuota Privacy Policy

Last updated: 2026-08-21

NetQuota is an open-source desktop application. NetQuota does not collect telemetry, advertising data, account information, or network-usage data for the project maintainers.

## Data processed locally

NetQuota reads operating-system network interface counters to calculate daily usage. Configuration and daily accounting state are stored locally on the user's device. They are not uploaded by NetQuota.

NetQuota may also store local operating-system startup settings when the user enables start on login.

## GitHub release check

While the tray application is running, NetQuota checks the public GitHub Releases API for a newer version. This request contains no usage counters, interface names, configuration, or account token. It uses the fixed public endpoint for this repository and the user agent `NetQuota-Update-Checker`.

As with any connection to GitHub, GitHub may receive standard request and service-usage information such as an IP address, device or application information, and the time of the request. This information is handled under [GitHub's General Privacy Statement](https://docs.github.com/en/site-policy/privacy-policies/github-general-privacy-statement), not by NetQuota or its maintainers.

If an update is available and the user confirms installation, NetQuota downloads the compatible release asset and its `SHA256SUMS` manifest directly from GitHub, verifies the asset locally, and starts the platform installer or portable replacement. If automatic installation is unavailable, the GitHub release page can be opened in the user's default browser. GitHub handles these connections under its own policies.

Update checks are not required for accounting. If GitHub is unavailable or the request is blocked, NetQuota continues monitoring local network counters.

## No analytics or maintainer backend

NetQuota does not include analytics, crash reporting, advertising, or a project-maintainer backend. It does not sell or share locally stored configuration or usage state.

This policy covers NetQuota itself. Operating systems, GitHub, the default browser, and third-party libraries may have their own privacy policies and data-processing practices.

## Contact

For privacy questions about NetQuota, contact the repository maintainer through the [GitHub issue tracker](https://github.com/KageRyo/netquota/issues). Please do not include private or sensitive information in a public issue.
