package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type UserQRSession struct {
	ent.Schema
}

func (UserQRSession) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id"),

		field.String("token_hash").NotEmpty().Sensitive(),

		field.Time("issued_at").Default(time.Now),
		field.Time("expires_at"),

		field.Bool("is_revoked").Default(false),

		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

func (UserQRSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("qr_sessions").
			Field("user_id").
			Unique().
			Required(),
	}
}

func (UserQRSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("token_hash").Unique(),
		index.Fields("user_id", "expires_at"),
	}
}
