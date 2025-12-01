# AchemDB

<div align="center"><img src="./docs/img/achemdb_logo.png" width=250 height=250></div>

AchemDB is a Go library that implements an **artificial chemistry database** - a novel approach to data processing inspired by chemical reactions. Instead of traditional database queries, data entities (molecules) interact through reactions that transform them based on patterns, rates, and environmental conditions.

## Overview

In AchemDB, data is represented as **molecules** that belong to different **species**. These molecules can undergo **reactions** that:

- Transform molecules into new species
- Update molecule properties (energy, stability, payload)
- Consume or create molecules
- React based on probabilistic rates and environmental context

This model is particularly well-suited for:

- Event processing and pattern detection
- Security alert systems
- Anomaly detection
- Complex state machines
- Reactive data pipelines

## Core Concepts

### Molecules

Molecules are the fundamental data entities in AchemDB. Each molecule has:

- **ID**: Unique identifier
- **Species**: The type/class of the molecule (e.g., "Event", "Suspicion", "Alert")
- **Payload**: Arbitrary key-value data
- **Energy**: A numeric value that can decay over time
- **Stability**: Affects how the molecule behaves in reactions
- **Tags**: Optional string tags for categorization
- **Timestamps**: `CreatedAt` and `LastTouchedAt` for lifecycle tracking

### Species

Species define the types of molecules in your system. They're registered in a **Schema** along with their descriptions and metadata.

### Reactions

Reactions define how molecules transform. Each reaction implements:

- **InputPattern**: Determines which molecules the reaction applies to
- **Rate**: Base probability (0.0-1.0) that the reaction occurs
- **EffectiveRate**: Calculates the effective rate considering catalysts
- **Apply**: The transformation logic that produces a `ReactionEffect`

A `ReactionEffect` can:

- Consume molecules (specify IDs in `ConsumedIDs`)
- Update molecules (specify changes in `Changes`)
- Create new molecules (add to `NewMolecules`)
- Perform additional operations (via `AdditionalOps`)

### Catalysts

Catalysts are molecules that increase the reaction rate without being consumed. They can:

- Match by species and optional where conditions
- Boost the base rate by a specified amount
- Set a maximum effective rate limit
- Support multiple catalysts (additive boosts)

### Environment

The **Environment** manages all molecules and applies reactions over discrete time steps. Each `Step()`:

1. Increments the environment time
2. Iterates through all molecules
3. Applies matching reactions probabilistically (using effective rate with catalysts)
4. Updates the molecule state based on reaction effects

### Multiple Environments

AchemDB supports **multiple isolated environments** per database instance. Each environment:

- Has a unique identifier
- Maintains its own molecule collection
- Has its own schema and reactions
- Operates independently from other environments

Molecules can only interact within the same environment - complete isolation is guaranteed.

## Configuration-Based DSL

AchemDB supports a powerful JSON-based DSL for defining schemas and reactions. This allows you to define complex reaction logic without writing Go code.

### Basic Reaction Example

```json
{
  "id": "login_failure_to_suspicion",
  "name": "Promote login failures to suspicion",
  "input": {
    "species": "Event",
    "where": {
      "type": { "eq": "login_failed" }
    }
  },
  "rate": 1.0,
  "effects": [
    { "consume": true },
    {
      "create": {
        "species": "Suspicion",
        "payload": {
          "ip": "$m.ip",
          "kind": "login_failed"
        },
        "energy": 1.0,
        "stability": 1.0
      }
    }
  ]
}
```

### Conditional Effects (If/Then/Else)

```json
{
  "effects": [
    {
      "if": {
        "field": "energy",
        "op": "gt",
        "value": 3.0
      },
      "then": [
        {
          "create": {
            "species": "HighEnergy",
            "payload": { "level": "high" }
          }
        }
      ],
      "else": [
        {
          "create": {
            "species": "LowEnergy",
            "payload": { "level": "low" }
          }
        }
      ]
    }
  ]
}
```

### Count Molecules Aggregation

```json
{
  "effects": [
    {
      "if": {
        "count_molecules": {
          "species": "Suspicion",
          "where": { "ip": { "eq": "$m.ip" } },
          "op": { "gte": 3 }
        }
      },
      "then": [
        {
          "create": {
            "species": "Alert",
            "payload": { "type": "multiple_suspicions" }
          }
        }
      ]
    }
  ]
}
```

### Partner Molecules

