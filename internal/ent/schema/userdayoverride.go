package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type UserDayOverride struct {
	ent.Schema
}

func (UserDayOverride) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id"),

		// opcional: si ese día reemplaza el turno base
		field.Int("shift_id").
			Optional().
			Nillable(),

		field.Time("date"),

		field.Bool("is_day_off").Default(false),

		// onsite, remote, hybrid_office, hybrid_home, off
		field.String("mode").
			Default("onsite"),

		field.String("notes").
			Optional().
			Nillable(),

		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

func (UserDayOverride) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("day_overrides").
			Field("user_id").
			Unique().
			Required(),

		edge.From("shift", Shift.Type).
			Ref("day_overrides").
			Field("shift_id").
			Unique(),
	}
}

func (UserDayOverride) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "date").Unique(),
	}
}
