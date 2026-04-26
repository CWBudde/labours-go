# Plan: Complete `labours-go` as a Drop-in `labours` Replacement for Hercules

Date: 2026-04-26

## Target

`labours-go` should be usable anywhere Hercules currently invokes Python `labours`, especially from `../hercules/cmd/hercules/report.go`, without changing user-facing Hercules workflows.

The practical target is:

```bash
hercules --pb <analysis flags> <repo> | labours -f pb -m <mode> -o <plot.png>
hercules report [--all] -o ./report <repo>
```

where `labours` can be the Go binary and produces equivalent files, exits, warnings, and reports for every mode Hercules asks for.

## Current Status Report

The repository is not yet a complete drop-in replacement despite the optimistic README/CLAUDE wording.

Working or partially working pieces already present:

- CLI skeleton with Cobra/Viper and Python-compatible flag names for common plotting flags.
- YAML and protobuf readers for the older/core Hercules payloads.
- Burndown project/file/person plotting paths.
- Core coupling, ownership, overwrites, devs, devs-efforts, old-vs-new, languages, devs-parallel, run-times, shotness, sentiment mode stubs or implementations.
- Theme and chart infrastructure around `gonum/plot`.
- Integration and visual testing scaffolding.
- Example Hercules data under `example_data/` and `data/`.

Critical gaps found during inspection:

- Hercules `report` default modes include modes this repo does not implement: `temporal-activity`, `bus-factor`, `ownership-concentration`, `knowledge-diffusion`, `hotspot-risk`.
- Hercules `report --all` additionally includes `burndown-repository` and `burndown-repos-combined`, also missing here.
- The local `pb.proto` is behind `../hercules/internal/pb/pb.proto`. Missing messages include repository burndown fields, temporal activity, bus factor, ownership concentration, knowledge diffusion, hotspot risk, refactoring proxy, onboarding, code churn, commits, sentiment, imports, and others.
- Reader interface does not expose the data needed by the missing report modes.
- Some implemented modes are semantic approximations, not Python-labours ports. Notable examples: `sentiment` derives heuristic sentiment from dev/language stats instead of reading `CommentSentimentResults`; `devs-parallel` can synthesize data instead of using the same ownership/coupling/devs calculations as Python.
- Coupling modes currently generate Go-native plots/assets, while Python `labours` trains embeddings and writes projector assets unless disabled.
- Several modes assume `output` is a directory and write nested files, but Hercules `report` passes a concrete file path per mode.
- `go test ./...` currently fails: 145 passed, 15 failed, 1 skipped. Failures are in `internal/modes` language output tests and `test/visual` Python compatibility/regression tests.
- README and CLAUDE status claims were corrected in Phase 0 so they no longer describe the project as production-ready.

## Hercules Contract to Match

From `../hercules/cmd/hercules/report.go`:

Default Hercules report analysis flags:

- `burndown`
- `burndown-files`
- `burndown-people`
- `couples`
- `devs`
- `temporal-activity`
- `bus-factor`
- `ownership-concentration`
- `knowledge-diffusion`
- `hotspot-risk`

All-report additionally enables:

- `shotness`
- `sentiment`

Default report modes:

- `burndown-project`
- `burndown-file`
- `burndown-person`
- `overwrites-matrix`
- `ownership`
- `couples-files`
- `couples-people`
- `devs`
- `devs-efforts`
- `old-vs-new`
- `languages`
- `temporal-activity`
- `bus-factor`
- `ownership-concentration`
- `knowledge-diffusion`
- `hotspot-risk`

All report modes additionally include:

- `burndown-repository`
- `burndown-repos-combined`
- `couples-shotness`
- `shotness`
- `sentiment`
- `devs-parallel`

Invocation shape used by Hercules report:

```bash
labours -f pb -i <report.pb> -o <charts>/<mode>.<png|svg> -m <mode> --backend Agg
```

Therefore every mode must accept a single output file path and write that path, or a predictable asset bundle compatible with report asset collection.

## Mode Parity Matrix

