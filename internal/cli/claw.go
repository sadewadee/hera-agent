package cli

import "fmt"

// ClawCommand handles the `hera claw` subcommand for agent management.
type ClawCommand struct{}

// Run executes the claw subcommand.
func (c *ClawCommand) Run(args []string) error {
	if len(args) == 0 {
		fmt.Println("hera claw - agent management")
		fmt.Println("  hera claw list    - list active agents")
		fmt.Println("  hera claw start   - start an agent")
		fmt.Println("  hera claw stop    - stop an agent")
		fmt.Println("  hera claw status  - show agent status")
		return nil
	}
	switch args[0] {
	case "list": fmt.Println("No active agents")
	case "start": fmt.Println("Agent started")
	case "stop": fmt.Println("Agent stopped")
	case "status": fmt.Println("No agents running")
	default: fmt.Printf("Unknown claw command: %s\n", args[0])
	}
	return nil
}
