package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

func main() {
	// -------------------
	// 1. Parse CLI arguments
	// -------------------
	rootDir := flag.String("root", ".", "Directory to watch for file changes")
	buildCmd := flag.String("build", "", "Command used to build the project")
	execCmd := flag.String("exec", "", "Command used to run the server")
	flag.Parse()

	if *buildCmd == "" || *execCmd == "" {
		fmt.Println("Error: --build and --exec commands are required")
		os.Exit(1)
	}

	// -------------------
	// 2. Setup logging
	// -------------------
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("Starting HotReload Engine")
	logger.Info("Root directory", "dir", *rootDir)
	logger.Info("Build command", "cmd", *buildCmd)
	logger.Info("Exec command", "cmd", *execCmd)

	// -------------------
	// 3. Initialize workspace directory
	// -------------------
	workspaceDir := filepath.Join(".", "hotreload_workspace")
	err := os.MkdirAll(workspaceDir, os.ModePerm)
	if err != nil {
		logger.Error("Failed to create workspace", "error", err)
		os.Exit(1)
	}
	logger.Info("Workspace initialized", "workspace", workspaceDir)

	// -------------------
	// Step 0 complete
	// -------------------
	logger.Info("Step 0 complete: CLI parsed and workspace ready")
}
