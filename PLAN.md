# Plan: `labours-go` Visual Parity with Python `labours`

Date: 2026-04-26 · Rebalanced: 2026-05-25

## Target

`labours-go` should be a drop-in replacement for Python `labours` wherever Hercules invokes it:

```bash
hercules --pb <flags> <repo> | labours -f pb -m <mode> -o <plot.png>
hercules report [--all] --labours-cmd ./labours -o ./report <repo>
```

The structural drop-in works today. **The remaining work is almost entirely visual:** the Go binary runs every report mode and writes the expected paths, but the rendered plots still do not *look like* Python's on real repositories. This document is now organized around closing that gap.

Parity bar: **perceptual first, RMSE later.** A mode is "good" when its chart reads as equivalent to Python's at a glance — same composition, file set, palette, and layout. Driving pixel RMSE to zero is a tracked goal, not a release gate (font/DPI rendering between `matplotlib-go` and real matplotlib will never be pixel-identical).

## Completed Foundation

Phases 0–8 of the original plan are done and stable. Summary of what no longer needs active work:

- Docs corrected (no false "production-ready" claims); `just build` produces a `./labours` drop-in binary; `install` and `version` (with Hercules schema compatibility) targets exist.
- `pb.proto` synced from `../hercules/internal/pb/pb.proto` and `internal/pb/pb.pb.go` regenerated.
- `ProtobufReader` exposes typed accessors for every report payload (repository burndown, sentiment, temporal activity, bus factor, ownership concentration, knowledge diffusion, hotspot risk, refactoring proxy); parse helpers return typed missing-vs-malformed errors.
- CLI parity: `-m/--mode` alias, mode validation, `--input-format` validation, `--backend Agg` hint, temporal legend thresholds, Python-style missing-data warnings, JSON output writes real data.
- Output planning + a central convention table: single-file modes write the requested path; multi-asset modes fan out to stable sibling/dir names; PNG and SVG both supported.
- Default and `--all` report modes run with **zero hard mode errors** on the copied Hercules SIVA fixture (`hercules report [--all] --strict --labours-cmd ./labours`).
- TensorFlow projector is intentionally **not** implemented; `couples-*` still emit the `*_vocabulary.tsv` / `*_vectors.tsv` / `*_metadata.tsv` asset names so Hercules report asset collection succeeds.
- Reader extraction goldens + structural visual tests pass by default; Python-parity visual checks are opt-in (`LABOURS_GO_VISUAL_PARITY=1`, `LABOURS_GO_PYTHON_PARITY=1`).

These are stable scaffolding. **Everything below is the active work.**

## Mode Parity Matrix

Renderer: which library draws the plot today (`mpl` = `matplotlib-go`, `gonum` = `gonum/plot`, `mixed` = both in the mode path). File-set: does Go emit the same basenames Python does for single-mode `-o $BASE.png`. Perceptual: RMSE on `system-optimiser-core` where a matched pair exists, else the gap.

| Mode | Renderer | File-set vs Python | Perceptual state |
| --- | --- | --- | --- |
| `burndown-project` | gonum | ✅ | RMSE 0.144 |
| `burndown-file` | gonum | ✅ (dir fanout) | not yet compared per-file |
| `burndown-person` | gonum | ✅ (dir fanout) | not yet compared per-file |
| `burndown-repository` | gonum | ✅ | needs multi-repo fixture |
| `burndown-repos-combined` | gonum | ✅ | Python errors on single-repo (intentional match) |
| `ownership` | mpl | ✅ | RMSE 0.112 (closest) |
| `overwrites-matrix` | mixed | ✅ | RMSE 0.503 (worst); Python needs TF |
| `couples-files` | mpl heatmap | ✅ (dir) | Python needs TF |
| `couples-people` | mpl heatmap | ✅ (dir) | Python needs TF |
| `couples-shotness` | gonum | ✅ (dir) | shotness-gated; not compared |
| `shotness` | gonum | ✅ | not compared |
| `sentiment` | gonum | ✅ | no real fixture (empty ticks) |
| `temporal-activity` | mixed | ❌ 1 composite vs **10** siblings | biggest file-set gap |
| `devs` | mpl | ✅ | RMSE 0.116 |
| `devs-efforts` | gonum | ✅ | dims differ; scatter vs time-series |
| `old-vs-new` | mpl | ✅ | RMSE 0.216 |
| `languages` | mpl | ✅ | RMSE 0.429 (stack order/palette) |
| `devs-parallel` | gonum | ✅ | Go-only (Python needs couples TSV); content TODO |
| `run-times` | mpl | Go emits PNG, Python text-only | intentional extra |
| `bus-factor` | mixed | ❌ missing `_gauge` panel | subsystems RMSE 0.335 |
| `ownership-concentration` | mixed | ✅ (`_timeline`,`_subsystems`) | dims differ |
| `knowledge-diffusion` | mixed | ✅ (`_distribution`,`_silos`,`_lorenz`) | silos 0.213, distribution 0.184; lorenz dims differ |
| `hotspot-risk` | mpl | Go-only (no Python PB decoder) | no parity baseline |
| `refactoring-proxy` | mpl | ✅ | RMSE 0.144 |

