#!/usr/bin/env bash
set -uo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
python_labours_dir="${PYTHON_LABOURS_DIR:-$repo_root/../hercules/python}"
ref_dir="$repo_root/analysis_results/reference"
work_dir="${LABOURS_PYTHON_PARITY_TMPDIR:-/tmp/labours-python-baselines}"
mpl_dir="${MPLCONFIGDIR:-/tmp/labours-mplconfig}"
report_fixture="$repo_root/test/testdata/hercules/report_default.pb"
sentiment_fixture="$repo_root/test/testdata/hercules/sentiment.pb"

mkdir -p "$ref_dir" "$work_dir" "$mpl_dir"

export PYTHONPATH="$python_labours_dir${PYTHONPATH:+:$PYTHONPATH}"
export MPLCONFIGDIR="$mpl_dir"
export COUPLES_SERVER_TIME="${COUPLES_SERVER_TIME:-0}"

run_labours() {
  local timeout_s="$1"
  shift
  timeout "$timeout_s" python3 -m labours --backend Agg "$@"
}

copy_if_present() {
  local src="$1"
  local dst="$2"
  if [[ -f "$src" ]]; then
    cp "$src" "$dst"
    echo "OK $(basename "$dst")"
  else
    echo "MISSING $(basename "$dst") from $src" >&2
    return 1
  fi
}

render_simple() {
  local name="$1"
  local input="$2"
  local mode="$3"
  shift 3

  local output="$ref_dir/python_$name.png"
  echo "Generating $output"
  if ! run_labours 90s -i "$input" -m "$mode" -o "$output" "$@" >"$work_dir/$name.log" 2>&1; then
    echo "FAILED $name" >&2
    tail -20 "$work_dir/$name.log" >&2
    return 1
  fi
}

render_and_copy() {
  local name="$1"
  local input="$2"
  local mode="$3"
  local generated="$4"
  shift 4

  local dir="$work_dir/$name"
  rm -rf "$dir"
  mkdir -p "$dir"
  echo "Generating python_$name.png from $mode"
  run_labours 90s -f pb -i "$input" -m "$mode" -o "$dir/out.png" --disable-projector --max-people 20 "$@" >"$dir/log.txt" 2>&1
  local status=$?
  if [[ $status -ne 0 ]]; then
    echo "WARN $name exited with $status; trying to use any plot produced before exit." >&2
    tail -20 "$dir/log.txt" >&2
  fi
  copy_if_present "$dir/$generated" "$ref_dir/python_$name.png"
}

render_optional_timeout() {
  local name="$1"
  local input="$2"
  local mode="$3"
  local generated="$4"
  local timeout_s="$5"
  shift 5

  local dir="$work_dir/$name"
  rm -rf "$dir"
  mkdir -p "$dir"
  echo "Generating optional python_$name.png from $mode"
  run_labours "$timeout_s" -f pb -i "$input" -m "$mode" -o "$dir/out.png" --disable-projector --max-people 5 "$@" >"$dir/log.txt" 2>&1
  local status=$?
  if [[ $status -ne 0 ]]; then
    echo "WARN $name exited with $status; using partial plot if one was produced." >&2
    tail -20 "$dir/log.txt" >&2
  fi
  copy_if_present "$dir/$generated" "$ref_dir/python_$name.png"
}

