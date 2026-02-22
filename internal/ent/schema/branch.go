package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Branch struct {
	ent.Schema
}

func (Branch) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.String("code").Optional().Nillable(), // opcional
		field.Bool("is_active").Default(true),
	}
}

func (Branch) Edges() []ent.Edge {
	return []ent.Edge{
		// 1:1 con BranchAddress
		edge.To("address", BranchAddress.Type).Unique(),

		// 1:N access points
		edge.To("access_points", AccessPoint.Type),

		// M:N users via UserBranch
		edge.To("user_branches", UserBranch.Type),
	}
}

func (Branch) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code").Unique().StorageKey("ux_branch_code"),
	}
}
