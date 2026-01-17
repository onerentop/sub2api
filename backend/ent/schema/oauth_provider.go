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

// OAuthProvider holds the schema definition for the OAuthProvider entity.
//
// OAuth 提供商配置：存储第三方登录提供商的配置信息
// 如 Google, GitHub, QQ, WeChat 等
//
// 删除策略：硬删除（配置表无需软删除）
type OAuthProvider struct {
	ent.Schema
}

func (OAuthProvider) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "oauth_providers"},
	}
}

func (OAuthProvider) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(50).
			NotEmpty().
			Unique().
			Comment("提供商标识: google, github, qq, wechat"),
		field.String("display_name").
			MaxLen(100).
			NotEmpty().
			Comment("显示名称: Google, GitHub"),
		field.String("client_id").
			MaxLen(255).
			Default("").
			Comment("OAuth Client ID"),
		field.String("client_secret").
			MaxLen(500).
			Default("").
			Sensitive().
			Comment("OAuth Client Secret"),
		field.Bool("enabled").
			Default(false).
			Comment("是否启用"),
		field.JSON("config", map[string]any{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Comment("额外配置: scopes, endpoints 等"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (OAuthProvider) Edges() []ent.Edge {
	return nil
}

func (OAuthProvider) Indexes() []ent.Index {
	return []ent.Index{
		// name 字段已在 Fields() 中声明 Unique()，无需重复索引
		index.Fields("enabled"),
	}
}
