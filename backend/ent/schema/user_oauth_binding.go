package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserOAuthBinding holds the schema definition for the UserOAuthBinding entity.
//
// 用户 OAuth 绑定：记录用户与第三方账号的绑定关系
// 一个用户可以绑定多个不同提供商的账号，但同一提供商只能绑定一个
//
// 删除策略：硬删除（随用户删除级联删除）
type UserOAuthBinding struct {
	ent.Schema
}

func (UserOAuthBinding) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_oauth_bindings"},
	}
}

func (UserOAuthBinding) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Comment("用户ID"),
		field.String("provider").
			MaxLen(50).
			NotEmpty().
			Comment("提供商标识: google, github"),
		field.String("provider_user_id").
			MaxLen(255).
			NotEmpty().
			Comment("第三方平台用户ID"),
		field.String("provider_email").
			MaxLen(255).
			Optional().
			Nillable().
			Comment("第三方平台邮箱"),
		field.String("provider_username").
			MaxLen(255).
			Optional().
			Nillable().
			Comment("第三方平台用户名"),
		field.String("provider_avatar").
			MaxLen(500).
			Optional().
			Nillable().
			Comment("第三方平台头像URL"),
		field.Text("access_token").
			Optional().
			Nillable().
			Sensitive().
			Comment("访问令牌（可选存储）"),
		field.Text("refresh_token").
			Optional().
			Nillable().
			Sensitive().
			Comment("刷新令牌（可选存储）"),
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

func (UserOAuthBinding) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("oauth_bindings").
			Field("user_id").
			Required().
			Unique(),
	}
}

func (UserOAuthBinding) Indexes() []ent.Index {
	return []ent.Index{
		// 同一提供商同一用户只能绑定一次（全局唯一）
		index.Fields("provider", "provider_user_id").Unique(),
		// 同一用户只能绑定一个该提供商账号
		index.Fields("user_id", "provider").Unique(),
		// 按用户查询绑定
		index.Fields("user_id"),
	}
}
