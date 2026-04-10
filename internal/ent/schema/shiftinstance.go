package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ShiftInstance struct {
	ent.Schema
}

func (ShiftInstance) Fields() []ent.Field {
	return []ent.Field{
		field.Int("shift_id"),
		field.Time("date").SchemaType(map[string]string{"postgres": "date"}),
		field.String("state").Default("scheduled"),
		field.String("mode").Default("onsite"),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (ShiftInstance) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("shift", Shift.Type).
			Ref("instances").
			Field("shift_id").
			Unique().
			Required(),
	}
}

func (ShiftInstance) Indexes() []ent.Index {
	return []ent.Index{
		// Evita que un mismo turno tenga dos registros el mismo día
		index.Fields("shift_id", "date").Unique(),
	}
}