| Mode | Current state | Required work |
| --- | --- | --- |
| `burndown-project` | Implemented, Python-compatible path exists | Verify raw/no/month/year resampling, start/end filters, JSON output, image parity. |
| `burndown-file` | Implemented | Ensure output-file behavior matches Python and report expectations for many files. |
| `burndown-person` | Implemented | Verify per-person output naming and date filtering. |
| `burndown-repository` | Missing | Add protobuf/YAML reader support for `repositories` and `repository_sequence`; port Python behavior. |
| `burndown-repos-combined` | Missing | Add combined repository burndown loader/plotter equivalent. |
| `overwrites-matrix` | Implemented | Verify data source: Python uses `Burndown.people_interaction`, not couples; add embedding asset behavior if required. |
| `ownership` | Implemented | Verify against `files_ownership`/people burndown Python logic and `--order-ownership-by-time`. |
| `couples-files` | Implemented differently | Decide compatibility target: projector embeddings/assets, static plots, or both. Ensure report file output works. |
| `couples-people` | Implemented differently | Same as couples-files; verify matrix preprocessing and projector behavior. |
| `couples-shotness` | Partial | Python uses shotness co-occurrence embeddings; protobuf reader currently reports not implemented. |
| `shotness` | Implemented | Verify printed stats and optional output behavior against Python. |
| `sentiment` | Not compatible | Parse `CommentSentimentResults`; port Python chart/stats behavior. Remove heuristic replacement from compatibility path. |
| `temporal-activity` | Missing | Implement reader, mode, chart layout, date filters, legend threshold flags. |
| `devs` | Implemented | Verify aggregate/time-series math, language parsing, `--max-people`, JSON output. |
| `devs-efforts` | Implemented | Verify Python parity and output names. |
| `old-vs-new` | Implemented | Verify against Python resampling and line classification. |
| `languages` | Implemented but tests fail | Fix reader data extraction and output behavior; Python derives language stats from devs data. |
| `devs-parallel` | Approximate | Port Python `load_devs_parallel` and `show_devs_parallel`; remove synthetic fallback from compatibility path. |
| `run-times` | Implemented | Verify text output and JSON behavior. Not used by report. |
| `bus-factor` | Missing | Parse `BusFactorAnalysisResults`; implement chart/stats mode. |
| `ownership-concentration` | Missing | Parse `OwnershipConcentrationResults`; implement Gini/HHI charts and subsystem output. |
| `knowledge-diffusion` | Missing | Parse `KnowledgeDiffusionResults`; implement distribution/time charts. |
| `hotspot-risk` | Missing | Parse `HotspotRiskResults`; implement ranking output/chart. |
| `refactoring-proxy` | Missing locally, in Python CLI | Parse `RefactoringProxyResults`; implement if aiming beyond Hercules report. |

## Phase 0: Baseline and Truth Cleanup

Goal: make progress measurable and stop relying on outdated status text.

Status as of 2026-04-26:

- README and CLAUDE now describe the port as in progress.
- `just build` now builds the drop-in `./labours` binary.
- `compat/README.md` records current fixture provenance gaps and future fixture requirements.
- CI now checks that `PLAN.md` exists and names build artifacts as `labours-*`.
- Current test baseline remains tracked: `go test ./...` reports 145 passed, 15 failed, 1 skipped.

Tasks:

- Replace README/CLAUDE "production-ready" claims with an accurate compatibility status.
- Add this plan to CI expectations and keep it updated as work lands.
- Record the current `go test ./...` failures in an issue or TODO section until fixed.
- Add a small `compat/` or `testdata/hercules/` README describing which `.pb` files came from which Hercules command.
- Decide whether the binary name should be `labours`, `labours-go`, or both. For drop-in use, produce `labours`.

Exit criteria:

- Documentation no longer claims complete parity.
- `just build` creates a binary usable as `labours`.
- Current failing tests are either fixed or explicitly tracked by test name.

## Phase 1: Protocol Buffer Schema Parity

Goal: the Go reader can parse every protobuf payload that current Hercules can emit for report modes.

Tasks:

- Replace local `pb.proto` with the current schema from `../hercules/internal/pb/pb.proto`, preserving Go package options.
- Regenerate `internal/pb/pb.pb.go`.
- Add reader methods and structs for:
  - repository burndown sequence and matrices
  - sentiment by tick
  - temporal activity aggregates and per-tick data
  - bus factor snapshots
  - ownership concentration snapshots
  - knowledge diffusion files/distribution
  - hotspot risk files
  - refactoring proxy
  - commits/file history if needed by downstream modes
