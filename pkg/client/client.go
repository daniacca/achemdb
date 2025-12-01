package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/daniacca/achemdb/internal/achem"
)

// SchemaBuilder provides a fluent API for building schemas
type SchemaBuilder struct {
	name      string
	species   []achem.SpeciesConfig
	reactions []*ReactionBuilder
}

// NewSchema creates a new schema builder
func NewSchema(name string) *SchemaBuilder {
	return &SchemaBuilder{
		name:      name,
		species:   make([]achem.SpeciesConfig, 0),
		reactions: make([]*ReactionBuilder, 0),
	}
}

// Species adds a species to the schema
func (sb *SchemaBuilder) Species(name, description string, meta map[string]any) *SchemaBuilder {
	sb.species = append(sb.species, achem.SpeciesConfig{
		Name:        name,
		Description: description,
		Meta:        meta,
	})
	return sb
}

// Reaction adds a reaction to the schema
func (sb *SchemaBuilder) Reaction(rb *ReactionBuilder) *SchemaBuilder {
	sb.reactions = append(sb.reactions, rb)
	return sb
}

// Build converts the builder to a SchemaConfig
func (sb *SchemaBuilder) Build() achem.SchemaConfig {
	reactions := make([]achem.ReactionConfig, 0, len(sb.reactions))
	for _, rb := range sb.reactions {
		reactions = append(reactions, rb.Build())
	}

	return achem.SchemaConfig{
		Name:      sb.name,
		Species:   sb.species,
		Reactions: reactions,
	}
}

// ReactionBuilder provides a fluent API for building reactions
type ReactionBuilder struct {
	id        string
	name      string
	input     *InputBuilder
	rate      float64
	catalysts []*CatalystBuilder
	effects   []*EffectBuilder
}

// NewReaction creates a new reaction builder
func NewReaction(id string) *ReactionBuilder {
	return &ReactionBuilder{
		id:        id,
		name:      id, // Default name to ID
		rate:      1.0,
		catalysts: make([]*CatalystBuilder, 0),
		effects:   make([]*EffectBuilder, 0),
	}
}

// Name sets the reaction name
func (rb *ReactionBuilder) Name(name string) *ReactionBuilder {
	rb.name = name
	return rb
}

// Input sets the input species and optional where conditions
func (rb *ReactionBuilder) Input(species string, whereEqs ...func(*InputBuilder)) *ReactionBuilder {
	ib := NewInput(species)
	for _, fn := range whereEqs {
		fn(ib)
	}
	rb.input = ib
	return rb
}

// Rate sets the reaction rate
func (rb *ReactionBuilder) Rate(rate float64) *ReactionBuilder {
	rb.rate = rate
	return rb
}

// Catalyst adds a catalyst to the reaction
func (rb *ReactionBuilder) Catalyst(cb *CatalystBuilder) *ReactionBuilder {
	rb.catalysts = append(rb.catalysts, cb)
	return rb
}

// Effect adds effects to the reaction
// Accepts EffectBuilder, CreateEffectBuilder, UpdateEffectBuilder, or IfEffectBuilder
func (rb *ReactionBuilder) Effect(ebs ...interface{}) *ReactionBuilder {
	for _, e := range ebs {
		switch v := e.(type) {
		case *EffectBuilder:
			rb.effects = append(rb.effects, v)
		case *CreateEffectBuilder:
			rb.effects = append(rb.effects, &EffectBuilder{create: v})
		case *UpdateEffectBuilder:
			rb.effects = append(rb.effects, &EffectBuilder{update: v})
		case *IfEffectBuilder:
			rb.effects = append(rb.effects, &EffectBuilder{ifCond: v.ifCond})
		}
	}
	return rb
}

// Build converts the builder to a ReactionConfig
func (rb *ReactionBuilder) Build() achem.ReactionConfig {
	input := achem.InputConfig{}
	if rb.input != nil {
		input = rb.input.Build()
	}

	catalysts := make([]achem.CatalystConfig, 0, len(rb.catalysts))
	for _, cb := range rb.catalysts {
		catalysts = append(catalysts, cb.Build())
	}

	effects := make([]achem.EffectConfig, 0, len(rb.effects))
	for _, eb := range rb.effects {
		effects = append(effects, eb.Build())
	}

	return achem.ReactionConfig{
		ID:        rb.id,
		Name:      rb.name,
		Input:     input,
		Rate:      rb.rate,
		Catalysts: catalysts,
		Effects:   effects,
	}
}

