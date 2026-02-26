package schema

import (
	"fmt"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Device struct {
	ent.Schema
}

func (Device) Fields() []ent.Field {
	return []ent.Field{
		field.Int("access_point_id"),

		field.String("name").NotEmpty(),
		field.String("serial").NotEmpty(), // identificador físico del equipo

		// "in" => dispositivo de entrada, "out" => dispositivo de salida
		field.String("direction").
			NotEmpty().
			Validate(func(s string) error {
				if s != "in" && s != "out" {
					return fmt.Errorf("direction must be 'in' or 'out'")
				}
				return nil
			}),

		field.Bool("is_active").Default(true),

		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Device) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("access_point", AccessPoint.Type).
			Ref("devices").
			Field("access_point_id").
			Unique().
			Required(),
	}
}

func (Device) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("serial").Unique().StorageKey("ux_device_serial"),

		// Clave: solo un device IN y uno OUT por access_point
		index.Fields("access_point_id", "direction").Unique().StorageKey("ux_device_access_direction"),

		index.Fields("access_point_id").StorageKey("ix_device_access_point"),
	}
}
