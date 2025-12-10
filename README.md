# AchemDB

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
![CI](https://github.com/daniacca/achemdb/actions/workflows/ci.yml/badge.svg)
![Github tag](https://img.shields.io/github/v/tag/daniacca/achemdb)
![Docker Pulls](https://img.shields.io/docker/pulls/kaelisra/achemdb)

<div align="center"><img src="./docs/img/achemdb_logo.png" width=250 height=250></div>

**An artificial chemistry database** – transform data through reactive patterns instead of queries.

---

## What is AchemDB?

AchemDB is a Go library and server that implements an **artificial chemistry database** – a novel approach to data processing inspired by chemical reactions. Instead of traditional database queries, data entities (molecules) interact through reactions that transform them based on patterns, rates, and environmental conditions.

- **Reactive data processing** – Molecules transform through reactions over discrete time steps
- **Pattern detection** – Reactions match molecules by species and conditions, enabling complex correlation
- **Probabilistic behavior** – Reactions fire with configurable rates, creating natural variability
- **Event-driven notifications** – Get notified when reactions fire, no polling required

Perfect for: security alert systems, anomaly detection, event correlation, complex state machines, and reactive data pipelines.

---

## Installation

### Go Package

```bash
go get github.com/daniacca/achemdb
```

### Server Binary

```bash
go install github.com/daniacca/achemdb/cmd/achemdb-server@latest
```

---

## Quickstart

### Minimal Example

Here's a simple example using the Go client:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/daniacca/achemdb/pkg/client"
)

func main() {
	// Create a schema with species and reactions
	schema := client.NewSchema("security-alerts").
		Species("Event", "Raw events", nil).
		Species("Suspicion", "Suspicious entities", nil).
		Reaction(client.NewReaction("login_failure_to_suspicion").
			Input("Event", client.WhereEq("type", "login_failed")).
			Rate(1.0).
			Effect(
				client.Consume(),
				client.Create("Suspicion").
					Payload("ip", client.Ref("m.ip")).
					Energy(1.0),
			),
		)

	// Apply schema to server
	ctx := context.Background()
	if err := client.ApplySchema(ctx, "http://localhost:8080", "production", schema); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Schema applied successfully!")
}
```

### Server Installation

#### From Source

```bash
go install github.com/daniacca/achemdb/cmd/achemdb-server@latest
```

#### Docker

You can run AChemDB as a standalone server using the official Docker image:

```bash
docker run -p 8080:8080 kaelisra/achemdb:latest
```

For advanced configuration (env vars, snapshots, docker-compose examples), see [docs/docker.md](./docs/docker.md).

### 1. Start the server

```bash
achemdb-server
# Server listening on :8080
```

### 2. Create a schema

```bash
curl -X POST http://localhost:8080/env/production/schema \
  -H "Content-Type: application/json" \
  -d '{
    "name": "security-alerts",
    "species": [
      {"name": "Event", "description": "Raw events"},
      {"name": "Suspicion", "description": "Suspicious entities"},
      {"name": "Alert", "description": "Alerts"}
    ],
    "reactions": [
      {
        "id": "login_failure_to_suspicion",
        "name": "Promote login failures",
        "input": {
          "species": "Event",
          "where": {"type": {"eq": "login_failed"}}
        },
        "rate": 1.0,
        "effects": [
          {"consume": true},
          {
            "create": {
              "species": "Suspicion",
              "payload": {"ip": "$m.ip"},
              "energy": 1.0
            }
          }
        ]
      }
    ]
  }'
```

### 3. Insert a molecule

```bash
curl -X POST http://localhost:8080/env/production/molecule \
  -H "Content-Type: application/json" \
  -d '{"species": "Event", "payload": {"type": "login_failed", "ip": "1.2.3.4"}}'
```

### 4. Run a tick

```bash
curl -X POST http://localhost:8080/env/production/tick
```

The reaction will fire, consuming the `Event` and creating a `Suspicion` molecule. Check results:

```bash
curl http://localhost:8080/env/production/molecules
```

---

## Core Concepts

### [Molecules](./docs/concepts.md#molecules)

Data entities with species, payload, energy, stability, and timestamps. Molecules are created, consumed, and transformed by reactions.

### [Reactions & DSL](./docs/dsl.md)

Reactions define how molecules transform. Use the JSON DSL to define:

- Input patterns (species + conditions)
- Probabilistic rates
- Effects (consume, create, update, conditional logic)
- Catalysts (rate boosters)
- Partners (multi-molecule reactions)

### [Notifications](./docs/notifications.md)

Get real-time notifications when reactions fire via webhooks or WebSocket. No polling needed.

### [Environments](./docs/concepts.md#environment-and-ticks)

Isolated containers for molecules and reactions. Each environment has its own schema, molecules, and time. Multiple environments can run on a single server.

---

## Usage as Go Package

AchemDB can also be used as a Go library:

```go
import "github.com/daniacca/achemdb/internal/achem"

schema := achem.NewSchema("my-system").WithSpecies(
    achem.Species{Name: "Event", Description: "Events"},
).WithReactions(...)

env := achem.NewEnvironment(schema)
env.Insert(achem.NewMolecule("Event", map[string]any{"type": "login_failed"}, 0))
env.Step()
```

See [Core Concepts](./docs/concepts.md) for more details on the Go API.

---

## Client Package

Use the fluent Go client to build schemas programmatically:

```go
import "github.com/daniacca/achemdb/pkg/client"

schema := client.NewSchema("security-alerts").
    Species("Event", "Raw events", nil).
    Reaction(client.NewReaction("login_failure_to_suspicion").
        Input("Event", client.WhereEq("type", "login_failed")).
        Rate(1.0).
        Effect(
            client.Consume(),
            client.Create("Suspicion").
                Payload("ip", client.Ref("m.ip")),
        ),
    )

err := client.ApplySchema(ctx, "http://localhost:8080", "production", schema)
```

See the [DSL Reference](./docs/dsl.md) for the equivalent JSON structure.

---

## Requirements

- Go 1.25.4 or later

### Run Demo

```bash
cd cmd/demo
go run .
```

---

## Documentation

- **[Overview](./docs/overview.md)** – High-level architecture, use cases, and design philosophy
- **[Core Concepts](./docs/concepts.md)** – Deep dive into molecules, reactions, environments, and notifications
- **[DSL Reference](./docs/dsl.md)** – Complete JSON schema and reaction syntax
- **[HTTP API](./docs/http-api.md)** – All endpoints, request/response formats, and examples
- **[Notifications](./docs/notifications.md)** – Notification system, event format, and notifier configuration
- **[Persistence](./docs/persistence.md)** – Design for snapshots and persistence (planned)

---

Special thanks to [Francesco Cacciante](./ACKNOWLEDGMENTS.md) and his book for inspiring this works.
