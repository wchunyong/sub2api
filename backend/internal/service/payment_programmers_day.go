package service

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

const (
	SettingProgrammersDayEnabled      = "PROGRAMMERS_DAY_ENABLED"
	SettingProgrammersDayBonusPercent = "PROGRAMMERS_DAY_BONUS_PERCENT"
)

type ProgrammersDayState struct {
	Enabled      bool      `json:"enabled"`
	Active       bool      `json:"active"`
	Visible      bool      `json:"visible"`
	BonusPercent float64   `json:"bonus_percent"`
	StartsAt     time.Time `json:"starts_at"`
	EndsAt       time.Time `json:"ends_at"`
	ServerNow    time.Time `json:"server_now"`
}

func programmersDayWindow(now time.Time) (time.Time, time.Time) {
	loc := time.FixedZone("Asia/Shanghai", 8*3600)
	local := now.In(loc)
	start := time.Date(local.Year(), time.October, 24, 0, 0, 0, 0, loc)
	return start, start.AddDate(0, 0, 3)
}

func programmersDayWindowActive(now time.Time) bool {
	start, end := programmersDayWindow(now)
	return !now.Before(start) && now.Before(end)
}

func validateProgrammersDayBonusPercent(percent float64) error {
	if !rechargeBonusValueValid(percent, maxRechargeBonusPercent) {
		return fmt.Errorf("bonus percent must be between 0 and 1000 with at most two decimal places")
	}
	return nil
}

func parseProgrammersDayBonusPercent(raw string) float64 {
	percent, err := strconv.ParseFloat(raw, 64)
	if err != nil || validateProgrammersDayBonusPercent(percent) != nil {
		return 24
	}
	return percent
}

func (s *PaymentConfigService) GetProgrammersDay(ctx context.Context, now time.Time) (*ProgrammersDayState, error) {
	values, err := s.settingRepo.GetMultiple(ctx, []string{SettingProgrammersDayEnabled, SettingProgrammersDayBonusPercent})
	if err != nil {
		return nil, err
	}
	start, end := programmersDayWindow(now)
	percent := parseProgrammersDayBonusPercent(values[SettingProgrammersDayBonusPercent])
	enabled := values[SettingProgrammersDayEnabled] != "false"
	return &ProgrammersDayState{Enabled: enabled, Active: enabled && programmersDayWindowActive(now), Visible: enabled && !now.Before(start.AddDate(0, 0, -1)) && now.Before(end), BonusPercent: percent, StartsAt: start, EndsAt: end, ServerNow: now}, nil
}

func (s *PaymentConfigService) UpdateProgrammersDay(ctx context.Context, enabled bool, percent float64) error {
	if err := validateProgrammersDayBonusPercent(percent); err != nil {
		return err
	}
	return s.settingRepo.SetMultiple(ctx, map[string]string{
		SettingProgrammersDayEnabled:      strconv.FormatBool(enabled),
		SettingProgrammersDayBonusPercent: strconv.FormatFloat(percent, 'f', -1, 64),
	})
}
