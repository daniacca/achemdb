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

// SchemaBuilder provides a fluent API for building schemas.
// Use it to define species and reactions that describe how molecules
// interact in an artificial chemistry system.
type SchemaBuilder struct {
	name      string
	species   []achem.SpeciesConfig
	reactions []*ReactionBuilder
}

// NewSchema creates a new schema builder with the given name.
// The name identifies the schema and is used for organization purposes.
func NewSchema(name string) *SchemaBuilder {
	return &SchemaBuilder{
		name:      name,
		species:   make([]achem.SpeciesConfig, 0),
		reactions: make([]*ReactionBuilder, 0),
	}
}

// Species adds a species definition to the schema.
// A species represents a type of molecule in the system.
// The meta parameter can be nil or contain additional metadata.
func (sb *SchemaBuilder) Species(name, description string, meta map[string]any) *SchemaBuilder {
	sb.species = append(sb.species, achem.SpeciesConfig{
		Name:        name,
		Description: description,
		Meta:        meta,
	})
	return sb
}

// Reaction adds a reaction definition to the schema.
// Reactions define how molecules transform when they interact.
func (sb *SchemaBuilder) Reaction(rb *ReactionBuilder) *SchemaBuilder {
	sb.reactions = append(sb.reactions, rb)
	return sb
}

// Build converts the builder to a SchemaConfig that can be used
// with ApplySchema or other AChemDB APIs.
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

// ReactionBuilder provides a fluent API for building reaction configurations.
// Reactions define how molecules of a specific species transform,
// including input patterns, rates, catalysts, and effects.
type ReactionBuilder struct {
	id        string
	name      string
	input     *InputBuilder
	rate      float64
	catalysts []*CatalystBuilder
	effects   []*EffectBuilder
	notify    *NotificationBuilder
}

// NewReaction creates a new reaction builder with the given ID.
// The ID must be unique within a schema. The name defaults to the ID
// but can be overridden with the Name method.
func NewReaction(id string) *ReactionBuilder {
	return &ReactionBuilder{
		id:        id,
		name:      id, // Default name to ID
		rate:      1.0,
		catalysts: make([]*CatalystBuilder, 0),
		effects:   make([]*EffectBuilder, 0),
	}
}

// Name sets the human-readable name for the reaction.
// If not set, the name defaults to the reaction ID.
func (rb *ReactionBuilder) Name(name string) *ReactionBuilder {
	rb.name = name
	return rb
}

// Input sets the input species and optional where conditions for the reaction.
// The whereEqs parameter allows chaining WhereEq calls to filter which
// molecules of the species can trigger this reaction.
func (rb *ReactionBuilder) Input(species string, whereEqs ...func(*InputBuilder)) *ReactionBuilder {
	ib := NewInput(species)
	for _, fn := range whereEqs {
		fn(ib)
	}
	rb.input = ib
	return rb
}

// Rate sets the base reaction rate, a value between 0.0 and 1.0.
// This represents the probability that the reaction will fire when
// a matching molecule is available. The effective rate can be modified
// by catalysts.
func (rb *ReactionBuilder) Rate(rate float64) *ReactionBuilder {
	rb.rate = rate
	return rb
}

// Catalyst adds a catalyst configuration to the reaction.
// Catalysts increase the reaction rate when matching molecules are present.
func (rb *ReactionBuilder) Catalyst(cb *CatalystBuilder) *ReactionBuilder {
	rb.catalysts = append(rb.catalysts, cb)
	return rb
}

// Effect adds one or more effects to the reaction.
// Effects define what happens when the reaction fires, such as consuming
// the input molecule, creating new molecules, or updating existing ones.
// Accepts EffectBuilder, CreateEffectBuilder, UpdateEffectBuilder, or IfEffectBuilder.
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

// Notify configures notification settings for this reaction.
// When enabled, notifications are sent when the reaction fires,
// allowing external systems to react to events in real-time.
func (rb *ReactionBuilder) Notify(nb *NotificationBuilder) *ReactionBuilder {
	rb.notify = nb
	return rb
}

// Build converts the builder to a ReactionConfig that can be used
// in schema definitions.
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

	reactionCfg := achem.ReactionConfig{
		ID:        rb.id,
		Name:      rb.name,
		Input:     input,
		Rate:      rb.rate,
		Catalysts: catalysts,
		Effects:   effects,
	}

	if rb.notify != nil {
		reactionCfg.Notify = rb.notify.Build()
	}

	return reactionCfg
}

