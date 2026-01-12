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

// Announcement holds the schema definition for the Announcement entity.
//
// 公告：用于在网站顶部显示滚动公告
// 支持富文本HTML、多种类型、定时发布
//
// 删除策略：软删除
type Announcement struct {
	ent.Schema
}

func (Announcement) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "announcements"},
	}
}

func (Announcement) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Optional().
			Nillable().
			MaxLen(255).
			Comment("公告标题，用于管理识别"),

		field.Text("content").
			NotEmpty().
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Comment("公告内容，富文本HTML"),

		field.Enum("type").
			Values("info", "success", "warning", "error").
			Default("info").
			Comment("公告类型: info(信息), success(成功), warning(警告), error(紧急)"),

		field.Int("sort_order").
			Default(0).
			Comment("排序权重，越小越靠前"),

		field.Bool("enabled").
			Default(true).
			Comment("是否启用"),

		field.Time("start_time").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("生效时间，null表示立即生效"),

		field.Time("end_time").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("过期时间，null表示永不过期"),

		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),

		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),

		field.Time("deleted_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("软删除时间"),
	}
}

func (Announcement) Edges() []ent.Edge {
	return nil
}

func (Announcement) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("enabled"),
		index.Fields("type"),
		index.Fields("sort_order"),
		index.Fields("start_time"),
		index.Fields("end_time"),
		index.Fields("deleted_at"),
	}
}
