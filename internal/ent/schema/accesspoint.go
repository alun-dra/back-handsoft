package schema

import (
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
		field.String("name").NotEmpty(), // "Puerta principal", "Portón bodega", etc.
		field.Bool("is_active").Default(true),
	}
}

func (AccessPoint) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("branch", Branch.Type).
			Ref("access_points").
			Field("branch_id").
			Unique().
			Required(),

		// M:N users via UserAccessPoint
		edge.To("user_access_points", UserAccessPoint.Type),

		// Opcional: si quieres referenciar marcaciones a la entrada
		edge.To("attendance_days", AttendanceDay.Type),
	}
}

func (AccessPoint) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("branch_id", "name").Unique().StorageKey("ux_accesspoint_branch_name"),
	}
}
