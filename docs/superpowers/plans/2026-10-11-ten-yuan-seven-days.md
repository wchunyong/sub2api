# Ten Yuan Seven Day Activity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (- [ ]) syntax for tracking.

**Goal:** Let administrators schedule and stop repeatable ten-yuan, seven-day recharge activities and let users see and unlock each round safely.

**Architecture:** Persist each activity round and each successful user claim in dedicated Ent entities. Extend PaymentService to manage the schedule and grant the chosen subscription group during balance-order fulfillment, then expose admin and user APIs. Build separate admin and user Vue pages and a dynamic user navigation item.

**Tech Stack:** Go 1.27, Gin, Ent, PostgreSQL migrations, Vue 3, TypeScript, Pinia, Vitest, pnpm.

**Spec:** docs/superpowers/specs/2026-10-11-ten-yuan-seven-days-design.md

## Global Constraints

- Each round starts at a Beijing time selected to the second and ends exactly 168 hours later.
- Eligibility is one completed balance recharge order with PaymentOrder.Amount at least 10 during the round; use PaidAt, with inclusive start and exclusive end.
- Populate PaidAt from a verified provider payment timestamp when supplied by the callback or query; otherwise use the server receipt time. This fallback was confirmed during implementation for providers without a trustworthy payment timestamp.
- One claim per user per round. Repeat payments do not reset usage or extend expiry. Prior activity-owned expired subscriptions may be renewed for the next round.
- Closing early stops qualification for payments after DisabledAt; payments before DisabledAt may settle before the original end. Existing grants expire at the original end.
- A callback handled after the original end completes the order without an expired grant and records a skip reason.
- A round may cover up to eight midnight-reset quota windows. The UI must say that the first and last dates may be partial.
- Default group is the active subscription group named exactly 十元七天乐; never silently choose a different group.
- Do not expose the word 订阅 in user-facing activity UI. Do not reuse national-day 50% bonus or old-user extra quota.
- Keep API contracts, payment idempotency, auditability and existing national-day behavior. Do not edit repository skills or add lockfiles other than pnpm-lock.yaml if package.json changes.

## Review Focus

1. Concurrent admin scheduling: exactly one current round survives; Task 1 test TestTenYuanScheduleConcurrent.
2. Invalid or changed group at fulfillment: paid orders retain retryable status and a visible audit reason; Task 3 test TestTenYuanGrantFailureRetryable.
3. Two qualifying orders for one user arrive concurrently: only one claim and no quota reset; Task 2 test TestTenYuanConcurrentClaim.
4. Browser clock differs from server time: countdown uses server offset and rolls status at zero; Task 5 test countdownWithServerSkew.
5. Activity closes while a callback is delayed: PaidAt before close remains eligible until original end; Task 3 test TestTenYuanDelayedCallbackBeforeClose.

---

### Task 1: Persist and schedule activity rounds

**Files:**
- Create: backend/ent/schema/activity_run.go
- Create: backend/ent/schema/activity_claim.go
- Create: backend/migrations/243_ten_yuan_activity.sql
- Create: backend/internal/service/payment_ten_yuan_activity.go
- Create: backend/internal/service/payment_ten_yuan_activity_schedule_test.go
- Modify: backend/ent generated files after go generate ./ent
- Test: backend/migrations/ten_yuan_activity_migration_test.go

**Interfaces:**
- Produces: TenYuanActivityRun {ID, GroupID, StartsAt, EndsAt, DisabledAt}; TenYuanActivityScheduleInput {GroupID, StartsAt}; PaymentService.GetTenYuanActivityAdmin(ctx context.Context, now time.Time) (*TenYuanActivityAdminState, error); PaymentService.ScheduleTenYuanActivity(ctx context.Context, input TenYuanActivityScheduleInput, adminID int64, now time.Time) (*TenYuanActivityRun, error); PaymentService.CloseTenYuanActivity(ctx context.Context, runID int64, now time.Time) error. AdminState contains current run, recent runs, and default_group_id.
- Consumes: PaymentService.entClient and groupRepo.GetByID.

- [ ] **Step 1: Write failing schedule and migration tests.** Test exact 168-hour end from a second-precision start; start/end boundary; invalid/non-subscription/inactive group; exact-name default discovery; edit only before start; close without shortening EndsAt; new round only after previous EndsAt; concurrent schedule leaves one current round. Migration test checks unique current marker and run/claim indexes.
- [ ] **Step 2: Run tests red.** Run from backend: go test -tags=unit ./internal/service -run 'TestTenYuanSchedule|TestTenYuanActivityStatus'. Expected: FAIL because activity methods/entities are absent. Run: go test ./migrations -run TestTenYuanActivityMigration. Expected: FAIL.
- [ ] **Step 3: Add two Ent schemas, SQL migration and schedule service.** ActivityRun fields: code, nullable unique current_marker, group_id, starts_at, ends_at, nullable disabled_at, created_by, created_at, updated_at. ActivityClaim fields: run_id, user_id, order_id, subscription_id, claimed_at; unique (run_id,user_id) and order_id. Schedule in a transaction: archive the prior current marker only if its original end has passed; otherwise edit a not-yet-started round or reject. The unique current_marker enforces the one-current-round invariant under concurrent writes. Parse/validate offset-bearing timestamps at HTTP boundary; service uses time.Time.
- [ ] **Step 4: Generate Ent code and run tests green.** Run from backend: go generate ./ent. Expected: exit 0. Run the two test commands in Step 2. Expected: PASS.
- [ ] **Step 5: Commit.** Stage the two schemas, generated Ent files, migration, service and tests. Commit: feat: persist ten yuan activity rounds.

