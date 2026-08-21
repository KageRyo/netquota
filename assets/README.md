# NetQuota icon assets

The icon represents daily network quota usage with a quota ring and upload/download arrows.

- `icon.svg` is the canonical editable source.
- `icon-256.png` is generated from the canonical source for application metadata and packaging tools.
- `windows/icon.ico` is generated with 16/32/48/64/128/256px entries for Windows resources and installers.
- `tray/icon-16.png` is the small system-tray raster asset.
- The Go application embeds both assets at build time, so installed builds do not need a separate asset directory.

Run `make icons` after changing `icon.svg` to regenerate the derived PNG and ICO files. The CI workflow checks that the committed generated files are up to date.
