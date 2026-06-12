#!/bin/sh
assert_success() {
  if [ $1 -ne 0 ]; then
    echo "FAIL: command failed with exit code $1"
    exit 1
  fi
}
