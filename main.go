// Package main serves as the primary entry point for httpee.
// It automatically detects `.http` files in the current directory or takes files as arguments,
// parses them sequentially, and passes them into the TUI process.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"httpee/pkg/config"
	"httpee/pkg/parser"
	"httpee/pkg/tui"
)

func main() {
	args := os.Args[1:]
	var files []string

	if len(args) > 0 {
		files = args
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
			os.Exit(1)
		}
		entries, err := os.ReadDir(cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
			os.Exit(1)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				ext := filepath.Ext(entry.Name())
				if ext == ".http" || ext == ".rest" {
					files = append(files, entry.Name())
				}
			}
		}
	}

	if len(files) == 0 {
		fmt.Println("No .http or .rest files found in the current directory.")
		fmt.Println("Usage: httpee [file1.http file2.rest ...]")
		os.Exit(1)
	}

	var allReqs []parser.Request
	// Merge variable definitions from all files. Later files can add keys, but
	// the first definition of a key wins (consistent with REST Client behaviour).
	mergedVars := make(map[string]string)

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", file, err)
			continue
		}

		fileDir := filepath.Dir(file)
		if !filepath.IsAbs(fileDir) {
			if abs, err := filepath.Abs(fileDir); err == nil {
				fileDir = abs
			}
		}

		reqs, vars, err := parser.Parse(f, fileDir)
		f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", file, err)
			continue
		}
		allReqs = append(allReqs, reqs...)
		for k, v := range vars {
			if _, exists := mergedVars[k]; !exists {
				mergedVars[k] = v
			}
		}
	}

	if len(allReqs) == 0 {
		fmt.Println("No valid HTTP requests found in the provided files.")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
	}

	p := tea.NewProgram(tui.New(allReqs, mergedVars, cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting httpee: %v\n", err)
		os.Exit(1)
	}
}
