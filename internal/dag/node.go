package dag

import (
	"context"
	"sync"
)

type Status int

const (
	StatusPending Status = iota
	StatusRunning
	StatusDone
	StatusFailed
	StatusDirty
)

type Node struct {
	Name    string
	Path    string
	Build   string
	Deps    []string

	mu      sync.Mutex
	status  Status
}

func (n *Node) SetStatus(s Status) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.status = s
}

func (n *Node) GetStatus() Status {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.status
}

func (n *Node) IsDirty() bool {
	return n.GetStatus() == StatusDirty
}

func (n *Node) Reset() {
	n.SetStatus(StatusPending)
}

// BuildFunc is what the DAG calls to build a node
type BuildFunc func(ctx context.Context, node *Node) error
