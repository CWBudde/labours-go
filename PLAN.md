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

Critical gaps found during inspection and follow-up implementation:

- Hercules `report` default modes originally included missing Go modes: `temporal-activity`, `bus-factor`, `ownership-concentration`, `knowledge-diffusion`, `hotspot-risk`. These now have protobuf-backed implementations and pass the current default-report smoke path.
- Hercules `report --all` additionally includes `burndown-repository` and `burndown-repos-combined`. These are now wired to protobuf repository burndown data, but the current copied Hercules SIVA fixture does not contain repository matrices, so full multi-repository parity still needs a dedicated fixture.
- The local `pb.proto` was behind `../hercules/internal/pb/pb.proto` when this plan started. Phase 1 has synced the schema and regenerated Go bindings; remaining work is using those payloads in modes.
- Reader accessors for default report payloads now exist, and the default report modes consume them. Additional compatibility work remains for parity and fixtures.
- Some implemented modes are semantic approximations, not Python-labours ports. Notable examples: `sentiment` has an explicit `--sentiment-fallback` heuristic path for legacy payloads, and `devs-parallel` has an explicit `--devs-parallel-fallback` synthetic path instead of using the same ownership/coupling/devs calculations as Python.
- Coupling modes currently generate Go-native plots/assets, while Python `labours` trains embeddings and writes projector assets unless disabled.
- The CLI now normalizes output paths before dispatch: single-file modes receive a concrete file path, and multi-asset modes receive the requested directory or the parent directory of a requested file path.
- The default-report protobuf fixture now runs through every default mode without CLI-level mode failures; visual parity remains to be proven.
- `hercules report --all --strict` with the copied SIVA fixture now exits successfully with no "mode not implemented" or hard mode errors from the Go binary. It still prints expected missing-data warnings for repository/file/person burndown analyses absent from that fixture.
- `go test ./...` now passes by default; visual Python compatibility/regression checks are opt-in.
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

| Mode                      | Current state                              | Required work                                                                                                                                          |
| ------------------------- | ------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `burndown-project`        | Implemented, Python-compatible path exists | Verify raw/no/month/year resampling, start/end filters, JSON output, image parity.                                                                     |
| `burndown-file`           | Implemented                                | Ensure output-file behavior matches Python and report expectations for many files.                                                                     |
| `burndown-person`         | Implemented                                | Verify per-person output naming and date filtering.                                                                                                    |
| `burndown-repository`     | Initial implementation                     | Validate with a real multi-repository payload; verify output naming, matrix orientation, and resampling against Python.                                |
| `burndown-repos-combined` | Initial implementation                     | Validate combined matrix semantics with a real multi-repository payload and Python parity fixture.                                                     |
| `overwrites-matrix`       | Implemented                                | Verify data source: Python uses `Burndown.people_interaction`, not couples; add embedding asset behavior if required.                                  |
| `ownership`               | Implemented                                | Verify against `files_ownership`/people burndown Python logic and `--order-ownership-by-time`.                                                         |
| `couples-files`           | Implemented differently                    | Decide compatibility target: projector embeddings/assets, static plots, or both. Ensure report file output works.                                      |
| `couples-people`          | Implemented differently                    | Same as couples-files; verify matrix preprocessing and projector behavior.                                                                             |
| `couples-shotness`        | Partial                                    | Python uses shotness co-occurrence embeddings; protobuf reader now exposes shotness co-occurrence data, but mode/output parity remains.                |
| `shotness`                | Implemented                                | Verify printed stats and optional output behavior against Python.                                                                                      |
| `sentiment`               | Partial                                    | Uses `CommentSentimentResults` by default and gates legacy heuristics behind `--sentiment-fallback`. Validate against a real sentiment payload.        |
| `temporal-activity`       | Basic implementation                       | Improve chart parity, date filters, and legend threshold behavior.                                                                                     |
| `devs`                    | Implemented                                | Verify aggregate/time-series math, language parsing, `--max-people`, JSON output.                                                                      |
| `devs-efforts`            | Implemented                                | Verify Python parity and output names.                                                                                                                 |
| `old-vs-new`              | Implemented                                | Verify against Python resampling and line classification.                                                                                              |
| `languages`               | Implemented                                | Language totals are now derived from Devs ticks for protobuf and compact YAML; temporal chart parity remains.                                          |
| `devs-parallel`           | Approximate                                | NaN failures are guarded and synthetic fallback is explicitly gated, but Python `load_devs_parallel` and `show_devs_parallel` still need to be ported. |
| `run-times`               | Implemented                                | Verify text output and JSON behavior. Not used by report.                                                                                              |
| `bus-factor`              | Basic implementation                       | Improve Python parity and subsystem output.                                                                                                            |
| `ownership-concentration` | Basic implementation                       | Improve Python parity and subsystem output.                                                                                                            |
| `knowledge-diffusion`     | Basic implementation                       | Improve optional top files/time trend parity.                                                                                                          |
| `hotspot-risk`            | Basic implementation                       | Improve table-like output and risk metric parity.                                                                                                      |
| `refactoring-proxy`       | Implemented                                | Parses `RefactoringProxyResults` and writes a rename-ratio chart. Not used by Hercules report.                                                         |

