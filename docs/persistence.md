# Persistence and Snapshots

## Purpose of Snapshots

Snapshots in AChemDB serve two primary purposes:

1. **Cold Restore**: Allow environments to be restored from a saved state after a server restart or shutdown. This enables long-running systems to maintain state across restarts.

2. **Crash Recovery**: Provide a recovery point in case of unexpected process termination. The latest snapshot can be loaded to restore the environment to a known good state.

**Important**: Snapshots are **not a Write-Ahead Log (WAL)**. They represent point-in-time state captures, not a sequential log of all changes. Snapshots do not provide transaction-level durability or exact replay capabilities.

## What is Persisted

A snapshot captures the essential state of an environment:

- **`env_id`**: The unique identifier of the environment (type: `EnvironmentID`)
- **`env_time`**: The current time/tick counter of the environment (type: `int64`)
- **`molecules`**: A complete list of all molecules currently in the environment (type: `[]Molecule`)

Each molecule includes:
- `id`: Unique molecule identifier
- `species`: The species name
- `payload`: Arbitrary key-value data
- `energy`: Energy value (float64)
- `stability`: Stability value (float64)
- `tags`: List of string tags
- `created_at`: Timestamp when molecule was created
- `last_touched_at`: Timestamp when molecule was last modified

## What is NOT Persisted

The following state is **not** included in snapshots and will be reset on restore:

- **Notification queue**: Pending notification jobs are lost
- **Reaction engine state**: Internal reaction processing state is not preserved
- **Notifiers**: Active notifier connections and configurations are not saved
- **HTTP server state**: Server connections, sessions, and in-flight requests are not persisted

After restore, the environment will:
- Continue processing reactions from the restored state
- Generate new notifications as reactions fire
- Require re-registration of notifiers and callbacks

## JSON Schema of the Snapshot Format

### Full Example

```json
{
  "environment_id": "security-alerts",
  "time": 12345,
  "molecules": [
    {
      "id": "mol_abc123",
      "species": "Event",
      "payload": {
        "type": "login_failed",
        "ip": "1.2.3.4",
        "user_id": 42
      },
      "energy": 1.0,
      "stability": 1.0,
      "tags": ["security", "authentication"],
      "created_at": 42,
      "last_touched_at": 100
    },
    {
      "id": "mol_def456",
      "species": "Alert",
      "payload": {
        "severity": "high",
        "message": "Multiple failed login attempts"
      },
      "energy": 0.8,
      "stability": 0.9,
      "tags": [],
      "created_at": 100,
      "last_touched_at": 100
    }
  ]
}
```

### Schema Definition

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["environment_id", "time", "molecules"],
  "properties": {
    "environment_id": {
      "type": "string",
      "description": "Unique identifier for the environment"
    },
    "time": {
      "type": "integer",
      "description": "Environment time/tick counter"
    },
    "molecules": {
      "type": "array",
      "description": "List of all molecules in the environment",
      "items": {
        "type": "object",
        "required": ["id", "species", "payload", "energy", "stability", "tags", "created_at", "last_touched_at"],
        "properties": {
          "id": {
            "type": "string",
            "description": "Unique molecule identifier"
          },
          "species": {
            "type": "string",
            "description": "Species name"
          },
          "payload": {
            "type": "object",
            "description": "Arbitrary key-value data",
            "additionalProperties": true
          },
          "energy": {
            "type": "number",
            "description": "Energy value"
          },
          "stability": {
            "type": "number",
            "description": "Stability value"
          },
          "tags": {
            "type": "array",
            "description": "List of string tags",
            "items": {
              "type": "string"
            }
          },
          "created_at": {
            "type": "integer",
            "description": "Timestamp when molecule was created"
          },
          "last_touched_at": {
            "type": "integer",
            "description": "Timestamp when molecule was last modified"
          }
        }
      }
    }
  }
}
```

## Future Extensions

### Write-Ahead Log (WAL)

Future versions may introduce a Write-Ahead Log to complement snapshots:

- **Purpose**: Provide fine-grained replay capabilities and transaction-level durability
- **Content**: Sequential log of all reaction effects, molecule changes, and notification events
- **Use Cases**: 
  - Exact replay for debugging
  - Time-travel queries
  - Full recovery with minimal data loss
- **Interaction with Snapshots**: WAL entries would be applied on top of the latest snapshot during restore

### Sharded Restore

For large environments with many molecules:

- **Sharding Strategy**: Split molecules across multiple snapshot files based on criteria (e.g., species, ID hash)
- **Parallel Loading**: Load shards concurrently to reduce restore time
- **Incremental Snapshots**: Only persist molecules that have changed since the last snapshot
- **Compression**: Apply compression to snapshot files to reduce storage requirements
