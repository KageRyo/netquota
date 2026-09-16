
# Network Interface Selection Resilience Design

## Goal

Make interface selection safe across route changes, IPv6-only hosts, adapter
renames, and temporary disappearance without silently moving accounting to a
different adapter.

## Chosen approach

The network.Interface type will expose the standard interface index, both
address families, and whether the operating system currently has a default
route on that interface. The production provider will enrich gopsutil data
with the standard library interface index and platform route-table discovery.
Linux and Windows will use native route-table readers; unsupported targets
simply leave the route marker unset and retain deterministic selection for an
unconfigured monitor.

Selection has two modes:

- With a saved selection, match the saved stable index first, then hardware
  address, then a legacy name-only selection. A saved selection that does not
  match returns a typed unavailable error. It never falls through to another
  interface.
- With no saved selection, choose a non-loopback default-route interface,
  then a non-loopback interface with either IPv4 or IPv6, then the first
  remaining interface. After the first automatic choice, the monitor keeps
  that runtime identity until it is explicitly changed in Settings.

The monitor will not mutate usage or persist state on an unavailable-interface
error. Saving a different interface remains the only operation that resets the
usage baseline; the UI will show a confirmation explaining that traffic from
the previous interface cannot be reconstructed. The dashboard will expose a
localized reselect action when the selected interface is unavailable.

## Identity and counter behavior

The persisted selection adds index and ipv6 while retaining existing name,
hardware-address, and IPv4 fields for backward-compatible JSON decoding. A
matching stable index or hardware address may survive a rename; a same-name
adapter with a changed hardware identity is not treated as the old adapter.
When a matching adapter reappears, the normal cumulative-counter decrease
logic remains authoritative: a decreased counter contributes zero and marks a
counter reset, preventing traffic from being invented.

## User-visible behavior

The dashboard and Settings dialog will identify IPv6-only interfaces, show a
localized unavailable status, and provide a direct path to Settings. Changing
the selected interface requires confirmation before SetConfig resets the
baseline. All new copy must exist in English, Traditional Chinese, and
Japanese catalogs.

## Verification

Automated tests will cover default-route preference, virtual-interface order,
IPv6-only selection, stable identity matching, saved-interface disappearance,
runtime disappearance/reappearance, counter re-baselining, localized status
copy, and the explicit Settings confirmation path. The full Linux/Windows CI
matrix remains the authority for Fyne and platform-specific compilation.

## Non-goals

- Do not capture packets, contact a server, or infer usage from a remote
  service.
- Do not automatically change a user's saved selection or reset its baseline.
- Do not redesign the tray layout beyond the unavailable/reselect affordance
  and the address text needed for IPv6.
