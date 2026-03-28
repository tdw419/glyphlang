package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/glyphlang/glyph/pkg/ai"
	"github.com/spf13/cobra"
)

// runAI handles the ai command — prompts an LLM to output GlyphLang and executes it.
func runAI(cmd *cobra.Command, args []string) error {
	prompt := strings.Join(args, " ")
	provider, _ := cmd.Flags().GetString("provider")
	model, _ := cmd.Flags().GetString("model")
	baseURL, _ := cmd.Flags().GetString("base-url")
	showCode, _ := cmd.Flags().GetBool("show-code")
	codeOnly, _ := cmd.Flags().GetBool("code-only")

	// Resolve API key from environment (optional for local providers)
	apiKey := resolveAPIKey(provider)
	if apiKey == "" && provider != "ollama" && baseURL == "" {
		return fmt.Errorf("no API key found for provider %q — set %s", provider, envVarForProvider(provider))
	}

	var pipeline *ai.Pipeline
	var err error
	if baseURL != "" {
		pipeline, err = ai.NewPipelineWithBaseURL(provider, apiKey, model, baseURL)
		if err != nil {
			return err
		}
	} else {
		pipeline = ai.NewPipeline(provider, apiKey, model)
	}

	if !codeOnly {
		color.New(color.FgCyan).Printf("⚡ AI → GlyphLang → Execute\n")
		color.New(color.FgWhite).Printf("   Prompt: %s\n", prompt)
		color.New(color.FgWhite).Printf("   Model:  %s (%s)\n\n", model, provider)
	}

	start := time.Now()
	result, err := pipeline.Run(prompt)
	elapsed := time.Since(start)

	if err != nil {
		return err
	}

	if codeOnly {
		// Just print the generated code
		fmt.Println(result.GlyphCode)
		return nil
	}

	if showCode {
		color.New(color.FgYellow, color.Bold).Println("── Generated GlyphLang ──")
		fmt.Println(result.GlyphCode)
		fmt.Println()
	}

	color.New(color.FgGreen, color.Bold).Println("── Result ──")

	// Pretty-print the result
	switch v := result.Output.(type) {
	case map[string]interface{}:
		out, _ := json.MarshalIndent(v, "", "  ")
		fmt.Println(string(out))
	case nil:
		fmt.Println("(no return value)")
	default:
		fmt.Println(v)
	}

	fmt.Println()
	color.New(color.FgWhite).Printf("   Method: %s | Time: %s\n", result.Method, elapsed.Round(time.Millisecond))

	return nil
}

// resolveAPIKey looks up the API key from environment variables.
func resolveAPIKey(provider string) string {
	switch provider {
	case "anthropic":
		return os.Getenv("ANTHROPIC_API_KEY")
	case "openai":
		return os.Getenv("OPENAI_API_KEY")
	case "ollama":
		return "ollama" // Ollama doesn't need a key
	default:
		return os.Getenv("ANTHROPIC_API_KEY")
	}
}

// envVarForProvider returns the expected environment variable name for a provider.
func envVarForProvider(provider string) string {
	switch provider {
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "openai":
		return "OPENAI_API_KEY"
	case "ollama":
		return "(none needed)"
	default:
		return "ANTHROPIC_API_KEY"
	}
}
