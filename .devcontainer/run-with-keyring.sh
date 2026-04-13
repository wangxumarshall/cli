#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -eq 0 ]; then
  set -- mise run test:ci
fi

export ENTIRE_DEVCONTAINER_KEYRING_PASSWORD="${ENTIRE_DEVCONTAINER_KEYRING_PASSWORD:-entire-devcontainer}"

exec dbus-run-session -- bash -lc '
  set -euo pipefail
  printf "%s" "$ENTIRE_DEVCONTAINER_KEYRING_PASSWORD" | gnome-keyring-daemon --unlock >/dev/null
  exec "$@"
' bash "$@"
