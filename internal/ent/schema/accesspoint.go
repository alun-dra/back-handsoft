package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type AccessPoint struct {
	ent.Schema
}

func (AccessPoint) Fields() []ent.Field {
	return []ent.Field{
		field.Int("branch_id"),

		field.String("name").
			NotEmpty(), // "Puerta principal", "Portón bodega", etc.

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

func (AccessPoint) Edges() []ent.Edge {
	return []ent.Edge{
		// AccessPoint pertenece a una Branch (N:1)
		edge.From("branch", Branch.Type).
			Ref("access_points").
			Field("branch_id").
			Unique().
			Required(),

		// M:N users via UserAccessPoint
		edge.To("user_access_points", UserAccessPoint.Type),

		// Marcaciones asociadas a esta entrada
		edge.To("attendance_days", AttendanceDay.Type),

		// NUEVO: dispositivos asociados a esta entrada (1:N)
		edge.To("devices", Device.Type),
	}
}

func (AccessPoint) Indexes() []ent.Index {
	return []ent.Index{
		// No permitir dos entradas con mismo nombre en la misma sucursal
		index.Fields("branch_id", "name").
			Unique().
			StorageKey("ux_accesspoint_branch_name"),

		index.Fields("branch_id").
			StorageKey("ix_accesspoint_branch"),
	}
}
