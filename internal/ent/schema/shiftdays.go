package schema

import (
	"fmt"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ShiftDay struct {
	ent.Schema
}

func (ShiftDay) Fields() []ent.Field {
	return []ent.Field{
		field.Int("shift_id"),

		// 1=lunes ... 7=domingo
		field.Int("weekday").
			Validate(func(v int) error {
				if v < 1 || v > 7 {
					return fmt.Errorf("weekday must be between 1 and 7")
				}
				return nil
			}),

		field.Bool("is_working_day").Default(true),

		// onsite, remote, hybrid_office, hybrid_home, off
		field.String("mode").
			Default("onsite"),

		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

func (ShiftDay) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("shift", Shift.Type).
			Ref("days").
			Field("shift_id").
			Unique().
			Required(),
	}
}

func (ShiftDay) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("shift_id", "weekday").Unique(),
	}
}
