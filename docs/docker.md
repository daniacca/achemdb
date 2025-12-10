# Running AChemDB in Docker / Docker Compose

AChemDB can be run as a containerized server using Docker or Docker Compose. This guide covers how to configure and run AChemDB in Docker environments.

## Quick Start

### Using Docker

Run the official image directly:

```bash
docker run -p 8080:8080 kaelisra/achemdb:latest
```

This starts AChemDB on port 8080 with default settings. The server will be accessible at `http://localhost:8080`.

### Using Docker Compose

For a more complete setup with persistent data and configuration, use Docker Compose:

```bash
docker-compose up -d
```

See the [Docker Compose Example](#docker-compose-example) section below for a complete configuration.

## Configuration via Environment Variables

AChemDB can be configured entirely through environment variables, making it ideal for containerized deployments. All configuration options support both CLI flags and environment variables, with environment variables taking precedence over defaults (but CLI flags take precedence over environment variables if both are provided).

### Available Environment Variables

#### `ACHEMDB_ADDR`

HTTP listen address inside the container.

- **Default**: `:8080`
- **Example**: `:8080`, `0.0.0.0:8080`, `:9090`
- **Description**: The address and port the server listens on inside the container. Typically `:8080` for containerized deployments.

```bash
docker run -p 8080:8080 -e ACHEMDB_ADDR=":8080" kaelisra/achemdb:latest
```

#### `ACHEMDB_ENV_ID`

Default environment ID used when loading an initial schema at startup.

- **Default**: `default`
- **Example**: `production`, `staging`, `development`
- **Description**: When `ACHEMDB_SCHEMA_FILE` is set, this environment ID is used to create/update the environment with the loaded schema.

```bash
docker run -p 8080:8080 -e ACHEMDB_ENV_ID="production" kaelisra/achemdb:latest
```

#### `ACHEMDB_SCHEMA_FILE`

Optional path to a JSON schema configuration file to load at startup.

- **Default**: (empty, disabled)
- **Example**: `/config/schema.json`
- **Description**: If provided, the server will load and validate the schema from this file at startup, creating or updating the environment specified by `ACHEMDB_ENV_ID`. The file must be mounted as a volume.

```bash
docker run -p 8080:8080 \
  -e ACHEMDB_SCHEMA_FILE="/config/schema.json" \
  -e ACHEMDB_ENV_ID="production" \
  -v $(pwd)/config/schema.json:/config/schema.json:ro \
  kaelisra/achemdb:latest
```

#### `ACHEMDB_SNAPSHOT_DIR`

Directory where environment snapshots are stored.

- **Default**: `./data`
- **Example**: `/data`, `/var/lib/achemdb/snapshots`
- **Description**: The directory inside the container where snapshots are persisted. Should be mounted as a volume to persist data across container restarts.

```bash
docker run -p 8080:8080 \
  -e ACHEMDB_SNAPSHOT_DIR="/data" \
  -v $(pwd)/data:/data \
  kaelisra/achemdb:latest
```

#### `ACHEMDB_SNAPSHOT_EVERY_TICKS`

How often to write snapshots (in number of ticks).

- **Default**: `1000`
- **Example**: `500`, `1000`, `0` (disables periodic snapshots)
- **Description**: The number of ticks between automatic snapshots. Set to `0` to disable periodic snapshots (snapshots can still be triggered manually via the API).

```bash
docker run -p 8080:8080 \
  -e ACHEMDB_SNAPSHOT_EVERY_TICKS="500" \
  kaelisra/achemdb:latest
```

#### `ACHEMDB_LOG_LEVEL`

Log level for server output.

- **Default**: `info`
- **Values**: `debug`, `info`, `warn`, `error`
- **Description**: Controls the verbosity of server logs. Case-insensitive.

```bash
docker run -p 8080:8080 -e ACHEMDB_LOG_LEVEL="debug" kaelisra/achemdb:latest
```

## Docker Compose Example

Here's a complete `docker-compose.yml` example with all configuration options:

```yaml
version: "3.9"

services:
  achemdb:
    image: kaelisra/achemdb:latest
    container_name: achemdb
    ports:
      - "8080:8080"
    environment:
      # HTTP listen address (inside the container)
      ACHEMDB_ADDR: ":8080"

      # Default environment ID used at startup
      ACHEMDB_ENV_ID: "production"

      # Optional: load an initial schema at startup
      # (this file will be mounted via volume below)
      ACHEMDB_SCHEMA_FILE: "/config/schema.json"

      # Snapshot configuration
      ACHEMDB_SNAPSHOT_DIR: "/data"
      ACHEMDB_SNAPSHOT_EVERY_TICKS: "1000"

      # Log level: debug, info, warn, error
      ACHEMDB_LOG_LEVEL: "info"

    volumes:
      # Persist snapshots on the host
      - ./data:/data
      # Mount a schema file from the host
      - ./config/schema.json:/config/schema.json:ro
```

### Understanding the Configuration

#### Ports

- **`8080:8080`**: Maps port 8080 on the host to port 8080 in the container. Change the first number to use a different host port (e.g., `9090:8080` to access via `localhost:9090`).

#### Volumes

- **`./data:/data`**: Persists snapshot data to `./data` on the host. This ensures snapshots survive container restarts and updates.
- **`./config/schema.json:/config/schema.json:ro`**: Mounts a schema file from the host. The `:ro` flag makes it read-only. Create this file following the [DSL Reference](./dsl.md).

#### Environment Variables

All configuration is done via environment variables:

- **`ACHEMDB_ADDR`**: Set to `:8080` to listen on all interfaces inside the container
- **`ACHEMDB_ENV_ID`**: Change to `staging`, `development`, or any identifier you prefer
- **`ACHEMDB_SCHEMA_FILE`**: Path to the mounted schema file (must match the volume mount)
- **`ACHEMDB_SNAPSHOT_DIR`**: Where snapshots are stored (must match the volume mount)
- **`ACHEMDB_SNAPSHOT_EVERY_TICKS`**: Adjust based on your snapshot frequency needs
- **`ACHEMDB_LOG_LEVEL`**: Set to `debug` for verbose logging, `error` for minimal output

### Setting Up the Schema File

Before starting the container, create a schema file at `./config/schema.json`:

```bash
mkdir -p config
cat > config/schema.json << 'EOF'
{
  "name": "my-schema",
  "species": [
    {"name": "Event", "description": "Events"}
  ],
  "reactions": []
}
EOF
```

See the [DSL Reference](./dsl.md) for complete schema syntax.

### Running with Docker Compose

1. **Create the schema file** (see above)

2. **Start the service**:

   ```bash
   docker-compose up -d
   ```

3. **Check logs**:

   ```bash
   docker-compose logs -f achemdb
   ```

4. **Stop the service**:

   ```bash
   docker-compose down
   ```

5. **View snapshots**:
   ```bash
   ls -la ./data/
   ```

## Customizing the Configuration

### Change the Environment ID

Edit `docker-compose.yml`:

```yaml
environment:
  ACHEMDB_ENV_ID: "staging"
```

### Adjust Log Level

For more verbose logging during development:

```yaml
environment:
  ACHEMDB_LOG_LEVEL: "debug"
```

For production, use `info` or `warn`:

```yaml
environment:
  ACHEMDB_LOG_LEVEL: "warn"
```

### Disable Periodic Snapshots

Set `ACHEMDB_SNAPSHOT_EVERY_TICKS` to `0`:

```yaml
environment:
  ACHEMDB_SNAPSHOT_EVERY_TICKS: "0"
```

Snapshots can still be triggered manually via the HTTP API.

### Use a Different Host Port

Change the port mapping:

```yaml
ports:
  - "9090:8080" # Access via localhost:9090
```

### Skip Initial Schema Loading

Remove or comment out the schema file environment variable and volume:

```yaml
environment:
  # ACHEMDB_SCHEMA_FILE: "/config/schema.json"  # Commented out

volumes:
  - ./data:/data
  # - ./config/schema.json:/config/schema.json:ro  # Commented out
```

You can still create schemas via the HTTP API after the server starts.

## Data Persistence

Snapshots are stored in the `./data` directory (or whatever path you configure). Each environment gets its own snapshot file:

```
./data/
  production.snapshot.json
  staging.snapshot.json
  ...
```

**Important**: Always mount the snapshot directory as a volume to ensure data persists across container restarts.

## Health Checks

The server exposes a health check endpoint at `/healthz`. You can verify the container is running:

```bash
curl http://localhost:8080/healthz
# Should return: ok
```

## Troubleshooting

### Container exits immediately

Check the logs:

```bash
docker-compose logs achemdb
```

Common issues:

- Invalid schema file (if `ACHEMDB_SCHEMA_FILE` is set)
- Port already in use (change the host port in `docker-compose.yml`)
- Permission issues with volume mounts

### Snapshots not persisting

Ensure the volume mount is correct and the directory exists:

```bash
mkdir -p ./data
docker-compose up -d
```

### Schema file not found

If `ACHEMDB_SCHEMA_FILE` is set but the file doesn't exist, the server will fail to start. Either:

- Create the schema file at the specified path
- Remove `ACHEMDB_SCHEMA_FILE` from the environment variables

## Building from Source

To build the Docker image from source:

```bash
docker build -t kaelisra/achemdb:latest .
docker run -p 8080:8080 kaelisra/achemdb:latest
```

See the `Dockerfile` in the repository root for build details.