// InputBuilder provides a fluent API for building input configurations
type InputBuilder struct {
	species  string
	where    achem.WhereConfig
	partners []*PartnerBuilder
}

// NewInput creates a new input builder
func NewInput(species string) *InputBuilder {
	return &InputBuilder{
		species:  species,
		where:    make(achem.WhereConfig),
		partners: make([]*PartnerBuilder, 0),
	}
}

// WhereEq is a helper function that returns a function to add where conditions
func WhereEq(field string, value any) func(*InputBuilder) {
	return func(ib *InputBuilder) {
		if ib.where == nil {
			ib.where = make(achem.WhereConfig)
		}
		ib.where[field] = achem.EqCondition{Eq: value}
	}
}

// WhereEq adds an equality condition to the where clause (method on InputBuilder)
func (ib *InputBuilder) WhereEq(field string, value any) *InputBuilder {
	if ib.where == nil {
		ib.where = make(achem.WhereConfig)
	}
	ib.where[field] = achem.EqCondition{Eq: value}
	return ib
}

// Partner adds a partner requirement
func (ib *InputBuilder) Partner(pb *PartnerBuilder) *InputBuilder {
	ib.partners = append(ib.partners, pb)
	return ib
}

// Build converts the builder to an InputConfig
func (ib *InputBuilder) Build() achem.InputConfig {
	partners := make([]achem.PartnerConfig, 0, len(ib.partners))
	for _, pb := range ib.partners {
		partners = append(partners, pb.Build())
	}

	return achem.InputConfig{
		Species:  ib.species,
		Where:    ib.where,
		Partners: partners,
	}
}

// PartnerBuilder provides a fluent API for building partner configurations
type PartnerBuilder struct {
	species string
	where   achem.WhereConfig
	count   int
}

// NewPartner creates a new partner builder
func NewPartner(species string) *PartnerBuilder {
	return &PartnerBuilder{
		species: species,
		where:   make(achem.WhereConfig),
		count:   1, // Default count
	}
}

// WhereEq adds an equality condition
func (pb *PartnerBuilder) WhereEq(field string, value any) *PartnerBuilder {
	if pb.where == nil {
		pb.where = make(achem.WhereConfig)
	}
	pb.where[field] = achem.EqCondition{Eq: value}
	return pb
}

// Count sets the required count of partners
func (pb *PartnerBuilder) Count(count int) *PartnerBuilder {
	pb.count = count
	return pb
}

// Build converts the builder to a PartnerConfig
func (pb *PartnerBuilder) Build() achem.PartnerConfig {
	return achem.PartnerConfig{
		Species: pb.species,
		Where:   pb.where,
		Count:   pb.count,
	}
}

// CatalystBuilder provides a fluent API for building catalyst configurations
type CatalystBuilder struct {
	species   string
	where     achem.WhereConfig
	rateBoost float64
	maxRate   *float64
}

// NewCatalyst creates a new catalyst builder
func NewCatalyst(species string) *CatalystBuilder {
	return &CatalystBuilder{
		species:   species,
		where:     make(achem.WhereConfig),
		rateBoost: 0.1, // Default boost
	}
}

// WhereEq adds an equality condition
func (cb *CatalystBuilder) WhereEq(field string, value any) *CatalystBuilder {
	if cb.where == nil {
		cb.where = make(achem.WhereConfig)
	}
	cb.where[field] = achem.EqCondition{Eq: value}
	return cb
}

// RateBoost sets the rate boost amount
func (cb *CatalystBuilder) RateBoost(boost float64) *CatalystBuilder {
	cb.rateBoost = boost
	return cb
}

// MaxRate sets the maximum effective rate
func (cb *CatalystBuilder) MaxRate(max float64) *CatalystBuilder {
	cb.maxRate = &max
	return cb
}

// Build converts the builder to a CatalystConfig
func (cb *CatalystBuilder) Build() achem.CatalystConfig {
	return achem.CatalystConfig{
		Species:   cb.species,
		Where:     cb.where,
		RateBoost: cb.rateBoost,
		MaxRate:   cb.maxRate,
	}
}

