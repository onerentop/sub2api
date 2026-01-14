-- 036_add_announcements.sql
-- 公告表：用于网站滚动公告功能

CREATE TABLE IF NOT EXISTS announcements (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255),
    content TEXT NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'info',
    sort_order INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_announcements_enabled ON announcements(enabled);
CREATE INDEX IF NOT EXISTS idx_announcements_type ON announcements(type);
CREATE INDEX IF NOT EXISTS idx_announcements_sort_order ON announcements(sort_order);
CREATE INDEX IF NOT EXISTS idx_announcements_start_time ON announcements(start_time);
CREATE INDEX IF NOT EXISTS idx_announcements_end_time ON announcements(end_time);
CREATE INDEX IF NOT EXISTS idx_announcements_deleted_at ON announcements(deleted_at);

-- 复合索引：用于查询当前生效的公告
CREATE INDEX IF NOT EXISTS idx_announcements_active ON announcements(enabled, deleted_at, start_time, end_time, sort_order);

-- 添加公告相关的系统设置
INSERT INTO settings (key, value, updated_at)
VALUES
    ('announcement.enabled', 'true', NOW()),
    ('announcement.interval', '5000', NOW())
ON CONFLICT (key) DO NOTHING;

COMMENT ON TABLE announcements IS '公告表';
COMMENT ON COLUMN announcements.title IS '公告标题，用于管理识别';
COMMENT ON COLUMN announcements.content IS '公告内容，富文本HTML';
COMMENT ON COLUMN announcements.type IS '公告类型: info(信息), success(成功), warning(警告), error(紧急)';
COMMENT ON COLUMN announcements.sort_order IS '排序权重，越小越靠前';
COMMENT ON COLUMN announcements.enabled IS '是否启用';
COMMENT ON COLUMN announcements.start_time IS '生效时间，null表示立即生效';
COMMENT ON COLUMN announcements.end_time IS '过期时间，null表示永不过期';
COMMENT ON COLUMN announcements.deleted_at IS '软删除时间';
