#!/usr/bin/env bash
set -euo pipefail

cmd="${1:-test}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
plugins_root="${PLUGINS_ROOT:-$(cd "${repo_root}/.." && pwd)/plugins}"
plugin_test_timeout="${PLUGIN_TEST_TIMEOUT:-600s}"

if [[ -n "${PLUGIN_REPOS:-}" ]]; then
	read -r -a repos <<<"${PLUGIN_REPOS}"
else
	repos=()
	while IFS= read -r repo; do
		repos+=("$(basename "${repo}")")
	done < <(find "${plugins_root}" -maxdepth 1 -type d -name 'brainkit-plugin-*' | sort)
fi

if [[ ${#repos[@]} -eq 0 ]]; then
	echo "No plugin repos found under ${plugins_root}"
	exit 0
fi

build_dir=""
if [[ "${cmd}" == "build" || "${cmd}" == "smoke" ]]; then
	build_dir="${PLUGIN_BUILD_DIR:-$(mktemp -d)}"
	mkdir -p "${build_dir}"
	echo "Plugin build output: ${build_dir}"
fi

run_build() {
	local dir="$1"
	local repo="$2"
	local out="${build_dir}/${repo}"
	echo "==> build ${repo}"
	(cd "${dir}" && go build -o "${out}" .)
}

run_test() {
	local dir="$1"
	local repo="$2"
	echo "==> test ${repo}"
	(cd "${dir}" && go test ./... -count=1 -timeout="${plugin_test_timeout}")
}

for repo in "${repos[@]}"; do
	dir="${plugins_root}/${repo}"
	if [[ ! -d "${dir}" ]]; then
		echo "skip ${repo}: missing directory"
		continue
	fi
	if [[ ! -f "${dir}/go.mod" ]]; then
		echo "skip ${repo}: no go.mod"
		continue
	fi
	case "${cmd}" in
		build)
			run_build "${dir}" "${repo}"
			;;
		test)
			run_test "${dir}" "${repo}"
			;;
		smoke)
			run_build "${dir}" "${repo}"
			run_test "${dir}" "${repo}"
			;;
		*)
			echo "usage: $0 [build|test|smoke]" >&2
			exit 2
			;;
	esac
done
