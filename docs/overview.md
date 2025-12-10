# AChemDB Overview

AChemDB is an **artificial chemistry database** and simulation engine written in Go.

Instead of querying static rows in tables, you inject **molecules** into a dynamic environment and let **reactions** transform them over time. The engine repeatedly applies reactions according to patterns, probabilities, and contextual rules, producing emergent behaviour.

You can think of it as:

> a small “metabolic runtime” for your data.

---

## Why AChemDB?

Traditional databases are great for:

- durable storage,
- exact queries,
- transactional guarantees.

They are less natural for:

- **pattern detection over streams of events**,
- **multi-step correlation** (e.g. “if these 3 things happen in this order within 5 minutes…”),
- **state that evolves probabilistically over time** (decay, aging, suspicion that grows/shrinks),
- **reactive pipelines** where the system should _push_ interesting events to you.

AChemDB is designed precisely for those cases:

- You model your domain as **species** (types of molecules).
- You define **reactions** that describe how molecules transform.
- You let an **environment** tick over time and fire reactions.
- You optionally subscribe to **notifications** for reactions that matter.

---

## High-level model

At the highest level, AChemDB is built around a few core concepts:

- **Molecule**  
  A data entity with:

  - `species` (type),
  - `payload` (arbitrary key–value),
  - `energy`, `stability`,
  - timestamps and tags.

- **Species**  
  A named family of molecules (e.g. `Event`, `Suspicion`, `Alert`).

- **Reaction**  
  A rule that says _when_ and _how_ molecules transform:

  - which molecules it applies to (`input` + optional `where`),
  - how likely it is to fire (`rate`, `catalysts`),
  - what happens when it does (`effects`: consume, update, create).

- **Environment**  
  A self-contained “broth” of molecules:

  - owns a schema (species + reactions),
  - advances in discrete time steps (`ticks`),
  - applies reactions on a snapshot at each tick.

- **Notifications**  
  When a reaction fires, AChemDB can emit a **notification event** to one or more notifiers (webhook, WebSocket, …). This lets you integrate AChemDB into larger systems without polling.

---

## Typical use cases

AChemDB is not meant to replace your primary database. It is a **runtime** you run alongside your systems.

Some concrete use cases:

### 1. Security / event correlation

- Species:
  - `Event` (raw logs, login failures, access attempts),
  - `Suspicion` (IPs/users under suspicion),
  - `Alert` (security alerts you want to act on).
- Reactions:
  - transform repeated `login_failed` events into `Suspicion`,
  - if enough `Suspicion` accumulate for the same IP/user → create an `Alert`,
  - periodically decay `Suspicion` / `Alert` energy and remove old ones.

The engine becomes a _correlator_ that raises alerts and notifies your systems via webhooks.

### 2. Reactive pipelines / enrichment

- Ingest raw events as `Event` molecules.
- Use reactions to:
  - attach derived fields (geo lookup, risk level),
  - fan out into different species (`Metric`, `Signal`, `Anomaly`),
  - drive follow-up notifications or downstream jobs.

### 3. Complex state machines

- Model states as species (`Pending`, `InProgress`, `Completed`, `Failed`).
- Reactions:
  - move molecules from one species to another when certain conditions are met,
  - apply probabilistic behaviour (e.g. retries, backoff),
  - use `count_molecules` conditions to trigger behaviour when enough entities share a state.

---

## How AChemDB is used

AChemDB can be used in two main ways:

1. **As a Go library**

   - Import `github.com/daniacca/achemdb/internal/achem`.
   - Build a `Schema` in Go or from JSON.
   - Create an `Environment`.
   - Insert molecules, call `Step()` or run with a ticker.
   - Integrate directly in your Go application.

2. **As a standalone HTTP server**

   - Run `cmd/achemdb-server`.
   - Create environments and schemas via HTTP.
   - Insert molecules via REST.
   - Start/stop automatic ticking.
   - Configure notifiers (webhook, WebSocket).
   - Use the `pkg/client` fluent client to build schemas and apply them to the server.

---

## Relationship with your “real” database

AChemDB **does not aim to be your main durable store**. The typical architecture looks like:

- Your app / event bus / main DB:
  - stores raw data durably,
  - forwards selected events to AChemDB (e.g. via HTTP).
- AChemDB:
  - maintains an in-memory population of molecules,
  - runs reactions and generates derived molecules (alerts, signals, …),
  - emits notifications when interesting reactions fire.
- Your app or other services:
  - consume notifications,
  - optionally persist the results back into your primary DB.

Later versions of AChemDB will support **snapshots** and **persistence** to make restarts safer. See `docs/persistence.md` for the design.

---

## Where to go next

- [Core concepts](./concepts.md) – deeper dive into molecules, reactions, environments.
- [DSL reference](./dsl.md) – JSON structures for schemas and reactions.
- [HTTP API](./http-api.md) – how to run AChemDB as a server.
- [Notifications](./notifications.md) – how to receive events when reactions fire.
- [Container & Docker reference](./docker.md) - how to run achemdb in a container.
