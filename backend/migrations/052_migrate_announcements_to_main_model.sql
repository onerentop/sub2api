-- 052_migrate_announcements_to_main_model.sql
-- 目的：将旧公告模型（type/sort_order/enabled/start_time/end_time）兼容升级到 main 分支新模型
-- 说明：
--   1) 保留旧列，避免破坏性变更；新增新模型需要的列与索引
--   2) 补齐 announcement_reads 表
--   3) 尽量幂等，支持重复执行

-- 若 announcements 表不存在，则按新模型创建
CREATE TABLE IF NOT EXISTS announcements (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    targeting JSONB NOT NULL DEFAULT '{}'::jsonb,
    starts_at TIMESTAMPTZ DEFAULT NULL,
    ends_at TIMESTAMPTZ DEFAULT NULL,
    created_by BIGINT DEFAULT NULL,
    updated_by BIGINT DEFAULT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 新模型字段（兼容旧表）
ALTER TABLE announcements ADD COLUMN IF NOT EXISTS status VARCHAR(20);
ALTER TABLE announcements ADD COLUMN IF NOT EXISTS targeting JSONB;
ALTER TABLE announcements ADD COLUMN IF NOT EXISTS starts_at TIMESTAMPTZ;
ALTER TABLE announcements ADD COLUMN IF NOT EXISTS ends_at TIMESTAMPTZ;
ALTER TABLE announcements ADD COLUMN IF NOT EXISTS created_by BIGINT;
ALTER TABLE announcements ADD COLUMN IF NOT EXISTS updated_by BIGINT;

-- 从旧字段回填新字段
DO $$
DECLARE
    has_enabled BOOLEAN;
    has_deleted_at BOOLEAN;
    has_start_time BOOLEAN;
    has_end_time BOOLEAN;
BEGIN
    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'announcements' AND column_name = 'enabled'
    ) INTO has_enabled;

    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'announcements' AND column_name = 'deleted_at'
    ) INTO has_deleted_at;

    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'announcements' AND column_name = 'start_time'
    ) INTO has_start_time;

    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'announcements' AND column_name = 'end_time'
    ) INTO has_end_time;

    IF has_start_time THEN
        EXECUTE 'UPDATE announcements SET starts_at = start_time WHERE starts_at IS NULL';
    END IF;

    IF has_end_time THEN
        EXECUTE 'UPDATE announcements SET ends_at = end_time WHERE ends_at IS NULL';
    END IF;

    IF has_enabled AND has_deleted_at THEN
        EXECUTE 'UPDATE announcements
                 SET status = CASE
                     WHEN status IS NULL OR status = '''' THEN
                         CASE
                             WHEN COALESCE(enabled, TRUE) = TRUE AND deleted_at IS NULL THEN ''active''
                             ELSE ''archived''
                         END
                     ELSE status
                 END';
    ELSIF has_enabled THEN
        EXECUTE 'UPDATE announcements
                 SET status = CASE
                     WHEN status IS NULL OR status = '''' THEN
                         CASE
                             WHEN COALESCE(enabled, TRUE) = TRUE THEN ''active''
                             ELSE ''archived''
                         END
                     ELSE status
                 END';
    ELSE
        EXECUTE 'UPDATE announcements SET status = ''draft'' WHERE status IS NULL OR status = ''''';
    END IF;
END $$;

-- 约束与默认值
UPDATE announcements SET title = '系统公告' WHERE title IS NULL OR btrim(title) = '';
UPDATE announcements SET status = 'draft' WHERE status IS NULL OR status = '';
UPDATE announcements SET targeting = '{}'::jsonb WHERE targeting IS NULL;

ALTER TABLE announcements ALTER COLUMN title SET NOT NULL;
ALTER TABLE announcements ALTER COLUMN content SET NOT NULL;
ALTER TABLE announcements ALTER COLUMN status SET DEFAULT 'draft';
ALTER TABLE announcements ALTER COLUMN status SET NOT NULL;
ALTER TABLE announcements ALTER COLUMN targeting SET DEFAULT '{}'::jsonb;
ALTER TABLE announcements ALTER COLUMN targeting SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_announcements_created_by') THEN
        ALTER TABLE announcements
            ADD CONSTRAINT fk_announcements_created_by
            FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_announcements_updated_by') THEN
        ALTER TABLE announcements
            ADD CONSTRAINT fk_announcements_updated_by
            FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;

-- 公告已读表
CREATE TABLE IF NOT EXISTS announcement_reads (
    id BIGSERIAL PRIMARY KEY,
    announcement_id BIGINT NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(announcement_id, user_id)
);

-- 索引（新模型）
CREATE INDEX IF NOT EXISTS idx_announcements_status ON announcements(status);
CREATE INDEX IF NOT EXISTS idx_announcements_starts_at ON announcements(starts_at);
CREATE INDEX IF NOT EXISTS idx_announcements_ends_at ON announcements(ends_at);
CREATE INDEX IF NOT EXISTS idx_announcements_created_at ON announcements(created_at);

CREATE INDEX IF NOT EXISTS idx_announcement_reads_announcement_id ON announcement_reads(announcement_id);
CREATE INDEX IF NOT EXISTS idx_announcement_reads_user_id ON announcement_reads(user_id);
CREATE INDEX IF NOT EXISTS idx_announcement_reads_read_at ON announcement_reads(read_at);

COMMENT ON TABLE announcements IS '系统公告';
COMMENT ON COLUMN announcements.status IS '状态: draft, active, archived';
COMMENT ON COLUMN announcements.targeting IS '展示条件（JSON 规则）';
COMMENT ON COLUMN announcements.starts_at IS '开始展示时间（为空表示立即生效）';
COMMENT ON COLUMN announcements.ends_at IS '结束展示时间（为空表示永久生效）';

COMMENT ON TABLE announcement_reads IS '公告已读记录';
COMMENT ON COLUMN announcement_reads.read_at IS '用户首次已读时间';
