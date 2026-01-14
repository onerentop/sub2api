-- 038_balance_group_quota_index.sql
-- 为余额模式按分组限额功能添加复合索引

-- 用于按用户+分组+时间范围查询用量的复合索引
-- 支持 SumActualCostByUserGroupAndTimeRange 和 SumActualCostByUserGroupedByGroup 查询
--
-- 注意: 生产环境如果 usage_logs 表数据量很大，建议手动执行以下命令（非事务模式）:
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_user_group_created
-- ON usage_logs(user_id, group_id, created_at);
--
-- 此处使用普通 CREATE INDEX 因为 migration runner 在事务中执行，
-- 而 CREATE INDEX CONCURRENTLY 不支持事务内运行。
CREATE INDEX IF NOT EXISTS idx_usage_logs_user_group_created
ON usage_logs(user_id, group_id, created_at);

COMMENT ON INDEX idx_usage_logs_user_group_created IS '余额模式按分组限额查询优化索引';
