# SDV's feature flag and release config location
This directory contains the feature flag declarations and values for the Software Defined Vehicle (SDV) project.

Since SDV is a subproject of AAOS (Android Automotive OS), we inherit from the release configuration
of AAOS, we will use the release configuration from Core Android as a fallback.

## Target Mapping

Here, we store the flags that are part of our `*s2a` & `trunk_staging` release configs.

We follow the following inheritance tree:
* `cs2a` -> `cp2a`: These inherit from the Core Android release config, as there is
    currently no AAOS release configuration available.
* `trunk_staging` -> `trunk_staging` (here we rather extend the `trunk_staging` config with the SDV feature flags)

This inheritance makes it possible to extend, but also override feature flags that are used in Core Android.

These feature flags are then used in the SDV devices defined in `device/google/sdv`.