```json
{
  "input": {
    "species": "Suspicion",
    "partners": [
      {
        "species": "Suspicion",
        "where": { "ip": { "eq": "$m.ip" } },
        "count": 1
      }
    ]
  },
  "rate": 1.0,
  "effects": [
    {
      "create": {
        "species": "Alert",
        "payload": { "type": "partner_found" }
      }
    }
  ]
}
```

### Catalysts

```json
{
  "id": "catalyzed_reaction",
  "input": {
    "species": "Input"
  },
  "rate": 0.3,
  "catalysts": [
    {
      "species": "Catalyst",
      "where": { "type": { "eq": "$m.type" } },
      "rate_boost": 0.5,
      "max_rate": 0.9
    }
  ],
  "effects": [
    {
      "create": {
        "species": "Output"
      }
    }
  ]
}
```

## Usage as Go package

AchemDB can also be integrated as a Go package directly into your application.

### Example: Security Alert System

The included demo shows how to build a security alert system using Go code:

```go
schema := achem.NewSchema("security-alerts").WithSpecies(
    achem.Species{
        Name:        "Event",
        Description: "Raw security/ops events",
    },
    achem.Species{
        Name:        "Suspicion",
        Description: "Suspicious entities (IP/user/etc.)",
    },
    achem.Species{
        Name:        "Alert",
        Description: "Alerts generated by the system",
    },
).WithReactions(
    NewLoginFailureToSuspicionReaction(),
    NewSuspicionToAlertReaction(),
    NewDecayReaction(),
)

env := achem.NewEnvironment(schema)

// Insert login failure events
for i := 0; i < 5; i++ {
    env.Insert(achem.NewMolecule("Event", map[string]any{
        "type": "login_failed",
        "ip":   "1.2.3.4",
    }, 0))
}

// Run simulation
for range 100 {
    env.Step()
}
```

#### How It Works

1. **LoginFailureToSuspicionReaction**: Converts `Event` molecules with `type="login_failed"` into `Suspicion` molecules
2. **SuspicionToAlertReaction**: When 3+ `Suspicion` molecules exist for the same IP, creates an `Alert` molecule
3. **DecayReaction**: Gradually reduces energy of `Suspicion` and `Alert` molecules; removes them when energy reaches zero

### Usage

#### Creating a Schema

```go
schema := achem.NewSchema("my-system").WithSpecies(
    achem.Species{
        Name:        "MySpecies",
        Description: "Description of this species",
    },
).WithReactions(
    myReaction1,
    myReaction2,
)
```

#### Using Configuration-Based Reactions

You can define reactions using JSON configuration:

```go
import "github.com/daniacca/achemdb/internal/achem"

cfg := achem.SchemaConfig{
    Name: "my-system",
    Species: []achem.SpeciesConfig{
        {
            Name:        "MySpecies",
            Description: "Description of this species",
        },
    },
    Reactions: []achem.ReactionConfig{
        {
            ID:   "my_reaction",
            Name: "My Reaction",
            Input: achem.InputConfig{
                Species: "MySpecies",
                Where: achem.WhereConfig{
                    "status": achem.EqCondition{Eq: "active"},
                },
            },
            Rate: 0.8,
            Effects: []achem.EffectConfig{
                {
                    Consume: true,
                },
                {
                    Create: &achem.CreateEffectConfig{
                        Species: "NewSpecies",
                        Payload: map[string]any{
                            "source": "$m.id",
                        },
                        Energy:    func() *float64 { v := 1.0; return &v }(),
                        Stability: func() *float64 { v := 1.0; return &v }(),
                    },
                },
            },
        },
    },
}

schema, err := achem.BuildSchemaFromConfig(cfg)
if err != nil {
    log.Fatal(err)
}
```

#### Implementing a Custom Reaction (Go)

For advanced use cases, you can implement the `Reaction` interface directly:

```go
type MyReaction struct {
    baseRate float64
}

func (r *MyReaction) ID() string   { return "my_reaction" }
func (r *MyReaction) Name() string { return "My Reaction" }
func (r *MyReaction) Rate() float64 { return r.baseRate }
func (r *MyReaction) EffectiveRate(m achem.Molecule, env achem.EnvView) float64 {
    // Return base rate, or calculate with catalysts
    return r.baseRate
}

func (r *MyReaction) InputPattern(m achem.Molecule) bool {
    return m.Species == "MySpecies"
}

func (r *MyReaction) Apply(m achem.Molecule, env achem.EnvView, ctx achem.ReactionContext) achem.ReactionEffect {
    // Example: Transform molecule (consume input, create output)
    newMol := achem.NewMolecule("NewSpecies", map[string]any{
        "source": m.ID,
    }, ctx.EnvTime)

    return achem.ReactionEffect{
        ConsumedIDs:  []achem.MoleculeID{m.ID}, // Consume the input molecule
        NewMolecules: []achem.Molecule{newMol}, // Create new molecules
    }

    // Alternative: Update molecule without consuming
    // updated := m
    // updated.Energy += 1.0
    // updated.LastTouchedAt = ctx.EnvTime
    // return achem.ReactionEffect{
    //     ConsumedIDs: []achem.MoleculeID{},
    //     Changes: []achem.MoleculeChange{
    //         {ID: m.ID, Updated: &updated},
    //     },
    // }
}
```