// InputBuilder provides a fluent API for building input configurations.
// Inputs define which molecules can trigger a reaction, including
// species filtering, where conditions, and partner requirements.
type InputBuilder struct {
	species  string
	where    achem.WhereConfig
	partners []*PartnerBuilder
}

// NewInput creates a new input builder for the specified species.
func NewInput(species string) *InputBuilder {
	return &InputBuilder{
		species:  species,
		where:    make(achem.WhereConfig),
		partners: make([]*PartnerBuilder, 0),
	}
}

// WhereEq is a helper function that returns a function to add equality
// conditions to an input builder. This is useful for chaining conditions
// when calling ReactionBuilder.Input.
func WhereEq(field string, value any) func(*InputBuilder) {
	return func(ib *InputBuilder) {
		if ib.where == nil {
			ib.where = make(achem.WhereConfig)
		}
		ib.where[field] = achem.EqCondition{Eq: value}
	}
}

// WhereEq adds an equality condition to the where clause.
// Only molecules with the specified field matching the value will match.
func (ib *InputBuilder) WhereEq(field string, value any) *InputBuilder {
	if ib.where == nil {
		ib.where = make(achem.WhereConfig)
	}
	ib.where[field] = achem.EqCondition{Eq: value}
	return ib
}

// Partner adds a partner molecule requirement to the input.
// Partners are additional molecules that must be present for the reaction to fire.
func (ib *InputBuilder) Partner(pb *PartnerBuilder) *InputBuilder {
	ib.partners = append(ib.partners, pb)
	return ib
}

// Build converts the builder to an InputConfig.
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

// PartnerBuilder provides a fluent API for building partner molecule configurations.
// Partners are additional molecules required for a reaction to fire.
type PartnerBuilder struct {
	species string
	where   achem.WhereConfig
	count   int
}

// NewPartner creates a new partner builder for the specified species.
func NewPartner(species string) *PartnerBuilder {
	return &PartnerBuilder{
		species: species,
		where:   make(achem.WhereConfig),
		count:   1, // Default count
	}
}

// WhereEq adds an equality condition to filter partner molecules.
func (pb *PartnerBuilder) WhereEq(field string, value any) *PartnerBuilder {
	if pb.where == nil {
		pb.where = make(achem.WhereConfig)
	}
	pb.where[field] = achem.EqCondition{Eq: value}
	return pb
}

// Count sets the required number of partner molecules.
// The default is 1 if not specified.
func (pb *PartnerBuilder) Count(count int) *PartnerBuilder {
	pb.count = count
	return pb
}

// Build converts the builder to a PartnerConfig.
func (pb *PartnerBuilder) Build() achem.PartnerConfig {
	return achem.PartnerConfig{
		Species: pb.species,
		Where:   pb.where,
		Count:   pb.count,
	}
}

// CatalystBuilder provides a fluent API for building catalyst configurations.
// Catalysts increase the reaction rate when matching molecules are present
// in the environment.
type CatalystBuilder struct {
	species   string
	where     achem.WhereConfig
	rateBoost float64
	maxRate   *float64
}

// NewCatalyst creates a new catalyst builder for the specified species.
func NewCatalyst(species string) *CatalystBuilder {
	return &CatalystBuilder{
		species:   species,
		where:     make(achem.WhereConfig),
		rateBoost: 0.1, // Default boost
	}
}

// WhereEq adds an equality condition to filter catalyst molecules.
func (cb *CatalystBuilder) WhereEq(field string, value any) *CatalystBuilder {
	if cb.where == nil {
		cb.where = make(achem.WhereConfig)
	}
	cb.where[field] = achem.EqCondition{Eq: value}
	return cb
}

// RateBoost sets the amount by which the reaction rate is increased
// for each matching catalyst molecule. The default is 0.1.
func (cb *CatalystBuilder) RateBoost(boost float64) *CatalystBuilder {
	cb.rateBoost = boost
	return cb
}

// MaxRate sets the maximum effective reaction rate, even when multiple
// catalysts are present. If not set, the rate can exceed 1.0.
func (cb *CatalystBuilder) MaxRate(max float64) *CatalystBuilder {
	cb.maxRate = &max
	return cb
}

// Build converts the builder to a CatalystConfig.
func (cb *CatalystBuilder) Build() achem.CatalystConfig {
	return achem.CatalystConfig{
		Species:   cb.species,
		Where:     cb.where,
		RateBoost: cb.rateBoost,
		MaxRate:   cb.maxRate,
	}
}

