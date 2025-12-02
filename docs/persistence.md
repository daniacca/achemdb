# Persistence (Design)

> **Status:** design document – persistence is not implemented yet.  
> This file describes the planned approach for snapshots and restarts.

AChemDB is primarily an **in-memory engine**. By default:

- molecules live only in memory,
- if the process is restarted, all environments start empty and `EnvTime` resets.

For many scenarios this is OK (e.g. purely derived alerts), but for long-running systems we want:

- the ability to **restart without losing all state**,
- predictable behaviour across restarts.

This document outlines the planned design for persistence.

---

## Goals

- Provide **periodic snapshots** of each environment:
  - include all molecules,
  - include the environment time.
- On restart, the server can **load the latest snapshot** for each environment.
- Keep implementation simple:
  - no transactional log, no full ACID semantics,
  - “best effort” persistence that is good enough for many use cases.
- Avoid impacting the core reaction engine’s performance too much.

Later versions may add an **event log / WAL** for finer-grained replay, but that is out of scope for the initial persistence.

---

## Snapshot model

We introduce a concept of **environment snapshot**:

```json
{
  "env_id": "security-alerts",
  "schema_name": "security-alerts",
  "time": 12345,
  "molecules": [
    {
      "id": "...",
      "species": "Event",
      "payload": { "type": "login_failed", "ip": "1.2.3.4" },
      "energy": 1.0,
      "stability": 1.0,
      "tags": [],
      "created_at": 42,
      "last_touched_at": 100
    }
    // ...
  ]
}
```

Notes:

- The snapshot is **per environment**.
- It is logically independent of how the schema is defined:
  - to restore an environment correctly, you must use a compatible schema.

The on-disk representation can be:

- JSON (human-readable, easier to debug),
- or a binary format later for performance.

The first version will likely use JSON for simplicity.

---

## When snapshots are taken

Snapshots are planned to be taken in one of two ways:

1. **Periodic snapshots** (recommended)
   - Each environment can be configured with:
     - `snapshot_interval` (in wall-clock time), **or**
     - `snapshot_every_n_ticks` (in ticks).
   - A background goroutine:
     - sleeps for the configured interval,
     - grabs the environment lock,
     - copies all molecules into a slice,
     - writes a snapshot file to disk.
2. **Manual snapshots** (optional)
   - Provide an HTTP endpoint or Go API:
     - `POST /env/{envID}/snapshot`
   - This forces an immediate snapshot.

To avoid blocking the engine:

- we can copy the in-memory state under lock,
- but write the file **asynchronously** in a separate goroutine.

---

## Where snapshots are stored

AchemDB server will be configurable with:

- a base directory for snapshots, e.g. `--snapshot-dir=./data`.
  For each environment, we can use:
- a single file that is periodically overwritten, e.g.:
  - `data/env_<envID>_snapshot.json`
- or a rolling set of snapshot files, e.g.: - `data/env_<envID>_snapshot_0001.json` - `data/env_<envID>_snapshot_0002.json`
  The first version will likely use a single “latest” file per environment:
- simpler to manage,
- avoids unbounded disk growth.

---

## Restore on startup

On startup, the server will:

1. Look in the snapshot directory for files matching existing environment IDs.
2. For each snapshot file:
   - parse JSON into a `Snapshot` struct,
   - create an environment with the given `env_id` and schema,
   - set `EnvTime` from the snapshot,
   - populate `mols` map with the snapshot molecules.

The schema used for restoration must be **compatible**:

- same species names,
- same expectations for payload fields.

The responsibility of providing the right schema (e.g. via `/env/{envID}/schema`) remains with the user:

- In a simple setup, you apply your schema first, then AChemDB loads snapshots.
- In more advanced setup, the schema could be loaded from its own config file.

---

## Interaction with notifications

Snapshots are **not** planned to capture:

- pending notification jobs,
- retry state for notifiers,
- open WebSocket connections.

After a restart:

- reactions will continue from the restored state,
- new notifications will be emitted as reactions fire,
- in-flight notifications at the time of crash may be lost.

For systems that need stronger guarantees, future work may add:

- a persistent log of `NotificationEvent`s,
- explicit ack/retry mechanisms on the consumer side.

---

## Limitations and trade-offs

The snapshot-based persistence intentionally does **not** provide:

- transaction-level durability (no guarantee about the exact point-in-time consistency),
- exact replay of random behaviour (randomness per tick is not logged),
- full audit log of all reactions.

Instead, it aims to be:

- **simple**, easy to reason about,
- good enough to avoid “complete loss of state” on restart,
- optional – you can run AChemDB fully in-memory if you prefer.

---

## Future directions

Potential future improvements:

- **Event log / WAL**:
  - log every `ReactionEffect` or `NotificationEvent`,
  - support replay for debugging, time-travel or full recovery.
- **Configurable snapshot policies** per environment:
  - different snapshot intervals for hot vs cold environments.
- **Pluggable storage backends**:
  - write snapshots to S3, GCS, or other object stores.

For now, the first implementation will focus on:

- per-environment JSON snapshots,
- periodic snapshotting,
- best-effort restore on startup.
