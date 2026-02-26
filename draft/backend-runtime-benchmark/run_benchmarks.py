#!/usr/bin/env python3
import json
import os
import platform
import shutil
import statistics
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path


ROOT = Path(__file__).resolve().parent
RESULTS_DIR = ROOT / "results"
LATEST_JSON = RESULTS_DIR / "latest.json"
LATEST_REPORT = ROOT / "report.md"


def run_command(cmd, cwd=None):
    started = time.perf_counter()
    proc = subprocess.run(
        cmd,
        cwd=cwd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=False,
    )
    elapsed = time.perf_counter() - started
    if proc.returncode != 0:
        raise RuntimeError(
            f"command failed ({' '.join(cmd)}):\nstdout:\n{proc.stdout}\nstderr:\n{proc.stderr}"
        )
    return elapsed, proc.stdout.strip()


def ensure_typescript_toolchain():
    node_modules = ROOT / "node_modules"
    if node_modules.exists():
        return
    run_command(["npm", "install"], cwd=ROOT)


def cleanup_for_compile():
    build_dir = ROOT / "build"
    if build_dir.exists():
        shutil.rmtree(build_dir)
    pycache = ROOT / "__pycache__"
    if pycache.exists():
        shutil.rmtree(pycache)
    rust_target = ROOT / "rust_bench" / "target"
    if rust_target.exists():
        shutil.rmtree(rust_target)


def bench_go():
    bin_path = ROOT / "build" / "go_bench"
    bin_path.parent.mkdir(parents=True, exist_ok=True)
    compile_s, _ = run_command(["go", "build", "-o", str(bin_path), "./go_bench"], cwd=ROOT)
    _, out = run_command([str(bin_path)], cwd=ROOT)
    result = json.loads(out)
    return {"runtime": "go", "compile_seconds": compile_s, "benchmark": result}


def bench_python():
    compile_s, _ = run_command(["python3", "-m", "py_compile", "python_bench.py"], cwd=ROOT)
    _, out = run_command(["python3", "python_bench.py"], cwd=ROOT)
    result = json.loads(out)
    return {"runtime": "python", "compile_seconds": compile_s, "benchmark": result}


def bench_typescript():
    ensure_typescript_toolchain()
    tsc_path = ROOT / "node_modules" / ".bin" / "tsc"
    compile_s, _ = run_command([str(tsc_path), "--project", "tsconfig.json"], cwd=ROOT)
    js_path = ROOT / "build" / "ts" / "node_bench.js"
    _, out = run_command(["node", str(js_path)], cwd=ROOT)
    result = json.loads(out)
    return {"runtime": "typescript-node", "compile_seconds": compile_s, "benchmark": result}


def bench_rust():
    compile_s, _ = run_command(
        ["cargo", "build", "--release", "--manifest-path", "rust_bench/Cargo.toml"],
        cwd=ROOT,
    )
    bin_path = ROOT / "rust_bench" / "target" / "release" / "tabura_runtime_bench"
    _, out = run_command([str(bin_path)], cwd=ROOT)
    result = json.loads(out)
    return {"runtime": "rust", "compile_seconds": compile_s, "benchmark": result}


def format_float(value, digits=2):
    return f"{value:.{digits}f}"


