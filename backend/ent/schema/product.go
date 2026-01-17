package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Product holds the schema definition for the Product entity.
// 商品表：用于在线充值的商品（余额包/订阅套餐）
type Product struct {
	ent.Schema
}

func (Product) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "products"},
	}
}

func (Product) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (Product) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100).
			NotEmpty().
			Comment("商品名称"),
		field.String("description").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Comment("商品描述"),
		field.Enum("type").
			Values("balance", "subscription").
			Default("balance").
			Comment("商品类型：balance=余额充值，subscription=订阅套餐"),
		field.Float("price_cny").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Comment("人民币价格"),
		field.Float("value").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Comment("商品价值：余额类型为USD金额，订阅类型为天数"),
		field.Int64("group_id").
			Optional().
			Nillable().
			Comment("订阅类型关联的分组ID"),
		field.Bool("is_active").
			Default(true).
			Comment("是否上架"),
		field.Int("sort_order").
			Default(0).
			Comment("排序权重，数值越小越靠前"),
	}
}

func (Product) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("orders", PaymentOrder.Type),
		edge.From("group", Group.Type).
			Ref("products").
			Field("group_id").
			Unique(),
	}
}

func (Product) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("type"),
		index.Fields("is_active"),
		index.Fields("sort_order"),
		index.Fields("deleted_at"),
	}
}
