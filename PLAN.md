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

All modes now render through `matplotlib-go` (Stage B complete; no `gonum/plot` remains in the plot path).

| Mode | Renderer | File-set vs Python | Perceptual state |
| --- | --- | --- | --- |
| `burndown-project` | mpl | ✅ | RMSE 0.144 |
| `burndown-file` | mpl | ✅ (dir fanout) | not yet compared per-file |
| `burndown-person` | mpl | ✅ (dir fanout) | not yet compared per-file |
| `burndown-repository` | mpl | ✅ | needs multi-repo fixture |
| `burndown-repos-combined` | mpl | ✅ | Python errors on single-repo (intentional match) |
| `ownership` | mpl | ✅ | RMSE 0.112 (closest) |
| `overwrites-matrix` | mpl | ✅ | RMSE 0.503 (worst); Python needs TF |
| `couples-files` | mpl | ✅ (dir) | Python needs TF |
| `couples-people` | mpl | ✅ (dir) | Python needs TF |
| `couples-shotness` | mpl | ✅ (dir) | shotness-gated; not compared |
| `shotness` | mpl | ✅ | not compared |
| `sentiment` | mpl | ✅ | no real fixture (empty ticks) |
| `temporal-activity` | mpl | ✅ 10 siblings (Stage A) | per-sibling perceptual TBD |
| `devs` | mpl | ✅ | RMSE 0.116 |
| `devs-efforts` | mpl | ✅ | dual mirror stackplot w/ DPSS smoothing (Python-parity) |
| `old-vs-new` | mpl | ✅ | RMSE 0.216 |
| `languages` | mpl | ✅ | RMSE 0.429 (stack order/palette) |
| `devs-parallel` | mpl | ✅ | parallel-coordinates (Python design); Go-only (Python needs couples TSV) |
| `run-times` | mpl | Go text-only by default | intentional extra (`--run-times-detail`) |
| `bus-factor` | mpl | ✅ incl. `_gauge` (Stage A) | subsystems RMSE 0.335 |
| `ownership-concentration` | mpl | ✅ (`_timeline`,`_subsystems`) | dims differ |
| `knowledge-diffusion` | mpl | ✅ (`_distribution`,`_silos`,`_lorenz`) | silos 0.213, distribution 0.184; lorenz dims differ |
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

- [x] Add missing primitives. Most already existed in `matplotlib-go` (gauge via GridSpec, `pie.go`, `gridspec.go`/`subplots.go`, and the full matplotlib colormap library incl. viridis/plasma/inferno in `color/listed_colormaps.go`). Added the remaining bridge helpers in `internal/graphics/matplotlib_plots.go`: `PlotScatterMatplotlib` (annotations, zero-line, categorical x-ticks), `PlotStackedBarChartMatplotlib`, rotated per-bar labels on `PlotBarChartMatplotlib`, and per-series `Dashes`/`Fill` on the line chart. Parallel-coordinates and a Kaplan-Meier survival *curve* are intentionally **not** added — no mode renders them today (devs-parallel is a concurrency timeline; burndown survival is text-only), so they would be speculative; revisit if Stage C introduces a true parallel-coordinates plot.
- [x] Migrate `gonum/plot` modes onto the compat layer: burndown (gonum `stacked-plot.go` deleted; legacy/fallback path now `PlotStackedBurndownMatplotlib`), couples heatmaps + bar charts, `devs-efforts`, `devs-parallel`, `sentiment`, `shotness`, `overwrites-matrix`, and the gonum halves of `report_metrics.go`. Dead gonum heatmap/bar code removed; `internal/graphics/size.go` reworked to return float64 inches (no `vg`); `gonum.org/v1/plot` removed from `go.mod`.
- [x] Centralize style in `internal/graphics/theme.go`. The duplicated render literals are now single-sourced as exported symbols every mode/bridge reads from: `PythonPlotFontFamily`/`PythonPlotMonoFontFamily` (`DejaVu Sans`/`DejaVu Sans Mono`), `PythonPlotFontSize()` (resolves `--font-size`, default 12), `InchesToPixels()` (DPI-100 inch→px, the one place `* 100` lives), and `PythonPlotDefaultWidthInches`/`PythonPlotDefaultHeightInches` (16×12 fallback). Retired the per-mode literals across `burndown_matplotlib.go`, `matplotlib_plots.go`, `ownership.go`, `overwrites.go`, `runTimes.go`, `report_metrics.go`, and de-duplicated the copy-pasted `ownershipPlotColors` (now `graphics.LaboursPlotColors`). Side benefit: modes that previously hardcoded font 12 now honor `--font-size` (default render unchanged). **Intentionally left per-mode:** fill opacity (Python uses different alphas per chart kind — stackplot 0.8 vs fill_between 0.3) and legend placement (upper-left / lower-left / lower-right vary by mode) are semantic Python-parity choices, not shared constants. The legacy `CurrentTheme` palette/opacity path (`GetBurndownColors`, live in `burndown-project`) is already a single global source and is parity-validated by the golden tests, so it was left as-is rather than re-routed. Verified: `go build/vet/test ./...` green, plus opt-in `LABOURS_GO_VISUAL_PARITY`/`LABOURS_GO_PYTHON_PARITY` golden + Python-reference regressions pass (burndown abs/rel, ownership) — confirming byte/perceptually-stable output.

