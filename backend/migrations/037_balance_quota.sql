-- 037_balance_quota.sql
-- 为余额计费模式添加每日/每周限额功能

-- 分组表：添加余额计费限额字段
-- 注：区别于 subscription 模式的 daily_limit_usd/weekly_limit_usd
ALTER TABLE groups ADD COLUMN IF NOT EXISTS balance_daily_quota DECIMAL(20,8) NULL;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS balance_weekly_quota DECIMAL(20,8) NULL;

COMMENT ON COLUMN groups.balance_daily_quota IS '余额计费模式的每日限额（金额）';
COMMENT ON COLUMN groups.balance_weekly_quota IS '余额计费模式的每周限额（金额）';

-- 用户表：添加限额覆盖字段
ALTER TABLE users ADD COLUMN IF NOT EXISTS balance_daily_quota DECIMAL(20,8) NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS balance_weekly_quota DECIMAL(20,8) NULL;

COMMENT ON COLUMN users.balance_daily_quota IS '用户每日限额覆盖（余额计费模式）';
COMMENT ON COLUMN users.balance_weekly_quota IS '用户每周限额覆盖（余额计费模式）';
