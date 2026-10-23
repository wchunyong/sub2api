package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProgrammersDayWindowUsesBeijingCalendar(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	for _, tc := range []struct {
		at     time.Time
		active bool
	}{
		{time.Date(2026, 10, 23, 23, 59, 59, 0, loc), false},
		{time.Date(2026, 10, 24, 0, 0, 0, 0, loc), true},
		{time.Date(2026, 10, 26, 23, 59, 59, 0, loc), true},
		{time.Date(2026, 10, 27, 0, 0, 0, 0, loc), false},
		{time.Date(2027, 10, 24, 0, 0, 0, 0, loc), true},
	} {
		require.Equal(t, tc.active, programmersDayWindowActive(tc.at))
	}
}

func TestProgrammersDayAdminSwitchPersistsAndControlsWindow(t *testing.T) {
	repo := &paymentConfigSettingRepoStub{values: map[string]string{}}
	svc := &PaymentConfigService{settingRepo: repo}
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.FixedZone("CST", 8*3600))
	state, err := svc.GetProgrammersDay(context.Background(), now)
	require.NoError(t, err)
	require.True(t, state.Active)
	require.Equal(t, 24.0, state.BonusPercent)
	require.NoError(t, svc.UpdateProgrammersDay(context.Background(), false, 30))
	state, err = svc.GetProgrammersDay(context.Background(), now)
	require.NoError(t, err)
	require.False(t, state.Active)
	require.False(t, state.Visible)
	require.Equal(t, 30.0, state.BonusPercent)
}

func TestProgrammersDayOverridesRegularRechargeBonus(t *testing.T) {
	cfg := &PaymentConfig{BalanceRechargeMultiplier: 1, RechargeBonusTiers: []RechargeBonusTier{{MinAmount: 100, BonusPercent: 5}}, ProgrammersDayEnabled: true, ProgrammersDayBonusPercent: 24}
	active := time.Date(2026, 10, 24, 12, 0, 0, 0, time.FixedZone("CST", 8*3600))
	amount, payBase, bonus := calculateBalanceRechargeOrderAmounts(cfg, 100, "CNY", active)
	require.Equal(t, 124.0, amount)
	require.Equal(t, 100.0, payBase)
	require.Equal(t, 24.0, bonus)
	cfg.BalanceRechargeMultiplier = 2
	amount, payBase, bonus = calculateBalanceRechargeOrderAmounts(cfg, 100, "CNY", active)
	require.Equal(t, 124.0, amount)
	require.Equal(t, 100.0, payBase)
	require.Equal(t, 24.0, bonus)
	cfg.BalanceRechargeMultiplier = 1
	cfg.ProgrammersDayEnabled = false
	amount, _, _ = calculateBalanceRechargeOrderAmounts(cfg, 100, "CNY", active)
	require.Equal(t, 105.0, amount)
}

func TestProgrammersDaySettingsDefaultAndValidation(t *testing.T) {
	svc := &PaymentConfigService{}
	cfg := svc.parsePaymentConfig(map[string]string{})
	require.True(t, cfg.ProgrammersDayEnabled)
	require.Equal(t, 24.0, cfg.ProgrammersDayBonusPercent)
	require.Error(t, validateProgrammersDayBonusPercent(-1))
	require.Error(t, validateProgrammersDayBonusPercent(1001))
	require.NoError(t, validateProgrammersDayBonusPercent(24))
	require.NoError(t, validateProgrammersDayBonusPercent(24.01))
}