- Make parse helpers return typed errors that distinguish "analysis missing" from "malformed payload".
- Add tests that unmarshal sample protobufs generated by current Hercules for each report analysis flag.

Exit criteria:

- A protobuf produced by `../hercules` with `reportDefaultAnalysisFlags` can be read without unknown/missing schema failures.
- Reader tests cover all report payloads.
- No mode needs to guess data that exists explicitly in protobuf.

## Phase 2: CLI Drop-in Compatibility

Goal: command-line behavior matches Python `labours` closely enough for scripts and Hercules report.

Tasks:

- Add missing `-m/--mode` alias behavior while preserving current `--modes` if desired.
- Add missing mode choices and reject unknown modes consistently.
- Add missing flags used by Python:
  - `--temporal-legend-threshold`
  - `--temporal-legend-single-col-threshold`
- Validate supported `--input-format` values: `yaml`, `pb`, `auto`.
- Implement Python-compatible date parsing tolerance where practical, or document accepted date formats and test them.
- Ensure `--backend Agg` is accepted as a rendering backend hint and does not change extension detection incorrectly.
- Preserve Python behavior for no modes, warnings, stdout summaries, and non-fatal missing analyses.
- Normalize output handling:
  - single mode with file path writes that file
  - mode that naturally creates multiple assets writes next to that file with stable names
  - directory output remains supported
  - JSON extension writes data instead of image where Python does

Exit criteria:

- Every invocation shape used by Hercules report succeeds for implemented modes.
- CLI compatibility tests compare help, flag acceptance, missing-data warnings, and output locations.

## Phase 3: Core Report Modes

Goal: `hercules report` default modes produce useful assets with no failures.

Priority order:

1. Fix currently implemented default modes.
2. Implement missing default modes.
3. Align visual/data parity with Python.

Tasks:

- Fix `languages`:
  - derive language totals from `DevsAnalysisResults` ticks/languages
  - support YAML and protobuf consistently
  - fix current output-file test failures
- Fix output-file semantics in coupling and multi-asset modes.
- Verify/fix burndown modes against current Hercules protobuf, especially tick size and matrix orientation.
- Verify/fix ownership and overwrites matrix against Python calculations.
- Port `temporal-activity`:
  - parse `TemporalActivityResults`
  - support aggregate and per-tick formats
  - respect date filters and legend threshold flags
- Port `bus-factor`:
  - parse snapshots/subsystems/threshold/tick size
  - plot time series and subsystem summary
- Port `ownership-concentration`:
  - parse Gini/HHI snapshots and subsystem metrics
  - plot both concentration metrics
- Port `knowledge-diffusion`:
  - parse file diffusion and editor count distribution
  - plot distribution plus optional top files/time trend
- Port `hotspot-risk`:
  - parse file risks
  - plot ranked risk bars/table-like output

Exit criteria:

- `hercules report -o /tmp/report <repo>` using the Go labours binary has zero failed default modes.
- Generated `index.html` references actual chart files for every default mode.
- `go test ./...` passes outside visual parity tests, or visual failures are marked separately with clear reasons.

## Phase 4: All Report Modes

Goal: `hercules report --all` succeeds.

Tasks:

- Implement `burndown-repository`.
- Implement `burndown-repos-combined`.
- Complete `couples-shotness` from real shotness co-occurrence data or define the exact Go equivalent.
- Replace heuristic `sentiment` with a real `CommentSentimentResults` implementation.
- Port `devs-parallel` from Python logic:
  - ownership burndown
  - people co-occurrence
  - devs time series
  - same filtering/max-people behavior
- Decide whether `refactoring-proxy` is in scope for "complete port" even though Hercules report currently does not list it; implement after report-all if yes.

Exit criteria:

- `hercules report --all -o /tmp/report <repo>` has zero failed modes with Go labours.
- `labours -f pb -m all` matches Python `all` mode composition and output behavior.

## Phase 5: Compatibility Test Harness

Goal: prevent regressions and quantify differences from Python labours.

Tasks:

- Add a fixture generator script that runs `../hercules` to create `.pb` files for:
  - default report flags
  - all report flags
  - burndown-only with files/people/repositories
  - couples-only
  - devs-only
  - shotness-only
  - sentiment-only
