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

		// Datos personales
		field.String("first_name").
			Optional().
			Nillable(),

		field.String("last_name").
			Optional().
			Nillable(),

		field.String("middle_name").
			Optional().
			Nillable(),

		field.String("email").
			Optional().
			Nillable(),

		field.String("employee_code").
			Optional().
			Nillable(),

		// Código físico para tarjeta, chip o código de barras
		field.String("access_code").
			Optional().
			Nillable(),

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
		index.Fields("email").Unique(),
		index.Fields("access_code").Unique(),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("refresh_tokens", RefreshToken.Type),
		edge.To("addresses", Address.Type),

		edge.To("user_branches", UserBranch.Type),
		edge.To("user_access_points", UserAccessPoint.Type),
		edge.To("attendance_days", AttendanceDay.Type),

		edge.To("shift_assignments", UserShiftAssignment.Type),
		edge.To("day_overrides", UserDayOverride.Type),
		edge.To("qr_sessions", UserQRSession.Type),
	}
}
