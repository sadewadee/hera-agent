#!/bin/bash
# ============================================================================
# Sample and Compress HuggingFace Datasets
#
# Downloads trajectories from HuggingFace datasets, randomly samples them,
# and runs trajectory compression to fit within a target token budget.
#
# Usage:
#   bash scripts/sample_and_compress.sh
#   bash scripts/sample_and_compress.sh --total-samples 5000
#   bash scripts/sample_and_compress.sh --output-name compressed_16k
#   bash scripts/sample_and_compress.sh --config datagen-config-examples/trajectory_compression.yaml
# ============================================================================

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

# ──────────────────────────────────────────────────────────────────────
# Defaults
# ──────────────────────────────────────────────────────────────────────
TOTAL_SAMPLES=1000
OUTPUT_NAME="compressed"
CONFIG_FILE="datagen-config-examples/trajectory_compression.yaml"
INPUT_DIR=""
OUTPUT_DIR=""
TARGET_TOKENS=29000
NUM_WORKERS=4

# ──────────────────────────────────────────────────────────────────────
# Parse arguments
# ──────────────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --total-samples)
            TOTAL_SAMPLES="$2"
            shift 2
            ;;
        --output-name)
            OUTPUT_NAME="$2"
            shift 2
            ;;
        --config)
            CONFIG_FILE="$2"
            shift 2
            ;;
        --input)
            INPUT_DIR="$2"
            shift 2
            ;;
        --output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --target-tokens)
            TARGET_TOKENS="$2"
            shift 2
            ;;
        --workers)
            NUM_WORKERS="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --total-samples N     Number of trajectories to sample (default: 1000)"
            echo "  --output-name NAME    Output directory suffix (default: compressed)"
            echo "  --config FILE         Compression config YAML (default: datagen-config-examples/trajectory_compression.yaml)"
            echo "  --input DIR           Input directory with trajectories.jsonl"
            echo "  --output DIR          Output directory for compressed data"
            echo "  --target-tokens N     Target max tokens per trajectory (default: 29000)"
            echo "  --workers N           Parallel workers (default: 4)"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# ──────────────────────────────────────────────────────────────────────
# Validate
# ──────────────────────────────────────────────────────────────────────
if [ -z "$INPUT_DIR" ]; then
    echo "No --input directory specified."
    echo "Please provide a directory containing trajectories.jsonl"
    echo ""
    echo "Example:"
    echo "  $0 --input data/trajectories --output data/${OUTPUT_NAME}"
    exit 1
fi

if [ -z "$OUTPUT_DIR" ]; then
    OUTPUT_DIR="${INPUT_DIR}_${OUTPUT_NAME}"
fi

if [ ! -f "$INPUT_DIR/trajectories.jsonl" ]; then
    echo "ERROR: $INPUT_DIR/trajectories.jsonl not found"
    exit 1
fi

# ──────────────────────────────────────────────────────────────────────
# Run compression
# ──────────────────────────────────────────────────────────────────────
echo "=== Sample and Compress ==="
echo "Input:          $INPUT_DIR"
echo "Output:         $OUTPUT_DIR"
echo "Total samples:  $TOTAL_SAMPLES"
echo "Target tokens:  $TARGET_TOKENS"
echo "Workers:        $NUM_WORKERS"
echo "Config:         $CONFIG_FILE"
echo ""

mkdir -p "$OUTPUT_DIR"

# Sample random trajectories
echo "Sampling $TOTAL_SAMPLES trajectories..."
TOTAL_LINES=$(wc -l < "$INPUT_DIR/trajectories.jsonl")
if [ "$TOTAL_LINES" -le "$TOTAL_SAMPLES" ]; then
    echo "  Dataset has $TOTAL_LINES lines (less than $TOTAL_SAMPLES), using all."
    cp "$INPUT_DIR/trajectories.jsonl" "$OUTPUT_DIR/sampled.jsonl"
else
    # Use shuf for random sampling
    shuf -n "$TOTAL_SAMPLES" "$INPUT_DIR/trajectories.jsonl" > "$OUTPUT_DIR/sampled.jsonl"
    echo "  Sampled $TOTAL_SAMPLES from $TOTAL_LINES trajectories."
fi

SAMPLED_COUNT=$(wc -l < "$OUTPUT_DIR/sampled.jsonl")
echo "  Sampled: $SAMPLED_COUNT trajectories"

echo ""
echo "Compressing trajectories to fit within $TARGET_TOKENS tokens..."

# If hera-batch has a compress subcommand, use it; otherwise note manual step
if command -v hera-batch &>/dev/null; then
    hera-batch compress \
        --input "$OUTPUT_DIR/sampled.jsonl" \
        --output "$OUTPUT_DIR/trajectories.jsonl" \
        --target-tokens "$TARGET_TOKENS" \
        --workers "$NUM_WORKERS" \
        --config "$CONFIG_FILE"
else
    echo "  hera-batch not found in PATH."
    echo "  Build it with: go build -o hera-batch ./cmd/hera-batch"
    echo "  Then re-run this script."
    echo ""
    echo "  Alternatively, the sampled (uncompressed) data is at:"
    echo "    $OUTPUT_DIR/sampled.jsonl"
    exit 1
fi

echo ""
echo "=== Done ==="
echo "Compressed output: $OUTPUT_DIR/trajectories.jsonl"

# Print stats
if [ -f "$OUTPUT_DIR/compression_metrics.json" ]; then
    echo ""
    echo "Compression metrics:"
    cat "$OUTPUT_DIR/compression_metrics.json"
fi