// EffectBuilder provides a fluent API for building effects
type EffectBuilder struct {
	consume bool
	create  *CreateEffectBuilder
	update  *UpdateEffectBuilder
	ifCond  *IfConditionBuilder
}

// Consume creates a consume effect
func Consume() *EffectBuilder {
	return &EffectBuilder{
		consume: true,
	}
}

// Create creates a create effect builder
func Create(species string) *CreateEffectBuilder {
	return &CreateEffectBuilder{
		species: species,
		payload: make(map[string]any),
	}
}

// Update creates an update effect builder
func Update() *UpdateEffectBuilder {
	return &UpdateEffectBuilder{}
}

// If creates a conditional effect builder
func If(icb *IfConditionBuilder) *IfEffectBuilder {
	return &IfEffectBuilder{
		ifCond: icb,
	}
}

// IfEffectBuilder wraps an IfConditionBuilder to provide Then/Else methods
type IfEffectBuilder struct {
	ifCond *IfConditionBuilder
}

// Then adds effects to execute if condition is true
func (ieb *IfEffectBuilder) Then(ebs ...interface{}) *IfEffectBuilder {
	for _, e := range ebs {
		switch v := e.(type) {
		case *EffectBuilder:
			ieb.ifCond.then = append(ieb.ifCond.then, v)
		case *CreateEffectBuilder:
			ieb.ifCond.then = append(ieb.ifCond.then, &EffectBuilder{create: v})
		case *UpdateEffectBuilder:
			ieb.ifCond.then = append(ieb.ifCond.then, &EffectBuilder{update: v})
		}
	}
	return ieb
}

// Else adds effects to execute if condition is false
func (ieb *IfEffectBuilder) Else(ebs ...interface{}) *IfEffectBuilder {
	for _, e := range ebs {
		switch v := e.(type) {
		case *EffectBuilder:
			ieb.ifCond.else_ = append(ieb.ifCond.else_, v)
		case *CreateEffectBuilder:
			ieb.ifCond.else_ = append(ieb.ifCond.else_, &EffectBuilder{create: v})
		case *UpdateEffectBuilder:
			ieb.ifCond.else_ = append(ieb.ifCond.else_, &EffectBuilder{update: v})
		}
	}
	return ieb
}

// Build converts the builder to an EffectConfig
func (eb *EffectBuilder) Build() achem.EffectConfig {
	effect := achem.EffectConfig{
		Consume: eb.consume,
	}

	if eb.create != nil {
		effect.Create = eb.create.Build()
	}

	if eb.update != nil {
		effect.Update = eb.update.Build()
	}

	if eb.ifCond != nil {
		effect.If = eb.ifCond.Build()
		effect.Then = make([]achem.EffectConfig, 0, len(eb.ifCond.then))
		effect.Else = make([]achem.EffectConfig, 0, len(eb.ifCond.else_))
		for _, teb := range eb.ifCond.then {
			effect.Then = append(effect.Then, teb.Build())
		}
		for _, eeb := range eb.ifCond.else_ {
			effect.Else = append(effect.Else, eeb.Build())
		}
	}

	return effect
}

// CreateEffectBuilder provides a fluent API for building create effects
type CreateEffectBuilder struct {
	species   string
	payload   map[string]any
	energy    *float64
	stability *float64
}

// Payload adds a payload field
func (ceb *CreateEffectBuilder) Payload(field string, value any) *CreateEffectBuilder {
	if ceb.payload == nil {
		ceb.payload = make(map[string]any)
	}
	ceb.payload[field] = value
	return ceb
}

// Energy sets the energy value
func (ceb *CreateEffectBuilder) Energy(energy float64) *CreateEffectBuilder {
	ceb.energy = &energy
	return ceb
}

// Stability sets the stability value
func (ceb *CreateEffectBuilder) Stability(stability float64) *CreateEffectBuilder {
	ceb.stability = &stability
	return ceb
}

// Build converts the builder to a CreateEffectConfig
func (ceb *CreateEffectBuilder) Build() *achem.CreateEffectConfig {
	return &achem.CreateEffectConfig{
		Species:   ceb.species,
		Payload:   ceb.payload,
		Energy:    ceb.energy,
		Stability: ceb.stability,
	}
}

