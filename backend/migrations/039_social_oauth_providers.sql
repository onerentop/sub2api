-- Migration: Social OAuth Providers
-- Description: Add tables for social OAuth login (Google, GitHub, etc.)

-- OAuth 提供商配置表
CREATE TABLE IF NOT EXISTS oauth_providers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    client_id VARCHAR(255) NOT NULL DEFAULT '',
    client_secret VARCHAR(500) NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_oauth_providers_enabled ON oauth_providers(enabled);

-- 插入初始提供商配置
INSERT INTO oauth_providers (name, display_name, config) VALUES
    ('google', 'Google', '{"scopes": "openid email profile", "auth_url": "https://accounts.google.com/o/oauth2/v2/auth", "token_url": "https://oauth2.googleapis.com/token", "userinfo_url": "https://www.googleapis.com/oauth2/v2/userinfo"}'),
    ('github', 'GitHub', '{"scopes": "read:user user:email", "auth_url": "https://github.com/login/oauth/authorize", "token_url": "https://github.com/login/oauth/access_token", "userinfo_url": "https://api.github.com/user", "emails_url": "https://api.github.com/user/emails"}')
ON CONFLICT (name) DO NOTHING;

-- 用户 OAuth 绑定表
CREATE TABLE IF NOT EXISTS user_oauth_bindings (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    provider_email VARCHAR(255),
    provider_username VARCHAR(255),
    provider_avatar VARCHAR(500),
    access_token TEXT,
    refresh_token TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 创建索引
-- 同一提供商同一用户只能绑定一次（全局唯一）
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_oauth_bindings_provider_user ON user_oauth_bindings(provider, provider_user_id);
-- 同一用户只能绑定一个该提供商账号
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_oauth_bindings_user_provider ON user_oauth_bindings(user_id, provider);
-- 按用户查询绑定
CREATE INDEX IF NOT EXISTS idx_user_oauth_bindings_user_id ON user_oauth_bindings(user_id);

-- 添加注释
COMMENT ON TABLE oauth_providers IS 'OAuth 提供商配置表';
COMMENT ON COLUMN oauth_providers.name IS '提供商标识: google, github, qq, wechat';
COMMENT ON COLUMN oauth_providers.display_name IS '显示名称';
COMMENT ON COLUMN oauth_providers.client_id IS 'OAuth Client ID';
COMMENT ON COLUMN oauth_providers.client_secret IS 'OAuth Client Secret';
COMMENT ON COLUMN oauth_providers.enabled IS '是否启用';
COMMENT ON COLUMN oauth_providers.config IS '额外配置: scopes, endpoints 等';

COMMENT ON TABLE user_oauth_bindings IS '用户 OAuth 绑定表';
COMMENT ON COLUMN user_oauth_bindings.user_id IS '用户ID';
COMMENT ON COLUMN user_oauth_bindings.provider IS '提供商标识';
COMMENT ON COLUMN user_oauth_bindings.provider_user_id IS '第三方平台用户ID';
COMMENT ON COLUMN user_oauth_bindings.provider_email IS '第三方平台邮箱';
COMMENT ON COLUMN user_oauth_bindings.provider_username IS '第三方平台用户名';
COMMENT ON COLUMN user_oauth_bindings.provider_avatar IS '第三方平台头像URL';
