# NetQuota icon assets

The icon represents daily network quota usage with a quota ring and upload/download arrows on a transparent background.

- `icon.svg` is the canonical editable source.
- `icon-256.png` is generated from the canonical source for application metadata and packaging tools.
- `tray/icon-16.png` is the dedicated small system-tray raster asset.
- `fonts/NotoSansCJKtc-Regular.otf` and `fonts/NotoSansCJKjp-Regular.otf` are
  embedded for reliable 正體中文 and 日本語 rendering. They are distributed under
  the SIL Open Font License 1.1 in `fonts/OFL.txt`.
- `windows/icon.ico` is generated on demand with 16/32/48/64/128/256px entries for Windows resources and installers; it is intentionally not committed.
- The Go application embeds the SVG and tray PNG at build time, so installed builds do not need a separate asset directory.

Run `make icons` after changing `icon.svg` to regenerate the high-resolution PNG and Windows ICO. CI checks that the committed 256px PNG is up to date; release and Windows builds generate the ICO before packaging.
