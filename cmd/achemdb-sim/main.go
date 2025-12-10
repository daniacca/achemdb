package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/daniacca/achemdb/internal/achem"
)

type seedMolecule struct {
	Species string         `json:"species"`
	Payload map[string]any `json:"payload"`
}

func main() {
	var (
		schemaFile = flag.String("schema-file", "", "path to schema JSON file (required)")
		ticks      = flag.Int("ticks", 100, "number of ticks to run")
		seedFile   = flag.String("seed", "", "path to seed molecules JSON file (optional)")
		envID      = flag.String("env-id", "simulation", "environment ID")
	)
	flag.Parse()

	if *schemaFile == "" {
		fmt.Fprintf(os.Stderr, "error: --schema-file is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load and validate schema
	cfg, schema, err := loadSchemaFromFile(*schemaFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading schema: %v\n", err)
		os.Exit(1)
	}

	// Create environment
	env := achem.NewEnvironment(schema)
	env.SetEnvironmentID(achem.EnvironmentID(*envID))

	// Load seed molecules if provided
	if *seedFile != "" {
		if err := loadSeedMolecules(env, *seedFile); err != nil {
			fmt.Fprintf(os.Stderr, "error loading seed molecules: %v\n", err)
			os.Exit(1)
		}
	}

	// Run simulation
	for i := 0; i < *ticks; i++ {
		env.Step()
	}

	// Print summary
	printSummary(cfg.Name, *ticks, env)
}

func loadSchemaFromFile(path string) (achem.SchemaConfig, *achem.Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return achem.SchemaConfig{}, nil, fmt.Errorf("reading schema file: %w", err)
	}

	var cfg achem.SchemaConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return achem.SchemaConfig{}, nil, fmt.Errorf("parsing schema JSON: %w", err)
	}

	// Validate the configuration
	if err := achem.ValidateSchemaConfig(cfg); err != nil {
		return achem.SchemaConfig{}, nil, fmt.Errorf("validating schema: %w", err)
	}

	// Build the schema
	schema, err := achem.BuildSchemaFromConfig(cfg)
	if err != nil {
		return achem.SchemaConfig{}, nil, fmt.Errorf("building schema: %w", err)
	}

	return cfg, schema, nil
}

func loadSeedMolecules(env *achem.Environment, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading seed file: %w", err)
	}

	var seeds []seedMolecule
	if err := json.Unmarshal(data, &seeds); err != nil {
		return fmt.Errorf("parsing seed JSON: %w", err)
	}

	// Insert each seed molecule with CreatedAt and LastTouchedAt = 0
	for _, seed := range seeds {
		m := achem.NewMolecule(achem.SpeciesName(seed.Species), seed.Payload, 0)
		// Override timestamps to 0 as requested
		m.CreatedAt = 0
		m.LastTouchedAt = 0
		env.Insert(m)
	}

	return nil
}

func printSummary(schemaName string, ticks int, env *achem.Environment) {
	molecules := env.AllMolecules()

	// Count molecules by species
	counts := make(map[achem.SpeciesName]int)
	for _, m := range molecules {
		counts[m.Species]++
	}

	fmt.Printf("Simulation finished (schema=%s, ticks=%d)\n", schemaName, ticks)
	fmt.Println("Species counts:")
	
	// Print in a consistent order (sorted by species name)
	speciesList := make([]achem.SpeciesName, 0, len(counts))
	for species := range counts {
		speciesList = append(speciesList, species)
	}
	
	// Simple sort by string value
	for i := 0; i < len(speciesList); i++ {
		for j := i + 1; j < len(speciesList); j++ {
			if speciesList[i] > speciesList[j] {
				speciesList[i], speciesList[j] = speciesList[j], speciesList[i]
			}
		}
	}

	for _, species := range speciesList {
		fmt.Printf("  %s: %d\n", species, counts[species])
	}
}

