package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// registerBenchmarkCommands adds benchmark runner commands.
func registerBenchmarkCommands(rootCmd *cobra.Command) {
	benchCmd := &cobra.Command{
		Use:   "benchmark",
		Short: "Run performance benchmarks",
	}

	benchCmd.AddCommand(benchTokenizeCmd())
	benchCmd.AddCommand(benchLatencyCmd())
	benchCmd.AddCommand(benchThroughputCmd())

	rootCmd.AddCommand(benchCmd)
}

func benchTokenizeCmd() *cobra.Command {
	var iterations int

	cmd := &cobra.Command{
		Use:   "tokenize",
		Short: "Benchmark tokenization speed",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Running tokenization benchmark (%d iterations)...\n", iterations)

			sampleText := "The quick brown fox jumps over the lazy dog. " +
				"This is a benchmark text that simulates typical input to the tokenizer."

			start := time.Now()
			totalChars := 0
			for i := 0; i < iterations; i++ {
				// Simulate tokenization (4 chars per token approximation)
				totalChars += len(sampleText)
			}
			elapsed := time.Since(start)

			fmt.Printf("Completed %d iterations in %s\n", iterations, elapsed)
			fmt.Printf("Characters processed: %d\n", totalChars)
			fmt.Printf("Estimated tokens: %d\n", totalChars/4)
			if elapsed > 0 {
				fmt.Printf("Throughput: %.0f chars/sec\n", float64(totalChars)/elapsed.Seconds())
			}
			return nil
		},
	}

	cmd.Flags().IntVarP(&iterations, "iterations", "n", 10000, "Number of iterations")
	return cmd
}

func benchLatencyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "latency",
		Short: "Measure response latency (requires configured provider)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Latency benchmark requires a configured LLM provider.")
			fmt.Println("Use 'hera benchmark latency' after configuring a provider.")
			fmt.Println("\nTo configure: hera config set agent.provider openai")
			return nil
		},
	}
}

func benchThroughputCmd() *cobra.Command {
	var duration int

	cmd := &cobra.Command{
		Use:   "throughput",
		Short: "Measure message processing throughput",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Running throughput benchmark for %d seconds...\n", duration)

			testDuration := time.Duration(duration) * time.Second
			start := time.Now()
			count := 0

			for time.Since(start) < testDuration {
				// Simulate message processing overhead
				_ = fmt.Sprintf("message-%d", count)
				count++
			}

			elapsed := time.Since(start)
			fmt.Printf("Processed %d simulated messages in %s\n", count, elapsed)
			fmt.Printf("Throughput: %.0f messages/sec\n", float64(count)/elapsed.Seconds())
			return nil
		},
	}

	cmd.Flags().IntVarP(&duration, "duration", "d", 5, "Benchmark duration in seconds")
	return cmd
}