## Phase 0: Baseline and Truth Cleanup

Goal: make progress measurable and stop relying on outdated status text.

Status as of 2026-04-26:

- README and CLAUDE now describe the port as in progress.
- `just build` now builds the drop-in `./labours` binary.
- `compat/README.md` records current fixture provenance gaps and future fixture requirements.
- CI now checks that `PLAN.md` exists and names build artifacts as `labours-*`.
- Current test baseline remains tracked: `go test ./...` reports 145 passed, 15 failed, 1 skipped.

Tasks:

- [x] Replace README/CLAUDE "production-ready" claims with an accurate compatibility status.
- [x] Add this plan to CI expectations and keep it updated as work lands.
- [x] Record the current `go test ./...` failures in an issue or TODO section until fixed.
- [x] Add a small `compat/` or `testdata/hercules/` README describing which `.pb` files came from which Hercules command.
- [x] Decide whether the binary name should be `labours`, `labours-go`, or both. For drop-in use, produce `labours`.

Acceptance criteria:

- [x] Documentation no longer claims complete parity.
- [x] `just build` creates a binary usable as `labours`.
- [x] Current failing tests are either fixed or explicitly tracked by test name.

## Phase 1: Protocol Buffer Schema Parity

Goal: the Go reader can parse every protobuf payload that current Hercules can emit for report modes.

Status as of 2026-04-26:

- Local `pb.proto` has been synced from `../hercules/internal/pb/pb.proto` with Go package options.
- `internal/pb/pb.pb.go` has been regenerated from the synced schema.
- `ProtobufReader` now exposes typed accessors for repository burndown, sentiment, temporal activity, bus factor, ownership concentration, knowledge diffusion, hotspot risk, and refactoring proxy payloads.
- `internal/readers/report_payloads_test.go` verifies unmarshalling, typed missing/malformed errors, and accessor conversion for the current Hercules report payload shapes using synthetic protobuf messages.
- `test/testdata/hercules/report_default.pb` is a real current-Hercules protobuf fixture for the default report analysis flag set.
- `test/testdata/hercules/shotness.pb` is a real current-Hercules protobuf fixture for the `shotness` report-all analysis.
- `test/testdata/hercules/sentiment.pb` is a synthetic non-empty `CommentSentimentResults` fixture for reader and plot parity. A TensorFlow-enabled neighboring Hercules binary is now available locally, but small real fixture runs produced empty `sentiment_by_tick` payloads.
- Protobuf developer stats now aggregate explicit `Devs` tick commits, line stats, and language stats instead of returning zero-filled synthetic placeholders.
- `go test ./internal/readers` passes.
- Current full test baseline after Phase 1 cleanup: `go test ./...` reports 268 passed in 12 packages.

Tasks:

- [x] Replace local `pb.proto` with the current schema from `../hercules/internal/pb/pb.proto`, preserving Go package options.
- [x] Regenerate `internal/pb/pb.pb.go`.
- [x] Add reader methods and structs for repository burndown sequence and matrices.
- [x] Add reader methods and structs for sentiment by tick.
- [x] Add reader methods and structs for temporal activity aggregates and per-tick data.
- [x] Add reader methods and structs for bus factor snapshots.
- [x] Add reader methods and structs for ownership concentration snapshots.
- [x] Add reader methods and structs for knowledge diffusion files/distribution.
- [x] Add reader methods and structs for hotspot risk files.
- [x] Add reader methods and structs for refactoring proxy.
- [x] Add reader methods and structs for commits/file history if needed by downstream modes.
- [x] Make parse helpers return typed errors that distinguish "analysis missing" from "malformed payload".
- [x] Add tests that unmarshal synthetic current-Hercules-shaped protobufs for report payload accessors.
- [x] Add tests that unmarshal a current-Hercules default report protobuf fixture.
- [x] Add tests that unmarshal a current-Hercules `shotness` protobuf fixture.
- [x] Add or document a current-Hercules `sentiment` protobuf fixture. Current coverage uses a synthetic non-empty `CommentSentimentResults` payload until a compact real repository produces non-empty sentiment ticks.

