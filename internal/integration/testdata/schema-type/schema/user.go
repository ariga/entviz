package schema

import (
	"database/sql"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.Other("create_time", &sql.NullTime{}).SchemaType(map[string]string{"not": "defined"}),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return nil
}
