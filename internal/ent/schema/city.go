package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type City struct {
	ent.Schema
}

func (City) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (City) Indexes() []ent.Index {
	return []ent.Index{
		// Opcional: evita ciudades duplicadas dentro de la misma región
		index.Edges("region").Fields("name").Unique(),
	}
}

func (City) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("region", Region.Type).
			Ref("cities").
			Unique().
			Required(),

		edge.To("communes", Commune.Type),
	}
}
