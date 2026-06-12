#!/bin/sh

assert_file_exists() {
  if [ ! -f "$1" ]; then
    echo "FAIL: File $1 does not exist"
    exit 1
  fi
}

assert_dir_exists() {
  if [ ! -d "$1" ]; then
    echo "FAIL: Directory $1 does not exist"
    exit 1
  fi
}

assert_contains() {
  if ! grep -q "$2" "$1"; then
    echo "FAIL: File $1 does not contain '$2'"
    exit 1
  fi
}
