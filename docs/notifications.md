# Notifications

AChemDB includes a **modular notification system** that allows you to be informed when reactions fire.

Instead of polling the API to see if anything interesting happened, you can configure reactions to emit **notification events** towards one or more **notifiers** (webhook, WebSocket, etc.).

This document explains:

- how the notification system is structured,
- how to configure notifications in the DSL and via the client package,
- how to manage notifiers via HTTP,
- what the notification event payload looks like,
- what reliability guarantees you can expect.

---

## High-level architecture

Conceptually, the notification path looks like this:

1. A reaction is defined with `notify.enabled = true` and a list of `notifiers`.
2. During a tick, when that reaction **fires** (i.e. produces a non-empty `ReactionEffect`):
   - AChemDB constructs a `NotificationEvent` that describes what happened.
3. The `NotificationEvent` is **enqueued** into the `NotificationManager`.
4. The `NotificationManager`:
   - dispatches the event to the configured notifiers asynchronously,
   - applies retries with backoff for transient failures (e.g. webhooks down),
   - logs failures.

The notification system is **decoupled** from the reaction engine:

- The environment’s `Step()` does _not_ block on HTTP calls or WebSocket writes.
- Delivery is best-effort with retries, but failures do not stop the environment from ticking.

---

## What is a notifier?

A **notifier** is a pluggable component that knows how to deliver a `NotificationEvent` to some target:

- Webhook notifier → HTTP POST JSON to a configured URL.
- WebSocket notifier → send JSON over a WebSocket connection.
- Future notifiers may include: RabbitMQ, Kafka, raw TCP, etc.

Internally, each notifier implements a simple interface (conceptually):

```go
type Notifier interface {
    Notify(ctx context.Context, event NotificationEvent) error
}
```

The `NotificationManager` holds a registry:

- `id` → `Notifier` instance

and uses it to dispatch events.

---

## Configuring notifications in reactions

Notifications are configured **per reaction**.

### JSON DSL

In the JSON DSL (`ReactionConfig`), you add a `notify` section:

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

Fields:

- `enabled`
  - `true` → notification events are emitted for this reaction.
  - `false` or omitted → no notification events.
- `notifiers`
  - list of notifier IDs (strings) that should receive the event.
  - Each ID must match a notifier registered on the server.

A reaction only generates a notification event when it actually produces effects:

- `ConsumedIDs` non-empty, **or**
- `Changes` non-empty, **or**
- `NewMolecules` non-empty.

If the reaction “matches” but does nothing (empty effect), no notification is generated.

---

### Go client (pkg/client)

Using the fluent client, you configure notifications with `Notify(...)` on the reaction builder:

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

The client will serialize this to the same `notify` JSON structure as above.

---

## Notification event structure

When a reaction fires, AChemDB builds a `NotificationEvent` object. Serialized to JSON, it looks roughly like this:

```json
{
  "environment_id": "production",
  "reaction_id": "login_failure_to_suspicion",
  "reaction_name": "Promote login failures to suspicion",
  "timestamp": 1730300000,
  "env_time": 42,
  "input_molecule": {
    "id": "mol-123",
    "species": "Event",
    "payload": {
      "type": "login_failed",
      "ip": "1.2.3.4"
    },
    "energy": 1.0,
    "stability": 1.0,
    "tags": [],
    "created_at": 40,
    "last_touched_at": 42
  },
  "partners": [
    {
      "id": "mol-456",
      "species": "Suspicion",
      "payload": {
        "ip": "1.2.3.4",
        "kind": "login_failed"
      },
      "energy": 1.5,
      "stability": 1.0,
      "created_at": 30,
      "last_touched_at": 42
    }
  ],
  "consumed_molecules": [
    {
      "id": "mol-123",
      "species": "Event",
      "payload": {
        "type": "login_failed",
        "ip": "1.2.3.4"
      },
      "energy": 1.0,
      "stability": 1.0,
      "created_at": 40,
      "last_touched_at": 42
    }
  ],
  "created_molecules": [
    {
      "id": "mol-789",
      "species": "Suspicion",
      "payload": {
        "ip": "1.2.3.4",
        "kind": "login_failed"
      },
      "energy": 1.0,
      "stability": 1.0,
      "created_at": 42,
      "last_touched_at": 42
    }
  ],
  "updated_molecules": [
    {
      "id": "mol-999",
      "species": "Suspicion",
      "payload": {
        "ip": "1.2.3.4",
        "kind": "login_failed"
      },
      "energy": 0.8,
      "stability": 1.0,
      "created_at": 10,
      "last_touched_at": 42
    }
  ],
  "effect": {
    "consumed_ids": ["mol-123"],
    "changes": [
      {
        "id": "mol-999",
        "updated": {
          "id": "mol-999",
          "species": "Suspicion",
          "payload": {
            "ip": "1.2.3.4",
            "kind": "login_failed"
          },
          "energy": 0.8,
          "stability": 1.0,
          "created_at": 10,
          "last_touched_at": 42
        }
      }
    ],
    "new_molecules": [
      {
        "id": "mol-789",
        "species": "Suspicion",
        "payload": {
          "ip": "1.2.3.4",
          "kind": "login_failed"
        },
        "energy": 1.0,
        "stability": 1.0,
        "created_at": 42,
        "last_touched_at": 42
      }
    ]
  }
}
```

Field summary:

