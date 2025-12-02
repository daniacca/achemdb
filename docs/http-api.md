# HTTP API Reference

AChemDB includes an HTTP server for remote access. All endpoints are environment-specific, allowing you to manage multiple isolated environments from a single server instance.

---

## Base URL

By default, the server listens on `:8080`. All endpoints are prefixed with the base URL:

```
http://localhost:8080
```

---

## Endpoints

### Health Check

**GET** `/healthz`

Check if the server is running.

**Response:**

- `200 OK` – Server is healthy

**Example:**

```bash
curl http://localhost:8080/healthz
```

---

### Environment Management

#### List All Environments

**GET** `/envs`

List all environment IDs.

**Response:**

```json
["production", "staging", "test"]
```

**Example:**

```bash
curl http://localhost:8080/envs
```

#### Create/Update Environment Schema

**POST** `/env/{envID}/schema`

Create or update an environment with a schema definition.

**Path Parameters:**

- `envID` (string) – Environment identifier

**Request Body:**
JSON `SchemaConfig` object (see [DSL Reference](./dsl.md))

**Response:**

- `200 OK` – Schema applied successfully
- `400 Bad Request` – Invalid schema
- `500 Internal Server Error` – Server error

**Example:**

```bash
curl -X POST http://localhost:8080/env/production/schema \
  -H "Content-Type: application/json" \
  -d @schema.json
```

**Schema JSON example:**

```json
{
  "name": "security-alerts",
  "species": [
    {
      "name": "Event",
      "description": "Raw security events"
    },
    {
      "name": "Suspicion",
      "description": "Suspicious entities"
    }
  ],
  "reactions": [
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
            },
            "energy": 1.0
          }
        }
      ]
    }
  ]
}
```

#### Delete Environment

**DELETE** `/env/{envID}`

Delete an environment and all its molecules.

**Path Parameters:**

- `envID` (string) – Environment identifier

**Response:**

- `200 OK` – Environment deleted
- `404 Not Found` – Environment does not exist

**Example:**

```bash
curl -X DELETE http://localhost:8080/env/production
```

---

### Molecule Operations

#### Insert Molecule

**POST** `/env/{envID}/molecule`

Insert a molecule into an environment.

**Path Parameters:**

- `envID` (string) – Environment identifier

**Request Body:**

```json
{
  "species": "Event",
  "payload": {
    "type": "login_failed",
    "ip": "1.2.3.4"
  },
  "energy": 1.0,
  "stability": 1.0,
  "tags": ["security", "login"]
}
```

**Fields:**

- `species` (string, required) – Species name
- `payload` (object, optional) – Payload data
- `energy` (float, optional) – Initial energy (default: 0.0)
- `stability` (float, optional) – Initial stability (default: 0.0)
- `tags` (array, optional) – String tags

**Response:**

- `200 OK` – Molecule created
- `400 Bad Request` – Invalid molecule data
- `404 Not Found` – Environment does not exist

**Example:**

```bash
curl -X POST http://localhost:8080/env/production/molecule \
  -H "Content-Type: application/json" \
  -d '{
    "species": "Event",
    "payload": {
      "type": "login_failed",
      "ip": "1.2.3.4"
    }
  }'
```

#### List All Molecules

**GET** `/env/{envID}/molecules`

List all molecules in an environment.

**Path Parameters:**

- `envID` (string) – Environment identifier

**Response:**

```json
[
  {
    "id": "mol-123",
    "species": "Event",
    "payload": {
      "type": "login_failed",
      "ip": "1.2.3.4"
    },
    "energy": 1.0,
    "stability": 1.0,
    "tags": [],
    "created_at": 42,
    "last_touched_at": 42
  }
]
```

**Example:**

```bash
curl http://localhost:8080/env/production/molecules
```

---

### Simulation Control

#### Manual Tick

**POST** `/env/{envID}/tick`

Manually trigger one simulation step (tick).

**Path Parameters:**

- `envID` (string) – Environment identifier

**Response:**

- `200 OK` – Tick completed
- `404 Not Found` – Environment does not exist

**Example:**

```bash
curl -X POST http://localhost:8080/env/production/tick
```

#### Start Auto-Running

**POST** `/env/{envID}/start?interval={ms}`

Start automatic ticking at a specified interval.

