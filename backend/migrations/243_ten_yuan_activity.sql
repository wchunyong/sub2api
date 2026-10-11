CREATE TABLE IF NOT EXISTS activity_runs (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    current_marker VARCHAR(64),
    group_id BIGINT NOT NULL REFERENCES groups(id),
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    disabled_at TIMESTAMPTZ,
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT activity_runs_valid_period CHECK (ends_at > starts_at)
);
CREATE UNIQUE INDEX IF NOT EXISTS activityrun_current_marker ON activity_runs(current_marker);
CREATE INDEX IF NOT EXISTS activityrun_code_starts_at ON activity_runs(code, starts_at);
CREATE INDEX IF NOT EXISTS activityrun_ends_at ON activity_runs(ends_at);

CREATE TABLE IF NOT EXISTS activity_claims (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL REFERENCES activity_runs(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    order_id BIGINT NOT NULL REFERENCES payment_orders(id),
    subscription_id BIGINT NOT NULL REFERENCES user_subscriptions(id),
    status VARCHAR(20) NOT NULL DEFAULT 'claimed',
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS activityclaim_run_id_user_id ON activity_claims(run_id, user_id);
CREATE UNIQUE INDEX IF NOT EXISTS activityclaim_order_id ON activity_claims(order_id);
CREATE INDEX IF NOT EXISTS activityclaim_user_id ON activity_claims(user_id);
