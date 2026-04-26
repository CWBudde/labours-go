# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working in this repository.

## Project Overview

Labours-go is a Go port of Python `labours`, the visualization/reporting command used by Hercules. The goal is a drop-in `labours` binary that can be used by `../hercules`, especially through:

```bash
hercules report --labours-cmd ./labours -o ./report <repo>
```

The port is in progress. Do not describe it as production-ready or fully compatible until the gates in `PLAN.md` are satisfied.

## Current Compatibility Status

Implemented or partially implemented modes:

- `burndown-project`
- `burndown-file`
- `burndown-person`
- `overwrites-matrix`
- `ownership`
- `couples-files`
- `couples-people`
- `couples-shotness`
- `shotness`
- `devs`
- `devs-efforts`
- `old-vs-new`
- `languages`
- `temporal-activity`
- `bus-factor`
- `ownership-concentration`
- `knowledge-diffusion`
- `hotspot-risk`
- `burndown-repository`
- `burndown-repos-combined`
- `devs-parallel`
- `run-times`
- `sentiment`
- `refactoring-proxy`

Known compatibility risks:

- Repository burndown modes need validation with a real multi-repository Hercules payload.
- `sentiment` requires collected `CommentSentimentResults` by default; legacy heuristic charts require explicit `--sentiment-fallback`.
- `devs-parallel` is approximate; synthetic charts require explicit `--devs-parallel-fallback`.
- The current test baseline includes visual compatibility failures. See `PLAN.md`.

## Build and Development Commands

Prefer `just` recipes:

```bash
just              # list recipes
just build        # build ./labours
just test         # run go test ./...
just check        # run lint/format checks when tools are installed
just clean        # remove local generated artifacts
```

Direct fallback commands:

```bash
go build -o labours
./labours --help
go test ./...
```

## Architecture

Core packages:

- `main.go`: entry point calling `cmd.Execute()`.
- `cmd/`: CLI command structure using Cobra.
  - `root.go`: root command, flags, config, theme setup.
  - `modes.go`: mode handler map and dispatch.
  - `helpers.go`: date parsing, input detection, output path helpers, Hercules integration helpers.
- `internal/readers/`: data input handling.
  - `reader.go`: reader interface.
  - `pb_reader.go`: protobuf reader.
  - `yaml_reader.go`: YAML reader.
- `internal/modes/`: analysis and chart implementations.
- `internal/graphics/`: plotting primitives, theme system, heatmaps, and chart saving.
- `internal/pb/`: generated protobuf types from local `pb.proto`.
- `test/`: integration and visual regression tests.

Key libraries:

- `github.com/spf13/cobra`
- `github.com/spf13/viper`
- `gonum.org/v1/plot`
- `github.com/schollz/progressbar/v3`
- `google.golang.org/protobuf`
- `gopkg.in/yaml.v3`

## Development Priorities

Use `PLAN.md` as the source of truth. Near-term priorities are:

1. Keep documentation honest about compatibility.
2. Build a binary named `labours` for drop-in testing.
3. Sync protobuf schema with current Hercules.
4. Fix reader extraction and output-file behavior for existing modes.
5. Implement missing Hercules report default modes.
6. Add report integration tests against `../hercules`.

## Testing Notes

The intended final baseline is `go test ./...` passing. At the start of Phase 0, the observed baseline was:

- 145 passed
- 15 failed
- 1 skipped

Failures were concentrated in:

- `internal/modes` language output tests
- `test/visual` Python compatibility/regression tests

Do not hide these failures by weakening tests without recording the compatibility decision in `PLAN.md`.

## Working Conventions

- Prefer existing package patterns and reader/mode boundaries.
- Keep compatibility behavior explicit. If a mode diverges from Python `labours`, document the reason and expected output.
- Avoid adding synthetic or heuristic data paths where Hercules provides real protobuf data.
- Make output paths deterministic because Hercules report collects generated assets.
- Preserve user changes in the working tree.