def make_report(payload):
    rows = payload["runs"]
    go_row = next((r for r in rows if r["runtime"] == "go"), None)
    if go_row is None:
        raise RuntimeError("go result missing")
    go_tp = float(go_row["benchmark"]["throughput_ops_per_sec"])
    go_p95 = float(go_row["benchmark"]["latency_us"]["p95"])
    go_compile = float(go_row["compile_seconds"])

    lines = []
    lines.append("# Backend Runtime Benchmark (Draft)")
    lines.append("")
    lines.append(f"- Timestamp (UTC): `{payload['metadata']['timestamp_utc']}`")
    lines.append(f"- Host: `{payload['metadata']['host']}`")
    lines.append(f"- OS: `{payload['metadata']['os']}`")
    lines.append(f"- Go: `{payload['metadata']['go']}`")
    lines.append(f"- Node: `{payload['metadata']['node']}`")
    lines.append(f"- Rust: `{payload['metadata']['rust']}`")
    lines.append(f"- Python: `{payload['metadata']['python']}`")
    lines.append(f"- Workload: `{payload['metadata']['workload']}`")
    lines.append("")
    lines.append("## Method")
    lines.append("")
    lines.append("- Synthetic backend-like CPU path per request: JSON decode, intent classification, token counting, SHA-256, JSON encode.")
    lines.append("- Throughput measured on 300,000 ops; latency measured on 50,000 per-op samples.")
    lines.append("- Compile/build time measured before execution for each runtime in this folder.")
    lines.append("- Rust compile measured from a clean local `target/` in this benchmark folder; Go/TS/Python use fresh local build artifacts.")
    lines.append("")
    lines.append("## Results")
    lines.append("")
    lines.append("| Runtime | Compile Time (s) | Throughput (ops/s) | p50 (us) | p95 (us) | p99 (us) | Throughput vs Go | p95 vs Go |")
    lines.append("|---|---:|---:|---:|---:|---:|---:|---:|")

    for row in rows:
        bench = row["benchmark"]
        tp = float(bench["throughput_ops_per_sec"])
        p50 = float(bench["latency_us"]["p50"])
        p95 = float(bench["latency_us"]["p95"])
        p99 = float(bench["latency_us"]["p99"])
        compile_s = float(row["compile_seconds"])
        tp_vs_go = tp / go_tp
        p95_vs_go = p95 / go_p95 if go_p95 else 0.0
        lines.append(
            f"| {row['runtime']} | {format_float(compile_s, 3)} | {format_float(tp)} | "
            f"{format_float(p50, 3)} | {format_float(p95, 3)} | {format_float(p99, 3)} | "
            f"{format_float(tp_vs_go, 2)}x | {format_float(p95_vs_go, 2)}x |"
        )

    lines.append("")
    lines.append("## Porting Decision Notes")
    lines.append("")
    lines.append("- Primary target for a Go port replacement should beat Go on both p95 latency and throughput, or provide clear non-performance benefits.")
    lines.append("- Compile-time impact matters for dev loop and CI cost.")
    lines.append("")

    ranking = sorted(rows, key=lambda r: float(r["benchmark"]["throughput_ops_per_sec"]), reverse=True)
    best = ranking[0]
    lines.append(f"- Highest throughput in this run: `{best['runtime']}`.")

    # Recommendation logic.
    recommendation = "Keep Go as default for backend runtime."
    rationale = []
    top_non_go = next((r for r in ranking if r["runtime"] != "go"), None)
    if top_non_go is not None:
        top_tp = float(top_non_go["benchmark"]["throughput_ops_per_sec"])
        top_p95 = float(top_non_go["benchmark"]["latency_us"]["p95"])
        top_compile = float(top_non_go["compile_seconds"])
        if top_tp > go_tp * 1.10 and top_p95 < go_p95 * 0.95:
            recommendation = f"Evaluate `{top_non_go['runtime']}` for a focused prototype."
            rationale.append("It materially outperformed Go on throughput and p95 latency in this synthetic workload.")
        else:
            rationale.append("No alternative showed a decisive win over Go on both throughput and p95 latency.")
        if top_compile > go_compile * 3.0:
            rationale.append("Compile-time cost is significantly higher than Go for at least one high-performance alternative.")

    lines.append(f"- Recommendation: **{recommendation}**")
    for item in rationale:
        lines.append(f"- {item}")
    lines.append("")
    lines.append("## Caveats")
    lines.append("")
    lines.append("- This is a synthetic microbenchmark, not an end-to-end benchmark with WebSocket I/O, disk I/O, or SQLite contention.")
    lines.append("- Use this as a directional signal; confirm with production-like load tests before committing to a full port.")
    lines.append("")
    return "\n".join(lines) + "\n"


def main():
    RESULTS_DIR.mkdir(parents=True, exist_ok=True)
    cleanup_for_compile()

    _, go_version = run_command(["go", "version"], cwd=ROOT)
    _, node_version = run_command(["node", "--version"], cwd=ROOT)
    _, rust_version = run_command(["rustc", "--version"], cwd=ROOT)

    runs = [bench_go(), bench_python(), bench_typescript(), bench_rust()]

    payload = {
        "metadata": {
            "timestamp_utc": datetime.now(timezone.utc).isoformat(),
            "host": platform.node(),
            "os": f"{platform.system()} {platform.release()}",
            "python": platform.python_version(),
            "go": go_version,
            "node": node_version,
            "rust": rust_version,
            "workload": "json decode/classify/hash/json encode",
        },
        "runs": runs,
    }

    LATEST_JSON.write_text(json.dumps(payload, indent=2), encoding="utf-8")
    report = make_report(payload)
    LATEST_REPORT.write_text(report, encoding="utf-8")
    print(report)
    print(f"\nWrote:\n- {LATEST_JSON}\n- {LATEST_REPORT}")


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"benchmark run failed: {exc}", file=sys.stderr)
        sys.exit(1)