// UpdateEffectBuilder provides a fluent API for building update effects
type UpdateEffectBuilder struct {
	energyAdd *float64
}

// EnergyAdd sets the energy addition amount
func (ueb *UpdateEffectBuilder) EnergyAdd(amount float64) *UpdateEffectBuilder {
	ueb.energyAdd = &amount
	return ueb
}

// Build converts the builder to an UpdateEffectConfig
func (ueb *UpdateEffectBuilder) Build() *achem.UpdateEffectConfig {
	return &achem.UpdateEffectConfig{
		EnergyAdd: ueb.energyAdd,
	}
}

// IfConditionBuilder provides a fluent API for building conditional effects
type IfConditionBuilder struct {
	field          string
	op             string
	value          any
	countMolecules *CountMoleculesBuilder
	then           []*EffectBuilder
	else_          []*EffectBuilder
}

// NewIfField creates a field-based condition
func NewIfField(field, op string, value any) *IfConditionBuilder {
	return &IfConditionBuilder{
		field: field,
		op:    op,
		value: value,
		then:  make([]*EffectBuilder, 0),
		else_: make([]*EffectBuilder, 0),
	}
}

// NewIfCount creates a count_molecules condition
func NewIfCount(cmb *CountMoleculesBuilder) *IfConditionBuilder {
	return &IfConditionBuilder{
		countMolecules: cmb,
		then:           make([]*EffectBuilder, 0),
		else_:          make([]*EffectBuilder, 0),
	}
}

// Then adds effects to execute if condition is true
func (icb *IfConditionBuilder) Then(eb ...*EffectBuilder) *IfConditionBuilder {
	icb.then = append(icb.then, eb...)
	return icb
}

// Else adds effects to execute if condition is false
func (icb *IfConditionBuilder) Else(eb ...*EffectBuilder) *IfConditionBuilder {
	icb.else_ = append(icb.else_, eb...)
	return icb
}

// Build converts the builder to an IfConditionConfig
func (icb *IfConditionBuilder) Build() *achem.IfConditionConfig {
	cond := &achem.IfConditionConfig{}

	if icb.countMolecules != nil {
		cond.CountMolecules = icb.countMolecules.Build()
	} else {
		cond.Field = icb.field
		cond.Op = icb.op
		cond.Value = icb.value
	}

	return cond
}

// CountMoleculesBuilder provides a fluent API for building count_molecules conditions
type CountMoleculesBuilder struct {
	species string
	where   achem.WhereConfig
	op      map[string]any
}

// NewCountMolecules creates a new count molecules builder
func NewCountMolecules(species string) *CountMoleculesBuilder {
	return &CountMoleculesBuilder{
		species: species,
		where:   make(achem.WhereConfig),
		op:      make(map[string]any),
	}
}

// WhereEq adds an equality condition
func (cmb *CountMoleculesBuilder) WhereEq(field string, value any) *CountMoleculesBuilder {
	if cmb.where == nil {
		cmb.where = make(achem.WhereConfig)
	}
	cmb.where[field] = achem.EqCondition{Eq: value}
	return cmb
}

// Op sets the comparison operator and value
func (cmb *CountMoleculesBuilder) Op(operator string, value any) *CountMoleculesBuilder {
	if cmb.op == nil {
		cmb.op = make(map[string]any)
	}
	cmb.op[operator] = value
	return cmb
}

// Build converts the builder to a CountMoleculesConfig
func (cmb *CountMoleculesBuilder) Build() *achem.CountMoleculesConfig {
	return &achem.CountMoleculesConfig{
		Species: cmb.species,
		Where:   cmb.where,
		Op:      cmb.op,
	}
}

// Ref creates a reference to a molecule field
// Accepts either "field" (becomes "$m.field") or "m.field" (becomes "$m.field")
func Ref(field string) string {
	if len(field) > 2 && field[:2] == "m." {
		return "$" + field
	}
	return "$m." + field
}

// ApplySchema sends the schema to the server
func ApplySchema(ctx context.Context, baseURL, envID string, schema *SchemaBuilder) error {
	cfg := schema.Build()

	// Convert to JSON
	jsonData, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	// Build URL
	u, err := url.JoinPath(baseURL, "env", envID, "schema")
	if err != nil {
		return fmt.Errorf("failed to build URL: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
