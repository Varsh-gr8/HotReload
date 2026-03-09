package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Varsh-gr8/HotReload/internal/dag"
	"github.com/Varsh-gr8/HotReload/internal/engine"
)

func main() {
	rootDir := flag.String("root", ".", "Directory to watch")
	buildCmd := flag.String("build", "", "Build command")
	execCmd := flag.String("exec", "", "Exec command")
	configFile := flag.String("config", "hotreload.yaml", "Path to config file")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	d := dag.New(logger)

	// Try loading yaml config first
	cfg, err := loadConfig(*configFile)
	if err == nil {
		// YAML config found — build DAG from it
		logger.Info("loading config", "file", *configFile)
		for _, n := range cfg.Nodes {
			d.AddNode(&dag.Node{
				Name:  n.Name,
				Path:  n.Path,
				Build: n.Build,
				Deps:  n.Deps,
			})
		}
		*execCmd = cfg.Exec
	} else {
		// No config — fall back to single node from flags
		if *buildCmd == "" || *execCmd == "" {
			fmt.Fprintln(os.Stderr, "Error: --build and --exec are required when no config file is present")
			fmt.Fprintln(os.Stderr, "Usage: hotreload --root <dir> --build <cmd> --exec <cmd>")
			fmt.Fprintln(os.Stderr, "       hotreload --config hotreload.yaml")
			os.Exit(1)
		}
		logger.Info("no config file found, using single node DAG from flags")
		d.AddNode(&dag.Node{
			Name:  "root",
			Path:  *rootDir,
			Build: *buildCmd,
			Deps:  []string{},
		})
	}

	if err := d.Build(); err != nil {
		logger.Error("failed to build DAG", "error", err)
		os.Exit(1)
	}

	logger.Info("DAG initialized", "nodes", len(d.Nodes))

	eng, err := engine.New(*rootDir, *execCmd, d, logger)
	if err != nil {
		logger.Error("failed to initialize engine", "error", err)
		os.Exit(1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	engineErr := make(chan error, 1)
	go func() {
		engineErr <- eng.Run()
	}()

	select {
	case sig := <-quit:
		logger.Info("shutdown signal received", "signal", sig.String())
		os.Exit(0)
	case err := <-engineErr:
		if err != nil {
			logger.Error("engine error", "error", err)
			os.Exit(1)
		}
	}
}
