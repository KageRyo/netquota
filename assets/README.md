# NetQuota icon assets

The icon represents daily network quota usage with a quota ring and upload/download arrows.

- `icon.svg` is the canonical editable source.
- `tray/icon-16.png` is the small system-tray raster asset.
- The Go application embeds both assets at build time, so installed builds do not need a separate asset directory.

Additional raster sizes and Windows `.ico` files should be generated from the canonical source as needed by packaging/build workflows instead of committing redundant copies by default.