Acceptance: `grep -rl 'gonum.org/v1/plot' internal/` returns no files in the plot path. ✅ (also clean in `cmd/`; `go build/vet/test ./...` green)

### Stage C — Per-mode perceptual fixes

Walk the divergence list worst-first; target perceptual equivalence (not RMSE yet):

- [x] `overwrites-matrix` (0.503): colormap/scale/ticks already matched; the real gap was layout — Go shrank the matrix into the lower-right. Retuned `WithGridSpecPadding` to fill the canvas like Python's `tight_layout`. Now perceptually equivalent.
- [x] `languages` (0.429): data + stack order + tab20 palette were already correct; the dominant divergence was a **renderer bug** — semi-transparent (`alpha<1`) stacked fills over a *transparent* AGG surface get their color channels corrupted on PNG save (straight-alpha buffer encoded as premultiplied `image.RGBA`), painting the C++ band red. Fixed perceptually by rendering languages opaque (Python's `alpha=0.8` over white is visually near-identical) and dropping the grid (Python draws none). **Known renderer issue (tracked separately):** the semi-transparent transparent-surface PNG path in `matplotlib-go`'s agg backend mis-encodes alpha; other modes setting `alpha<1` (refactoring-proxy, some report_metrics line fills) may be subtly affected. Proper fix belongs in the renderer, not per-mode.
- [x] `bus-factor_subsystems` (0.335): already faithful after Stage B — matches Python on figsize `(12, n*0.4+2)`, ascending sort with insertion-order tie-break, exact risk colors (`#F44336/#FF9800/#FFC107/#4CAF50`), bar height 0.6, right-aligned value labels, and the x=1 critical line. Rendered output is perceptually equivalent; residual RMSE is antialiasing/font/line-alpha, not composition/ordering.
- [x] `knowledge-diffusion_silos` (0.213): already faithful after Stage B (figsize `(14, n*0.35+2)`, ascending sort + InvertY worst-at-top, colors `#90CAF9`/`#1565C0`, bar height 0.35, monospace labels, dual value labels, legend lower-right). Removed a stray `AddXGrid()` Python doesn't draw. Now perceptually equivalent.
- [x] Fix the dimension-mismatch modes by matching Python figure-size defaults: `bus-factor_timeline` 1400×600, `ownership-concentration_timeline` 1400×600, `knowledge-diffusion_lorenz` 800×800 (square), `ownership-concentration_subsystems` `n*0.5+2` (was `n*0.35+2`) → 1200×7350. All now match Python pixel dimensions exactly. (`devs-efforts` dims are tied to the scatter→time-series content rewrite below, not a pure figsize fix.)
- [x] Then the cheap ≤0.22 modes (`old-vs-new`, `burndown-project`, `refactoring-proxy`, `devs`, `ownership`): verified perceptually equivalent — composition, palette, layout, legend order/colors, and figure dimensions all match Python. No legend/line-width/font changes were needed (already correct from the mpl migration). **Residual RMSE is axis-label formatting**, which needs a renderer feature rather than per-mode edits: (1) Python's `apply_plot_style` forces scientific *offset* notation on the y-axis (single `1e4`/`1e5` corner label + small ticks); `matplotlib-go`'s `ScalarFormatter` has no offset-text support (would print `1e4` on every tick), so Go shows full numbers — which also causes the `ownership` y-labels to clip. (2) Date x-tick format differs per mode (year vs `YYYY-01-01`). Both are cross-cutting matplotlib-go formatter features → tracked as renderer/Stage-D work, explicitly **not** a release gate.
- [x] Content TODOs surfaced by parity:
  - `devs-efforts` is now the Python-parity **dual mirror stackplot** ("Efforts through time (changed lines of code)"): per-day changed-lines per developer from the time series, top-N by total effort with an aggregated "others" row, cumulative sums, and Slepian/DPSS smoothing (`slepianWindow(10, 0.5)` via shifted power iteration on the standard DPSS tridiagonal matrix; the existing symmetric `convolveSame` matches numpy "same" for the symmetric taper, with Python's edge-preserving tail restore). Cumulative layers stack upward, negated/scaled instantaneous layers stack downward; tab20 palette with the color cycle continuing across both stackplots, non-negative y-ticks, legend upper-left `ncol=2`, 16×10. The unidentified-author sentinel (dev index −1) is labelled "Unidentified" (cleaner than Python's accidental `people[-1]`). Falls back to the commits-vs-lines scatter when per-day data or a valid header range is absent.
  - `devs-parallel` now renders the Python-parity **parallel-coordinates** chart ("Developers"): each developer is a zero-slope cubic-spline curve across the 5 rank axes (commits/lines/ownership/couples/commit-cooccurrence, normalized by developer count), drawn as viridis-gradient segments, `xlim 0..6`, `ylim -0.1..1.1`. This is the default output now; the Go-only concurrency timeline moved to a `_concurrency_timeline` sibling behind `--devs-parallel-detail`. (No Python pixel baseline — Python's `load_devs_parallel` `sys.exit`s without `couples_people_data.tsv`; this matches Python's *design*.)

### Stage D — RMSE tightening (later)

Only after perceptual parity holds and only where cheap. Drive matched-pair RMSE toward ≤ 0.10 via `cmd/parityviewer`. Explicitly **not** a blocker for "done."

**Scale note.** `cmd/parityviewer --print` reports RMSE on a **0–255** byte scale (`rmse=58.6` etc.); the normalized 0–1 figures used in this doc and the HTML badges are that value **÷255**. The ≤ 0.10 goal therefore means **raw RMSE ≤ 25.5**. (Housekeeping: the printed and displayed scales disagree — normalize `--print` output to 0–1 so the gate is directly readable. See D7.)

**Current matched-pair RMSE** — `system-optimiser-core`, current Go vs current Python baseline (regenerated 2026-05-25; Python baseline unchanged since 2025-12). Worst-first, normalized ÷255. `diff%` = fraction of pixels that differ at all (a high `diff%` with a low RMSE means a *pervasive but small* per-pixel shift — a renderer/alpha signature, not a layout error).

| Mode | RMSE | raw | dims | diff% | likely root cause |
| --- | --- | --- | --- | --- | --- |
| `temporal-activity_heatmap_commits` | 0.230 | 58.6 | match (fixed) | 92% | heatmap colormap-scale / cell annotations / colorbar |
| `temporal-activity_weekdays_lines` | 0.202 | 51.5 | match | 19% | stacked-bar style (D3) |
| `bus-factor_subsystems` | 0.196 (was 0.282) | 50.1 | match | 18% | D2 done; residual = TightLayout left-margin undershoot |
| `temporal-activity_heatmap_lines` | 0.193 | 49.1 | match (fixed) | 98% | heatmap content (as above) |
| `temporal-activity_months_lines` | 0.177 | 45.2 | match | 20% | stacked-bar style (D3) |
| `temporal-activity_hours_lines` | 0.176 | 45.0 | match | 19% | stacked-bar style (D3) |
| `temporal-activity_weekdays_commits` | 0.170 | 43.4 | match | 19% | stacked-bar style (D3) |
| `temporal-activity_weeks_lines` | 0.166 | 42.3 | match | 18% | stacked-bar style (D3) |
| `knowledge-diffusion_distribution` | 0.163 | 41.6 | match | 14% | font/AA polish (D6) |
| `knowledge-diffusion_silos` | 0.156 | 39.7 | match | 16% | font/AA polish (D6) |
| `ownership-concentration_subsystems` | 0.152 (was 0.239) | 38.8 | match | 22% | D5 encoding fixed; residual = bars matted over opaque white axes (D5b) + TightLayout margin |
| `temporal-activity_hours_commits` | 0.152 | 38.7 | match | 20% | stacked-bar style (D3) |
| `old-vs-new` | 0.147 | 37.6 | match | 21% | y-axis offset notation (D4) |
| `temporal-activity_weeks_commits` | 0.142 | 36.3 | match | 19% | stacked-bar style (D3) |
| `knowledge-diffusion_lorenz` | 0.139 | 35.6 | match (fixed) | 42% | diagonal reference-line AA (D6) |
| `devs-efforts` | 0.139 | 35.3 | match (fixed) | 13% | DPSS smoothing / stack content |
| `refactoring-proxy` | 0.121 (was 0.130) | 30.9 | match | 11% | D5 encoding fixed; residual = white-matte fill (D5b) + x-ticks (D4) |
| `languages` | 0.126 | 32.2 | match | 74% | NOT D5 (buffer opaque); Go fill opaque A255 vs Python alpha 0.8 A204 — alpha-application gap |
| `bus-factor_timeline` | 0.123 | 31.5 | match (fixed) | 41% | line AA + date x-ticks (D4) |
| `temporal-activity_months_commits` | 0.123 | 31.4 | match | 15% | stacked-bar style (D3) |
| `ownership-concentration_timeline` | 0.121 | 30.8 | match (fixed) | 40% | line AA + date x-ticks (D4) |
| `bus-factor_gauge` | 0.111 | 28.2 | match | 7% | gauge text/AA polish (D6) |
| `burndown-project` | 0.101 | 25.6 | match | 9% | y-axis offset notation (D4) |
| `ownership` | 0.090 ✅ | 23.0 | match | 6% | at goal (D4 closes y-label clip) |
| `overwrites-matrix` | 0.082 ✅ | 21.0 | match | 72% | NOT D5 (buffer opaque); pervasive ±1 colormap rounding over heatmap |
| `devs` | 0.078 ✅ | 20.0 | match | 5% | at goal |

Artifact-only (no Python pair, RMSE n/a): `temporal-activity` (Go-only composite — Stage A says drop it), `devs-parallel`, `hotspot-risk`, `run-times`.

Refined tasks, **worst-first**. Several modes share one root cause, so a few fixes clear many rows — tasks are ordered by the worst RMSE they touch.

- [x] **D1 — `temporal-activity` heatmap figure size** (was 0.556 / 0.525, the two worst). Go hardcoded `19.2×8` (1920×800); Python sets `figsize=(19.2,8)` at creation but `apply_plot_style` then overrides it to `args.size or "16,10"` → effective 1600×1000. Fixed in `report_metrics.go:plotTemporalHeatmap` to use `reportPlotInches("temporal-activity.png")` (16×10). Re-measured: 0.230 / 0.193 — still high because the *content* differs (next task). Also drop the Go-only `temporal-activity.png` composite to finish Stage A file-set parity.
- [x] **D2 — `_subsystems` horizontal-bar lists** (`bus-factor_subsystems` 0.282→**0.196**; `ownership-concentration_subsystems` 0.239→**0.153**). Root causes found and fixed:
  - `ownership-concentration_subsystems` was the **wrong chart entirely** — Go drew a single-series *vertical* green Gini bar chart; Python (`ownership_concentration.py:_plot_subsystems`) draws a **two-series horizontal grouped** chart (Gini `#E91E63` + HHI `#3F51B5`, alpha 0.8, alphabetical sort, bar_height 0.35, xlim 0–1.1, value labels, legend). Rewrote `plotOwnershipSubsystemsBar` to match (now uses `SubsystemHHI`, which the reader already exposed but the old code ignored). diff% 99%→22%.
  - `bus-factor_subsystems` was structurally correct (sort/colors/height/labels/locator all matched), but its dominant residual was the **axes-box position**: Go used fixed subplot padding (left 0.24 → bars at x=288) while Python's `tight_layout` puts the long subsystem labels at left 0.56 (bars at x=672) — a 384px horizontal misalignment of large solid bars. Switched it to the measured `TightLayout` save path (left → 0.46), matched Python's 9.6 y-label font, replaced the hand-tuned y-limits with matplotlib's actual 5%-margin autoscale, and matched the critical line (pure-red `alpha=0.4`). diff% 35%→18%.
  - **Remaining residual (both, → D6/renderer):** matplotlib-go's `TightLayout` consistently reserves ~124px less left margin than matplotlib's `tight_layout` (it also pads for `ax.text` value-label annotations, which matplotlib ignores), so bars stay ~0.10·width left of Python. Plus the D5 alpha channel on ownership (Python stores straight-alpha `a=204`; Go opaque would match RGB but measured *worse* because position—not fill—dominates, so alpha 0.8 was kept). Driving these two under 0.10 needs the renderer-layout fix, not per-mode tuning.
- [ ] **D3 — `temporal-activity` stacked bar charts** (8 siblings, 0.12–0.20; worst `weekdays_lines` 0.202). All ~15–20% diff% → one shared stacked-bar styling delta (bar width, edge width, tab20 cycling, legend placement, y-tick step). Fix once in the temporal bar path; re-measure all 8 together. Also resolve the heatmap *content* residual here (colormap normalization/`vmax`, per-cell annotation threshold/font, colorbar presence) since it shares the temporal code.
- [ ] **D4 — matplotlib-go: scientific-offset y-axis + per-mode date x-ticks** (cross-cutting; clears the 0.10–0.15 cluster). Python's `apply_plot_style` forces `ScalarFormatter` *offset* text (one `1e4`/`1e5` corner label + small ticks); `matplotlib-go` has none, so Go prints full numbers — inflating axis-label pixel diffs and clipping `ownership` y-labels. Date x-tick format also differs per mode (year vs `YYYY-01-01`). Add offset-text support + a per-mode date formatter in `../matplotlib-go`, then re-measure `old-vs-new`, `burndown-project`, `ownership`, `refactoring-proxy`, `bus-factor_timeline`, `ownership-concentration_timeline`, `devs`. Biggest single lever; renderer work, not per-mode.
- [x] **D5 — matplotlib-go: alpha-on-transparent-surface PNG encoding** (`../matplotlib-go/backends/agg/agg_text.go:SavePNG`). Root cause confirmed by reproduction (`backends/agg/alpha_transparent_repro_test.go`): the AGG render buffer holds **straight** (non-premultiplied) alpha, but `ToGoImage()` copies those bytes verbatim into an `image.RGBA` (premultiplied **by Go's contract**, i.e. channel ≤ alpha). `png.Encode` trusts that invariant and un-premultiplies with unclamped 16-bit math (`c*0xffff/a`): for any channel where `value > alpha` it **overflows uint16 and wraps** (a fill `(200,120,40)@0.5` round-tripped to `(141,237,77)`), and channels with `value < alpha` are wrongly brightened. **Fix:** reinterpret the straight buffer as `*image.NRGBA` (identical R,G,B,A byte layout) so `png.Encode` writes the values verbatim — this also extends the codebase's own `TestAggSavePNGRoundTripsGetImageRGBA` invariant (decoded PNG == straight GetImage) to `alpha<255`. Added a regression test; `go test ./...` green in both repos.
  - **Re-measured (raw 0–255):** `refactoring-proxy` 33.0→**30.9**, `ownership-concentration_subsystems` 39.1→**38.8**, `languages` 32.4→**32.2**, `overwrites-matrix` 21.0→**21.0**. All improved-or-flat, none regressed — but the deltas are small, because the PLAN's *attribution* of these modes to the encoding bug was largely wrong (corrected below by pixel sampling).
  - **The encoding fix was real but not the dominant residual on these modes:**
    - `languages` / `overwrites-matrix` buffers are **opaque** (`A=255`) where the bug needs `A<255`, so D5 never triggered there. Go's `languages` fill RGB already matched Python exactly (`174,199,232`); the residual is that **Go draws the fill opaque (`A=255`) while Python uses `alpha=0.8` (`A=204`)** — an alpha-application styling gap. `overwrites-matrix` is just ±1 colormap rounding over a 72%-area heatmap.
    - `ownership-concentration_subsystems` / `refactoring-proxy` **do** have `alpha<1` content and the fix corrected their previously overflow-garbled bar/fill pixels — but the bars are a small % of these tall sparse canvases, so RMSE barely moves. Pixel sampling shows the real residual is a **separate compositing bug**: Go's `0.8` bars render with RGB **matted toward white** (pink `#E91E63` → `(237,75,130,204)` = `0.8·color + 0.2·white`; blue similarly) while Python keeps them pure (`233,33,101 @ A206`). Isolation test (`TestAggFillOverWhiteTransparentClear`) proves the surface *clear* is innocent (a fill over a `(255,255,255,0)` clear stays pure) — so the white matte comes from the **axes-background patch being rendered opaque white** even though report figures set `WithAxesBackground(1,1,1,0)`. That alpha-0 axes facecolor is not honored during AGG compositing. **→ new follow-up (D5b/renderer): honor a fully-transparent axes/figure patch so semi-transparent fills composite over transparent, not white.** This is the actual high-diff% driver for the report charts.
- [ ] **D6 — AA/font polish** (≤0.16, already perceptually equivalent): `knowledge-diffusion_{distribution,silos,lorenz}`, `bus-factor_gauge`, and whatever D4 leaves on the timelines. Residual is antialiasing, font hinting, and line-alpha — diminishing returns; only attempt after D1–D5 and only where a single rcParam-level change helps.
- [ ] **D7 — housekeeping**: normalize `cmd/parityviewer --print` RMSE to 0–1 so printed output, HTML badges, and this doc share one scale and the ≤0.10 gate reads directly. Then refresh `compare/index.html` for the record.

Stop conditions: D1–D3 are cheap and mode-local; D4–D5 are `../matplotlib-go` changes that each clear several rows at once and are the only realistic path under 0.10 for the line/fill modes. RMSE ≤ 0.10 remains a tracked goal, **not** a release gate.

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