### Working with Environments

#### Single Environment

```go
env := achem.NewEnvironment(schema)

// Insert molecules
molecule := achem.NewMolecule("MySpecies", map[string]any{
    "key": "value",
}, env.now())
env.Insert(molecule)

// Run simulation steps
for i := 0; i < 100; i++ {
    env.Step()
}

// Query molecules
allMolecules := env.AllMolecules()
```

#### Multiple Environments

```go
manager := achem.NewEnvironmentManager()

// Create environments
envID1 := achem.EnvironmentID("production")
envID2 := achem.EnvironmentID("staging")

manager.CreateEnvironment(envID1, schema1)
manager.CreateEnvironment(envID2, schema2)

// Get environment
env, exists := manager.GetEnvironment(envID1)
if exists {
    env.Insert(molecule)
    env.Step()
}

// List all environments
envIDs := manager.ListEnvironments()

// Delete environment
manager.DeleteEnvironment(envID1)
```

## Running the DB as standalone application: HTTP Server API

AchemDB includes an HTTP server for remote access. All endpoints are environment-specific.

### Endpoints

- `GET /healthz` - Health check
- `GET /envs` - List all environment IDs
- `POST /env/{envID}/schema` - Create/update environment schema (JSON SchemaConfig)
- `POST /env/{envID}/molecule` - Insert a molecule (JSON: `{"species": "...", "payload": {...}}`)
- `GET /env/{envID}/molecules` - List all molecules in environment
- `POST /env/{envID}/tick` - Manually trigger one step
- `POST /env/{envID}/start?interval=1000` - Start auto-running (interval in milliseconds)
- `POST /env/{envID}/stop` - Stop auto-running
- `DELETE /env/{envID}` - Delete environment

### Example Usage

```bash
# Create environment
curl -X POST http://localhost:8080/env/production/schema \
  -H "Content-Type: application/json" \
  -d @schema.json

# Insert molecule
curl -X POST http://localhost:8080/env/production/molecule \
  -H "Content-Type: application/json" \
  -d '{"species": "Event", "payload": {"type": "login_failed", "ip": "1.2.3.4"}}'

# List molecules
curl http://localhost:8080/env/production/molecules

# Start auto-running
curl -X POST http://localhost:8080/env/production/start?interval=1000

# List all environments
curl http://localhost:8080/envs
```

## Architecture

```
internal/achem/
├── molecule.go              # Molecule data structure
├── species.go              # Species definitions
├── schema.go               # Schema management
├── reaction.go             # Reaction interface
├── environment.go          # Environment and simulation engine
├── environment_manager.go  # Multiple environment management
├── config.go               # Configuration structures (DSL)
├── config_schema_builder.go # Configuration-based reaction builder
└── utils.go                # Utility functions

cmd/
├── achemdb-server/         # HTTP server
│   └── main.go
└── demo/                   # Example application
    ├── main.go
    ├── login_failure_reaction.go
    ├── suspicion_alert_reaction.go
    └── decay_reaction.go
```

## DSL Features

### Field References

In effects and conditions, you can reference molecule fields using `$m.field`:

- `$m.energy` - molecule energy
- `$m.stability` - molecule stability
- `$m.id` - molecule ID
- `$m.species` - molecule species
- `$m.field` - payload field (e.g., `$m.ip`, `$m.type`)

### Comparison Operators

Supported operators for conditions:

- `eq` - equals
- `ne` - not equals
- `gt` - greater than
- `gte` - greater than or equal
- `lt` - less than
- `lte` - less than or equal

### Where Conditions

Where conditions support equality matching:

```json
{
  "where": {
    "field1": { "eq": "value1" },
    "field2": { "eq": "$m.field1" }
  }
}
```

## Requirements

- Go 1.25.4 or later

## Installation (Go package)

```bash
go get github.com/daniacca/achemdb
```

## Running the Demo application

