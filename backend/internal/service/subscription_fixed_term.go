package service

import (
	"context"
	"strings"
	"time"
)

// assignSubscriptionUntil grants a fixed-term benefit without renewal or quota resets.
func (s *SubscriptionService) assignSubscriptionUntil(ctx context.Context, input *AssignSubscriptionInput, expiresAt time.Time) (*UserSubscription, error) {
	if input == nil {
		return nil, ErrSubscriptionNilInput
	}
	now := s.now()
	if !expiresAt.After(now) {
		return nil, ErrSubscriptionExpired
	}
	group, err := s.groupRepo.GetByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}
	if !group.IsSubscriptionType() {
		return nil, ErrGroupNotSubscriptionType
	}

	var subID int64
	err = s.withSubscriptionUpdateTx(ctx, func(txCtx context.Context) error {
		exists, err := s.userSubRepo.ExistsByUserIDAndGroupID(txCtx, input.UserID, input.GroupID)
		if err != nil {
			return err
		}
		if exists {
			sub, err := s.userSubRepo.GetByUserIDAndGroupID(txCtx, input.UserID, input.GroupID)
			if err != nil {
				return err
			}
			sub, err = s.userSubRepo.GetByIDForUpdate(txCtx, sub.ID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(sub.Notes) != strings.TrimSpace(input.Notes) {
				return ErrSubscriptionAssignConflict
			}
			subID = sub.ID
			// Repair legacy rounded expiry without reviving suspended grants or
			// modifying usage counters and window anchors on repeated payments.
			if sub.ExpiresAt.After(expiresAt) {
				return s.userSubRepo.ExtendExpiry(txCtx, sub.ID, expiresAt)
			}
			return nil
		}

		sub := &UserSubscription{
			UserID: input.UserID, GroupID: input.GroupID,
			StartsAt: now, ExpiresAt: expiresAt, Status: SubscriptionStatusActive,
			AssignedAt: now, Notes: input.Notes, CreatedAt: now, UpdatedAt: now,
		}
		if input.AssignedBy > 0 {
			sub.AssignedBy = &input.AssignedBy
		}
		if err := s.userSubRepo.Create(txCtx, sub); err != nil {
			return err
		}
		subID = sub.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.invalidateSubscriptionCaches(input.UserID, input.GroupID); err != nil {
		return nil, err
	}
	return s.userSubRepo.GetByID(ctx, subID)
}