Acceptance criteria:

- [x] A protobuf produced by `../hercules` with `reportDefaultAnalysisFlags` can be read without unknown/missing schema failures.
- [x] Reader tests cover all default report payload shapes with synthetic protobuf fixtures.
- [x] Reader tests cover default report payloads with a real `../hercules` fixture.
- [x] Reader tests cover `shotness` report-all payloads with a real `../hercules` fixture.
- [x] Reader tests cover `sentiment` report-all payloads with synthetic current-Hercules-shaped data and parity coverage exercises the synthetic `sentiment.pb` fixture.
- [x] No mode needs to guess data that exists explicitly in protobuf.

## Phase 2: CLI Drop-in Compatibility

Goal: command-line behavior matches Python `labours` closely enough for scripts and Hercules report.

Status as of 2026-04-26:

- `--mode` is registered as a Python-compatible alias alongside existing `--modes`/`-m`.
- `--temporal-legend-threshold` and `--temporal-legend-single-col-threshold` are registered.
- Mode parsing now handles repeated values and comma-separated values through one testable resolver.
- Known Python/Hercules mode names are validated before input files are read; unknown modes fail early.
- `--input-format` is validated as `auto`, `yaml`, or `pb` before input files are read.
- `--backend Agg` is treated as a rendering backend hint and leaves output extension detection to the requested output path.
- Output planning now preserves a single-mode file path, expands directory output to a per-mode file path, makes multi-mode file output use sibling per-mode files, and passes multi-asset modes a directory so their assets are written next to the requested file path.
- `languages` directory output now writes `languages.png` and `languages.svg`, fixing the prior `internal/modes` language output failures.
- No-mode invocations are accepted as a Python-compatible no-op after input parsing.
- Missing analyses now print Python-style guidance warnings and continue instead of reporting hard mode errors.
- `.json` output now serializes reader data directly for CLI modes instead of rendering charts through a temporary output path.
- Current full test baseline after Phase 2 CLI work: `go test ./...` reports 201 passed, 12 failed, 1 skipped. Remaining failures are the pre-existing visual compatibility failures.

Tasks:

- [x] Add missing `-m/--mode` alias behavior while preserving current `--modes` if desired.
- [x] Add missing mode choices and reject unknown modes consistently.
- [x] Add `--temporal-legend-threshold`.
- [x] Add `--temporal-legend-single-col-threshold`.
- [x] Validate supported `--input-format` values: `yaml`, `pb`, `auto`.
- [x] Implement Python-compatible date parsing tolerance where practical, or document accepted date formats and test them.
- [x] Ensure `--backend Agg` is accepted as a rendering backend hint and does not change extension detection incorrectly.
- [x] Preserve Python behavior for no modes, warnings, stdout summaries, and non-fatal missing analyses.
- [x] Normalize single-mode output so a file path writes that file.
- [x] Normalize multi-asset mode output so assets are written next to the requested file path with stable names.
- [x] Keep directory output supported.
- [x] Ensure JSON extension writes real data instead of an image where Python does.

Acceptance criteria:

- [x] Every invocation shape used by Hercules report succeeds for implemented modes at the CLI/output dispatch layer.
- [x] CLI compatibility tests compare important help/flag registration.
- [x] CLI compatibility tests compare flag acceptance.
- [x] CLI compatibility tests compare missing-data warnings.
- [x] CLI compatibility tests compare output locations.

## Phase 3: Core Report Modes

Goal: `hercules report` default modes produce useful assets with no failures.

Status as of 2026-04-26:

