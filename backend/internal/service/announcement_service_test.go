package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type announcementRepoStub struct {
	item        *Announcement
	activeItems []Announcement
}

func (s *announcementRepoStub) Create(_ context.Context, a *Announcement) error {
	s.item = a
	return nil
}

func (s *announcementRepoStub) GetByID(_ context.Context, _ int64) (*Announcement, error) {
	if s.item == nil {
		return nil, ErrAnnouncementNotFound
	}
	return s.item, nil
}

func (s *announcementRepoStub) Update(_ context.Context, a *Announcement) error {
	s.item = a
	return nil
}

func (*announcementRepoStub) Delete(context.Context, int64) error {
	return nil
}

func (*announcementRepoStub) List(context.Context, pagination.PaginationParams, AnnouncementListFilters) ([]Announcement, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *announcementRepoStub) ListActive(context.Context, time.Time) ([]Announcement, error) {
	return s.activeItems, nil
}

func TestAnnouncementServiceCreateRejectsEqualStartEndTimes(t *testing.T) {
	repo := &announcementRepoStub{}
	svc := NewAnnouncementService(repo, nil, nil, nil)
	now := time.Unix(1776790020, 0)

	_, err := svc.Create(context.Background(), &CreateAnnouncementInput{
		Title:      "公告",
		Content:    "内容",
		Status:     AnnouncementStatusActive,
		NotifyMode: AnnouncementNotifyModePopup,
		StartsAt:   &now,
		EndsAt:     &now,
	})
	require.ErrorIs(t, err, ErrAnnouncementInvalidSchedule)
}

func TestAnnouncementServiceCreateAcceptsBannerAnnouncements(t *testing.T) {
	repo := &announcementRepoStub{}
	svc := NewAnnouncementService(repo, nil, nil, nil)

	created, err := svc.Create(context.Background(), &CreateAnnouncementInput{
		Title:      " 顶部横幅 ",
		Content:    " 💪 2024 年度白皮书发布 ",
		Status:     AnnouncementStatusActive,
		NotifyMode: AnnouncementNotifyModeBanner,
		BannerConfig: AnnouncementBannerConfig{
			WholeBannerClickEnabled: true,
			ClickURL:                " https://hovxm.com ",
			ButtonEnabled:           true,
			ButtonText:              " 了解详情 ",
			ButtonURL:               " https://hovxm.com/docs ",
		},
	})

	require.NoError(t, err)
	require.Equal(t, AnnouncementNotifyModeBanner, created.NotifyMode)
	require.True(t, created.BannerConfig.WholeBannerClickEnabled)
	require.Equal(t, "https://hovxm.com", created.BannerConfig.ClickURL)
	require.True(t, created.BannerConfig.ButtonEnabled)
	require.Equal(t, "了解详情", created.BannerConfig.ButtonText)
	require.Equal(t, "https://hovxm.com/docs", created.BannerConfig.ButtonURL)
}

func TestAnnouncementServiceCreateBannerUsesDefaultTitleWhenBlank(t *testing.T) {
	repo := &announcementRepoStub{}
	svc := NewAnnouncementService(repo, nil, nil, nil)

	created, err := svc.Create(context.Background(), &CreateAnnouncementInput{
		Title:      "",
		Content:    "横幅内容",
		Status:     AnnouncementStatusActive,
		NotifyMode: AnnouncementNotifyModeBanner,
	})

	require.NoError(t, err)
	require.Equal(t, DefaultBannerAnnouncementTitle, created.Title)
}

func TestAnnouncementServiceListForAnonymousShowsOnlyUntargetedBanners(t *testing.T) {
	repo := &announcementRepoStub{
		activeItems: []Announcement{
			{
				ID:         1,
				Title:      "全员横幅",
				Content:    "访客可见",
				Status:     AnnouncementStatusActive,
				NotifyMode: AnnouncementNotifyModeBanner,
			},
			{
				ID:         2,
				Title:      "弹窗",
				Content:    "访客不需要",
				Status:     AnnouncementStatusActive,
				NotifyMode: AnnouncementNotifyModePopup,
			},
			{
				ID:         3,
				Title:      "条件横幅",
				Content:    "需要余额条件",
				Status:     AnnouncementStatusActive,
				NotifyMode: AnnouncementNotifyModeBanner,
				Targeting: AnnouncementTargeting{
					AnyOf: []AnnouncementConditionGroup{{
						AllOf: []AnnouncementCondition{{
							Type:     AnnouncementConditionTypeBalance,
							Operator: AnnouncementOperatorGT,
							Value:    0,
						}},
					}},
				},
			},
		},
	}
	svc := NewAnnouncementService(repo, nil, nil, nil)

	items, err := svc.ListForAnonymous(context.Background())

	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(1), items[0].Announcement.ID)
	require.Nil(t, items[0].ReadAt)
}

func TestAnnouncementServiceCreateRejectsBannerInvalidURL(t *testing.T) {
	repo := &announcementRepoStub{}
	svc := NewAnnouncementService(repo, nil, nil, nil)

	_, err := svc.Create(context.Background(), &CreateAnnouncementInput{
		Title:      "顶部横幅",
		Content:    "白皮书发布",
		Status:     AnnouncementStatusActive,
		NotifyMode: AnnouncementNotifyModeBanner,
		BannerConfig: AnnouncementBannerConfig{
			WholeBannerClickEnabled: true,
			ClickURL:                "javascript:alert(1)",
		},
	})

	require.ErrorIs(t, err, ErrAnnouncementInvalidBannerConfig)
}

func TestAnnouncementServiceUpdateRejectsEqualStartEndTimes(t *testing.T) {
	repo := &announcementRepoStub{
		item: &Announcement{
			ID:         1,
			Title:      "公告",
			Content:    "内容",
			Status:     AnnouncementStatusActive,
			NotifyMode: AnnouncementNotifyModePopup,
		},
	}
	svc := NewAnnouncementService(repo, nil, nil, nil)
	now := time.Unix(1776790020, 0)
	startsAt := &now
	endsAt := &now

	_, err := svc.Update(context.Background(), 1, &UpdateAnnouncementInput{
		StartsAt: &startsAt,
		EndsAt:   &endsAt,
	})
	require.ErrorIs(t, err, ErrAnnouncementInvalidSchedule)
}