// EffectBuilder provides a fluent API for building reaction effects.
// Effects define what happens when a reaction fires, such as consuming
// molecules, creating new ones, or updating existing ones.
type EffectBuilder struct {
	consume bool
	create  *CreateEffectBuilder
	update  *UpdateEffectBuilder
	ifCond  *IfConditionBuilder
}

// Consume creates an effect that consumes (removes) the input molecule
// when the reaction fires.
func Consume() *EffectBuilder {
	return &EffectBuilder{
		consume: true,
	}
}

// Create creates an effect builder for creating new molecules of the
// specified species when the reaction fires.
func Create(species string) *CreateEffectBuilder {
	return &CreateEffectBuilder{
		species: species,
		payload: make(map[string]any),
	}
}

// Update creates an effect builder for updating existing molecules
// when the reaction fires.
func Update() *UpdateEffectBuilder {
	return &UpdateEffectBuilder{}
}

// If creates a conditional effect builder that executes different effects
// based on a condition. Use NewIfField or NewIfCount to create the condition.
func If(icb *IfConditionBuilder) *IfEffectBuilder {
	return &IfEffectBuilder{
		ifCond: icb,
	}
}

// IfEffectBuilder wraps an IfConditionBuilder to provide Then/Else methods
// for conditional effect execution.
type IfEffectBuilder struct {
	ifCond *IfConditionBuilder
}

// Then adds effects to execute if the condition is true.
// Accepts EffectBuilder, CreateEffectBuilder, or UpdateEffectBuilder.
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

// Else adds effects to execute if the condition is false.
// Accepts EffectBuilder, CreateEffectBuilder, or UpdateEffectBuilder.
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

// Build converts the builder to an EffectConfig.
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

// CreateEffectBuilder provides a fluent API for building create effects.
// Create effects generate new molecules when a reaction fires.
type CreateEffectBuilder struct {
	species   string
	payload   map[string]any
	energy    *float64
	stability *float64
}

// Payload adds a field to the payload of the created molecule.
// The value can be a literal or a reference using Ref() to copy
// values from the input molecule.
func (ceb *CreateEffectBuilder) Payload(field string, value any) *CreateEffectBuilder {
	if ceb.payload == nil {
		ceb.payload = make(map[string]any)
	}
	ceb.payload[field] = value
	return ceb
}

// Energy sets the initial energy value for the created molecule.
// If not set, the molecule will use the default energy value.
func (ceb *CreateEffectBuilder) Energy(energy float64) *CreateEffectBuilder {
	ceb.energy = &energy
	return ceb
}

// Stability sets the initial stability value for the created molecule.
// If not set, the molecule will use the default stability value.
func (ceb *CreateEffectBuilder) Stability(stability float64) *CreateEffectBuilder {
	ceb.stability = &stability
	return ceb
}

// Build converts the builder to a CreateEffectConfig.
func (ceb *CreateEffectBuilder) Build() *achem.CreateEffectConfig {
	return &achem.CreateEffectConfig{
		Species:   ceb.species,
		Payload:   ceb.payload,
		Energy:    ceb.energy,
		Stability: ceb.stability,
	}
}

// UpdateEffectBuilder provides a fluent API for building update effects.
// Update effects modify existing molecules when a reaction fires.
type UpdateEffectBuilder struct {
	energyAdd *float64
}

// EnergyAdd sets the amount to add to the molecule's energy.
// The energy is modified in place when the reaction fires.
func (ueb *UpdateEffectBuilder) EnergyAdd(amount float64) *UpdateEffectBuilder {
	ueb.energyAdd = &amount
	return ueb
}

// Build converts the builder to an UpdateEffectConfig.
func (ueb *UpdateEffectBuilder) Build() *achem.UpdateEffectConfig {
	return &achem.UpdateEffectConfig{
		EnergyAdd: ueb.energyAdd,
	}
}

// IfConditionBuilder provides a fluent API for building conditional effects.
// Conditions can check molecule fields or count molecules in the environment.
type IfConditionBuilder struct {
	field          string
	op             string
	value          any
	countMolecules *CountMoleculesBuilder
	then           []*EffectBuilder
	else_          []*EffectBuilder
}

// NewIfField creates a field-based condition that compares a molecule field
// with a value. Supported operators: "eq", "ne", "gt", "gte", "lt", "lte".
func NewIfField(field, op string, value any) *IfConditionBuilder {
	return &IfConditionBuilder{
		field: field,
		op:    op,
		value: value,
		then:  make([]*EffectBuilder, 0),
		else_: make([]*EffectBuilder, 0),
	}
}

