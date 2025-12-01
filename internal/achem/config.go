package achem

type SpeciesConfig struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Meta        map[string]any `json:"meta,omitempty"`
}

// Where is a map of field -> { eq: value }
type EqCondition struct {
	Eq any `json:"eq"`
}

// Where config can at the moment only use Equality conditions
type WhereConfig map[string]EqCondition

type InputConfig struct {
	Species string       `json:"species"`
	Where   WhereConfig  `json:"where,omitempty"`
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
	// todo: add other effects, like If, Log, etc.
}

type ReactionConfig struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Input   InputConfig    `json:"input"`
	Rate    float64        `json:"rate"`
	Effects []EffectConfig `json:"effects"`
}

type SchemaConfig struct {
	Name      string           `json:"name"`
	Species   []SpeciesConfig  `json:"species"`
	Reactions []ReactionConfig `json:"reactions"`
}