## Current Reality: the Visual Parity Gap

On the canonical dataset `system-optimiser-core` (1,994 commits, C++, HEAD `b7f510a`), the latest showcase run is **labours-go 21 OK / 0 failed**, **Python 15 OK** (sentiment/shotness disabled in that run). No matched pair is pixel-identical. The concrete divergences:

- **Wrong file set (blocks automated comparison):**
  - `temporal-activity` — Go writes 1 composite; Python writes 10: `_{weekdays,hours,months,weeks}_{commits,lines}.png` (8) + `_heatmap_{commits,lines}.png` (2).
  - `bus-factor` — Go writes `_timeline` + `_subsystems`; Python also writes `_gauge` (big colored BF number + line-ownership pie).
- **Dimension mismatches (matched names, different canvas):** `bus-factor_timeline`, `devs-efforts`, `knowledge-diffusion_lorenz`, `ownership-concentration_{timeline,subsystems}`.
- **RMSE offenders (matched pairs):** `overwrites-matrix` 0.503, `languages` 0.429, `bus-factor_subsystems` 0.335, `old-vs-new` 0.216, `knowledge-diffusion_silos` 0.213, `knowledge-diffusion_distribution` 0.184, `burndown-project` 0.144, `refactoring-proxy` 0.144, `devs` 0.116, `ownership` 0.112.
- **Intentional Python-side gaps — keep Go behavior, document, do not "fix":**
  - TF-only in Python: `couples-files`, `couples-people`, `overwrites-matrix` (Python raises `ImportError: TensorFlow required`). For a golden reference, run Python with `uv pip install tensorflow` in the `.venv`.
  - `hotspot-risk` — no Python PB decoder; Go-only. Optionally contribute the decoder upstream to `../hercules/python/labours/_pb.py`.
  - `burndown-repos-combined` — Python raises `ValueError: No repository data available` on single-repo; Go now matches by erroring out.
  - `devs-parallel` — Python's `load_devs_parallel` needs `couples_people_data.tsv` from a shared tmpdir; Go runs standalone (keep it).

## Visual Parity Plan

### Stage A — Composition & file-set parity

Cheap, high-leverage: until Go emits the same basenames, the parity viewer cannot even compare. No deep rendering work.

- [x] `temporal-activity`: stop emitting the single composite; emit the 10 Python siblings (8 bar charts + 2 heatmaps). Largest single item.
- [x] `bus-factor`: add the `_gauge` panel (colored bus-factor number + line-ownership pie).
- [x] Settle one filename convention across the showcase harness. `scripts/full_showcase.sh` already invokes both backends with hyphenated `-o <mode>.png`; Go siblings now use Python's underscore suffixes (`_timeline`, `_gauge`, `_weekdays_commits`, …) and `cmd/parityviewer` matches by exact basename. (`ewws-events-parity` underscore fixture is a stale dataset to regenerate under Stage E, not a harness change.)
- [x] Gate Go-only auxiliary plots behind off-by-default `--<mode>-detail` flags so default output mirrors Python: `run-times` (text-only by default), `devs-efforts` productivity ranking, `devs-parallel` concurrency timeline (text-only by default), `knowledge-diffusion` trend.

Acceptance: for every showcase mode, `ls plots-go/<mode>*` and `ls plots-python/<mode>*` produce the same basename set.

### Stage B — Renderer consolidation on `matplotlib-go`

Root cause of much divergence is the two-renderer split. Make `matplotlib-go` (local `replace => ../matplotlib-go`) the single style source of truth, then fix style once instead of per mode.

