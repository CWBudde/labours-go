#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
hercules_bin="${HERCULES_BIN:-$repo_root/../hercules/hercules}"
source_repo="${HERCULES_FIXTURE_REPO:-$repo_root/../hercules/cmd/hercules/test_data/hercules.siva}"
out_dir="${1:-$repo_root/test/testdata/hercules}"

mkdir -p "$out_dir"

if [[ ! -x "$hercules_bin" ]]; then
  echo "Hercules binary not found or not executable: $hercules_bin" >&2
  echo "Set HERCULES_BIN=/path/to/hercules to override." >&2
  exit 1
fi

if [[ ! -e "$source_repo" ]]; then
  echo "Fixture repository not found: $source_repo" >&2
  echo "Set HERCULES_FIXTURE_REPO=/path/to/repo-or-siva to override." >&2
  exit 1
fi

run_fixture() {
  local name="$1"
  shift
  echo "Generating $out_dir/$name.pb"
  "$hercules_bin" --pb --quiet "$@" "$source_repo" > "$out_dir/$name.pb"
}

run_optional_fixture() {
  local name="$1"
  shift
  echo "Generating $out_dir/$name.pb"
  if ! "$hercules_bin" --pb --quiet "$@" "$source_repo" > "$out_dir/$name.pb"; then
    rm -f "$out_dir/$name.pb"
    echo "Optional fixture $name failed to generate." >&2
  fi
}

run_fixture report_default \
  --burndown --burndown-files --burndown-people \
  --couples \
  --devs \
  --temporal-activity \
  --bus-factor \
  --ownership-concentration \
  --knowledge-diffusion \
  --hotspot-risk

run_optional_fixture report_all \
  --burndown --burndown-files --burndown-people \
  --couples \
  --devs \
  --temporal-activity \
  --bus-factor \
  --ownership-concentration \
  --knowledge-diffusion \
  --hotspot-risk \
  --shotness \
  --sentiment

run_fixture burndown_all --burndown --burndown-files --burndown-people
run_fixture couples --couples
run_fixture devs --devs
run_fixture shotness --shotness

run_optional_fixture sentiment --sentiment
