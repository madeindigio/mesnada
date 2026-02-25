// Package main provides the mcp-convert CLI tool for converting MCP config files
// between formats used by different AI coding assistants.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sevir/mesnada/internal/mcpconv"
)

func main() {
	var (
		from        = flag.String("from", "", "Input format: mesnada|copilot|vscode|zed|antigravity")
		to          = flag.String("to", "", "Output format: mesnada|copilot|vscode|claude|gemini|opencode|vibe|zed|antigravity")
		all         = flag.Bool("all", false, "Generate all supported formats into conventional project paths")
		showVersion = flag.Bool("version", false, "Show version and exit")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: mcp-convert [flags] <mcp-config-file>\n\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nSupported input formats (--from): mesnada, copilot, vscode, zed, antigravity\n")
		fmt.Fprintf(os.Stderr, "Supported output formats (--to):  mesnada, copilot, vscode, claude, gemini, opencode, vibe, zed, antigravity\n")
		fmt.Fprintf(os.Stderr, "\n--all output paths:\n")
		for _, e := range mcpconv.AllFormats {
			fmt.Fprintf(os.Stderr, "  %-12s -> %s\n", e.Format, e.Path)
		}
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("mcp-convert %s\n", mcpconv.Version)
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "error: MCP config file argument is required")
		flag.Usage()
		os.Exit(1)
	}

	inputFile := flag.Arg(0)
	runRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: get working directory: %v\n", err)
		os.Exit(1)
	}

	cfg, err := mcpconv.ParseCanonicalFileWithFormat(inputFile, *from)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *all {
		projectDir, err := findProjectDir(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		written, err := mcpconv.WriteAllFormatsSkippingSource(cfg, projectDir, runRoot, inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		for _, p := range written {
			fmt.Println(p)
		}
		return
	}

	payload, err := mcpconv.RenderByFormat(cfg, *to, runRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if s, ok := payload.(string); ok {
		fmt.Print(s)
		return
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// findProjectDir returns the project root directory relative to the input file.
// It walks up from the input file's directory looking for a .git directory,
// falling back to the current working directory.
func findProjectDir(inputFile string) (string, error) {
	abs, err := filepath.Abs(inputFile)
	if err != nil {
		return "", fmt.Errorf("resolve input path: %w", err)
	}
	dir := filepath.Dir(abs)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Fall back to CWD
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	return cwd, nil
}

// supportedFromFormats lists accepted --from values for validation.
var supportedFromFormats = []string{"mesnada", "copilot", "vscode", "zed", "antigravity"}

func isSupportedFromFormat(f string) bool {
	f = strings.ToLower(f)
	for _, s := range supportedFromFormats {
		if s == f {
			return true
		}
	}
	return f == ""
}