- [ ] Add missing primitives to `../matplotlib-go` (one-time): gauge, pie, parallel-coordinates, modern colormaps (viridis/plasma/inferno — only Reds/Greens/OrRd registered today), multi-panel/subplot grid, Kaplan-Meier survival curve.
- [ ] Migrate `gonum/plot` modes onto the compat layer: burndown (`internal/graphics/stacked-plot.go`), couples heatmaps, `devs-efforts`, `devs-parallel`, `sentiment`, `shotness`, and the gonum halves of `report_metrics.go`.
- [ ] Centralize style (figure size, DPI 100, fonts, fill opacity, legend placement) in `internal/graphics/theme.go` and have every mode read from it. Retire per-mode style constants.

Acceptance: `grep -rl 'gonum.org/v1/plot' internal/` returns no files in the plot path.

### Stage C — Per-mode perceptual fixes

Walk the divergence list worst-first; target perceptual equivalence (not RMSE yet):

- [ ] `overwrites-matrix` (0.503): heatmap colormap, value scale, tick labels.
- [ ] `languages` (0.429): stack order + tableau palette (Python sorts layers differently).
- [ ] `bus-factor_subsystems` (0.335): subsystem ordering and per-row color.
- [ ] `knowledge-diffusion_silos` (0.213): bar order, palette, label truncation.
- [ ] Fix the dimension-mismatch modes (`bus-factor_timeline`, `devs-efforts`, `knowledge-diffusion_lorenz`, `ownership-concentration_*`) by matching Python figure-size defaults.
- [ ] Then the cheap ≤0.22 modes (`old-vs-new`, `burndown-project`, `refactoring-proxy`, `devs`, `ownership`): legend ordering, line widths, axis fonts.
- [ ] Content TODOs surfaced by parity: `devs-efforts` true time-series (not scatter); `devs-parallel` true parallel-coordinates plot.

### Stage D — RMSE tightening (later)

Only after perceptual parity holds and only where cheap. Drive matched-pair RMSE toward ≤ 0.10 via `cmd/parityviewer`. Explicitly **not** a blocker for "done."

### Stage E — Regression surface

- [x] `scripts/full_showcase.sh` + `just showcase` / `just showcase-compare` + `cmd/parityviewer` exist and produce `compare/index.html`.
- [ ] Add a small fast fixture repo (~50 commits, 2 authors, mixed languages) so the comparison runs in <30 s per PR.
- [ ] Wire a nightly CI job that fails when any matched-name RMSE regresses, any Go-only artifact disappears, or any Python-supported mode starts failing in Go.
- [ ] Optionally auto-load the repo `.mailmap` into manifest output so users see merged identities (the `--people-dict` workflow is already documented with `system-optimiser-core/raw/people-dict.txt` as the worked example).

## Parity Datasets

- **`system-optimiser-core`** — canonical surface (1,994 commits, C++). `analysis_results/system-optimiser-core/compare/index.html` is the visual record; hyphenated filenames.
- **`ewws-events-parity`** — promote to second regression surface. Uses **underscore** filenames (reconcile under Stage A). Document the source repo/size in a short README; run `just parity-viewer-meko` / `showcase-compare` against it.
- **`meko-aggregate-parity`** — **broken/incomplete**: `meko-aggregate.pb` is 0 bytes; `artifacts-go/` and `baseline-python/` are empty (only `ewws-statistics-output/` per-module `.pb` files exist). Task: regenerate a non-empty aggregate protobuf, then run both backends to produce a real comparison — or retire it if aggregate analysis is out of scope.

## Definition of Done

The port is complete when all of these hold:

- `go test ./...` passes.
- `hercules report --strict --labours-cmd ./labours -o <dir> <repo>` and `... --all --strict ...` pass.
- Every showcase mode emits the **same file set** as Python (Stage A).
- Each matched pair is **perceptually equivalent** — same composition, palette, layout — on both `system-optimiser-core` and `ewws-events-parity`, judged via `compare/index.html`.
- All modes render through `matplotlib-go`; no `gonum/plot` remains in the plot path (Stage B).
- RMSE ≤ 0.10 per matched pair is a tracked goal, not a release gate (Stage D).
- Intentional Python divergences (TF modes, `hotspot-risk`, `burndown-repos-combined`, `devs-parallel`) are documented.
- Output paths/assets are deterministic and compatible with Hercules report collection.
