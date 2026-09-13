ALTER TABLE announcements
    ADD COLUMN IF NOT EXISTS banner_config JSONB NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN announcements.notify_mode IS '通知模式: silent(仅铃铛), popup(弹窗提醒), banner(顶部横幅)';
COMMENT ON COLUMN announcements.banner_config IS '顶部横幅配置（JSON）';