- `languages` now derives totals from `DevsAnalysisResults` tick language stats for protobuf input.
- YAML dev time-series parsing now supports current compact Hercules tick entries of the form `[commits, added, removed, changed, languages]`.
- Basic single-file plot modes are wired for `temporal-activity`, `bus-factor`, `ownership-concentration`, `knowledge-diffusion`, and `hotspot-risk`.
- The real default report fixture runs through the full default mode list with the local `./labours` binary and writes chart/assets under `/tmp/labours-go-phase3-default`.
- `hercules report --strict --labours-cmd ./labours` succeeds on the copied Hercules SIVA fixture and generates `index.html`, `report.pb`, and default chart assets under `/tmp/labours-go-hercules-report-default`.
- `temporal-activity` now uses per-tick data for date-filtered hour aggregation when `--start-date`/`--end-date` are supplied.
- `bus-factor` now writes a subsystem summary chart next to the main timeline.
- `knowledge-diffusion` now writes distribution, knowledge-silo, and trend charts.
- `hotspot-risk` now writes the ranked risk chart plus a TSV table and text summary.
- Current full test baseline after Phase 3 work: `go test ./...` reports 201 passed, 12 failed, 1 skipped. Remaining failures are the pre-existing visual compatibility failures.

Priority order:

1. Fix currently implemented default modes.
2. Implement missing default modes.
3. Align visual/data parity with Python.

Tasks:

- [x] Fix `languages` to derive language totals from `DevsAnalysisResults` ticks/languages.
- [x] Fix `languages` to support YAML and protobuf consistently.
- [x] Fix current `languages` output-file test failures.
- [x] Fix CLI output-file semantics in coupling and multi-asset modes.
- [ ] Verify/fix burndown modes against current Hercules protobuf tick size.
- [ ] Verify/fix burndown modes against current Hercules protobuf matrix orientation.
- [ ] Verify/fix ownership against Python calculations.
- [ ] Verify/fix overwrites matrix against Python calculations.
- [x] Port `temporal-activity` mode using `TemporalActivityResults`.
- [x] Make `temporal-activity` support aggregate and per-tick formats.
- [x] Make `temporal-activity` respect date filters and legend threshold flags.
- [x] Port `bus-factor` mode using snapshots/subsystems/threshold/tick size.
- [x] Make `bus-factor` plot time series and subsystem summary.
- [x] Port `ownership-concentration` mode using Gini/HHI snapshots and subsystem metrics.
- [x] Make `ownership-concentration` plot both concentration metrics.
- [x] Port `knowledge-diffusion` mode using file diffusion and editor count distribution.
- [x] Make `knowledge-diffusion` plot distribution plus optional top files/time trend.
- [x] Port `hotspot-risk` mode using file risks.
- [x] Make `hotspot-risk` plot ranked risk bars/table-like output.

Acceptance criteria:

- [x] `hercules report -o /tmp/report <repo>` using the Go labours binary has zero failed default modes.
- [x] Generated `index.html` references actual chart files for every default mode.
- [x] `go test ./...` passes outside visual parity tests, or visual failures are marked separately with clear reasons.

## Phase 4: All Report Modes

Goal: `hercules report --all` succeeds.

Status as of 2026-04-26:

- `burndown-repository` and `burndown-repos-combined` are registered as real mode handlers.
- `burndown-repository` writes one chart per repository to the requested report chart directory when repository matrices are available.
- `burndown-repos-combined` sums repository matrices and writes the requested combined chart path.
- The current copied Hercules SIVA fixture does not include repository burndown matrices, so both repository modes now report Python-style missing-data warnings instead of "Mode not implemented yet".
- `couples-shotness` now builds its Go-native coupling matrix from real shotness record counter overlap for both YAML and protobuf input, and the current Hercules `shotness.pb` fixture covers the protobuf path.
- `sentiment` now requires real `CommentSentimentResults` protobuf data by default and gates the legacy developer/language heuristic behind explicit `--sentiment-fallback`.
- `sentiment` and `devs-parallel` now sanitize zero/empty values so `gonum/plot` no longer rejects NaN bar data.
- `devs-parallel` no longer synthesizes data by default when people burndown is missing; the legacy synthetic path requires explicit `--devs-parallel-fallback`, and the current Go analysis respects `--max-people`.
- `devs-parallel` now loads the same required data groups as Python `load_devs_parallel`: ownership burndown, people co-occurrence, and dev time-series. It ranks developers by commits, lines, ownership, coupling, and commit co-occurrence.
- `refactoring-proxy` is in scope for the complete port and is now implemented as a protobuf-backed chart mode, although Hercules report does not invoke it.
- `labours -f pb -m all` expands to Python's `all` mode composition and the direct internal `all` handler uses the same list and output planning.
- `hercules report --all --strict --labours-cmd ./labours` exits 0 on `/tmp/labours-go-hercules.siva` and writes report assets under `/tmp/labours-go-hercules-report-all-phase4`.
- Current full test baseline after Phase 4 devs-parallel/refactoring-proxy work: `go test ./...` reports 217 passed, 11 failed, 1 skipped. Remaining failures are the pre-existing visual compatibility failures.
- Remaining Phase 4 work is fixture-gated: TensorFlow-enabled Hercules is available locally, but compact real runs produced empty sentiment ticks. Real sentiment validation still needs a small repository fixture that emits non-empty `sentiment_by_tick`, and zero-warning report-all coverage still needs a richer report-all fixture with repository burndown, people burndown, and sentiment payloads.

