package process

import (
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type Process struct {
	cmd    *exec.Cmd
	logger *slog.Logger
}

func Start(command string, logger *slog.Logger) (*Process, error) {
	parts := strings.Fields(command)
	cmd := exec.Command(parts[0], parts[1:]...)

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	logger.Info("process started",
		"pid", cmd.Process.Pid,
		"cmd", command,
	)

	return &Process{cmd: cmd, logger: logger}, nil
}

func (p *Process) Kill() {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return
	}

	pid := p.cmd.Process.Pid

	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		p.logger.Warn("could not get pgid, killing pid directly",
			"pid", pid,
			"error", err,
		)
		p.cmd.Process.Kill()
		return
	}

	p.logger.Info("sending SIGTERM", "pgid", pgid)
	syscall.Kill(-pgid, syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		p.cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("process exited cleanly after SIGTERM", "pgid", pgid)
	case <-time.After(2 * time.Second):
		p.logger.Warn("process ignored SIGTERM, sending SIGKILL", "pgid", pgid)
		syscall.Kill(-pgid, syscall.SIGKILL)
	}
}
