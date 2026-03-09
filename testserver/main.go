package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

var startTime = time.Now()

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Server started at: %s\n", startTime.Format(time.RFC3339))
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "ok\n")
	})

	logger.Info("testserver starting",
		"port", 8080,
		"started_at", startTime.Format(time.RFC3339),
	)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}
//test3
