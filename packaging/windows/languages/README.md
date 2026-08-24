# Inno Setup language files

`ChineseTraditional.isl` and `Japanese.isl` are the official Inno Setup
translations from the [`jrsoftware/issrc` repository](https://github.com/jrsoftware/issrc/tree/main/Files/Languages).
They are kept with the installer script so Windows release builds do not depend
on which optional translations a runner's Inno Setup installation includes.

The installer overrides the Chinese display name to `正體中文` in
`../netquota.iss`; the translation contents remain the upstream translation.
