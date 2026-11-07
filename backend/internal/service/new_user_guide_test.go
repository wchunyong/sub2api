//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewUserGuideDefaultsAndBlankReset(t *testing.T) {
	ctx := context.Background()
	repo := &newUserBenefitSettingRepo{values: map[string]string{}}
	svc := NewPaymentConfigService(nil, repo, nil)

	cfg, err := svc.GetNewUserGuide(ctx)
	require.NoError(t, err)
	require.Equal(t, defaultNewUserGuideVideoURL, cfg.VideoURL)
	require.Equal(t, defaultNewUserGuidePDFURL, cfg.PDFURL)

	require.NoError(t, svc.UpdateNewUserGuide(ctx, NewUserGuideConfig{
		VideoURL: " https://example.com/video ", PDFURL: "http://example.com/guide.pdf",
	}))
	cfg, err = svc.GetNewUserGuide(ctx)
	require.NoError(t, err)
	require.Equal(t, "https://example.com/video", cfg.VideoURL)
	require.Equal(t, "http://example.com/guide.pdf", cfg.PDFURL)

	require.NoError(t, svc.UpdateNewUserGuide(ctx, NewUserGuideConfig{}))
	cfg, err = svc.GetNewUserGuide(ctx)
	require.NoError(t, err)
	require.Equal(t, defaultNewUserGuideVideoURL, cfg.VideoURL)
	require.Equal(t, defaultNewUserGuidePDFURL, cfg.PDFURL)
}

func TestUpdateNewUserGuideRejectsNonHTTPLinks(t *testing.T) {
	ctx := context.Background()
	repo := &newUserBenefitSettingRepo{values: map[string]string{}}
	svc := NewPaymentConfigService(nil, repo, nil)

	require.Error(t, svc.UpdateNewUserGuide(ctx, NewUserGuideConfig{VideoURL: "javascript:alert(1)"}))
	require.Error(t, svc.UpdateNewUserGuide(ctx, NewUserGuideConfig{PDFURL: "ftp://example.com/a.pdf"}))
	require.Error(t, svc.UpdateNewUserGuide(ctx, NewUserGuideConfig{VideoURL: "not a url"}))
	require.Empty(t, repo.values)
}
