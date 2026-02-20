package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Region struct {
	ent.Schema
}

func (Region) Fields() []ent.Field {
	return []ent.Field{
		field.Int("country_id").Positive(),
		field.String("name").NotEmpty(),
		field.String("code").NotEmpty(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Region) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code").Unique(),
	}
}

func (Region) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("cities", City.Type),
	}
}
