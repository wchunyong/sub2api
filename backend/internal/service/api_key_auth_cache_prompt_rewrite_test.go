package service

import (
	"context"
	"testing"
)

// 防回归：分组级 prompt_rewrite 开关必须能通过认证缓存快照往返。
// 该字段曾只加在 Group 与快照结构体上，忘记在 snapshotFromAPIKey /
// snapshotToAPIKey 中赋值，导致真实请求里恒为 false、开关失效。
func TestAuthSnapshotKeepsPromptRewrite(t *testing.T) {
	const groupID = int64(9)

	for _, want := range []bool{true, false} {
		want := want
		t.Run(map[bool]string{true: "enabled", false: "disabled"}[want], func(t *testing.T) {
			svc := &APIKeyService{}
			gid := groupID
			apiKey := &APIKey{
				ID:      1,
				UserID:  2,
				GroupID: &gid,
				User:    &User{ID: 2, Status: StatusActive, Role: RoleUser},
				Group: &Group{
					ID:            groupID,
					Name:          "prompt-rewrite-group",
					Platform:      PlatformOpenAI,
					Status:        StatusActive,
					PromptRewrite: want,
				},
			}

			snapshot := svc.snapshotFromAPIKey(context.Background(), apiKey)
			if snapshot == nil || snapshot.Group == nil {
				t.Fatalf("snapshot missing group")
			}
			if snapshot.Group.PromptRewrite != want {
				t.Fatalf("snapshot lost prompt_rewrite: got %v, want %v", snapshot.Group.PromptRewrite, want)
			}

			restored, used, err := svc.applyAuthCacheEntry("test-key", &APIKeyAuthCacheEntry{Snapshot: snapshot})
			if err != nil {
				t.Fatalf("applyAuthCacheEntry: %v", err)
			}
			if !used || restored == nil || restored.Group == nil {
				t.Fatalf("snapshot not restored: used=%v", used)
			}
			if restored.Group.PromptRewrite != want {
				t.Fatalf("restored group lost prompt_rewrite: got %v, want %v", restored.Group.PromptRewrite, want)
			}
		})
	}
}
