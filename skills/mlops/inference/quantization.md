---
name: quantization_inference
description: "Model quantization for efficient inference"
version: "1.0"
trigger: "quantization int8 int4 inference optimization"
platforms: []
requires_tools: ["run_command"]
---

# Model Quantization for Inference

## Purpose
Reduce model size and improve inference speed by quantizing model weights and activations from full precision to lower bit widths.

## Instructions
1. Profile the baseline model (size, latency, accuracy)
2. Select quantization strategy (post-training vs quantization-aware training)
3. Choose target precision (INT8, INT4, FP8, mixed precision)
4. Apply quantization and measure accuracy degradation
5. Benchmark inference speed and memory usage on target hardware

## Quantization Types
- **Post-Training Quantization (PTQ)**: No retraining needed, fast to apply
- **Quantization-Aware Training (QAT)**: Simulates quantization during training, better accuracy
- **Dynamic quantization**: Quantize weights statically, activations dynamically at runtime
- **Static quantization**: Calibrate both weights and activations with representative data
- **Weight-only quantization**: Only quantize weights, keep activations in FP16

## Precision Levels
| Precision | Size Reduction | Speed Gain | Accuracy Impact |
|-----------|---------------|------------|-----------------|
| FP16      | 2x            | 1.5-2x    | Minimal         |
| INT8      | 4x            | 2-4x      | Low (<1% loss)  |
| INT4      | 8x            | 3-6x      | Moderate (1-3%)  |
| GPTQ/AWQ  | 4-8x          | 2-4x      | Low for LLMs    |

## LLM Quantization
- **GPTQ**: Layer-wise quantization with calibration data, good for INT4
- **AWQ (Activation-Aware)**: Preserves important weight channels, better quality
- **GGUF**: Format for llama.cpp, supports mixed precision per layer
- **bitsandbytes**: NF4/INT8 quantization integrated with HuggingFace

## Best Practices
- Always benchmark on your specific task, not just perplexity
- Use calibration datasets representative of production traffic
- Start with INT8 PTQ before trying more aggressive quantization
- Test edge cases where quantization may cause regression
- Profile on target hardware (CPU vs GPU quantization kernels differ)