### Task 2: Grant fixed-term subscription claims

**Files:**
- Create: backend/internal/service/subscription_activity_grant.go
- Create: backend/internal/service/subscription_activity_grant_test.go
- Modify: backend/internal/service/payment_ten_yuan_activity.go
- Modify: backend/internal/service/payment_ten_yuan_activity_schedule_test.go

**Interfaces:**
- Consumes: Task 1 ActivityRun and ActivityClaim entities.
- Produces: SubscriptionService.assignTenYuanActivityUntilInTx(ctx context.Context, userID, groupID, runID int64, expiresAt time.Time) (*UserSubscription, error); PaymentService.applyTenYuanActivityForOrder(ctx context.Context, order *ent.PaymentOrder) error.

- [ ] **Step 1: Write failing grant tests.** Test a first claim expires at round EndsAt; second order in the same round does not change DailyUsageUSD; duplicate callback stays one claim; expired prior activity-owned subscription renews and resets windows in a later round; active/manual subscription conflicts without mutation; TestTenYuanConcurrentClaim proves one claim under two concurrent orders.
- [ ] **Step 2: Run red.** Run from backend: go test -tags=unit ./internal/service -run 'TestTenYuanGrant|TestTenYuanConcurrentClaim'. Expected: FAIL because grant methods are absent.
- [ ] **Step 3: Implement grant transaction.** Lock the existing user/group subscription, call the new in-transaction subscription method, insert the unique claim, and commit together. Tag campaign ownership in subscription notes with round ID. Handle unique-claim collision as already claimed, never reset on same-round replay. Invalidate subscription and billing caches after commit. Preserve conflict errors for non-activity grants.
- [ ] **Step 4: Run green.** Run the command in Step 2. Expected: PASS.
- [ ] **Step 5: Commit.** Commit: feat: grant idempotent ten yuan activity claims.

### Task 3: Connect payment fulfillment and HTTP contracts

**Files:**
- Modify: backend/internal/service/payment_fulfillment.go
- Modify: backend/internal/handler/payment_handler.go
- Modify: backend/internal/handler/admin/payment_handler.go
- Modify: backend/internal/server/routes/payment.go
- Create: backend/internal/handler/payment_ten_yuan_activity_test.go
- Create: backend/internal/service/payment_ten_yuan_activity_fulfillment_test.go
- Modify: frontend/src/types/payment.ts

**Interfaces:**
- Consumes: Task 1 schedule methods and Task 2 applyTenYuanActivityForOrder.
- Produces: GET /api/v1/admin/payment/activities/ten-yuan-seven-days; PUT /api/v1/admin/payment/activities/ten-yuan-seven-days; POST /api/v1/admin/payment/activities/ten-yuan-seven-days/close; GET /api/v1/payment/activities/ten-yuan-seven-days. PaymentService.GetTenYuanActivityForUser(ctx context.Context, userID int64, now time.Time) (*TenYuanActivityUserState, error). User response includes status, starts_at, ends_at, server_now, group benefit summary and claimed. Checkout response adds ten_yuan_activity summary.

- [ ] **Step 1: Write failing service and HTTP tests.** Test insufficient amount and non-balance order; payment at exact start qualifies and exact end does not; TestTenYuanDelayedCallbackBeforeClose; callback after end skips with audit; TestTenYuanGrantFailureRetryable; replayed paid order; admin auth and malformed seconds/timezone; user response excludes admin fields; checkout exposes only live eligibility.
- [ ] **Step 2: Run red.** Run from backend: go test -tags=unit ./internal/service ./internal/handler -run 'TestTenYuan'. Expected: FAIL because endpoints and fulfillment hook are absent.
- [ ] **Step 3: Integrate before markCompleted.** For balance fulfillment call applyTenYuanActivityForOrder before markCompleted in both create/redeem and reused-redeem paths. A grant failure returns an error so the existing paid-order retry path remains available. A callback after EndsAt logs a skip and proceeds. Add the four routes to the existing payment handler instances; map service validation errors to 4xx and persistence errors to 5xx. Return server_now for countdown calibration.
- [ ] **Step 4: Run green and check existing national-day behavior.** Run Step 2 command. Expected: PASS. Run from backend: go test -tags=unit ./internal/service -run 'TestNationalDay|TestTenYuan'. Expected: PASS.
- [ ] **Step 5: Commit.** Commit: feat: expose ten yuan activity and fulfill recharge grants.