render_hotspot_risk() {
  local dir="$work_dir/hotspot_risk"
  rm -rf "$dir"
  mkdir -p "$dir"
  echo "Generating python_hotspot_risk.png from patched Python hotspot renderer"
  python3 - "$report_fixture" "$dir/out.png" >"$dir/log.txt" 2>&1 <<'PY'
import os
import sys
from argparse import Namespace

from labours import plotting, readers
import labours.modes.hotspot_risk as hotspot

fixture, output = sys.argv[1:3]

# The neighboring Python reader has the message type and renderer, but this
# checkout is missing the PB registry entry and still expects an old size_ field.
readers.PB_MESSAGES["HotspotRisk"] = "labours.pb_pb2.HotspotRiskResults"
hotspot.apply_plot_style = lambda *args, **kwargs: None
hotspot.deploy_plot = lambda title, path, _background: plotting.deploy_plot(title, path, "white")

reader = readers.ProtobufReader()
with open(fixture, "rb") as f:
    reader.read(f)

hr = reader.contents["HotspotRisk"]
files = []
for entry in hr.files:
    files.append({
        "path": str(entry.path),
        "risk_score": float(entry.risk_score),
        "size": int(entry.size),
        "churn": int(entry.churn),
        "coupling_degree": int(entry.coupling_degree),
        "ownership_gini": float(entry.ownership_gini),
        "size_normalized": float(entry.size_normalized),
        "churn_normalized": float(entry.churn_normalized),
        "coupling_normalized": float(entry.coupling_normalized),
        "ownership_normalized": float(entry.ownership_normalized),
    })

args = Namespace(
    output=output,
    backend="Agg",
    style="ggplot",
    background="white",
    font_size=12,
    text_size=12,
    size=None,
    relative=False,
)
hotspot.show_hotspot_risk(args, reader.get_name(), files, int(hr.window_days))
PY
  local status=$?
  if [[ $status -ne 0 ]]; then
    echo "FAILED hotspot_risk" >&2
    tail -30 "$dir/log.txt" >&2
    return 1
  fi
  copy_if_present "$dir/out/hotspot_risk_ranked.png" "$ref_dir/python_hotspot_risk.png"
}

render_sentiment() {
  local dir="$work_dir/sentiment"
  rm -rf "$dir"
  mkdir -p "$dir"
  echo "Generating python_sentiment.png from patched Python sentiment renderer"
  python3 - "$sentiment_fixture" "$ref_dir/python_sentiment.png" >"$dir/log.txt" 2>&1 <<'PY'
import sys
from argparse import Namespace
from types import SimpleNamespace

from labours import readers
from labours.modes.sentiment import show_sentiment_stats

fixture, output = sys.argv[1:3]

readers.PB_MESSAGES["Sentiment"] = "labours.pb_pb2.CommentSentimentResults"

reader = readers.ProtobufReader()
with open(fixture, "rb") as f:
    reader.read(f)

sentiment = reader.contents["Sentiment"]
data = {
    int(k): SimpleNamespace(
        Value=float(v.value),
        Comments=list(v.comments),
        Commits=list(v.commits),
    )
    for k, v in sentiment.sentiment_by_tick.items()
}
if not data:
    raise RuntimeError("sentiment fixture has no sentiment_by_tick entries")

args = Namespace(
    output=output,
    backend="Agg",
    style="ggplot",
    background="white",
    font_size=12,
    size=None,
    start_date=None,
    end_date=None,
    mode="sentiment",
)
show_sentiment_stats(args, reader.get_name(), "year", reader.get_header()[0], data)
PY
  local status=$?
  if [[ $status -ne 0 ]]; then
    echo "FAILED sentiment" >&2
    tail -30 "$dir/log.txt" >&2
    return 1
  fi
  echo "OK python_sentiment.png"
}

