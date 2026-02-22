package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("username").
			NotEmpty(),
		field.String("password_hash").
			NotEmpty().
			Sensitive(),
		field.String("role").
			Default("user").
			NotEmpty(),
		field.Bool("is_active").
			Default(true),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("username").Unique(),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		// existentes
		edge.To("refresh_tokens", RefreshToken.Type),
		edge.To("addresses", Address.Type),

		// NUEVOS: sucursales asignadas al usuario
		edge.To("user_branches", UserBranch.Type),

		// NUEVOS: entradas (access points) asignadas al usuario
		edge.To("user_access_points", UserAccessPoint.Type),

		// NUEVOS: marcaciones / asistencia por día
		edge.To("attendance_days", AttendanceDay.Type),
	}
}