- Add golden tests for reader extraction, not only rendered pixels.
- For each mode, compare Go extracted data against Python labours intermediate data where possible.
- Split visual tests into:
  - structural tests: file exists, decodes, non-empty, sane dimensions
  - parity tests: compare against Python references with tolerances
- Fix current visual dimension mismatch by making plot size defaults match Python or by making tests compare normalized renderings.
- Add report integration test:
  - build local `labours`
  - run `../hercules report --labours-cmd <local binary> --strict`
  - verify chart count and no failures in index data

Exit criteria:

- `go test ./...` passes.
- A dedicated compatibility suite can be run locally and in CI.
- Visual parity thresholds are documented per mode.

## Phase 6: Output and Asset Parity

Goal: files produced by Go labours are a drop-in replacement for Python labours outputs.

Tasks:

- Define output convention for each mode in a table and enforce it in tests.
- Support PNG and SVG consistently.
- Implement or intentionally disable TensorFlow projector behavior:
  - If implemented: write projector metadata/vector files compatible with Python labours.
  - If not implemented: document that `--disable-projector` is effectively always true and adjust Hercules report expectations if necessary.
- Ensure JSON output is real mode data, not placeholder extraction.
- Make stdout/stderr messages useful but quiet under `--quiet`.
- Remove progress bars from non-interactive/quiet report runs.

Exit criteria:

- Hercules report asset collection finds all intended charts/assets.
- Re-running the same command produces deterministic output file names.
- Missing data warnings match Python closely enough for existing scripts.

## Phase 7: Performance and Robustness

Goal: the Go port handles large repositories better than Python without sacrificing correctness.

Tasks:

- Avoid densifying large sparse matrices unless the mode genuinely needs dense data.
- Add memory benchmarks for Linux-scale burndown and couples payloads.
- Stream or chunk expensive multi-file/person chart generation.
- Add bounds checks and malformed protobuf tests for every sparse matrix parser.
- Audit all direct type assertions in YAML reader; replace panic-prone assertions with checked conversions.
- Make date filtering and resampling efficient on long histories.

Exit criteria:

- Large fixtures run within agreed memory/time limits.
- Fuzz or malformed-input tests cover readers.
- No reader panics on missing optional analyses.

## Phase 8: Packaging and Integration

Goal: users can install and Hercules can discover the Go replacement naturally.

Tasks:

- Build binary as `labours` by default.
- Optionally keep `labours-go` as an alias for development.
- Add install target and release workflow.
- Add version output that includes schema compatibility with Hercules.
- Document how to point Hercules report at this binary:

```bash
hercules report --labours-cmd ./labours -o ./report <repo>
```

- Test discovery through PATH, matching Hercules `resolveLaboursCommand()`.

Exit criteria:

- Fresh checkout can run `just build`, then `../hercules/hercules report --labours-cmd ./labours ...`.
- Release artifact includes Linux/macOS binaries if desired.

## Suggested Near-term Work Order

1. Update `pb.proto` from Hercules and regenerate Go protobuf code.
2. Fix reader extraction for `DevsAnalysisResults` language totals.
3. Fix output-file semantics for `languages` and coupling modes.
4. Add missing CLI flags and `--mode` alias.
5. Implement `temporal-activity`, because it is a default Hercules report mode and has a clear protobuf schema.
6. Implement `bus-factor`, `ownership-concentration`, `knowledge-diffusion`, and `hotspot-risk`.
7. Add the report integration test against `../hercules`.
8. Complete report-all modes.
9. Replace heuristic modes with true Python-compatible ports.
10. Tighten visual parity once data parity is stable.

## Definition of Done

The port is complete when all of these are true:

- `go test ./...` passes.
- `hercules report --strict --labours-cmd <go labours> -o <dir> <repo>` passes.
- `hercules report --all --strict --labours-cmd <go labours> -o <dir> <repo>` passes.
- Every Python labours CLI mode exposed in current `../hercules/python/labours/cli.py` is either implemented or explicitly documented as intentionally unsupported with a non-zero compatibility decision.
- Local `pb.proto` is in sync with current Hercules protobuf schema.
- No implemented mode uses synthetic or heuristic data when Hercules provides the real analysis payload.
- Output paths and generated assets are deterministic and compatible with Hercules report collection.
- Documentation accurately states compatibility status and known limitations.
