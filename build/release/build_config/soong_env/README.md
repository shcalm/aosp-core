# Soong env

This directory contains files that allow a phased rollout of a change in Soong
behavior that is otherwise controlled by an environment variable.

This approach **SHOULD ONLY BE USED** when "after the first pass of product
config" is TOO LATE to get the answer.  When feasible, use a Build Flag.

# How it works

At startup, Soong checks for a file named `soong_env/${TARGET_RELEASE}.json` in:
- `build/release`
- `vendor/google_shared/build/release`
- `vendor/google/release`
- any additional directories listed in `${SOONG_ENVVAR_FLAG_DIRS}`

Missing files are ignored.

The value of each environment variable used by Soong is:
- If the environment variable was already present in the environment when
  Soong started, it will not be changed from that value.
- If it was unset, its value will be the last value set in the locations
  above.

## Syntax

The file `soong_env/${TARGET_RELEASE}.json` contains a json object containing
the `env_default` map: the name of the environment variable is the key, and the
value to assign is the value.  For example, to set `SOONG_NINJA` to `siso` (when
not already set) on `trunk_staging`, then `soong_env/trunk_staging.json` would
contain:
```
{
  "env_default: {
    "SOONG_NINJA": "siso"
  }
}
```

## Caveats

This approach to guarding features for incremental rollout to release configs is
**extremely limited** in what it supports.  Compared to Build Flags:

- There is no fallback value: those are found in the Soong code.
- Inheritance is not honored: Soong only looks for `${TARGET_RELEASE}`.
- While setting `SOONG_ENVVAR_FLAG_DIRS` in one of these files will change the
  value, it will be ignored by this process.
