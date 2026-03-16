package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Shift struct {
	ent.Schema
}

func (Shift) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.String("description").Optional().Nillable(),

		field.String("start_time").NotEmpty(), // "08:00"
		field.String("end_time").NotEmpty(),   // "17:00"

		field.Int("break_minutes").Default(0),

		field.Bool("crosses_midnight").Default(false),
		field.Bool("is_active").Default(true),

		field.Time("created_at").
			Default(time.Now).
			Immutable(),

		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Shift) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name").Unique(),
	}
}

func (Shift) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("days", ShiftDay.Type),
		edge.To("user_assignments", UserShiftAssignment.Type),
		edge.To("day_overrides", UserDayOverride.Type),
	}
}
