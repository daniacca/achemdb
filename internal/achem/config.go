package achem

// SpeciesConfig represents a species configuration used in JSON schemas.
type SpeciesConfig struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Meta        map[string]any `json:"meta,omitempty"`
}

// EqCondition represents an equality condition for filtering molecules.
// Where is a map of field -> { eq: value }
type EqCondition struct {
	Eq any `json:"eq"`
}

// WhereConfig defines filtering conditions for molecules.
// Currently only equality conditions are supported.
type WhereConfig map[string]EqCondition

// ComparisonOp represents a comparison operator for field conditions.
type ComparisonOp string

const (
	OpEq  ComparisonOp = "eq"  // equals
	OpNe  ComparisonOp = "ne"  // not equals
	OpGt  ComparisonOp = "gt"  // greater than
	OpGte ComparisonOp = "gte" // greater than or equal
	OpLt  ComparisonOp = "lt"  // less than
	OpLte ComparisonOp = "lte" // less than or equal
)

// FieldCondition represents a condition on a molecule field
type FieldCondition struct {
	Field string `json:"field"` // field name (e.g., "energy", or "$m.field" for payload)
	Op    string `json:"op"`    // operator: "eq", "ne", "gt", "gte", "lt", "lte"
	Value any    `json:"value"` // comparison value
}

// CountMoleculesConfig represents a count aggregation operation
type CountMoleculesConfig struct {
	Species string      `json:"species"` // species to count
	Where   WhereConfig `json:"where,omitempty"`
	Op      map[string]any `json:"op"`   // operator and value, e.g., {"gte": 3}
}

// IfConditionConfig represents a conditional check
type IfConditionConfig struct {
	// Either a simple field condition
	Field string `json:"field,omitempty"`
	Op    string `json:"op,omitempty"`
	Value any    `json:"value,omitempty"`
	
	// Or a count_molecules aggregation
	CountMolecules *CountMoleculesConfig `json:"count_molecules,omitempty"`
}

// PartnerConfig represents a partner molecule requirement
type PartnerConfig struct {
	Species string      `json:"species"` // species of the partner
	Where   WhereConfig `json:"where,omitempty"`
	Count   int         `json:"count"`   // number of partners required (default: 1)
}

// CatalystConfig represents a catalyst molecule that increases reaction rate
type CatalystConfig struct {
	Species string      `json:"species"`              // species of the catalyst
	Where   WhereConfig `json:"where,omitempty"`      // conditions for catalyst matching
	RateBoost float64   `json:"rate_boost,omitempty"` // amount to add to rate (default: 0.1)
	MaxRate  *float64   `json:"max_rate,omitempty"`   // maximum effective rate (default: 1.0)
}

type InputConfig struct {
	Species  string          `json:"species"`
	Where    WhereConfig     `json:"where,omitempty"`
	Partners []PartnerConfig `json:"partners,omitempty"` // partner molecules required for the reaction
}

type CreateEffectConfig struct {
	Species   string         `json:"species"`
	Payload   map[string]any `json:"payload,omitempty"`
	Energy    *float64       `json:"energy,omitempty"`
	Stability *float64       `json:"stability,omitempty"`
}

type UpdateEffectConfig struct {
	EnergyAdd *float64 `json:"energy_add,omitempty"`
}

type EffectConfig struct {
	Consume bool                 `json:"consume,omitempty"`
	Create  *CreateEffectConfig  `json:"create,omitempty"`
	Update  *UpdateEffectConfig  `json:"update,omitempty"`
	
	// Conditional effects
	If   *IfConditionConfig `json:"if,omitempty"`   // condition to check
	Then []EffectConfig     `json:"then,omitempty"` // effects if condition is true
	Else []EffectConfig     `json:"else,omitempty"` // effects if condition is false
}

type ReactionConfig struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Input     InputConfig      `json:"input"`
	Rate      float64          `json:"rate"`
	Catalysts []CatalystConfig `json:"catalysts,omitempty"` // catalysts that increase reaction rate
	Effects   []EffectConfig   `json:"effects"`
	Notify    *NotificationConfig `json:"notify,omitempty"` // notification configuration
}

type SchemaConfig struct {
	Name      string           `json:"name"`
	Species   []SpeciesConfig  `json:"species"`
	Reactions []ReactionConfig `json:"reactions"`
}
