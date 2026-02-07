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

// PaymentOrder holds the schema definition for the PaymentOrder entity.
// 支付订单表：记录用户的充值订单
type PaymentOrder struct {
	ent.Schema
}

func (PaymentOrder) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "payment_orders"},
	}
}

func (PaymentOrder) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (PaymentOrder) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Comment("用户ID"),
		field.Int64("product_id").
			Optional().
			Nillable().
			Comment("商品ID，自定义金额充值时为空"),
		field.String("order_no").
			MaxLen(64).
			NotEmpty().
			Unique().
			Comment("平台订单号"),
		field.String("trade_no").
			MaxLen(64).
			Optional().
			Nillable().
			Comment("易支付订单号"),
		field.Float("amount_cny").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Comment("支付金额（人民币）"),
		field.Float("amount_value").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Comment("到账价值（余额USD/订阅天数）"),
		field.Enum("order_type").
			Values("balance", "subscription").
			Default("balance").
			Comment("订单类型：balance=余额充值，subscription=订阅购买"),
		field.Enum("payment_method").
			Values("wechat", "alipay").
			Default("alipay").
			Comment("支付方式"),
		field.Enum("status").
			Values("pending", "paid", "failed", "refunded", "auditing").
			Default("pending").
			Comment("订单状态"),
		field.Time("paid_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("支付时间"),
		field.JSON("callback_data", map[string]any{}).
			Optional().
			Comment("回调原始数据"),
		field.String("remark").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Comment("备注（审核用）"),
	}
}

func (PaymentOrder) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("payment_orders").
			Field("user_id").
			Required().
			Unique(),
		edge.From("product", Product.Type).
			Ref("orders").
			Field("product_id").
			Unique(),
	}
}

func (PaymentOrder) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("status"),
		index.Fields("order_type"),
		index.Fields("payment_method"),
		index.Fields("created_at"),
	}
}