// NewIfCount creates a count-based condition that checks the number of
// molecules matching certain criteria in the environment.
func NewIfCount(cmb *CountMoleculesBuilder) *IfConditionBuilder {
	return &IfConditionBuilder{
		countMolecules: cmb,
		then:           make([]*EffectBuilder, 0),
		else_:          make([]*EffectBuilder, 0),
	}
}

// Then adds effects to execute if the condition is true.
func (icb *IfConditionBuilder) Then(eb ...*EffectBuilder) *IfConditionBuilder {
	icb.then = append(icb.then, eb...)
	return icb
}

// Else adds effects to execute if the condition is false.
func (icb *IfConditionBuilder) Else(eb ...*EffectBuilder) *IfConditionBuilder {
	icb.else_ = append(icb.else_, eb...)
	return icb
}

// Build converts the builder to an IfConditionConfig.
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

// CountMoleculesBuilder provides a fluent API for building count_molecules conditions.
// These conditions count molecules matching certain criteria and compare
// the count with a threshold.
type CountMoleculesBuilder struct {
	species string
	where   achem.WhereConfig
	op      map[string]any
}

// NewCountMolecules creates a new count molecules builder for the specified species.
func NewCountMolecules(species string) *CountMoleculesBuilder {
	return &CountMoleculesBuilder{
		species: species,
		where:   make(achem.WhereConfig),
		op:      make(map[string]any),
	}
}

// WhereEq adds an equality condition to filter which molecules are counted.
func (cmb *CountMoleculesBuilder) WhereEq(field string, value any) *CountMoleculesBuilder {
	if cmb.where == nil {
		cmb.where = make(achem.WhereConfig)
	}
	cmb.where[field] = achem.EqCondition{Eq: value}
	return cmb
}

// Op sets the comparison operator and threshold value for the count.
// Supported operators: "eq", "ne", "gt", "gte", "lt", "lte".
// Example: Op("gte", 3) means "count >= 3".
func (cmb *CountMoleculesBuilder) Op(operator string, value any) *CountMoleculesBuilder {
	if cmb.op == nil {
		cmb.op = make(map[string]any)
	}
	cmb.op[operator] = value
	return cmb
}

// Build converts the builder to a CountMoleculesConfig.
func (cmb *CountMoleculesBuilder) Build() *achem.CountMoleculesConfig {
	return &achem.CountMoleculesConfig{
		Species: cmb.species,
		Where:   cmb.where,
		Op:      cmb.op,
	}
}

// Ref creates a reference to a molecule field that can be used in payload values.
// The reference will be resolved to the actual value from the input molecule
// when the reaction fires. Accepts either "field" (becomes "$m.field") or
// "m.field" (becomes "$m.field").
func Ref(field string) string {
	if len(field) > 2 && field[:2] == "m." {
		return "$" + field
	}
	return "$m." + field
}

// ApplySchema sends the schema configuration to an AChemDB server.
// The baseURL is the server's base URL (e.g., "http://localhost:8080"),
// and envID is the environment ID where the schema should be applied.
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

// NotificationBuilder provides a fluent API for building notification configurations.
// Notifications allow external systems to be notified when reactions fire,
// either through webhooks, WebSocket, or callbacks.
type NotificationBuilder struct {
	enabled   bool
	notifiers []string
}

// NewNotification creates a new notification builder with notifications
// enabled by default.
func NewNotification() *NotificationBuilder {
	return &NotificationBuilder{
		enabled:   true,
		notifiers: make([]string, 0),
	}
}

// Enabled sets whether notifications are enabled for this reaction.
func (nb *NotificationBuilder) Enabled(enabled bool) *NotificationBuilder {
	nb.enabled = enabled
	return nb
}

// Notifier adds a notifier ID to the list of notifiers to use.
// Notifiers must be registered with the server separately.
func (nb *NotificationBuilder) Notifier(id string) *NotificationBuilder {
	nb.notifiers = append(nb.notifiers, id)
	return nb
}

// Notifiers adds multiple notifier IDs to the list.
func (nb *NotificationBuilder) Notifiers(ids ...string) *NotificationBuilder {
	nb.notifiers = append(nb.notifiers, ids...)
	return nb
}

// Build converts the builder to a NotificationConfig.
func (nb *NotificationBuilder) Build() *achem.NotificationConfig {
	return &achem.NotificationConfig{
		Enabled:   nb.enabled,
		Notifiers: nb.notifiers,
	}
}
