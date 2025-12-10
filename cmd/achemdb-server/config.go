package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/daniacca/achemdb/internal/achem"
)

// ServerConfig holds the server configuration
type ServerConfig struct {
	Addr              string
	DefaultEnvID      string
	SchemaFile        string
	SnapshotDir       string
	SnapshotEveryTicks int
	LogLevel          string
}

// configResolver defines how to resolve a single configuration value
type configResolver struct {
	flagName    string
	envVarName  string
	defaultVal  string
	description string
	setter      func(*ServerConfig, string)
}

// loadServerConfig loads server configuration from CLI flags and environment variables.
// Uses a resolver pattern to make it easy to add new configuration options.
func loadServerConfig() ServerConfig {
	cfg := ServerConfig{}

	// Define all configuration resolvers
	// To add a new option, just add a new resolver here
	resolvers := []configResolver{
		{
			flagName:    "addr",
			envVarName:  "ACHEMDB_ADDR",
			defaultVal:  ":8080",
			description: "HTTP listen address (e.g. :8080, 0.0.0.0:8080)",
			setter:      func(c *ServerConfig, v string) { c.Addr = v },
		},
		{
			flagName:    "env-id",
			envVarName:  "ACHEMDB_ENV_ID",
			defaultVal:  "default",
			description: "default environment ID for initial schema",
			setter:      func(c *ServerConfig, v string) { c.DefaultEnvID = v },
		},
		{
			flagName:    "schema-file",
			envVarName:  "ACHEMDB_SCHEMA_FILE",
			defaultVal:  "",
			description: "optional path to a JSON schema config file to load at startup",
			setter:      func(c *ServerConfig, v string) { c.SchemaFile = v },
		},
		{
			flagName:    "snapshot-dir",
			envVarName:  "ACHEMDB_SNAPSHOT_DIR",
			defaultVal:  "./data",
			description: "Directory where environment snapshots are stored",
			setter:      func(c *ServerConfig, v string) { c.SnapshotDir = v },
		},
		{
			flagName:    "snapshot-every-ticks",
			envVarName:  "ACHEMDB_SNAPSHOT_EVERY_TICKS",
			defaultVal:  "1000",
			description: "How often to write snapshots (in number of ticks); 0 disables periodic snapshots",
			setter:      func(c *ServerConfig, v string) {
				// Parse int value, with error handling
				if val, err := strconv.Atoi(v); err == nil {
					c.SnapshotEveryTicks = val
				} else {
					log.Printf("Invalid value for snapshot-every-ticks: %s, using default 1000", v)
					c.SnapshotEveryTicks = 1000
				}
			},
		},
		{
			flagName:    "log-level",
			envVarName:  "ACHEMDB_LOG_LEVEL",
			defaultVal:  "info",
			description: "Log level: debug, info, warn, error",
			setter:      func(c *ServerConfig, v string) { c.LogLevel = v },
		},
	}

	// Register string flags first
	flagVars := make(map[string]*string)
	for _, resolver := range resolvers {
		flagVars[resolver.flagName] = flag.String(resolver.flagName, "", resolver.description)
	}

	// Parse flags once
	flag.Parse()

	// Resolve values for each resolver
	for _, resolver := range resolvers {
		var value string
		if *flagVars[resolver.flagName] != "" {
			value = *flagVars[resolver.flagName]
		} else if envValue := os.Getenv(resolver.envVarName); envValue != "" {
			value = envValue
		} else {
			value = resolver.defaultVal
		}
		resolver.setter(&cfg, value)
	}

	return cfg
}

// loadInitialSchemaFromFile loads a schema configuration from a JSON file.
// Returns the SchemaConfig and the built Schema, or an error.
func loadInitialSchemaFromFile(path string) (achem.SchemaConfig, *achem.Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return achem.SchemaConfig{}, nil, err
	}

	var cfg achem.SchemaConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return achem.SchemaConfig{}, nil, err
	}

	// Validate the configuration
	if err := achem.ValidateSchemaConfig(cfg); err != nil {
		return achem.SchemaConfig{}, nil, err
	}

	// Build the schema
	schema, err := achem.BuildSchemaFromConfig(cfg)
	if err != nil {
		return achem.SchemaConfig{}, nil, err
	}

	return cfg, schema, nil
}

// applyInitialSchemaToEnvironment loads a schema from a file and applies it to the environment manager.
// Creates or updates the environment with the given ID.
func applyInitialSchemaToEnvironment(manager *achem.EnvironmentManager, globalNotifierMgr *achem.NotificationManager, schemaFile string, envID achem.EnvironmentID, snapshotDir string, snapshotEveryTicks int) error {
	_, schema, err := loadInitialSchemaFromFile(schemaFile)
	if err != nil {
		return err
	}

	// Try to create the environment, or update if it already exists
	err = manager.CreateEnvironment(envID, schema)
	if err != nil {
		// Environment already exists, update its schema
		if err := manager.UpdateEnvironmentSchema(envID, schema); err != nil {
			return err
		}
	}

	// Set the notification manager and snapshot config for the environment
	env, exists := manager.GetEnvironment(envID)
	if exists {
		env.SetNotificationManager(globalNotifierMgr)
		if snapshotDir != "" {
			env.SetSnapshotDir(snapshotDir)
		}
		if snapshotEveryTicks >= 0 {
			env.SetSnapshotEveryNTicks(snapshotEveryTicks)
		}
	}

	return nil
}

