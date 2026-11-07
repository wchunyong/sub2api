package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

const (
	SettingNewUserGuideVideoURL = "NEW_USER_GUIDE_VIDEO_URL"
	SettingNewUserGuidePDFURL   = "NEW_USER_GUIDE_PDF_URL"

	defaultNewUserGuideVideoURL = "https://www.bilibili.com/video/BV1XgeB6UEQg"
	defaultNewUserGuidePDFURL   = "https://pub-17119e84a4e34086ab498f52e1943e3a.r2.dev/tutorial/win.pdf"
	maxNewUserGuideURLLength    = 2048
)

// NewUserGuideConfig holds the links shown by the long-running new user guide.
// Empty stored values fall back to the built-in defaults.
type NewUserGuideConfig struct {
	VideoURL string `json:"video_url"`
	PDFURL   string `json:"pdf_url"`
}

func validateNewUserGuideURL(raw string) error {
	if len(raw) > maxNewUserGuideURLLength {
		return fmt.Errorf("guide link must be at most %d characters", maxNewUserGuideURLLength)
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("guide link must be an absolute http or https URL")
	}
	return nil
}

func resolveNewUserGuideURL(raw, fallback string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	return fallback
}

// GetNewUserGuide returns the effective video and PDF links for the guide.
func (s *PaymentConfigService) GetNewUserGuide(ctx context.Context) (*NewUserGuideConfig, error) {
	values, err := s.settingRepo.GetMultiple(ctx, []string{SettingNewUserGuideVideoURL, SettingNewUserGuidePDFURL})
	if err != nil {
		return nil, err
	}
	return &NewUserGuideConfig{
		VideoURL: resolveNewUserGuideURL(values[SettingNewUserGuideVideoURL], defaultNewUserGuideVideoURL),
		PDFURL:   resolveNewUserGuideURL(values[SettingNewUserGuidePDFURL], defaultNewUserGuidePDFURL),
	}, nil
}

// UpdateNewUserGuide stores the guide links. A blank link resets it to the default.
func (s *PaymentConfigService) UpdateNewUserGuide(ctx context.Context, cfg NewUserGuideConfig) error {
	video := strings.TrimSpace(cfg.VideoURL)
	pdf := strings.TrimSpace(cfg.PDFURL)
	if video != "" {
		if err := validateNewUserGuideURL(video); err != nil {
			return fmt.Errorf("video url: %w", err)
		}
	}
	if pdf != "" {
		if err := validateNewUserGuideURL(pdf); err != nil {
			return fmt.Errorf("pdf url: %w", err)
		}
	}
	return s.settingRepo.SetMultiple(ctx, map[string]string{
		SettingNewUserGuideVideoURL: video,
		SettingNewUserGuidePDFURL:   pdf,
	})
}
