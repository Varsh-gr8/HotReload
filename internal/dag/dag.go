package dag

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
)

type DAG struct {
	Nodes  map[string]*Node
	edges  map[string][]string
	logger *slog.Logger
}

func New(logger *slog.Logger) *DAG {
	return &DAG{
		Nodes:  make(map[string]*Node),
		edges:  make(map[string][]string),
		logger: logger,
	}
}

func (d *DAG) AddNode(n *Node) {
	d.Nodes[n.Name] = n
}

func (d *DAG) Build() error {
	for name, node := range d.Nodes {
		for _, dep := range node.Deps {
			if _, ok := d.Nodes[dep]; !ok {
				return fmt.Errorf("node %q depends on unknown node %q", name, dep)
			}
			d.edges[dep] = append(d.edges[dep], name)
		}
	}
	return nil
}

func (d *DAG) OwnerOf(filePath string) *Node {
	cleanFile := filepath.Clean(filePath)
	for _, node := range d.Nodes {
		if node.Path == "" {
			continue
		}
		cleanNode := filepath.Clean(node.Path)
		if strings.HasPrefix(cleanFile, cleanNode) {
			return node
		}
	}
	return nil
}

func (d *DAG) MarkDirty(name string) {
	node, ok := d.Nodes[name]
	if !ok {
		return
	}
	node.SetStatus(StatusDirty)
	d.logger.Info("node marked dirty", "node", name)
	for _, dependent := range d.edges[name] {
		d.MarkDirty(dependent)
	}
}

func (d *DAG) RootNodes() []*Node {
	var roots []*Node
	for _, node := range d.Nodes {
		if len(node.Deps) == 0 {
			roots = append(roots, node)
		}
	}
	return roots
}

func (d *DAG) Execute(ctx context.Context, fn BuildFunc) error {
	completed := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errCh := make(chan error, len(d.Nodes))

	var runNode func(node *Node)
	runNode = func(node *Node) {
		defer wg.Done()

		for _, dep := range node.Deps {
			for {
				mu.Lock()
				done := completed[dep]
				mu.Unlock()
				if done {
					break
				}
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
		}

		if ctx.Err() != nil {
			return
		}

		node.SetStatus(StatusRunning)
		d.logger.Info("building node", "node", node.Name)

		if err := fn(ctx, node); err != nil {
			node.SetStatus(StatusFailed)
			d.logger.Error("node failed", "node", node.Name, "error", err)
			errCh <- err
			return
		}

		node.SetStatus(StatusDone)
		d.logger.Info("node done", "node", node.Name)

		mu.Lock()
		completed[node.Name] = true
		mu.Unlock()

		for _, dependent := range d.edges[node.Name] {
			depNode := d.Nodes[dependent]
			wg.Add(1)
			go runNode(depNode)
		}
	}

	roots := d.RootNodes()
	for _, root := range roots {
		wg.Add(1)
		go runNode(root)
	}

	wg.Wait()
	close(errCh)

	if err := <-errCh; err != nil {
		return err
	}
	return nil
}

func (d *DAG) ExecuteDirtyOnly(ctx context.Context, fn BuildFunc) error {
	for _, node := range d.Nodes {
		if node.GetStatus() != StatusDirty {
			node.SetStatus(StatusDone)
		}
	}
	return d.Execute(ctx, fn)
}
