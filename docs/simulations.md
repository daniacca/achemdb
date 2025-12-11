# Simulation Tool

The `achemdb-sim` CLI tool allows you to run simulations of AChemDB schemas locally without needing a running server. This is useful for:

- Testing schema behavior
- Understanding reaction dynamics
- Validating that schemas don't create unbounded molecule populations
- Debugging schema configurations

## Installation

Build the simulation tool:

```bash
go build ./cmd/achemdb-sim
```

Or run directly:

```bash
go run ./cmd/achemdb-sim --help
```

## Usage

### Basic Simulation

Run a simulation with a schema file:

```bash
go run ./cmd/achemdb-sim \
  --schema-file=examples/schema/security.json \
  --ticks=10
```

This will:

1. Load and validate the schema
2. Create an environment
3. Run 10 simulation ticks
4. Print a summary of molecule counts per species

### With Seed Molecules

You can seed the environment with initial molecules from a JSON file:

```bash
go run ./cmd/achemdb-sim \
  --schema-file=examples/schema/security.json \
  --ticks=10 \
  --seed=examples/seed/security.json
```

The seed file format is a JSON array of molecules:

```json
[
  {
    "species": "Event",
    "payload": {
      "type": "login_failed",
      "ip": "1.2.3.4"
    }
  },
  {
    "species": "Event",
    "payload": {
      "type": "login_failed",
      "ip": "1.2.3.4"
    }
  }
]
```

### Command-Line Options

- `--schema-file` (required): Path to schema JSON file
- `--ticks` (optional, default: 100): Number of simulation ticks to run
- `--seed` (optional): Path to seed molecules JSON file
- `--env-id` (optional, default: "simulation"): Environment ID (mainly for logging)

### Example Output

```
Simulation finished (schema=security-alerts, ticks=10)
Species counts:
  Event: 1
  Suspicion: 5
```

**Note:** The number of ticks matters! Molecules decay over time, so:

- Too few ticks: Reactions may not have time to fire
- Too many ticks: All molecules may decay away
- Optimal range: 5-30 ticks depending on the schema (see schema-specific examples below)

## Automated Tests

The example schemas in `examples/` are covered by automated tests in `internal/achem/simulation_test.go`. These tests:

- Load each example schema
- Seed appropriate initial molecules
- Run a moderate number of simulation steps (100 ticks)
- Assert basic sanity conditions:
  - Total molecule count doesn't exceed conservative bounds (1000)
  - No single species exceeds reasonable limits (500)
  - Higher-level species appear when expected
  - Decay reactions eventually remove temporary molecules

Run the tests:

```bash
go test ./internal/achem -v -run TestSimulation
```

These tests serve as a regression harness to ensure:

- Schemas remain stable and don't create infinite/unbounded molecule populations
- Future changes to the DSL or engine don't break existing schemas
- Decay and consumption patterns work correctly

## Schema-Specific Examples

Each schema has been tested with seed data. Here are working examples:

### Security Schema

```bash
go run ./cmd/achemdb-sim \
  --schema-file=examples/schema/security.json \
  --ticks=10 \
  --seed=examples/seed/security.json
```

**Optimal ticks:** 5-10 (Suspicion molecules decay after ~10 ticks)

### Ecommerce Schema

```bash
go run ./cmd/achemdb-sim \
  --schema-file=examples/schema/ecommerce.json \
  --ticks=20 \
  --seed=examples/seed/ecommerce.json
```

**Optimal ticks:** 5-30 (Shows CartItem → Purchase → Recommendation flow)

### Monitoring Schema

```bash
go run ./cmd/achemdb-sim \
  --schema-file=examples/schema/monitoring.json \
  --ticks=10 \
  --seed=examples/seed/monitoring.json
```

**Optimal ticks:** 5-30 (Shows Metric → Baseline → Anomaly → Incident progression)

### IoT Schema

```bash
go run ./cmd/achemdb-sim \
  --schema-file=examples/schema/iot.json \
  --ticks=10 \
  --seed=examples/seed/iot.json
```

**Optimal ticks:** 5-10 (Shows SensorReading → Alert → Maintenance flow)

### Default Schema

```bash
go run ./cmd/achemdb-sim \
  --schema-file=examples/schema/default.json \
  --ticks=20 \
  --seed=examples/seed/default.json
```

**Optimal ticks:** 5-20 (Shows Event → Processed transformation)

## Best Practices

1. **Start Small**: Begin with 5-10 ticks to see initial reactions, then increase to 20-30 for full progression
2. **Use Seeds**: Seed files help you test specific scenarios and ensure reactions trigger
3. **Monitor Counts**: Watch for species that grow unbounded - this indicates missing decay reactions or guard conditions
4. **Adjust Ticks**: If you see empty results, try fewer ticks (molecules may have decayed). If you see only initial species, try more ticks.
5. **Iterate**: Adjust reaction rates and conditions based on simulation results

## Limitations

The simulation tool:

- Runs in a single process (no HTTP server)
- Doesn't support snapshots or persistence
- Doesn't support notifications
- Is intended for testing and development, not production workloads
