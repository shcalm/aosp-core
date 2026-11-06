# Soong-only builds

Android builds can now be done without makefiles for generating build targets. (makefiles are
still used for product config) Doing so will halve the analysis time of the build, increasing
developer productivity.

To enable soong-only builds:

  1. Run `build/soong/scripts/soong_only_diff_test.py <product_name>`
    a. This will run both a soong-only and soong+make build, and report differences between them.
    b. It must be run in an envsetup'd shell.
  2. Fix any diffs reported by the script
    a. If you still have modules defined in Android.mk files, they must be converted to Android.bp
       files. See [our mk -> bp conversion guide](./mk2bp.md) for this.
    b. There may also be bugs or unconverted parts of the core build system code. Please reach out
       to the soong team if you encounter problems despite all Android.mk files being converted to
       soong.
  3. Add `PRODUCT_SOONG_ONLY := true` to your product config.

It's possible that your build doesn't work or produces incorrect results after enabling soong-only
mode. There can be issues in two main places:

 - If you still have modules defined in Android.mk files, they must be converted to Android.bp
   files. See [our mk -> bp conversion guide](./mk2bp.md) for this.
 - There may also be bugs or unconverted parts of the core build system code. Please reach out to
   the soong team if you encounter problems despite all Android.mk files being converted to soong.

To test that your build works with soong only, you can use
`build/soong/scripts/soong_only_diff_test.py <product name>`. That script will run both a soong-only
and soong+make build of your product, and report any differences found. The script must be run
in an envsetup'd shell.

Soong-only mode can also be controlled with the `SOONG_ONLY=true/false` environment variable.
