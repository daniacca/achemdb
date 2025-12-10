# DSL Reference

AChemDB provides a JSON-based DSL (Domain-Specific Language) for defining schemas and reactions. This allows you to define complex reaction logic without writing Go code.

---

## Schema Structure

A schema is defined as a JSON object with the following structure:

```json
{
  "name": "my-system",
  "species": [
    {
      "name": "Event",
      "description": "Raw events",
      "meta": {}
    }
  ],
  "reactions": [
    {
      "id": "my_reaction",
      "name": "My Reaction",
      "input": { ... },
      "rate": 0.8,
      "catalysts": [ ... ],
      "effects": [ ... ],
      "notify": { ... }
    }
  ]
}
```

### Schema Fields

- `name` (string, required) – Name of the schema
- `species` (array, required) – List of species definitions
- `reactions` (array, required) – List of reaction definitions

---

## Species

A species defines a family of molecules with a common role.

```json
{
  "name": "Event",
  "description": "Raw security events",
  "meta": {
    "category": "input"
  }
}
```

### Species Fields

- `name` (string, required) – Unique species name
- `description` (string, optional) – Human-readable description
- `meta` (object, optional) – Arbitrary metadata for tooling/documentation

---

## Reactions

A reaction defines how molecules transform. Each reaction has:

- **Input pattern** – which molecules it applies to
- **Rate** – probability of firing (0.0–1.0)
- **Catalysts** (optional) – molecules that boost the rate
- **Effects** – what happens when the reaction fires
- **Notifications** (optional) – whether to emit notification events

### Basic Reaction Structure

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

### Reaction Fields

- `id` (string, required) – Unique reaction identifier
- `name` (string, required) – Human-readable name
- `input` (object, required) – Input pattern (see below)
- `rate` (float, required) – Base probability (0.0–1.0)
- `catalysts` (array, optional) – Catalyst definitions (see below)
- `effects` (array, required) – Effect definitions (see below)
- `notify` (object, optional) – Notification configuration (see [Notifications](./notifications.md))

---

## Input Patterns

The `input` field defines which molecules a reaction applies to.

### Simple Input

```json
{
  "input": {
    "species": "Event"
  }
}
```

Matches all molecules of species `Event`.

### Input with Where Conditions

```json
{
  "input": {
    "species": "Event",
    "where": {
      "type": { "eq": "login_failed" },
      "ip": { "eq": "$m.ip" }
    }
  }
}
```

Matches molecules where:

- `species` is `Event`
- `payload.type` equals `"login_failed"`
- `payload.ip` equals the molecule's own `ip` field

### Input with Partners

```json
{
  "input": {
    "species": "Suspicion",
    "partners": [
      {
        "species": "Suspicion",
        "where": {
          "ip": { "eq": "$m.ip" }
        },
        "count": 1
      }
    ]
  }
}
```

Matches a `Suspicion` molecule that has at least 1 partner molecule (another `Suspicion` with the same `ip`).

**Note:** Partners are matched at reaction time, not during input pattern matching. The reaction will only fire if both the input molecule and the required partners exist.

### Input Fields

