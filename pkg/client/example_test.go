package client_test

import (
	"context"
	"fmt"

	"github.com/daniacca/achemdb/pkg/client"
)

func ExampleSchemaBuilder() {
	schema := client.NewSchema("security-alerts").
		Species("Event", "Raw events", nil).
		Species("Suspicion", "Suspicious stuff", nil).
		Species("Alert", "Alerts", nil).
		Reaction(client.NewReaction("login_failure_to_suspicion").
			Input("Event", client.WhereEq("type", "login_failed")).
			Rate(1.0).
			Effect(
				client.Consume(),
				client.Create("Suspicion").
					Payload("ip", client.Ref("m.ip")).
					Payload("kind", "login_failed").
					Energy(1.0).
					Stability(1.0),
			),
		)

	cfg := schema.Build()
	fmt.Printf("Schema: %s\n", cfg.Name)
	fmt.Printf("Species: %d\n", len(cfg.Species))
	fmt.Printf("Reactions: %d\n", len(cfg.Reactions))

	// Example: Apply to server (commented out for test)
	// ctx := context.Background()
	// err := client.ApplySchema(ctx, "http://localhost:8080", "production", schema)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	_ = schema
}

func ExampleApplySchema() {
	ctx := context.Background()
	schema := client.NewSchema("test").
		Species("Test", "Test species", nil)

	// This would send the schema to the server
	// Uncomment to actually send:
	// err := client.ApplySchema(ctx, "http://localhost:8080", "test-env", schema)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	_ = ctx
	_ = schema
}

func ExampleReactionBuilder_Notify() {
	schema := client.NewSchema("security-alerts").
		Species("Event", "Raw events", nil).
		Species("Alert", "Alerts", nil).
		Reaction(client.NewReaction("event_to_alert").
			Input("Event").
			Rate(1.0).
			Effect(
				client.Consume(),
				client.Create("Alert").
					Payload("message", "Event processed"),
			).
			Notify(client.NewNotification().
				Enabled(true).
				Notifiers("webhook-1", "websocket-1"),
			),
		)

	_ = schema
}

