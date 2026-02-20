package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Commune struct {
	ent.Schema
}

func (Commune) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Commune) Indexes() []ent.Index {
	return []ent.Index{
		// Opcional: evita comunas duplicadas dentro de la misma ciudad
		index.Edges("city").Fields("name").Unique(),
	}
}

func (Commune) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("city", City.Type).
			Ref("communes").
			Unique().
			Required(),

		edge.To("addresses", Address.Type),
	}
}
