package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wexinc/ralph/internal/agent/custom"
	"github.com/wexinc/ralph/internal/config"
)

// agentCmd represents the agent command group.
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage AI agents",
	Long:  `Commands for managing AI agents including listing, adding, and configuring agents.`,
}

// agentAddCmd represents the agent add command.
var agentAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a custom agent",
	Long: `Add a custom agent to Ralph.

This command prompts you for the agent configuration:
  - Name: Unique identifier for the agent
  - Command: The command used to run the agent
  - Detection method: How to detect if the agent is available
  - Model list command: Optional command to list available models

Custom agents are stored in .ralph/config.yaml.

Examples:
  ralph agent add                 # Interactive mode
  ralph agent add --name myagent  # Provide name, prompt for rest`,
	RunE: runAgentAdd,
}

// agentListCmd represents the agent list command.
var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available agents",
	Long:  `List all available agents including built-in and custom agents.`,
	RunE:  runAgentList,
}

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.AddCommand(agentAddCmd)
	agentCmd.AddCommand(agentListCmd)

	// Flags for agent add
	agentAddCmd.Flags().StringP("name", "n", "", "Agent name")
	agentAddCmd.Flags().StringP("command", "c", "", "Agent command")
	agentAddCmd.Flags().StringP("description", "d", "", "Agent description")
	agentAddCmd.Flags().String("detection", "command", "Detection method (command, path, env, always)")
	agentAddCmd.Flags().String("detection-value", "", "Value for detection (command name, path, or env var)")
	agentAddCmd.Flags().String("model-list-cmd", "", "Command to list available models")
	agentAddCmd.Flags().String("default-model", "", "Default model for the agent")
}

// runAgentAdd handles the agent add command.
func runAgentAdd(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	// Get or prompt for name
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		name = promptInput(cmd, reader, "Agent name", "")
		if name == "" {
			return fmt.Errorf("agent name is required")
		}
	}

	// Check if agent already exists
	if _, ok := DefaultRegistry.Get(name); ok {
		return fmt.Errorf("agent %q already exists", name)
	}

	// Get or prompt for command
	command, _ := cmd.Flags().GetString("command")
	if command == "" {
		command = promptInput(cmd, reader, "Agent command (e.g., 'my-cli run')", "")
		if command == "" {
			return fmt.Errorf("agent command is required")
		}
	}

	// Get or prompt for description
	description, _ := cmd.Flags().GetString("description")
	if description == "" {
		description = promptInput(cmd, reader, "Description (optional)", "")
	}

	// Get or prompt for detection method
	detectionStr, _ := cmd.Flags().GetString("detection")
	detection := config.DetectionMethod(detectionStr)

	// Validate detection method
	switch detection {
	case config.DetectionMethodCommand, config.DetectionMethodPath, config.DetectionMethodEnv, config.DetectionMethodAlways:
		// valid
	default:
		cmd.Printf("Invalid detection method %q, using 'command'\n", detection)
		detection = config.DetectionMethodCommand
	}

	// Get detection value if needed
	detectionValue, _ := cmd.Flags().GetString("detection-value")
	if detectionValue == "" && detection != config.DetectionMethodAlways {
		hint := getDetectionHint(detection)
		detectionValue = promptInput(cmd, reader, fmt.Sprintf("Detection value (%s)", hint), "")
	}

	// Get model list command
	modelListCmd, _ := cmd.Flags().GetString("model-list-cmd")
	if modelListCmd == "" {
		modelListCmd = promptInput(cmd, reader, "Model list command (optional)", "")
	}

	// Get default model
	defaultModel, _ := cmd.Flags().GetString("default-model")
	if defaultModel == "" {
		defaultModel = promptInput(cmd, reader, "Default model (optional)", "")
	}

	// Create the custom agent config
	customCfg := config.CustomAgentConfig{
		Name:             name,
		Description:      description,
		Command:          command,
		DetectionMethod:  detection,
		DetectionValue:   detectionValue,
		ModelListCommand: modelListCmd,
		DefaultModel:     defaultModel,
	}

	// Load current config
	cfg, err := config.Load("")
	if err != nil {
		// If config doesn't exist, create a new one
		cfg = config.NewConfig()
	}

	// Add the custom agent to config
	cfg.Agent.Custom = append(cfg.Agent.Custom, customCfg)

	// Save the config
	if err := config.Save(cfg, ""); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Register the agent
	a := custom.New(customCfg)
	DefaultRegistry.Register(a)

	cmd.Printf("Successfully added custom agent %q\n", name)
	cmd.Printf("Configuration saved to %s\n", config.DefaultConfigPath)

	// Show availability
	if a.IsAvailable() {
		cmd.Println("Agent is available and ready to use.")
	} else {
		cmd.Println("Note: Agent is not currently available on this system.")
	}

	return nil
}

// runAgentList handles the agent list command.
func runAgentList(cmd *cobra.Command, args []string) error {
	agents := DefaultRegistry.All()
	available := DefaultRegistry.Available()

	cmd.Printf("Registered agents: %d\n", len(agents))
	cmd.Printf("Available agents: %d\n\n", len(available))

	for _, a := range agents {
		status := "unavailable"
		if a.IsAvailable() {
			status = "available"
		}
		cmd.Printf("  %s - %s (%s)\n", a.Name(), a.Description(), status)
	}

	return nil
}

// promptInput prompts the user for input with a default value.
func promptInput(cmd *cobra.Command, reader *bufio.Reader, prompt string, defaultVal string) string {
	if defaultVal != "" {
		cmd.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		cmd.Printf("%s: ", prompt)
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultVal
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

// getDetectionHint returns a hint for the detection value based on method.
func getDetectionHint(method config.DetectionMethod) string {
	switch method {
	case config.DetectionMethodCommand:
		return "command name to check in PATH"
	case config.DetectionMethodPath:
		return "file or directory path to check"
	case config.DetectionMethodEnv:
		return "environment variable name"
	default:
		return ""
	}
}
