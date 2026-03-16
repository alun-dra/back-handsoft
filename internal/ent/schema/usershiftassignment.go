package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type UserShiftAssignment struct {
	ent.Schema
}

func (UserShiftAssignment) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id"),
		field.Int("shift_id"),

		field.Time("start_date"),
		field.Time("end_date").Optional().Nillable(),

		field.Bool("is_active").Default(true),

		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

func (UserShiftAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("shift_assignments").
			Field("user_id").
			Unique().
			Required(),

		edge.From("shift", Shift.Type).
			Ref("user_assignments").
			Field("shift_id").
			Unique().
			Required(),
	}
}

func (UserShiftAssignment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "shift_id", "start_date"),
	}
}
