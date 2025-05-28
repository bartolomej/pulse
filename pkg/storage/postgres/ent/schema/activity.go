package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Activity struct {
	ent.Schema
}

func (Activity) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique(),
		field.String("uid").Unique(),
		field.String("source_uid"),
		field.String("title"),
		field.String("body"),
		field.String("url"),
		field.String("image_url"),
		field.Time("created_at"),
		field.String("short_summary"),
		field.String("full_summary"),
	}
}

func (Activity) Edges() []ent.Edge {
	return nil
}
