#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

package_timeout="${BRAINKIT_PACKAGE_TEST_TIMEOUT:-300s}"
root_timeout="${BRAINKIT_ROOT_TEST_TIMEOUT:-600s}"
root_package="github.com/brainlet/brainkit"

/opt/homebrew/bin/go list ./... \
  | rg -v '/experiments/' \
  | rg -v "^${root_package}\$" \
  | xargs /opt/homebrew/bin/go test -count=1 -timeout "${package_timeout}"

# The root package carries the broad cross-subsystem suite, so it needs a
# larger timeout budget than the leaf packages.
/opt/homebrew/bin/go test -count=1 -timeout "${root_timeout}" "${root_package}"