**Path Parameters:**

- `envID` (string) – Environment identifier

**Query Parameters:**

- `interval` (integer, required) – Interval in milliseconds

**Response:**

- `200 OK` – Auto-running started
- `400 Bad Request` – Invalid interval
- `404 Not Found` – Environment does not exist

**Example:**

```bash
curl -X POST "http://localhost:8080/env/production/start?interval=1000"
```

This starts auto-running with a 1-second interval (1000ms).

#### Stop Auto-Running

**POST** `/env/{envID}/stop`

Stop automatic ticking.

**Path Parameters:**

- `envID` (string) – Environment identifier

**Response:**

- `200 OK` – Auto-running stopped
- `404 Not Found` – Environment does not exist

**Example:**

```bash
curl -X POST http://localhost:8080/env/production/stop
```

---

### Notifier Management

#### Register Notifier

**POST** `/notifiers`

Register a new notifier (webhook, WebSocket, etc.).

**Request Body:**

```json
{
  "type": "webhook",
  "id": "webhook-1",
  "config": {
    "url": "http://your-app.com/webhook",
    "headers": {
      "Authorization": "Bearer token"
    }
  }
}
```

**Fields:**

- `type` (string, required) – Notifier type (`"webhook"`, `"websocket"`, etc.)
- `id` (string, required) – Unique notifier identifier
- `config` (object, required) – Notifier-specific configuration

**Webhook Config:**

- `url` (string, required) – Target URL for HTTP POST
- `headers` (object, optional) – HTTP headers to include

**WebSocket Config:**

- `path` (string, optional) – WebSocket path (usually managed by server)

**Response:**

- `200 OK` – Notifier registered
- `400 Bad Request` – Invalid notifier configuration

**Example (Webhook):**

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

**Example (WebSocket):**

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

#### List All Notifiers

**GET** `/notifiers`

List all registered notifiers.

**Response:**

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

**Example:**

```bash
curl http://localhost:8080/notifiers
```

#### Delete Notifier

**DELETE** `/notifiers/{notifierID}`

Unregister a notifier.

**Path Parameters:**

- `notifierID` (string) – Notifier identifier

**Response:**

- `200 OK` – Notifier deleted
- `404 Not Found` – Notifier does not exist

**Example:**

```bash
curl -X DELETE http://localhost:8080/notifiers/webhook-1
```

**Note:** If a notifier is referenced by reactions but deleted, those reactions will log errors when trying to emit notifications.

---

## Complete Workflow Example

Here's a complete example of setting up an environment, inserting molecules, and running the simulation:

```bash
# 1. Create environment with schema
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

# 2. Insert some molecules
curl -X POST http://localhost:8080/env/production/molecule \
  -H "Content-Type: application/json" \
  -d '{"species": "Event", "payload": {"type": "login_failed", "ip": "1.2.3.4"}}'

curl -X POST http://localhost:8080/env/production/molecule \
  -H "Content-Type: application/json" \
  -d '{"species": "Event", "payload": {"type": "login_failed", "ip": "1.2.3.4"}}'

# 3. Run a few ticks manually
curl -X POST http://localhost:8080/env/production/tick
curl -X POST http://localhost:8080/env/production/tick

# 4. Check results
curl http://localhost:8080/env/production/molecules

# 5. Start auto-running (optional)
curl -X POST "http://localhost:8080/env/production/start?interval=1000"

# 6. Stop auto-running (when done)
curl -X POST http://localhost:8080/env/production/stop
```

---

## Error Responses

All endpoints may return standard HTTP error codes:

- `400 Bad Request` – Invalid request data
- `404 Not Found` – Resource (environment, notifier) does not exist
- `500 Internal Server Error` – Server error

Error responses typically include a JSON body with an error message:

```json
{
  "error": "Environment 'production' not found"
}
```

---

## WebSocket Notifications

When using WebSocket notifiers, clients can connect to receive real-time notification events. The WebSocket endpoint path is typically configured per notifier.

**Example WebSocket client (JavaScript):**

```javascript
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

For details on notification event format, see [Notifications](./notifications.md).

---

## See Also

- [DSL Reference](./dsl.md) – Schema and reaction JSON structure
- [Notifications](./notifications.md) – Notification system and event format
- [Overview](./overview.md) – High-level architecture and use cases
