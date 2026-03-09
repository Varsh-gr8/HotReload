package engine

import (
	"context"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/Varsh-gr8/HotReload/internal/dag"
	"github.com/Varsh-gr8/HotReload/internal/process"
	"github.com/Varsh-gr8/HotReload/internal/watcher"
)

const (
	crashThreshold = 3
	crashWindow    = 10 * time.Second
	crashBackoff   = 5 * time.Second
)

type Engine struct {
	rootDir string
	execCmd string
	dag     *dag.DAG
	watcher *watcher.Watcher
	logger  *slog.Logger

	mu          sync.Mutex
	cancelBuild context.CancelFunc
	current     *process.Process
	crashTimes  []time.Time
}

func New(rootDir, execCmd string, d *dag.DAG, logger *slog.Logger) (*Engine, error) {
	w, err := watcher.New(logger)
	if err != nil {
		return nil, err
	}
	return &Engine{
		rootDir: rootDir,
		execCmd: execCmd,
		dag:     d,
		watcher: w,
		logger:  logger,
	}, nil
}

func (e *Engine) Run() error {
	if err := e.watcher.AddRecursive(e.rootDir); err != nil {
		return err
	}

	trigger, err := e.watcher.Watch()
	if err != nil {
		return err
	}

	e.logger.Info("engine started, running full DAG build")
	e.startFullBuild()

	// Now receives actual file path
	for changedFile := range trigger {
		e.logger.Info("change detected",
			"file", changedFile,
		)
		e.handleChange(changedFile)
	}

	return nil
}

func (e *Engine) handleChange(filePath string) {
	owner := e.dag.OwnerOf(filePath)

	if owner == nil {
		e.logger.Info("no owner found, full rebuild")
		e.cancelCurrent()
		e.startFullBuild()
		return
	}

	e.logger.Info("incremental rebuild",
		"owner_node", owner.Name,
		"file", filePath,
	)
	e.dag.MarkDirty(owner.Name)
	e.cancelCurrent()
	e.startIncrementalBuild()
}

func (e *Engine) cancelCurrent() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cancelBuild != nil {
		e.cancelBuild()
	}
	if e.current != nil {
		e.current.Kill()
		e.current = nil
	}
}

func (e *Engine) startFullBuild() {
	for _, node := range e.dag.Nodes {
		node.SetStatus(dag.StatusPending)
	}
	e.runDAG(false)
}

func (e *Engine) startIncrementalBuild() {
	e.runDAG(true)
}

func (e *Engine) runDAG(incrementalOnly bool) {
	if e.isCrashLooping() {
		e.logger.Warn("crash loop detected, backing off",
			"duration", crashBackoff.String(),
		)
		time.Sleep(crashBackoff)
		e.mu.Lock()
		e.crashTimes = nil
		e.mu.Unlock()
	}

	ctx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	e.cancelBuild = cancel
	e.mu.Unlock()

	go func() {
		defer cancel()

		var err error
		if incrementalOnly {
			err = e.dag.ExecuteDirtyOnly(ctx, e.buildNode)
		} else {
			err = e.dag.Execute(ctx, e.buildNode)
		}

		if err != nil {
			if ctx.Err() != nil {
				e.logger.Info("build cancelled, new build taking over")
				return
			}
			e.logger.Error("dag execution failed", "error", err)
			e.recordCrash()
			return
		}

		e.logger.Info("all nodes built, starting server")
		proc, err := process.Start(e.execCmd, e.logger)
		if err != nil {
			e.logger.Error("failed to start server", "error", err)
			e.recordCrash()
			return
		}

		e.mu.Lock()
		e.current = proc
		e.mu.Unlock()

		e.logger.Info("server running")
	}()
}

func (e *Engine) buildNode(ctx context.Context, node *dag.Node) error {
	e.logger.Info("building node",
		"node", node.Name,
		"cmd", node.Build,
	)

	parts := strings.Fields(node.Build)
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Stdout = logWriter(e.logger, node.Name)
	cmd.Stderr = logWriter(e.logger, node.Name)

	return cmd.Run()
}

func (e *Engine) isCrashLooping() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.crashTimes) < crashThreshold {
		return false
	}
	windowStart := e.crashTimes[len(e.crashTimes)-crashThreshold]
	return time.Since(windowStart) < crashWindow
}

func (e *Engine) recordCrash() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.crashTimes = append(e.crashTimes, time.Now())
	if len(e.crashTimes) > 10 {
		e.crashTimes = e.crashTimes[1:]
	}
}