Tasks:

- [x] Implement `burndown-repository`.
- [x] Implement `burndown-repos-combined`.
- [x] Complete `couples-shotness` from real shotness co-occurrence data or define the exact Go equivalent.
- [x] Prefer real `CommentSentimentResults` in `sentiment` when protobuf data is present.
- [ ] Validate `sentiment` with a real current-Hercules sentiment protobuf fixture. Blocked locally: TensorFlow-enabled Hercules runs on the available compact inputs produced empty sentiment ticks.
- [x] Remove or explicitly gate heuristic `sentiment` fallback from the strict compatibility path.
- [x] Guard `sentiment` against NaN bar values on zero-activity fallback data.
- [x] Guard `devs-parallel` against NaN bar values on zero-activity fallback data.
- [x] Port `devs-parallel` ownership burndown logic from Python.
- [x] Port `devs-parallel` people co-occurrence logic from Python.
- [x] Port `devs-parallel` devs time-series logic from Python.
- [x] Port `devs-parallel` filtering/max-people behavior from Python.
- [x] Decide whether `refactoring-proxy` is in scope for "complete port" even though Hercules report currently does not list it.
- [x] Implement `refactoring-proxy` after report-all if it is in scope.

Acceptance criteria:

- [x] `hercules report --all --strict -o /tmp/report <repo>` has zero hard mode errors with Go labours on the copied Hercules SIVA fixture.
- [ ] `hercules report --all -o /tmp/report <repo>` has zero missing-data warnings with a fixture that includes repository burndown, people burndown, and sentiment payloads. Blocked locally: no such real fixture exists under `test/testdata/hercules/`, and compact TensorFlow-enabled sentiment runs produced empty sentiment ticks.
- [x] `labours -f pb -m all` matches Python `all` mode composition.
- [x] `labours -f pb -m all` matches Python `all` mode output behavior.

## Phase 5: Compatibility Test Harness

Goal: prevent regressions and quantify differences from Python labours.

Status as of 2026-04-26:

- `scripts/generate_hercules_fixtures.sh` now regenerates protobuf fixtures from a neighboring `../hercules` checkout, with optional `report_all` output so incomplete local fixture sets do not block the rest of the fixture set.
- `just fixtures` wraps fixture generation and `test/testdata/hercules/README.md` documents the generator, environment overrides, and extraction golden refresh command.
- Reader extraction goldens now cover the checked-in current-Hercules default report and shotness fixtures via `report_default_summary.golden.json` and `shotness_summary.golden.json`.
- Visual tests are split so structural image validation runs by default, while golden and Python parity comparisons are opt-in through `LABOURS_GO_VISUAL_PARITY=1` and `LABOURS_GO_PYTHON_PARITY=1`.
- `just test-visual-parity` runs the opt-in visual parity checks explicitly.
- An opt-in Hercules report integration smoke test builds the local `labours` binary and runs `../hercules report --labours-cmd <local binary> --strict`, then verifies `index.html` and chart asset count.
- Current full test baseline after the output convention work: `go test ./...` passes with 257 tests.

Tasks:

- [x] Add a fixture generator script that runs `../hercules`.
- [x] Generate default report flags fixture.
- [ ] Generate all report flags fixture.
- [ ] Generate burndown-only fixture with files/people/repositories.
- [ ] Generate couples-only fixture.
- [ ] Generate devs-only fixture.
- [x] Generate shotness-only fixture.
- [x] Generate sentiment-only fixture. Current fixture is synthetic and non-empty; replace it when a compact real Hercules input emits sentiment ticks.
- [x] Add golden tests for reader extraction, not only rendered pixels.
- [ ] Compare Go extracted data against Python labours intermediate data where possible for each mode.
- [x] Split visual tests into structural tests: file exists, decodes, non-empty, sane dimensions.
- [x] Split visual tests into parity tests: compare against Python references with tolerances.
- [ ] Fix current visual dimension mismatch by matching Python plot size defaults or comparing normalized renderings.
- [x] Add report integration test that builds local `labours`.
- [x] Add report integration test that runs `../hercules report --labours-cmd <local binary> --strict`.
- [x] Add report integration test that verifies chart count and no failures in index data.