- `species` (string, required) – Species name to match
- `where` (object, optional) – Conditions on payload fields (see [Where Conditions](#where-conditions))
- `partners` (array, optional) – Partner molecule requirements (see [Partners](#partners))

---

## Where Conditions

Where conditions filter molecules based on payload field values.

### Equality Conditions

```json
{
  "where": {
    "type": { "eq": "login_failed" },
    "status": { "eq": "active" }
  }
}
```

### Field References

You can reference other fields using `$m.field`:

```json
{
  "where": {
    "ip": { "eq": "$m.ip" }
  }
}
```

This matches molecules where `payload.ip` equals the molecule's own `ip` field.

### Supported Operators

Currently, only equality (`eq`) is supported in `where` conditions. Future versions may support additional operators.

---

## Partners

Partners are additional molecules that must exist for a reaction to fire.

```json
{
  "input": {
    "species": "Suspicion",
    "partners": [
      {
        "species": "Suspicion",
        "where": {
          "ip": { "eq": "$m.ip" }
        },
        "count": 2
      }
    ]
  }
}
```

### Partner Fields

- `species` (string, required) – Species of the partner molecule
- `where` (object, optional) – Conditions for matching partners
- `count` (integer, optional) – Minimum number of partners required (default: 1)

**Note:** Partner molecules are distinct from the input molecule. They are matched at reaction time and are not consumed unless explicitly specified in effects.

---

## Rate

The `rate` field specifies the base probability (0.0–1.0) that a reaction fires when its input pattern matches.

```json
{
  "rate": 0.8
}
```

- `0.0` – Never fires
- `1.0` – Always fires (if input matches)
- `0.5` – Fires 50% of the time

---

## Catalysts

Catalysts are molecules that increase the reaction rate without being consumed.

```json
{
  "rate": 0.3,
  "catalysts": [
    {
      "species": "Catalyst",
      "where": {
        "type": { "eq": "$m.type" }
      },
      "rate_boost": 0.5,
      "max_rate": 0.9
    }
  ]
}
```

### Catalyst Fields

- `species` (string, required) – Species of catalyst molecules
- `where` (object, optional) – Conditions for matching catalysts
- `rate_boost` (float, optional) – Amount to add to base rate (default: 0.1)
- `max_rate` (float, optional) – Maximum effective rate (default: 1.0)

### Catalyst Behavior

- Multiple catalysts can boost the rate additively
- The effective rate is: `min(base_rate + sum(rate_boosts), max_rate, 1.0)`
- Catalysts are not consumed by the reaction

---

## Effects

Effects define what happens when a reaction fires. Multiple effects can be specified and are applied in order.

### Consume Effect

Removes the input molecule from the environment:

```json
{
  "effects": [{ "consume": true }]
}
```

### Create Effect

Creates a new molecule:

```json
{
  "effects": [
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

#### Create Fields

- `species` (string, required) – Species of the new molecule
- `payload` (object, optional) – Payload data (supports field references)
- `energy` (float, optional) – Initial energy (default: 0.0)
- `stability` (float, optional) – Initial stability (default: 0.0)

### Update Effect

Updates the input molecule:

```json
{
  "effects": [
    {
      "update": {
        "energy_add": -0.1
      }
    }
  ]
}
```

#### Update Fields

- `energy_add` (float, optional) – Amount to add to energy (can be negative)

**Note:** Update effects do not consume the molecule. To both update and consume, use separate effects.

### Conditional Effects (If/Then/Else)

Apply different effects based on conditions:

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
            "payload": {
              "level": "high"
            }
          }
        }
      ],
      "else": [
        {
          "create": {
            "species": "LowEnergy",
            "payload": {
              "level": "low"
            }
          }
        }
      ]
    }
  ]
}
```

#### If Condition Fields

- `field` (string, required) – Field name (e.g., `"energy"`, `"stability"`, or `"$m.field"` for payload)
- `op` (string, required) – Operator: `"eq"`, `"ne"`, `"gt"`, `"gte"`, `"lt"`, `"lte"`
- `value` (any, required) – Comparison value

#### Count Molecules Condition

Check the count of molecules matching criteria:

```json
{
  "effects": [
    {
      "if": {
        "count_molecules": {
          "species": "Suspicion",
          "where": {
            "ip": { "eq": "$m.ip" }
          },
          "op": { "gte": 3 }
        }
      },
      "then": [
        {
          "create": {
            "species": "Alert",
            "payload": {
              "type": "multiple_suspicions"
            }
          }
        }
      ]
    }
  ]
}
```

#### Count Molecules Fields

- `species` (string, required) – Species to count
- `where` (object, optional) – Conditions for matching molecules
- `op` (object, required) – Operator and value, e.g., `{"gte": 3}` or `{"eq": 5}`

---

## Field References

Throughout the DSL, you can reference molecule fields using `$m.field` syntax:

- `$m.energy` – molecule energy
- `$m.stability` – molecule stability
- `$m.id` – molecule ID
- `$m.species` – molecule species
- `$m.created_at` / `$m.createdAt` / `$m.CreatedAt` – environment time when the molecule was created
- `$m.last_touched_at` / `$m.lastTouchedAt` / `$m.LastTouchedAt` – environment time when the molecule was last mutated
- `$m.field` – payload field (e.g., `$m.ip`, `$m.type`)

You can also use the shorthand form (without `$m.` prefix) for payload fields in some contexts, but `$m.field` is always supported and explicit.

**Note:** Timestamp fields (`created_at` and `last_touched_at`) return numeric values (int64) representing the environment time when the molecule was created or last updated. These values can be used in payload creation, conditions, and comparisons.

### Examples

```json
{
  "payload": {
    "source_ip": "$m.ip",
    "source_energy": "$m.energy",
    "original_id": "$m.id"
  }
}
```

```json
{
  "where": {
    "ip": { "eq": "$m.ip" }
  }
}
```

---

## Comparison Operators

Supported operators for conditional effects:

- `eq` – equals
- `ne` – not equals
- `gt` – greater than
- `gte` – greater than or equal
- `lt` – less than
- `lte` – less than or equal

---

## Complete Examples

### Example 1: Simple Transformation

```json
{
  "id": "event_to_suspicion",
  "name": "Convert events to suspicions",
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
        "energy": 1.0
      }
    }
  ]
}
```

### Example 2: Conditional Alert

```json
{
  "id": "suspicion_to_alert",
  "name": "Create alert from multiple suspicions",
  "input": {
    "species": "Suspicion"
  },
  "rate": 1.0,
  "effects": [
    {
      "if": {
        "count_molecules": {
          "species": "Suspicion",
          "where": {
            "ip": { "eq": "$m.ip" }
          },
          "op": { "gte": 3 }
        }
      },
      "then": [
        {
          "create": {
            "species": "Alert",
            "payload": {
              "ip": "$m.ip",
              "level": "high"
            },
            "energy": 5.0
          }
        }
      ]
    }
  ]
}
```

### Example 3: Decay Reaction

```json
{
  "id": "decay",
  "name": "Decay suspicion energy",
  "input": {
    "species": "Suspicion"
  },
  "rate": 1.0,
  "effects": [
    {
      "update": {
        "energy_add": -0.1
      }
    },
    {
      "if": {
        "field": "energy",
        "op": "lte",
        "value": 0.0
      },
      "then": [{ "consume": true }]
    }
  ]
}
```

### Example 4: Catalyzed Reaction

```json
{
  "id": "catalyzed_reaction",
  "name": "Reaction with catalyst",
  "input": {
    "species": "Input"
  },
  "rate": 0.3,
  "catalysts": [
    {
      "species": "Catalyst",
      "where": {
        "type": { "eq": "$m.type" }
      },
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

---

## See Also

- [Core Concepts](./concepts.md) – Understanding molecules, reactions, and environments
- [HTTP API](./http-api.md) – How to apply schemas via HTTP
- [Notifications](./notifications.md) – Configuring notifications in reactions
