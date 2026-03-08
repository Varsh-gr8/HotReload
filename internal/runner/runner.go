package runner

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/Varsh-gr8/HotReload/internal/process"
)

type Runner struct {
	buildCmd string
	execCmd  string
	logger   *slog.Logger
	current  *process.Process
}

func New(buildCmd, execCmd string, logger *slog.Logger) *Runner {
	return &Runner{
		buildCmd: buildCmd,
		execCmd:  execCmd,
		logger:   logger,
	}
}

func (r *Runner) StopCurrent() {
	if r.current != nil {
		r.logger.Info("stopping current server process")
		r.current.Kill()
		r.current = nil
	}
}

func (r *Runner) BuildAndRun(ctx context.Context) error {
	r.StopCurrent()

	r.logger.Info("build started", "cmd", r.buildCmd)

	parts := strings.Fields(r.buildCmd)
	buildCmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		if ctx.Err() != nil {
			r.logger.Info("build cancelled due to new file change")
			return ctx.Err()
		}
		r.logger.Error("build failed", "error", err)
		return err
	}

	r.logger.Info("build succeeded")

	if ctx.Err() != nil {
		r.logger.Info("context cancelled before server start, skipping exec")
		return ctx.Err()
	}

	r.logger.Info("starting server", "cmd", r.execCmd)

	proc, err := process.Start(r.execCmd, r.logger)
	if err != nil {
		r.logger.Error("failed to start server", "error", err)
		return err
	}

	r.current = proc
	return nil
}
