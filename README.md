# Labours-go

Labours-go is a Go port of the Python [labours](https://github.com/src-d/hercules/tree/master/python/labours) command used by Hercules to render repository analysis data. The target is a drop-in `labours` binary for pipelines such as:

```bash
hercules --burndown --pb /path/to/repo | labours -f pb -m burndown-project -o chart.png
hercules report --labours-cmd ./labours -o ./report /path/to/repo
```

## Project Status

This repository is an active port, not yet a complete replacement for Python `labours`.

What is already present:

- Core CLI plumbing for reading Hercules YAML and protobuf data.
- Burndown, ownership, overwrites, devs, coupling, shotness, old-vs-new, language, runtime, and related plotting code.
- A Go charting stack based on `gonum/plot`.
- Compatibility and visual test scaffolding.
- Example Hercules datasets under `example_data/` and `data/`.

Known gaps:

- Current Hercules report modes are not all implemented. Missing report modes include `temporal-activity`, `bus-factor`, `ownership-concentration`, `knowledge-diffusion`, `hotspot-risk`, `burndown-repository`, and `burndown-repos-combined`.
- The local protobuf schema is behind `../hercules/internal/pb/pb.proto`.
- Some modes are approximations rather than direct Python-labours ports, notably `sentiment` and `devs-parallel`.
- Some modes still need output-path compatibility work because Hercules report passes one concrete output file per mode.
- `go test ./...` currently has known failures in language output tests and visual compatibility tests. See [PLAN.md](./PLAN.md) for the current baseline and completion plan.

## Build

Build the drop-in binary:

```bash
just build
```

This creates `./labours`. A development alias can also be built manually when useful:

```bash
go build -o labours-go
```

Verify the CLI:

```bash
./labours --help
```

## Usage Examples

Generate a project burndown chart from a protobuf payload:

```bash
./labours -f pb -i analysis.pb -m burndown-project -o burndown.png
```

Generate charts from a saved YAML payload:

```bash
./labours -f yaml -i analysis.yaml -m devs -o devs.png
```

Run through Hercules report while testing this port:

```bash
hercules report --labours-cmd ./labours -o ./report /path/to/repo
```

## Supported Modes

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
- `devs-parallel`
- `run-times`
- `sentiment`

Missing modes needed for full current Hercules parity:

- `burndown-repository`
- `burndown-repos-combined`
- `temporal-activity`
- `bus-factor`
- `ownership-concentration`
- `knowledge-diffusion`
- `hotspot-risk`
- `refactoring-proxy`

## Development Workflow

This project uses [Just](https://github.com/casey/just) as a command runner and [treefmt](https://github.com/numtide/treefmt) for formatting.

Common commands:

```bash
just              # list recipes
just build        # build ./labours
just test         # run go test ./...
just check        # run lint/format checks where tools are installed
just clean        # remove generated local build/test artifacts
```

Run with arguments:

```bash
just run -f pb -i example_data/hercules_burndown.pb -m burndown-project -o out.png
just run-built --help
```

Visual and compatibility helpers:

```bash
just test-visual
just test-python-compat
just generate-all-charts
```

## Architecture

Main packages:

- `cmd/`: Cobra CLI, mode dispatch, input/output helpers.
- `internal/readers/`: YAML/protobuf reader implementations.
- `internal/modes/`: analysis mode implementations.
- `internal/graphics/`: plotting, themes, heatmaps, and chart helpers.
- `internal/pb/`: generated protobuf types from `pb.proto`.
- `test/`: integration and visual regression tests.

The intended data flow is:

```text
Hercules .yaml/.pb -> reader -> mode processor -> chart/data output
```

## Completion Plan

See [PLAN.md](./PLAN.md). It tracks the work required to make this a complete drop-in replacement as used by `../hercules`.