Acceptance criteria:

- [x] `go test ./...` passes.
- [x] A dedicated compatibility suite can be run locally.
- [ ] A dedicated compatibility suite can be run in CI.
- [x] Visual parity thresholds are documented per mode.

## Phase 6: Output and Asset Parity

Goal: files produced by Go labours are a drop-in replacement for Python labours outputs.

Status as of 2026-04-27:

- `README.md` now defines the output convention for every implemented mode.
- `cmd` has a central output convention table used by the output planner, and tests require every implemented mode to have a documented convention.
- `burndown-file` is now planned as a basename fanout mode instead of a directory-style asset mode, matching its handler behavior and preserving requested file basenames.
- Directory-style chart modes now write SVG companions next to PNG assets, and report metric companion assets are tested for both PNG and SVG output paths.

Tasks:

- [x] Define output convention for each mode in a table.
- [x] Enforce output convention for each mode in tests.
- [x] Support PNG consistently across modes.
- [x] Support SVG consistently across modes.
- [x] Decide whether TensorFlow projector behavior is implemented or intentionally disabled. Decision: intentionally disabled — Go port never trains TF embeddings; `couples-*` modes still emit the same `*_vocabulary.tsv` / `*_vectors.tsv` / `*_metadata.tsv` asset filenames so Hercules report asset collection still finds them.
- [ ] If projector behavior is implemented, write projector metadata/vector files compatible with Python labours.
- [x] If projector behavior is not implemented, document that `--disable-projector` is effectively always true and adjust Hercules report expectations if necessary. Documented in README "Compatibility Notes".
- [ ] Ensure JSON output is real mode data, not placeholder extraction.
- [ ] Make stdout/stderr messages useful but quiet under `--quiet`.
- [ ] Remove progress bars from non-interactive/quiet report runs.

Acceptance criteria:

- [ ] Hercules report asset collection finds all intended charts/assets.
- [ ] Re-running the same command produces deterministic output file names.
- [ ] Missing data warnings match Python closely enough for existing scripts.

## Phase 7: Performance and Robustness

Goal: the Go port handles large repositories better than Python without sacrificing correctness.

Tasks:

- [ ] Avoid densifying large sparse matrices unless the mode genuinely needs dense data.
- [ ] Add memory benchmarks for Linux-scale burndown payloads.
- [ ] Add memory benchmarks for Linux-scale couples payloads.
- [ ] Stream or chunk expensive multi-file/person chart generation.
- [ ] Add bounds checks for every sparse matrix parser.
- [ ] Add malformed protobuf tests for every sparse matrix parser.
- [ ] Audit all direct type assertions in YAML reader.
- [ ] Replace panic-prone YAML reader assertions with checked conversions.
- [ ] Make date filtering efficient on long histories.
- [ ] Make resampling efficient on long histories.

Acceptance criteria:

- [ ] Large fixtures run within agreed memory/time limits.
- [ ] Fuzz or malformed-input tests cover readers.
- [ ] No reader panics on missing optional analyses.

## Phase 8: Packaging and Integration

Goal: users can install and Hercules can discover the Go replacement naturally.

Tasks:

- [x] Build binary as `labours` by default.
- [x] Optionally keep `labours-go` as an alias for development.
- [x] Add install target.
- [ ] Add release workflow.
- [x] Add version output that includes schema compatibility with Hercules.
- [x] Document how to point Hercules report at this binary:

```bash
hercules report --labours-cmd ./labours -o ./report <repo>
```

- [ ] Test discovery through PATH, matching Hercules `resolveLaboursCommand()`.

Acceptance criteria:

- [ ] Fresh checkout can run `just build`, then `../hercules/hercules report --labours-cmd ./labours ...`.
- [ ] Release artifact includes Linux binaries if desired.
- [ ] Release artifact includes macOS binaries if desired.

## Phase 9: Side-by-Side Python Parity on a Real Repository

Goal: close the concrete divergences observed by running both backends against
`/mnt/projekte/Code/Coda/system-optimiser-core` (1,994 commits, 7.7 years, C++23)
on 2026-05-24. The end-to-end harness is `scripts/full_showcase.sh`; the
artifacts and side-by-side diff are at
`analysis_results/system-optimiser-core/{plots-go,plots-python,compare}`.
Open `compare/index.html` for the visual record.