render_python_derived_baselines() {
  local dir="$work_dir/derived"
  rm -rf "$dir"
  mkdir -p "$dir"
  echo "Generating derived Python baselines from Python labours readers"
  python3 - "$report_fixture" "$repo_root/test/testdata/hercules/shotness.pb" "$ref_dir" >"$dir/log.txt" 2>&1 <<'PY'
import os
import sys
from collections import defaultdict

import numpy as np

import matplotlib
matplotlib.use("Agg")
from matplotlib import pyplot as plt

from labours import readers

report_fixture, shotness_fixture, ref_dir = sys.argv[1:4]

# This neighboring Python checkout does not register HotspotRisk yet, but the
# default report fixture contains it. Registering it avoids noisy parse warnings.
readers.PB_MESSAGES["HotspotRisk"] = "labours.pb_pb2.HotspotRiskResults"


def load_reader(path):
    reader = readers.ProtobufReader()
    with open(path, "rb") as f:
        reader.read(f)
    return reader


def save(fig, name):
    path = os.path.join(ref_dir, "python_" + name + ".png")
    fig.tight_layout()
    fig.savefig(path, dpi=96)
    plt.close(fig)
    print("OK", os.path.basename(path))


def compact(label, limit=36):
    label = str(label)
    if len(label) <= limit:
        return label
    return "..." + label[-(limit - 3):]


def dense_matrix(matrix):
    if hasattr(matrix, "toarray"):
        return matrix.toarray()
    return np.asarray(matrix)


def top_pairs(names, matrix, limit):
    pairs = []
    for i in range(len(names)):
        for j in range(i + 1, len(names)):
            value = int(matrix[i, j]) if i < matrix.shape[0] and j < matrix.shape[1] else 0
            if value > 0:
                pairs.append((value, names[i], names[j]))
    pairs.sort(reverse=True, key=lambda p: p[0])
    return pairs[:limit]


def plot_coupling_heatmap(name, title, names, matrix, color):
    fig, ax = plt.subplots(figsize=(12, 12))
    shown = matrix
    shown_names = names
    if len(names) > 60:
        totals = np.asarray(matrix).sum(axis=1)
        order = np.argsort(-totals)[:60]
        shown = matrix[order][:, order]
        shown_names = [names[i] for i in order]
    im = ax.imshow(shown, cmap=color, aspect="auto")
    ax.set_title(title)
    ax.set_xticks(range(len(shown_names)))
    ax.set_yticks(range(len(shown_names)))
    ax.set_xticklabels([compact(n, 18) for n in shown_names], rotation=90, fontsize=6)
    ax.set_yticklabels([compact(n, 28) for n in shown_names], fontsize=6)
    fig.colorbar(im, ax=ax, fraction=0.046, pad=0.04)
    save(fig, name)


def plot_coupling_pairs(name, title, pairs, limit=20):
    pairs = pairs[:limit]
    values = [p[0] for p in pairs]
    labels = [compact(os.path.basename(p[1]) + "-" + os.path.basename(p[2]), 28) for p in pairs]
    fig, ax = plt.subplots(figsize=(16, 8))
    ax.bar(range(len(values)), values, color="#4C78A8")
    ax.set_title(title)
    ax.set_xlabel("Coupling Pair Rank")
    ax.set_ylabel("Coupling Score")
    ax.set_xticks(range(len(labels)))
    ax.set_xticklabels([str(i + 1) for i in range(len(labels))])
    for idx, label in enumerate(labels[:10]):
        ax.text(idx, values[idx], label, rotation=70, ha="left", va="bottom", fontsize=7)
    save(fig, name)


def plot_dev_efforts(reader):
    people, days = reader.get_devs()
    stats = defaultdict(lambda: [0, 0, 0, 0])
    for devs in days.values():
        for dev, stat in devs.items():
            stats[dev][0] += stat.Commits
            stats[dev][1] += stat.Added
            stats[dev][2] += stat.Removed
            stats[dev][3] += stat.Changed
    rows = []
    for dev, values in stats.items():
        lines = values[1] + values[2] + values[3]
        score = values[0] + lines * 0.01
        rows.append((score, people[dev], values[0], lines))
    rows.sort(reverse=True)
    rows = rows[:20]

    fig, ax = plt.subplots(figsize=(16, 8))
    ax.scatter([r[2] for r in rows], [r[3] for r in rows], color="#4C78A8")
    for _, person, commits, lines in rows[:10]:
        ax.annotate(compact(person, 22), (commits, lines), fontsize=7)
    ax.set_title("Developer Efforts: Commits vs Lines Changed")
    ax.set_xlabel("Total Commits")
    ax.set_ylabel("Total Lines Changed")
    save(fig, "devs_efforts_scatter")

    fig, ax = plt.subplots(figsize=(16, 8))
    ax.bar(range(len(rows)), [r[0] for r in rows], color="#F58518")
    ax.set_title("Developer Productivity Ranking")
    ax.set_xlabel("Developer Rank")
    ax.set_ylabel("Productivity Score (Commits + Lines/100)")
    ax.set_xticks(range(len(rows)))
    ax.set_xticklabels([str(i + 1) for i in range(len(rows))])
    save(fig, "devs_productivity_ranking")


def plot_runtime(reader):
    data = sorted(reader.get_run_times().items(), key=lambda kv: kv[1], reverse=True)
    data = [(k, float(v)) for k, v in data if float(v) > 0]
    if not data:
        return
    top = data[:15]
    total = sum(v for _, v in data)

    fig, ax = plt.subplots(figsize=(16, 8))
    ax.bar(range(len(top)), [v for _, v in top], color="#54A24B")
    ax.set_title("Runtime Analysis Breakdown")
    ax.set_xlabel("Operations (by time)")
    ax.set_ylabel("Time")
    ax.set_xticks(range(len(top)))
    ax.set_xticklabels([compact(k, 12) for k, _ in top], rotation=45, ha="right")
    save(fig, "runtime_breakdown")

    pct = [(k, v * 100 / total) for k, v in data[:10]]
    fig, ax = plt.subplots(figsize=(16, 10))
    ax.barh(range(len(pct)), [v for _, v in pct], color="#E45756")
    ax.set_title("Runtime Percentage Distribution")
    ax.set_xlabel("Cumulative Percentage")
    ax.set_ylabel("Operations")
    ax.set_yticks(range(len(pct)))
    ax.set_yticklabels([compact(k, 18) + " (%.1f%%)" % v for k, v in pct])
    save(fig, "runtime_percentage")


def plot_knowledge_trend(reader):
    files, _distribution, _people, _window_months, _tick_size = reader.get_knowledge_diffusion()
    trend = {}
    for data in files.values():
        for tick, editors in data.get("editors_over_time", {}).items():
            trend[tick] = max(trend.get(tick, 0), int(editors))
    if not trend:
        return
    ticks = sorted(trend)
    fig, ax = plt.subplots(figsize=(16, 8))
    ax.plot(ticks, [trend[t] for t in ticks], marker="o", color="#4C78A8")
    ax.set_title("Knowledge Diffusion Trend")
    ax.set_xlabel("Tick")
    ax.set_ylabel("Max Unique Editors")
    save(fig, "knowledge_diffusion_trend")


def plot_shotness(reader):
    records = list(reader.get_shotness())
    rows = []
    for record in records:
        total = sum(int(v) for v in record.counters.values())
        role = getattr(record, "internal_role", "") or getattr(record, "type", "")
        rows.append((total, str(record.file), str(record.name), str(role)))
    rows.sort(reverse=True)
    rows = rows[:20]
    fig, ax = plt.subplots(figsize=(16, 10))
    ax.bar(range(len(rows)), [r[0] for r in rows], color="#E45756")
    ax.set_title("Code Hotspots (Most Frequently Modified Structural Units)")
    ax.set_xlabel("Structural Units")
    ax.set_ylabel("Total Modifications")
    ax.set_xticks(range(len(rows)))
    ax.set_xticklabels(
        [compact("%s:%s" % (r[3], r[2]), 24) for r in rows],
        rotation=45,
        ha="right",
        fontsize=8,
    )
    save(fig, "shotness")


def plot_devs_parallel(reader, max_people=20):
    people, days = reader.get_devs()
    commits = defaultdict(int)
    active_days = defaultdict(set)
    for day, devs in days.items():
        for dev, stat in devs.items():
            name = people[dev]
            commits[name] += stat.Commits
            if stat.Commits or stat.Added or stat.Removed or stat.Changed:
                active_days[name].add(day)
    chosen = [name for _commits, name in sorted(((v, k) for k, v in commits.items()), reverse=True)[:max_people]]
    if not chosen:
        return

    all_days = sorted(set().union(*(active_days[name] for name in chosen)))
    concurrency = [
        sum(1 for name in chosen if day in active_days[name])
        for day in all_days
    ]
    if not concurrency:
        concurrency = [0]
    avg = sum(concurrency) / len(concurrency)

    fig, ax = plt.subplots(figsize=(16, 8))
    ax.plot(range(len(concurrency)), concurrency, color="#4C78A8", linewidth=2)
    ax.fill_between(range(len(concurrency)), concurrency, alpha=0.25, color="#4C78A8")
    ax.axhline(avg, color="#F58518", linestyle="--", label="Average (%.1f)" % avg)
    ax.set_title("Parallel Development Activity Over Time")
    ax.set_xlabel("Time Period")
    ax.set_ylabel("Number of Concurrent Developers")
    ax.legend()
    save(fig, "devs_parallel_activity")

    overlaps = []
    for name in chosen:
        values = []
        left = active_days[name]
        for other in chosen:
            if other == name:
                continue
            right = active_days[other]
            union = len(left | right)
            values.append((len(left & right) / union) if union else 0.0)
        overlaps.append(sum(values) / len(values) if values else 0.0)

    fig, ax = plt.subplots(figsize=(16, 8))
    ax.bar(range(len(chosen)), overlaps, color="#72B7B2")
    ax.set_title("Developer Collaboration Patterns")
    ax.set_xlabel("Developers")
    ax.set_ylabel("Average Overlap Coefficient")
    ax.set_xticks(range(len(chosen)))
    ax.set_xticklabels([compact(name, 18) for name in chosen], rotation=45, ha="right")
    save(fig, "devs_parallel_concurrency")


report = load_reader(report_fixture)
shotness = load_reader(shotness_fixture)

files, matrix = report.get_files_coocc()
matrix = dense_matrix(matrix)
plot_coupling_heatmap("couples_files_heatmap", "File Coupling Heatmap", files, matrix, "Reds")
plot_coupling_pairs("couples_files_top_pairs", "Top File Coupling Pairs", top_pairs(files, matrix, 20), 15)

entities, shotness_matrix = shotness.get_shotness_coocc()
shotness_matrix = dense_matrix(shotness_matrix)
plot_coupling_heatmap("couples_shotness_heatmap", "Shotness Coupling Heatmap", entities, shotness_matrix, "Greens")
plot_coupling_pairs("couples_shotness_top_pairs", "Top Shotness Coupling Pairs", top_pairs(entities, shotness_matrix, 25), 20)

plot_dev_efforts(report)
plot_runtime(report)
plot_knowledge_trend(report)
plot_shotness(shotness)
plot_devs_parallel(report)
PY
  local status=$?
  if [[ $status -ne 0 ]]; then
    echo "FAILED derived Python baselines" >&2
    tail -60 "$dir/log.txt" >&2
    return 1
  fi
}

