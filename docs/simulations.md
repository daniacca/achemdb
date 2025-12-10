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
  --schema-file=examples/schema_security.json \
  --ticks=200
```

This will:
1. Load and validate the schema
2. Create an environment
3. Run 200 simulation ticks
4. Print a summary of molecule counts per species

### With Seed Molecules

You can seed the environment with initial molecules from a JSON file:

```bash
go run ./cmd/achemdb-sim \
  --schema-file=examples/schema_security.json \
  --ticks=200 \
  --seed=examples/seeds_security.json
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
Simulation finished (schema=security-alerts, ticks=200)
Species counts:
  Alert: 1
  Event: 0
  Suspicion: 3
```

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

## Best Practices

1. **Start Small**: Begin with a small number of ticks (50-100) to understand behavior
2. **Use Seeds**: Seed files help you test specific scenarios and ensure reactions trigger
3. **Monitor Counts**: Watch for species that grow unbounded - this indicates missing decay reactions or guard conditions
4. **Iterate**: Adjust reaction rates and conditions based on simulation results

## Limitations

The simulation tool:
- Runs in a single process (no HTTP server)
- Doesn't support snapshots or persistence
- Doesn't support notifications
- Is intended for testing and development, not production workloads

