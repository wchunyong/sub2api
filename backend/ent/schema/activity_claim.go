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

type ActivityClaim struct{ ent.Schema }

func (ActivityClaim) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "activity_claims"}}
}

func (ActivityClaim) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("run_id"),
		field.Int64("user_id"),
		field.Int64("order_id"),
		field.Int64("subscription_id"),
		field.String("status").MaxLen(20).Default("claimed"),
		field.Time("claimed_at").Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz", dialect.SQLite: "DATETIME"}),
	}
}

func (ActivityClaim) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("run_id", "user_id").Unique(),
		index.Fields("order_id").Unique(),
		index.Fields("user_id"),
	}
}