main() {
  render_simple burndown_absolute "$repo_root/example_data/hercules_burndown.yaml" burndown-project || return 1
  render_simple burndown_relative "$repo_root/example_data/hercules_burndown.yaml" burndown-project --relative || return 1
  render_simple devs "$repo_root/example_data/hercules_devs.yaml" devs || return 1
  render_simple languages "$repo_root/example_data/hercules_devs.yaml" languages || return 1
  render_simple old_vs_new "$repo_root/example_data/hercules_devs.yaml" old-vs-new || return 1

  render_and_copy burndown_project "$report_fixture" burndown-project out.png || return 1
  render_and_copy burndown_file_sample "$report_fixture" burndown-file "out/CODE_OF_CONDUCT.md.png" || return 1
  render_and_copy burndown_person_sample "$report_fixture" burndown-person "out/alexander bezzubov|bzz@apache.org.png" || return 1
  render_and_copy ownership "$report_fixture" ownership out.png || return 1
  render_and_copy bus_factor "$report_fixture" bus-factor out_timeline.png || return 1
  render_and_copy bus_factor_subsystems "$report_fixture" bus-factor out_subsystems.png || return 1
  render_and_copy ownership_concentration "$report_fixture" ownership-concentration out_timeline.png || return 1
  render_and_copy knowledge_diffusion "$report_fixture" knowledge-diffusion out_distribution.png || return 1
  render_and_copy knowledge_diffusion_silos "$report_fixture" knowledge-diffusion out_silos.png || return 1
  render_and_copy temporal_activity "$report_fixture" temporal-activity out_hours_commits.png || return 1
  render_optional_timeout overwrites_matrix "$report_fixture" overwrites-matrix out.png 45s || return 1
  render_hotspot_risk || return 1
  render_sentiment || return 1
  render_python_derived_baselines || return 1
}

main
