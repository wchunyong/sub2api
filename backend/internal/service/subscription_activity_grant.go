package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const tenYuanActivityNotePrefix = "ten_yuan_seven_days:"

// assignTenYuanActivityUntilInTx is called inside the claim transaction. Its caller
// owns cache invalidation after commit.
func (s *SubscriptionService) assignTenYuanActivityUntilInTx(ctx context.Context, userID, groupID, runID int64, expiresAt time.Time) (*UserSubscription, error) {
	if s == nil || s.userSubRepo == nil || s.groupRepo == nil {
		return nil, ErrSubscriptionNilInput
	}
	now := s.now()
	if !expiresAt.After(now) {
		return nil, ErrSubscriptionExpired
	}
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil || !group.IsActive() || !group.IsSubscriptionType() {
		return nil, ErrGroupNotSubscriptionType
	}
	note := fmt.Sprintf("%s%d", tenYuanActivityNotePrefix, runID)
	existing, err := s.userSubRepo.GetByUserIDAndGroupID(ctx, userID, groupID)
	if err != nil && err != ErrSubscriptionNotFound {
		return nil, err
	}
	if existing == nil {
		sub := &UserSubscription{UserID: userID, GroupID: groupID, StartsAt: now, ExpiresAt: expiresAt, Status: SubscriptionStatusActive, AssignedAt: now, Notes: note, CreatedAt: now, UpdatedAt: now}
		if err := s.userSubRepo.Create(ctx, sub); err != nil {
			return nil, err
		}
		return sub, nil
	}
	existing, err = s.userSubRepo.GetByIDForUpdate(ctx, existing.ID)
	if err != nil {
		return nil, err
	}
	if tenYuanActivityNoteMatches(existing.Notes, note) {
		if existing.Status != SubscriptionStatusActive {
			return nil, ErrSubscriptionAssignConflict
		}
		if !existing.ExpiresAt.Equal(expiresAt) {
			existing.ExpiresAt = expiresAt
			if err := s.userSubRepo.Update(ctx, existing); err != nil {
				return nil, err
			}
		}
		return existing, nil
	}
	if !tenYuanActivityOwned(existing.Notes) || existing.ExpiresAt.After(now) {
		return nil, ErrSubscriptionAssignConflict
	}
	renewed := renewedSubscriptionTerm(existing, note, now, expiresAt)
	if err := s.userSubRepo.Update(ctx, renewed); err != nil {
		return nil, err
	}
	return renewed, nil
}

func tenYuanActivityNoteMatches(notes, marker string) bool {
	for _, line := range strings.Split(notes, "\n") {
		if strings.TrimSpace(line) == marker {
			return true
		}
	}
	return false
}

func tenYuanActivityOwned(notes string) bool {
	for _, line := range strings.Split(notes, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), tenYuanActivityNotePrefix) {
			return true
		}
	}
	return false
}
