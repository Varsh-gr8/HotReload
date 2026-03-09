package dag

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

func newTestDAG() *DAG {
	return New(slog.New(slog.NewTextHandler(os.Stdout, nil)))
}

func TestDAGRootRunsFirst(t *testing.T) {
	d := newTestDAG()

	order := []string{}

	d.AddNode(&Node{Name: "utils", Path: "./utils", Build: "echo utils", Deps: []string{}})
	d.AddNode(&Node{Name: "server", Path: "./server", Build: "echo server", Deps: []string{"utils"}})

	if err := d.Build(); err != nil {
		t.Fatal(err)
	}

	err := d.Execute(context.Background(), func(ctx context.Context, n *Node) error {
		order = append(order, n.Name)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	if order[0] != "utils" {
		t.Errorf("expected utils to run first, got %s", order[0])
	}
	if order[1] != "server" {
		t.Errorf("expected server to run second, got %s", order[1])
	}
}

func TestDirtyPropagation(t *testing.T) {
	d := newTestDAG()

	d.AddNode(&Node{Name: "utils", Path: "./utils", Build: "echo utils", Deps: []string{}})
	d.AddNode(&Node{Name: "server", Path: "./server", Build: "echo server", Deps: []string{"utils"}})

	if err := d.Build(); err != nil {
		t.Fatal(err)
	}

	d.MarkDirty("utils")

	if d.Nodes["utils"].GetStatus() != StatusDirty {
		t.Error("utils should be dirty")
	}
	if d.Nodes["server"].GetStatus() != StatusDirty {
		t.Error("server should be dirty because it depends on utils")
	}
}

func TestOwnerOf(t *testing.T) {
	d := newTestDAG()

	d.AddNode(&Node{Name: "utils", Path: "internal/utils", Build: "echo utils", Deps: []string{}})

	if err := d.Build(); err != nil {
		t.Fatal(err)
	}

	owner := d.OwnerOf("internal/utils/helper.go")
	if owner == nil {
		t.Fatal("expected owner to be found")
	}
	if owner.Name != "utils" {
		t.Errorf("expected utils, got %s", owner.Name)
	}
}

func TestUnknownDepReturnsError(t *testing.T) {
	d := newTestDAG()

	d.AddNode(&Node{Name: "server", Path: "./server", Build: "echo server", Deps: []string{"nonexistent"}})

	if err := d.Build(); err == nil {
		t.Error("expected error for unknown dependency")
	}
}
