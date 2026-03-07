"""
Vpay Test Data Generator
Generates inflow and outflow JSONL files at 1M and 10M record counts.
No external dependencies beyond Python stdlib.
"""

import json
import random
import uuid
import os
import sys
import time
from datetime import datetime, timedelta

# ── Constants ────────────────────────────────────────────────────────────────

NIGERIAN_BANKS = [
    "Access Bank", "First Bank", "GTBank", "Zenith Bank", "UBA",
    "Fidelity Bank", "Sterling Bank", "Stanbic IBTC", "Union Bank",
    "Ecobank", "Polaris Bank", "Wema Bank", "Heritage Bank", "Keystone Bank",
    "Unity Bank", "First City Monument Bank", "Jaiz Bank", "Titan Trust Bank"
]

EPOCH_START = datetime(2020, 1, 1)
EPOCH_END   = datetime(2023, 12, 31, 23, 59, 59)
EPOCH_SECS  = int((EPOCH_END - EPOCH_START).total_seconds())

OUTPUT_DIR = "./data"

# ── Helpers ──────────────────────────────────────────────────────────────────

def rand_account():
    return f"{random.randint(1000000000, 9999999999)}"

def rand_session():
    return f"{random.randint(0, 999999999999999999999999999999):030d}"

def rand_date():
    return EPOCH_START + timedelta(seconds=random.randint(0, EPOCH_SECS))

def rand_inflow_amount():
    val = random.uniform(100, 5_000_000)
    return f"{val:,.2f}"

def rand_outflow_amount():
    tiers = [(500, 5_000, 0.40), (5_000, 50_000, 0.35),
             (50_000, 500_000, 0.20), (500_000, 5_000_000, 0.05)]
    r = random.random()
    cumulative = 0.0
    for lo, hi, prob in tiers:
        cumulative += prob
        if r < cumulative:
            return str(random.randint(lo // 100, hi // 100) * 100)
    return "5000"

def rand_statement_ids():
    start = random.randint(10_000_000, 99_000_000)
    return f"{start}-{start + random.randint(1, 5)}"

def rand_response():
    r = random.random()
    if r < 0.90:  return "00"
    elif r < 0.95: return "91"
    else:          return "99"

# ── Record builders ──────────────────────────────────────────────────────────

def build_inflow():
    dt = rand_date()
    ts = dt.strftime("%Y%m%d%H%M%S")
    ms = random.randint(100, 999)
    return {
        "Amount":                rand_inflow_amount(),
        "From Account No":       rand_account(),
        "From Bank":             random.choice(NIGERIAN_BANKS),
        "Session ID":            rand_session(),
        "To Account No":         rand_account(),
        "Transaction Date":      dt.strftime("%Y-%m-%d %H:%M:%S"),
        "Transaction Reference": f"Zpay-{ts}{ms}",
        "Type":                  "INFLOW",
        "Wallet Name":           "Zpay",
    }

def build_outflow():
    dt = rand_date()
    return {
        "Session ID":             rand_session(),
        "Statement IDS":          rand_statement_ids(),
        "Transaction Amount":     rand_outflow_amount(),
        "Transaction Date":       dt.strftime("%Y-%m-%d %H:%M:%S"),
        "Transaction Reference":  f"v1-zpay-{uuid.uuid4()}",
        "Transaction Response":   rand_response(),
        "Type":                   "OUTFLOW",
    }

# ── Writer ───────────────────────────────────────────────────────────────────

def write_jsonl(filepath, builder_fn, total, chunk=50_000):
    """Write `total` records to a JSONL file, flushing every `chunk` records."""
    written = 0
    t0 = time.time()
    with open(filepath, "w", encoding="utf-8") as f:
        while written < total:
            batch = min(chunk, total - written)
            lines = []
            for _ in range(batch):
                lines.append(json.dumps(builder_fn(), separators=(",", ":")))
            f.write("\n".join(lines) + "\n")
            written += batch
            elapsed = time.time() - t0
            rate = written / elapsed if elapsed > 0 else 0
            pct = written / total * 100
            print(f"  {written:>10,} / {total:,}  ({pct:.1f}%)  {rate:,.0f} rec/s", end="\r")
    print(f"  {written:>10,} / {total:,}  (100.0%)  — done in {time.time()-t0:.1f}s      ")

# ── Main ─────────────────────────────────────────────────────────────────────

JOBS = [
    ("inflow_1M.jsonl",   build_inflow,   1_000_000),
    ("outflow_1M.jsonl",  build_outflow,  1_000_000),
    ("inflow_10M.jsonl",  build_inflow,  10_000_000),
    ("outflow_10M.jsonl", build_outflow, 10_000_000),
    ("inflow_100M.jsonl",  build_inflow,  100_000_000),
    ("outflow_100M.jsonl", build_outflow, 100_000_000),
]

os.makedirs(OUTPUT_DIR, exist_ok=True)

for filename, builder, count in JOBS:
    path = os.path.join(OUTPUT_DIR, filename)
    print(f"\n▶ Generating {filename}  ({count:,} records) ...")
    write_jsonl(path, builder, count)
    size_mb = os.path.getsize(path) / 1_024 / 1_024
    print(f"  ✓ Saved → {path}  ({size_mb:.1f} MB)")

print("\n✅ All files generated successfully.")
