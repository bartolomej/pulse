package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Source struct {
	ent.Schema
}

func (Source) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique(),
		field.String("name"),
		field.String("url"),
		field.String("type"),
		field.String("config_json").Optional().Nillable(),
	}
}

func (Source) Edges() []ent.Edge {
	return nil
}
