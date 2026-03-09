# HotReload

A CLI hot-reload engine for Go projects. Watches for file changes and automatically rebuilds and restarts your server.

## Architecture

HotReload models the build process as a **DAG (Directed Acyclic Graph)**:

- Each node represents a build command
- Parent nodes execute before their dependents
- Independent sibling nodes execute in parallel
- On file change, only the affected node and its dependents rebuild
```
file change detected
      ↓
identify owner node via path matching
      ↓
mark node dirty → propagate dirty flag downward
      ↓
kill running server (SIGTERM → SIGKILL)
      ↓
rebuild only dirty nodes in parallel where possible
      ↓
start server when all nodes done
```

## Usage

### Simple project (single build command)
```bash
hotreload --root ./myproject \
          --build "go build -o ./bin/server ./cmd/server" \
          --exec "./bin/server"
```

### Multi-package project (via config)

Create `hotreload.yaml`:
```yaml
exec: "./bin/server"
nodes:
  - name: utils
    path: ./internal/utils
    build: "go build ./internal/utils"
    deps: []

  - name: db
    path: ./internal/db
    build: "go build ./internal/db"
    deps: [utils]

  - name: server
    path: ./cmd/server
    build: "go build -o ./bin/server ./cmd/server"
    deps: [db, utils]
```

Then run:
```bash
hotreload --config hotreload.yaml
```

## Run the demo
```bash
make demo
```

## Run tests
```bash
make test
```

## Build
```bash
make build
```

## Features

- DAG-based execution — root nodes first, siblings in parallel
- Incremental rebuilds — only affected nodes rebuild on file change
- Dirty flag propagation — dependents of changed node rebuild automatically
- Debounced file watching — rapid saves trigger only one rebuild
- Process group killing — entire process tree killed on restart
- SIGTERM → SIGKILL — graceful shutdown with force kill fallback
- Crash loop protection — backs off after 3 crashes in 10 seconds
- New directory detection — watches folders created at runtime
- File filtering — only .go files trigger rebuilds
- YAML config — declare multi-node dependency trees
- CLI fallback — works without config via --build and --exec flags