- `environment_id` – ID of the environment where the reaction fired.
- `reaction_id` – reaction’s `ID`.
- `reaction_name` – human-friendly `Name`.
- `timestamp` – wall-clock time when the notification was created (epoch seconds or ms).
- `env_time` – environment’s internal time (`EnvTime`) at the tick when the reaction fired.
- `input_molecule` – the primary molecule that triggered the reaction.
- `partners` – partner molecules matched via `input.partners` (if any).
- `consumed_molecules` – molecules removed by this reaction.
- `created_molecules` – molecules created by this reaction.
- `updated_molecules` – molecules changed by this reaction.
- `effect` – raw `ReactionEffect`:
  - `consumed_ids`,
  - `changes` (with `updated`),
  - `new_molecules`.

Not all reactions will populate all arrays. For example:

- a pure “decay” reaction may only populate `updated_molecules`,
- a one-shot “promote to alert” reaction may populate `consumed_molecules` and `created_molecules`.

---

## Managing notifiers via API

Notifiers are managed via the HTTP API of the AChemDB server.

### Register a webhook notifier

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

Fields:

- `type` – `"webhook"` for HTTP POST.
- `id` – unique identifier used in reactions (`notifiers: ["webhook-1"]`).
- `config.url` – target URL.
- `config.headers` – optional map of HTTP headers to attach to each request.

The server will store this configuration and create a `Notifier` instance.

### Register a WebSocket notifier

Example payload (details may depend on your current implementation):

```bash
curl -X POST http://localhost:8080/notifiers \
  -H "Content-Type: application/json" \
  -d '{
    "type": "websocket",
    "id": "websocket-1",
    "config": {
      "path": "/ws/notifications"
    }
  }'
```

- The WebSocket notifier usually works together with a WebSocket endpoint exposed by the AChemDB server.
- Clients connect to that path and receive events as messages.

### List all notifiers

```bash
curl http://localhost:8080/notifiers
```

Typical response:

```json
[
  {
    "id": "webhook-1",
    "type": "webhook",
    "config": {
      "url": "http://your-app.com/webhook"
    }
  },
  {
    "id": "websocket-1",
    "type": "websocket",
    "config": {
      "path": "/ws/notifications"
    }
  }
]
```

### Delete a notifier

```bash
curl -X DELETE http://localhost:8080/notifiers/webhook-1
```

If you remove a notifier that is referenced by reactions:

- those reactions will still try to emit,
- but the missing notifier ID will be logged as an error and skipped.

---

## Delivery model and reliability

### Asynchronous queue

Reaction execution and notification delivery are decoupled via a queue:

- When a reaction fires, the environment:
  - builds the `NotificationEvent`,
  - calls `NotificationManager.Enqueue(event, notifierIDs)`.
- `Enqueue` is **non-blocking** (or best-effort non-blocking):
  - if the internal channel is full, the event may be dropped (logged).
- One or more worker goroutines consume the queue and dispatch events to notifiers.

This design ensures that:

- slow or failing notifiers do not block the simulation engine,
- high bursts of notifications are buffered up to the channel capacity.

### Retries and backoff

For each notifier ID, the `NotificationManager` attempts delivery with a simple retry policy, e.g.:

- up to `maxRetries` attempts (e.g., `3`),
- exponential backoff (e.g., 100ms → 200ms → 400ms → …),
- optional overall timeout per job.

If all attempts fail:

- the failure is logged (with notifier ID, attempts, error),
- the event is dropped (no retry beyond that point).

Future versions may introduce:

- pluggable retry policies,
- dead-letter queues for failed notifications.

### Guarantees

The current design favours **availability and non-blocking behaviour** over strong guarantees.

What you can expect:

- **At-most-once** delivery per notifier:
  - if a delivery fails after all retries, the event is lost (unless stored by the notifier itself).
- **Best-effort ordering**:
  - within a single notifier, events are typically delivered in the order they are enqueued,
  - across different notifiers, ordering is not guaranteed.
- **No persistence of the notification queue**:
  - if AChemDB restarts, in-flight events in memory are lost,
  - snapshots (planned) will not include notification job state.

If you need stronger semantics (exactly-once, durable queues), you can:

- configure a notifier that sends to a reliable external system (e.g. Kafka, RabbitMQ),
- treat that system as your durable notification bus.

---

## Consuming notifications

### Webhook consumer example

A simple HTTP endpoint that receives webhook notifications might look like:

```go
func webhookHandler(w http.ResponseWriter, r *http.Request) {
    defer r.Body.Close()

    var event struct {
        EnvironmentID string `json:"environment_id"`
        ReactionID    string `json:"reaction_id"`
        // ... other fields
    }

    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, "invalid payload", http.StatusBadRequest)
        return
    }

    // Process the event (store, trigger workflows, etc.)
    log.Printf("received notification from env=%s reaction=%s", event.EnvironmentID, event.ReactionID)

    w.WriteHeader(http.StatusOK)
}
```

### WebSocket consumer example

On the client side (pseudo-code):

```js
const ws = new WebSocket("ws://localhost:8080/ws/notifications");

ws.onopen = () => {
  console.log("Connected to AChemDB notifications");
};

ws.onmessage = (msg) => {
  const event = JSON.parse(msg.data);
  console.log("Notification event:", event);
};

ws.onclose = () => {
  console.log("WebSocket closed");
};
```

You can then filter events client-side based on:

- `environment_id`,
- `reaction_id`,
- contents of `input_molecule` or `created_molecules`.

---

## When should you use notifications?

Use AChemDB notifications when:

- you want to react **in real time** to specific reaction events,
- you do **not** want to poll `GET /env/{envID}/molecules` in a tight loop,
- you want to feed a downstream system (alerting, dashboards, workflows, etc.).

Good examples:

- Security alerts → send to SIEM or incident management system via webhook.
- Anomaly detection → push to a metric/monitoring stack.
- Domain workflows → trigger jobs in a worker system when certain patterns occur.

For more background on the overall runtime model, see:

- [Overview](./overview.md)
- [Core concepts](./concepts.md)
