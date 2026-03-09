package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Varsh-gr8/HotReload/internal/runner"
	"github.com/Varsh-gr8/HotReload/internal/watcher"
)

type State int

const (
	StateIdle     State = iota
	StateBuilding State = iota
	StateRunning  State = iota
)

func (s State) String() string {
	switch s {
	case StateIdle:
		return "IDLE"
	case StateBuilding:
		return "BUILDING"
	case StateRunning:
		return "RUNNING"
	default:
		return "UNKNOWN"
	}
}

const (
	crashThreshold = 3
	crashWindow    = 10 * time.Second
	crashBackoff   = 5 * time.Second
)

type Engine struct {
	rootDir     string
	runner      *runner.Runner
	watcher     *watcher.Watcher
	logger      *slog.Logger

	mu          sync.Mutex
	state       State
	cancelBuild context.CancelFunc
	crashTimes  []time.Time
}

func New(rootDir, buildCmd, execCmd string, logger *slog.Logger) (*Engine, error) {
	w, err := watcher.New(logger)
	if err != nil {
		return nil, err
	}

	return &Engine{
		rootDir: rootDir,
		runner:  runner.New(buildCmd, execCmd, logger),
		watcher: w,
		logger:  logger,
		state:   StateIdle,
	}, nil
}

func (e *Engine) setState(s State) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.logger.Info("state transition",
		"from", e.state.String(),
		"to", s.String(),
	)
	e.state = s
}

func (e *Engine) getState() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
}

func (e *Engine) Run() error {
	if err := e.watcher.AddRecursive(e.rootDir); err != nil {
		return err
	}

	trigger, err := e.watcher.Watch()
	if err != nil {
		return err
	}

	e.logger.Info("engine started, triggering initial build")
	e.startBuild()

	for range trigger {
		e.logger.Info("change detected",
			"current_state", e.getState().String(),
		)

		switch e.getState() {
		case StateBuilding:
			e.logger.Info("build in progress, cancelling and restarting")
			e.mu.Lock()
			if e.cancelBuild != nil {
				e.cancelBuild()
			}
			e.mu.Unlock()
			e.startBuild()

		case StateRunning, StateIdle:
			e.startBuild()
		}
	}

	return nil
}

func (e *Engine) startBuild() {
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
	e.state = StateBuilding
	e.mu.Unlock()

	e.logger.Info("starting build")

	go func() {
		defer cancel()

		err := e.runner.BuildAndRun(ctx)
		if err != nil {
			if ctx.Err() != nil {
				e.logger.Info("build was cancelled, new build taking over")
				return
			}
			e.logger.Error("build or server failed", "error", err)
			e.setState(StateIdle)
			e.recordCrash()
			return
		}

		e.setState(StateRunning)
		e.logger.Info("server is running")
	}()
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
