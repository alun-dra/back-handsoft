package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type UserAccessPoint struct {
	ent.Schema
}

func (UserAccessPoint) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id"),
		field.Int("access_point_id"),

		field.Bool("is_active").Default(true),

		// opcional: trazabilidad de asignación
		field.Time("assigned_at").Default(time.Now),
		field.Time("revoked_at").Optional().Nillable(),
	}
}

func (UserAccessPoint) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("user_access_points").
			Field("user_id").
			Unique().
			Required(),

		edge.From("access_point", AccessPoint.Type).
			Ref("user_access_points").
			Field("access_point_id").
			Unique().
			Required(),
	}
}

func (UserAccessPoint) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "access_point_id").Unique().StorageKey("ux_user_access_point"),
	}
}
