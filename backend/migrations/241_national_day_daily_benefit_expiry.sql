-- Cap only campaign-issued daily benefits. Leave balances, monthly gifts,
-- usage windows, suspended/revoked status and manually assigned grants intact.
UPDATE user_subscriptions
SET expires_at = '2026-10-07 16:00:00+00:00',
    updated_at = CURRENT_TIMESTAMP
WHERE notes = 'national_day_2026_daily_benefit'
  AND deleted_at IS NULL
  AND expires_at > '2026-10-07 16:00:00+00:00'
  AND group_id IN (
      SELECT id FROM groups
      WHERE name = '10 元解锁国庆七天福利'
        AND subscription_type = 'subscription'
  );
