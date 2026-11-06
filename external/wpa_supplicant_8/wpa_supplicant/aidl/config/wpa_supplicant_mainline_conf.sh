#!/bin/bash
#
# Copyright (C) 2025 The Android Open Source Project
#
# This software may be distributed under the terms of the BSD license.
# See README for more details.
#

# Removes comments, trailing whitespace, and empty lines from the template.
# Logic was copied from wpa_supplicant_conf.sh
# $1: the template file name
sed -e 's/#.*$//' -e 's/[ \t]*$//' -e '/^$/d' < $1
