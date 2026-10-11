package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ActivityRun struct{ ent.Schema }

func (ActivityRun) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "activity_runs"}}
}

func (ActivityRun) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").MaxLen(64),
		field.String("current_marker").MaxLen(64).Optional().Nillable(),
		field.Int64("group_id"),
		field.Time("starts_at").SchemaType(map[string]string{dialect.Postgres: "timestamptz", dialect.SQLite: "DATETIME"}),
		field.Time("ends_at").SchemaType(map[string]string{dialect.Postgres: "timestamptz", dialect.SQLite: "DATETIME"}),
		field.Time("disabled_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz", dialect.SQLite: "DATETIME"}),
		field.Int64("created_by"),
		field.Time("created_at").Default(time.Now).Immutable().SchemaType(map[string]string{dialect.Postgres: "timestamptz", dialect.SQLite: "DATETIME"}),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz", dialect.SQLite: "DATETIME"}),
	}
}

func (ActivityRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("current_marker").Unique(),
		index.Fields("code", "starts_at"),
		index.Fields("ends_at"),
	}
}