Headline result of the baseline run: **labours-go 21 OK / 0 failed**,
**Python labours 15 OK / 5 failed**. RMSE between matched PNGs ranges from
0.11 (`ownership`) to 0.50 (`overwrites-matrix`); no matched pair is
pixel-identical. Most divergences are filename/composition gaps rather than
data gaps, but a few are real.

### 9.a Filename and composition parity

Many modes produce *different artifact filenames* between Go and Python.
Hercules `report` and the parity viewer match by filename, so divergent
names break automated comparison even when the underlying data is correct.
Pick one canonical set per mode (prefer the Python names, since Hercules
report expects them) and emit the same files from labours-go.

**Python convention** — single-mode invocation (`-o $BASE.png`):

- Single-plot modes (`burndown-project`, `ownership`, `old-vs-new`,
  `devs-efforts`, `devs-parallel`, `devs`, `languages`, `refactoring-proxy`,
  `overwrites-matrix`, …) write directly to `args.output`.
- Multi-plot modes (`bus-factor`, `ownership-concentration`,
  `knowledge-diffusion`, `temporal-activity`) split the base name and write
  *sibling* files: `$BASE_{suffix}{ext}`. No subdirectory.
- The `--mode all` invocation uses a different convention
  (`get_plot_path()` writes into a subdir derived from the base) — that
  case is what Hercules `report --all` consumes. For now Phase 9.a targets
  the single-mode convention since that is what the showcase exercises.
- SVG sidecars: Python does not emit them; labours-go currently does.
  Drop the SVG companions to match Python (or gate them behind `--svg`).

| Mode                      | labours-go emits today                                                  | Python emits (single-mode `-o $BASE.png`)                                              | Action                                                                                                | Status |
| ------------------------- | ----------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- | ------ |
| `old-vs-new`              | `$BASE.png` (matches)                                                   | `$BASE.png` (single file, no transform)                                                | —                                                                                                     | ✅ done |
| `devs-efforts`            | `$BASE.png` (scatter; time-series TODO)                                 | `$BASE.png` (single time-series of efforts)                                            | Time-series implementation still needed; filename parity done.                                        | ✅ filename done, ⚠️ content TODO |
| `devs-parallel`           | `$BASE.png` (parallel-activity timeline)                                | `$BASE.png` (parallel-coordinates "Developers")                                        | Picked the closest existing Go plot. True parallel-coordinates port TODO.                              | ✅ filename done, ⚠️ content TODO |
| `run-times`               | `$BASE.png` (breakdown bar; text summary stdout)                        | no PNG (stdout table only)                                                              | Go emits one PNG when `-o` supplied; matches text behaviour.                                          | ✅ done |
| `bus-factor`              | `$BASE_timeline.png` + `$BASE_subsystems.png`                           | `$BASE_gauge.png` + `$BASE_timeline.png` + `$BASE_subsystems.png`                      | Gauge panel (big colored BF + line-ownership pie) still TODO; renames + timeline done.                | ⚠️ 2/3 — `_gauge` TODO |
| `ownership-concentration` | `$BASE_timeline.png` + `$BASE_subsystems.png`                           | `$BASE_timeline.png` + `$BASE_subsystems.png`                                          | —                                                                                                     | ✅ done |
| `knowledge-diffusion`     | `$BASE_distribution.png` + `$BASE_silos.png` + `$BASE_lorenz.png`       | `$BASE_distribution.png` + `$BASE_silos.png` + `$BASE_lorenz.png`                      | Lorenz curve implemented with trapezoidal-rule Gini. `_trend` removed (Go-only, gated behind future flag). | ✅ done |
| `temporal-activity`       | `$BASE.png` (single composite)                                          | `$BASE_{weekdays,hours,months,weeks}_{commits,lines}.png` (8) + `$BASE_heatmap_{commits,lines}.png` (2) | Stop writing single composite. Emit the 10 sibling files Python emits.                                | ❌ TODO — biggest scope |

Acceptance: for every mode in the showcase, `ls plots-go/<mode>*` and
`ls plots-python/<mode>*` produce the same set of basenames. The Go-only
auxiliary plots (`devs-efforts` productivity ranking, `devs-parallel`
developer concurrency, `run-times` percentage pie, `knowledge-diffusion`
trend) remain in the codebase behind `//nolint:unused` and are tracked for
re-exposure via mode-specific `--<mode>-detail` flags.

### 9.b Visual parity within matched modes

For every mode that already produces a same-named file in both backends,
`compare -metric RMSE` reports the following on the baseline dataset:

