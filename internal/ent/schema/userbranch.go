package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type UserBranch struct {
	ent.Schema
}

func (UserBranch) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id"),
		field.Int("branch_id"),

		field.Bool("is_active").Default(true),

		// opcional (si algún día quieres roles por sucursal)
		field.String("role_in_branch").Optional().Nillable(),
	}
}

func (UserBranch) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("user_branches").
			Field("user_id").
			Unique().
			Required(),

		edge.From("branch", Branch.Type).
			Ref("user_branches").
			Field("branch_id").
			Unique().
			Required(),
	}
}

func (UserBranch) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "branch_id").Unique().StorageKey("ux_user_branch"),
	}
}
