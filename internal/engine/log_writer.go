package engine

import (
	"bytes"
	"log/slog"
)

type slogWriter struct {
	logger *slog.Logger
	node   string
	buf    bytes.Buffer
}

func logWriter(logger *slog.Logger, node string) *slogWriter {
	return &slogWriter{logger: logger, node: node}
}

func (w *slogWriter) Write(p []byte) (n int, err error) {
	w.logger.Info(string(p), "node", w.node)
	return len(p), nil
}
