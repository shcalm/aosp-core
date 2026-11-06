# Siso config

This directory tells Siso how to handle RBE for the various Ninja rules.
It provides the configuration needed for supported projects in AOSP to use RBE,
 either via `rewrapper` or using Siso's native RBE support.

If `SISO_CONFIG_DIR` is set to:
- `${OUT_DIR}/siso_config`, that directory is used as-is.
- any other directory, `${OUT_DIR}/siso_config` is constructed with:
  - `${OUT_DIR}/siso_config/main` is a symlink to this directory.
  - `${OUT_DIR}/siso_config/extension` is a symlink to `${SISO_CONFIG_DIR}`.
  - `${SISO_CONFIG_DIR}/main.star` must contain a module "extension", similar
    to the "main" module in `build/soong/siso_config/main.star`

See Siso documentation for
[starlark config](https://chromium.googlesource.com/build/+/refs/heads/main/siso/docs/starlark_config.md)
for more details.