```bash
cd cmd/demo
go run .
```

This will simulate a security alert system and output the final counts of Events, Suspicions, and Alerts.

## Running as standalone DB Server

```bash
cd cmd/achemdb-server
go run .
```

The server will listen on `:8080` by default.

## Client Package

AchemDB provides a fluent client package (`pkg/client`) for building schemas programmatically and sending them to the server. This provides a type-safe, developer-friendly API for defining schemas in Go code.

### Installation

```bash
go get github.com/daniacca/achemdb/pkg/client
```

### Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/daniacca/achemdb/pkg/client"
)

func main() {
    schema := client.NewSchema("security-alerts").
        Species("Event", "Raw events", nil).
        Species("Suspicion", "Suspicious stuff", nil).
        Species("Alert", "Alerts", nil).
        Reaction(client.NewReaction("login_failure_to_suspicion").
            Input("Event", client.WhereEq("type", "login_failed")).
            Rate(1.0).
            Effect(
                client.Consume(),
                client.Create("Suspicion").
                    Payload("ip", client.Ref("m.ip")).
                    Payload("kind", "login_failed").
                    Energy(1.0).
                    Stability(1.0),
            ),
        )

    // Apply schema to server
    ctx := context.Background()
    err := client.ApplySchema(ctx, "http://localhost:8080", "production", schema)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Building Schemas

#### Adding Species

```go
schema := client.NewSchema("my-system").
    Species("Species1", "Description 1", nil).
    Species("Species2", "Description 2", map[string]any{"meta": "value"})
```

#### Building Reactions

```go
reaction := client.NewReaction("my_reaction").
    Name("My Reaction").
    Input("InputSpecies", client.WhereEq("status", "active")).
    Rate(0.8).
    Effect(
        client.Consume(),
        client.Create("OutputSpecies").
            Payload("field", "value").
            Energy(1.0),
    )
```

#### Input Conditions

```go
// Simple input
reaction.Input("Species")

// Input with where conditions
reaction.Input("Species",
    client.WhereEq("field1", "value1"),
    client.WhereEq("field2", 42),
)
```

#### Effects

**Consume:**

```go
client.Consume()
```

**Create:**

```go
client.Create("NewSpecies").
    Payload("key1", "value1").
    Payload("key2", client.Ref("m.field")).  // Reference molecule field
    Energy(2.0).
    Stability(1.5)
```

**Update:**

```go
client.Update().EnergyAdd(0.5)
```

**Conditional Effects (If/Then/Else):**

```go
client.If(client.NewIfField("energy", "gt", 3.0)).
    Then(
        client.Create("HighEnergy"),
    ).
    Else(
        client.Create("LowEnergy"),
    )
```

**Count Molecules Condition:**

```go
client.If(client.NewIfCount(
    client.NewCountMolecules("Suspicion").
        WhereEq("ip", client.Ref("m.ip")).
        Op("gte", 3),
)).
    Then(
        client.Create("Alert").
            Payload("level", "high"),
    )
```

#### Partners

```go
reaction.Input("Suspicion").
    Partner(client.NewPartner("Suspicion").
        WhereEq("ip", client.Ref("m.ip")).
        Count(1),
    )
```

#### Catalysts

```go
reaction.Catalyst(client.NewCatalyst("CatalystSpecies").
    WhereEq("type", "active").
    RateBoost(0.5).
    MaxRate(0.9),
)
```

### Field References

Use `client.Ref()` to reference molecule fields in payloads and conditions:

```go
client.Ref("m.ip")      // References payload field "ip" -> "$m.ip"
client.Ref("ip")        // Also works -> "$m.ip"
client.Ref("m.energy")  // References energy field -> "$m.energy"
```

### Complete Example

```go
schema := client.NewSchema("security-alerts").
    Species("Event", "Raw events", nil).
    Species("Suspicion", "Suspicious entities", nil).
    Species("Alert", "Security alerts", nil).
    Reaction(client.NewReaction("login_failure_to_suspicion").
        Input("Event", client.WhereEq("type", "login_failed")).
        Rate(1.0).
        Effect(
            client.Consume(),
            client.Create("Suspicion").
                Payload("ip", client.Ref("m.ip")).
                Payload("kind", "login_failed").
                Energy(1.0).
                Stability(1.0),
        ),
    ).
    Reaction(client.NewReaction("suspicion_to_alert").
        Input("Suspicion").
        Rate(0.8).
        Effect(
            client.If(client.NewIfCount(
                client.NewCountMolecules("Suspicion").
                    WhereEq("ip", client.Ref("m.ip")).
                    Op("gte", 3),
            )).
                Then(
                    client.Create("Alert").
                        Payload("ip", client.Ref("m.ip")).
                        Payload("level", "high").
                        Energy(5.0),
                ),
        ),
    ).
    Reaction(client.NewReaction("decay").
        Input("Suspicion").
        Rate(1.0).
        Effect(
            client.Update().EnergyAdd(-0.1),
        ),
    )

ctx := context.Background()
err := client.ApplySchema(ctx, "http://localhost:8080", "production", schema)
if err != nil {
    log.Fatal(err)
}
```

### Benefits of the Client Package

- **Type Safety**: Compile-time checking of schema structure
- **Fluent API**: Chainable methods for readable code
- **IDE Support**: Full autocomplete and documentation
- **Reusability**: Build schemas programmatically
- **Integration**: Easy to integrate into Go applications

## Notifications System

AchemDB includes a modular notification system that allows you to be notified when reactions fire. This eliminates the need for polling - the database proactively notifies your application when interesting events occur.

### Notification Architecture

The notification system is **modular** - you can implement different notification channels (webhooks, WebSocket, RabbitMQ, Kafka, etc.) and activate them as needed. Each reaction can be configured to send notifications to specific notifiers.

### Supported Notifiers

Currently implemented:

- **Webhook** - HTTP POST to a webhook URL
- **WebSocket** - Real-time notifications via WebSocket connections

Future notifiers (easy to add):

- RabbitMQ
- Kafka
- TCP Socket

### Configuring Notifications in Reactions

#### Using JSON DSL

```json
{
  "id": "login_failure_to_suspicion",
  "name": "Promote login failures to suspicion",
  "input": {
    "species": "Event",
    "where": {
      "type": { "eq": "login_failed" }
    }
  },
  "rate": 1.0,
  "effects": [
    { "consume": true },
    {
      "create": {
        "species": "Suspicion",
        "payload": {
          "ip": "$m.ip"
        }
      }
    }
  ],
  "notify": {
    "enabled": true,
    "notifiers": ["webhook-1", "websocket-1"]
  }
}
```

#### Using Client Package

```go
schema := client.NewSchema("security-alerts").
    Reaction(client.NewReaction("login_failure_to_suspicion").
        Input("Event", client.WhereEq("type", "login_failed")).
        Rate(1.0).
        Effect(
            client.Consume(),
            client.Create("Suspicion").
                Payload("ip", client.Ref("m.ip")),
        ).
        Notify(client.NewNotification().
            Enabled(true).
            Notifiers("webhook-1", "websocket-1"),
        ),
    )
```

### Managing Notifiers via API

#### Register a Webhook Notifier

```bash
curl -X POST http://localhost:8080/notifiers \
  -H "Content-Type: application/json" \
  -d '{
    "type": "webhook",
    "id": "webhook-1",
    "config": {
      "url": "http://your-app.com/webhook",
      "headers": {
        "Authorization": "Bearer token"
      }
    }
  }'
```

#### List All Notifiers

```bash
curl http://localhost:8080/notifiers
```

#### Unregister a Notifier

```bash
curl -X DELETE http://localhost:8080/notifiers/webhook-1
```

### Notification Event Format

When a reaction fires, a notification event is sent with the following structure:

```json
{
  "environment_id": "production",
  "reaction_id": "login_failure_to_suspicion",
  "reaction_name": "Promote login failures to suspicion",
  "timestamp": 1234567890,
  "env_time": 42,
  "input_molecule": {
    "id": "...",
    "species": "Event",
    "payload": { "type": "login_failed", "ip": "1.2.3.4" },
    "energy": 1.0
  },
  "partners": [],
  "consumed_molecules": [...],
  "created_molecules": [
    {
      "id": "...",
      "species": "Suspicion",
      "payload": { "ip": "1.2.3.4" }
    }
  ],
  "updated_molecules": [],
  "effect": {
    "consumed_ids": [...],
    "changes": [...],
    "new_molecules": [...]
  }
}
```

## Design Philosophy

AchemDB embraces a **reactive, chemistry-inspired** model where:

- Data flows through transformations rather than being queried
- Reactions are declarative and composable
- The system evolves over time through discrete steps
- Probabilistic rates add natural variability
- Molecules have lifecycle properties (energy, stability) that affect behavior

This approach is particularly powerful for systems that need to:

- Detect patterns across multiple data points
- Evolve state over time
- Handle complex, multi-stage transformations
- Model systems with natural decay or aging
