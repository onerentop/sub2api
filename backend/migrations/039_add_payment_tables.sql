-- 039_add_payment_tables.sql
-- 添加商品表和支付订单表，支持在线充值功能

-- 商品表：用于在线充值的商品（余额包/订阅套餐）
CREATE TABLE IF NOT EXISTS products (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    type VARCHAR(20) NOT NULL DEFAULT 'balance',
    price_cny DECIMAL(20,8) NOT NULL,
    value DECIMAL(20,8) NOT NULL,
    group_id BIGINT REFERENCES groups(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- 商品表索引
CREATE INDEX IF NOT EXISTS idx_products_type ON products(type);
CREATE INDEX IF NOT EXISTS idx_products_is_active ON products(is_active);
CREATE INDEX IF NOT EXISTS idx_products_sort_order ON products(sort_order);
CREATE INDEX IF NOT EXISTS idx_products_deleted_at ON products(deleted_at);
CREATE INDEX IF NOT EXISTS idx_products_group_id ON products(group_id);

-- 支付订单表：记录用户的充值订单
CREATE TABLE IF NOT EXISTS payment_orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id BIGINT REFERENCES products(id) ON DELETE SET NULL,
    order_no VARCHAR(64) NOT NULL UNIQUE,
    trade_no VARCHAR(64),
    amount_cny DECIMAL(20,8) NOT NULL,
    amount_value DECIMAL(20,8) NOT NULL,
    order_type VARCHAR(20) NOT NULL DEFAULT 'balance',
    payment_method VARCHAR(20) NOT NULL DEFAULT 'alipay',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    paid_at TIMESTAMPTZ,
    callback_data JSONB,
    remark TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 支付订单表索引
CREATE INDEX IF NOT EXISTS idx_payment_orders_user_id ON payment_orders(user_id);
CREATE INDEX IF NOT EXISTS idx_payment_orders_status ON payment_orders(status);
CREATE INDEX IF NOT EXISTS idx_payment_orders_order_type ON payment_orders(order_type);
CREATE INDEX IF NOT EXISTS idx_payment_orders_payment_method ON payment_orders(payment_method);
CREATE INDEX IF NOT EXISTS idx_payment_orders_created_at ON payment_orders(created_at);

-- 添加注释
COMMENT ON TABLE products IS '商品表：用于在线充值的商品（余额包/订阅套餐）';
COMMENT ON COLUMN products.type IS '商品类型：balance=余额充值，subscription=订阅套餐';
COMMENT ON COLUMN products.price_cny IS '人民币价格';
COMMENT ON COLUMN products.value IS '商品价值：余额类型为USD金额，订阅类型为天数';
COMMENT ON COLUMN products.group_id IS '订阅类型关联的分组ID';
COMMENT ON COLUMN products.is_active IS '是否上架';
COMMENT ON COLUMN products.sort_order IS '排序权重，数值越小越靠前';

COMMENT ON TABLE payment_orders IS '支付订单表：记录用户的充值订单';
COMMENT ON COLUMN payment_orders.order_no IS '平台订单号';
COMMENT ON COLUMN payment_orders.trade_no IS '易支付订单号';
COMMENT ON COLUMN payment_orders.amount_cny IS '支付金额（人民币）';
COMMENT ON COLUMN payment_orders.amount_value IS '到账价值（余额USD/订阅天数）';
COMMENT ON COLUMN payment_orders.order_type IS '订单类型：balance=余额充值，subscription=订阅购买';
COMMENT ON COLUMN payment_orders.payment_method IS '支付方式：wechat/alipay';
COMMENT ON COLUMN payment_orders.status IS '订单状态：pending/paid/failed/refunded/auditing';
COMMENT ON COLUMN payment_orders.callback_data IS '回调原始数据';
COMMENT ON COLUMN payment_orders.remark IS '备注（审核用）';
