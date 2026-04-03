#!/usr/bin/env python3
"""
GlyphLang Experiment Runner using AutoSpec's ExperimentLoop.

This script runs autonomous experiments to fix the SSA expression lowering bug.
The SSA compiler produces null+RETURN for standalone expressions while the
standard compiler produces correct PUSH_CONST/ADD/RETURN bytecode.

Hypotheses to test:
1. Fix SSA builder to handle ReturnStatement with expression value
2. Fix SSA builder's const lowering for IntLit/FloatLit
3. Add expression-to-SSA conversion for binary ops in return statements

Metric: number of passing tests (go test ./... -count=1 | grep -c "^ok")
"""
import sys
import os

# Add autospec to path
sys.path.insert(0, os.path.expanduser("~/zion/projects/autospec/autospec/src"))

from autospec.autoresearch.loop import ExperimentLoop, Hypothesis
from pathlib import Path

PROJECT = Path.home() / "zion" / "projects" / "glyphlang"

loop = ExperimentLoop(
    project_path=PROJECT,
    target_file="pkg/ssa/lower_cpu.go",
    eval_command="go test ./... -count=1 2>&1 | grep -c '^ok'",
    time_budget_minutes=30,
    lower_is_better=False,  # more passing tests = better
)

print(f"ExperimentLoop configured for {PROJECT}")
print(f"Eval: {loop.eval_command}")
print(f"Target: {loop.target_file}")
print(f"Time budget: {loop.time_budget_minutes} min")

# Get baseline
print("\n--- Baseline measurement ---")
try:
    metric, output = loop.run_experiment()
    print(f"Baseline: {metric} passing packages")
    print(f"Output: {output.strip()}")
except Exception as e:
    print(f"Baseline failed: {e}")
    metric = 0.0

print(f"\nReady to accept hypotheses.")
print(f"Usage: python3 {__file__} --hypothesis '<description>' --file <path> --content <changes>")
