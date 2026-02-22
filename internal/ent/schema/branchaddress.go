package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type BranchAddress struct {
	ent.Schema
}

func (BranchAddress) Fields() []ent.Field {
	return []ent.Field{
		field.Int("branch_id"),
		field.Int("commune_id"),

		field.String("street").NotEmpty(),
		field.String("number").NotEmpty(),
		field.String("apartment").Optional().Nillable(),

		field.String("extra").Optional().Nillable(), // ej: "Bodega, Piso 2, etc."
	}
}

func (BranchAddress) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("branch", Branch.Type).
			Ref("address").
			Field("branch_id").
			Unique().
			Required(),

		edge.To("commune", Commune.Type).
			Field("commune_id").
			Unique().
			Required(),
	}
}

func (BranchAddress) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("branch_id").Unique().StorageKey("ux_branch_address_branch_id"),
	}
}
