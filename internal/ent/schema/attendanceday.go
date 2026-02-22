package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type AttendanceDay struct {
	ent.Schema
}

func (AttendanceDay) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id"),
		field.Int("branch_id"),

		// acceso usado para iniciar el día (opcional)
		field.Int("access_point_id").Optional().Nillable(),

		// día laboral (sin hora)
		field.Time("work_date"),

		// marcaciones
		field.Time("work_in_at").Optional().Nillable(),
		field.Time("break_out_at").Optional().Nillable(),
		field.Time("break_in_at").Optional().Nillable(),
		field.Time("work_out_at").Optional().Nillable(),

		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (AttendanceDay) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).Field("user_id").Unique().Required(),
		edge.To("branch", Branch.Type).Field("branch_id").Unique().Required(),

		edge.To("access_point", AccessPoint.Type).
			Field("access_point_id").
			Unique(),
	}
}

func (AttendanceDay) Indexes() []ent.Index {
	return []ent.Index{
		// un registro por user + sucursal + día
		index.Fields("user_id", "branch_id", "work_date").Unique().StorageKey("ux_attendance_day"),
	}
}