| Mode                       | RMSE  | Likely cause                                                                                |
| -------------------------- | ----- | ------------------------------------------------------------------------------------------- |
| `ownership`                | 0.112 | Colour map / legend placement.                                                              |
| `devs`                     | 0.116 | Per-developer line widths and legend ordering.                                              |
| `burndown-project`         | 0.144 | Resampling smoothing + axis label fonts.                                                    |
| `refactoring-proxy`        | 0.144 | Threshold band tints differ from Python defaults.                                           |
| `knowledge-diffusion_silos`| 0.213 | Bar order, palette, label truncation.                                                       |
| `bus-factor_subsystems`    | 0.335 | Subsystem ordering and per-row colour.                                                      |
| `languages`                | 0.429 | Stack order of language layers; Python sorts differently and uses tableau palette.          |
| `overwrites-matrix`        | 0.503 | Most divergent: heatmap scale, colour ramp, and tick labels all differ.                     |

Acceptance: every matched-name pair on this dataset gets RMSE ≤ 0.10 with
the existing `cmd/parityviewer` toolchain. Track per-mode progress against
the table above.

### 9.c Modes where Python currently fails

Python labours fails on five modes against this real-world dataset; each is
either a Python-side limitation or an unsupported analysis. Decide per-mode
whether to keep Go-only support or to gate Go behind the same constraint.

- **`overwrites-matrix`, `couples-files`, `couples-people`** — Python raises
  `ImportError: TensorFlow is required for training embeddings`. labours-go
  already runs the analysis without TensorFlow. Action: document the
  intentional divergence; keep Go behaviour but ensure the produced plot is
  visually faithful to the Python output *when* TF is installed (use the
  `.venv` plus `uv pip install tensorflow` path for a golden reference run).
- **`hotspot-risk`** — Python prints `there is no registered PB decoder for
  HotspotRisk`. The analysis is labours-go-only. Action: contribute the
  decoder upstream to `../hercules/python/labours/_pb.py` (or document as a
  permanent Go-only mode). Until then `hotspot-risk` has no parity baseline.
- **`burndown-repos-combined`** — Python raises `ValueError: No repository
  data available` on single-repo input. labours-go now returns the same
  `No repository data available` message and exits non-zero, matching
  Python's strictness (was: "silently writes an empty PNG").
- **`devs-parallel`** — Python's `load_devs_parallel` reads
  `couples_people_data.tsv` from `args.tmpdir`, so it can only run *after*
  `couples-people`. Action: either (a) port the dependency into labours-go's
  in-process pipeline so `devs-parallel` works standalone (already the
  case — keep that), and update `scripts/full_showcase.sh` to share a
  tmpdir between `couples-people` and `devs-parallel` in Python so the
  Python comparison has data.

### 9.d Tooling

The showcase harness exists; lock it in as the regression surface.

- [x] `scripts/full_showcase.sh` runs hercules once + both labours backends
  + per-mode logs, with `BACKEND={go|python|both}`, `PEOPLE_DICT=…`,
  `ENABLE_SHOTNESS`, `ENABLE_SENTIMENT`, `SKIP_HERCULES` knobs.
- [x] `just showcase <repo>` recipe.
- [x] `just showcase-compare <repo-name>` recipe launches `cmd/parityviewer`
  with `--baseline-dir plots-python --artifact-dir plots-go`.
- [x] Static `compare/index.html` with side-by-side and diff PNGs.
- [ ] Wire `system-optimiser-core` into CI as a nightly job that fails when
  any matched-name RMSE regresses, when any Go-only artifact disappears,
  or when any Python-supported mode starts failing in Go.
- [ ] Add a small fixture repo (~50 commits, 2 authors, mixed languages) so
  the same comparison can run in <30 s on every PR.

### 9.e Author identity hygiene

The baseline run merged 9 raw signatures into 3 canonical names via
`raw/people-dict.txt`. Real-world repos almost always need this.

- [x] Document the `--people-dict` workflow in the README, with the
  `analysis_results/system-optimiser-core/raw/people-dict.txt` file as a
  worked example.
- [ ] Consider auto-loading the repo's `.mailmap` in labours-go's manifest
  output so users see exactly which identities were merged.

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
11. Address Phase 9.a filename divergences (cheap, unblocks automated parity).
12. Walk the Phase 9.b RMSE table top-to-bottom — start with `ownership`/`devs`/`burndown-project` (cheap), end with `overwrites-matrix` (largest delta).

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
