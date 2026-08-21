# NetQuota icon assets

The icon represents daily network quota usage with a quota ring and upload/download arrows on a transparent background.

- `icon.svg` is the canonical editable source.
- `icon-256.png` is generated from the canonical source for application metadata and packaging tools.
- `tray/icon-16.png` is generated from the same source for the system tray and desktop notifications.
- `windows/icon.ico` is generated on demand with 16/32/48/64/128/256px entries for Windows resources and installers; it is intentionally not committed.
- The Go application embeds the SVG and tray PNG at build time, so installed builds do not need a separate asset directory.

Run `make icons` after changing `icon.svg` to regenerate the derived assets. CI checks that the committed PNG assets are up to date, while release/Windows builds generate the ICO before packaging.