### Task 4: Admin activity page

**Files:**
- Create: frontend/src/api/admin/activities.ts
- Create: frontend/src/views/admin/ActivitiesView.vue
- Create: frontend/src/views/admin/__tests__/ActivitiesView.spec.ts
- Modify: frontend/src/router/index.ts
- Modify: frontend/src/components/layout/AppSidebar.vue
- Modify: frontend/src/i18n/locales/zh/common.ts
- Modify: frontend/src/i18n/locales/en/common.ts
- Create: frontend/src/i18n/locales/zh/admin/activity.ts and frontend/src/i18n/locales/en/admin/activity.ts
- Modify: frontend/src/i18n/locales/zh/admin/index.ts and frontend/src/i18n/locales/en/admin/index.ts

**Interfaces:**
- Consumes: Task 3 admin routes and task 1 group validation contract; existing admin/groups.getAll.
- Produces: admin route /admin/activities and navigation label 活动.

- [ ] **Step 1: Write failing Vue tests.** Verify one activity card; name-exact default group; missing default requires selection; second-precision Beijing input sent as offset-bearing ISO timestamp; derived end display; scheduled edit, active locked controls, and early-close warning. Include a server validation error case.
- [ ] **Step 2: Run red.** Run from frontend: pnpm run test:run -- src/views/admin/__tests__/ActivitiesView.spec.ts. Expected: FAIL because page does not exist.
- [ ] **Step 3: Implement the API module and page.** Reuse existing admin form/layout patterns. Place 活动 in admin sidebar with the PriceTagIcon and an admin-only route. Display current status, exact start and computed end, chosen group's actual daily limit, and action errors; do not offer unsupported activity types.
- [ ] **Step 4: Run green.** Run Step 2 command. Expected: PASS.
- [ ] **Step 5: Commit.** Commit: feat: add admin activity management page.

### Task 5: User activity page and dynamic navigation

**Files:**
- Create: frontend/src/api/activities.ts
- Create: frontend/src/views/user/TenYuanActivityView.vue
- Create: frontend/src/views/user/__tests__/TenYuanActivityView.spec.ts
- Modify: frontend/src/components/layout/AppSidebar.vue
- Modify: frontend/src/components/layout/__tests__/AppSidebar.spec.ts
- Modify: frontend/src/views/user/PaymentView.vue and frontend/src/views/user/__tests__/PaymentView.spec.ts
- Modify: frontend/src/router/index.ts
- Modify: frontend/src/i18n/locales/zh/common.ts and frontend/src/i18n/locales/en/common.ts
- Modify: frontend/src/i18n/locales/zh/misc.ts and frontend/src/i18n/locales/en/misc.ts

**Interfaces:**
- Consumes: Task 3 user status and checkout summary.
- Produces: authenticated route /ten-yuan-seven-days, user navigation label 十元七天乐 with PriceTagIcon, start/end countdowns, claimed status and recharge CTA.

- [ ] **Step 1: Write failing Vue tests.** Verify navigation visible only while enabled and before original end, including scheduled; hidden after close/end. Verify countdownWithServerSkew using fake timers and server_now; start-to-active and active-to-ended refresh; claimed messaging, accurate dates, 10-yuan threshold, partial first/last dates, and no user-facing 订阅 or national-day bonuses. Verify checkout shows active eligibility hint only.
- [ ] **Step 2: Run red.** Run from frontend: pnpm run test:run -- src/views/user/__tests__/TenYuanActivityView.spec.ts src/components/layout/__tests__/AppSidebar.spec.ts src/views/user/__tests__/PaymentView.spec.ts. Expected: FAIL on the new behavior.
- [ ] **Step 3: Implement page and navigation.** Adapt the existing national-day page layout to this one activity. Fetch status on mount, on visibility/focus and periodically while the shell is open; derive countdown from server_now offset. Re-fetch at zero. Keep user copy in Chinese and English locale files; no subscription navigation restoration.
- [ ] **Step 4: Run green.** Run Step 2 command. Expected: PASS.
- [ ] **Step 5: Commit.** Commit: feat: show ten yuan activity to users.

## Final verification

- Run from backend: go test -tags=unit ./internal/service ./internal/handler ./migrations. Expected: PASS.
- Run from backend: go test ./... . Expected: PASS. If external integration dependencies make this fail, capture the exact failure and run the largest relevant subset.
- Run from frontend: pnpm run lint:check; pnpm run typecheck; pnpm run test:run; pnpm run build. Expected: all exit 0.
- Review the diff against the spec, inspect generated Ent changes, verify no old national-day behavior or user subscription visibility changed, and run git diff --check.
