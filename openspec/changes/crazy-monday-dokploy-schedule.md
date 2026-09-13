# 疯狂星期一 Dokploy Schedule 方案

## 目标

每周一为每个有效用户发放一个仅周一当天可用的订阅活动。活动固定使用订阅分组 `疯狂星期一`，额度为 10 美元。

## 执行方式

新增一个后端一次性命令：

```bash
backend/cmd/crazy-monday
```

命令由 Dokploy Schedule 每周日晚触发。脚本默认 dry-run，只打印本次会处理的用户数量和时间窗；只有显式传入 `--execute` 才写入数据库。

后台已有的人工分配入口是 `POST /api/v1/admin/subscriptions/assign`，最终调用 `SubscriptionService.AssignSubscription`，适合“从当前时刻起 N 天”的人工操作。本活动脚本参考它的字段和分配语义，但为了严格满足“周一当天”窗口，写入时显式设置 `starts_at = 周一 00:00:00`、`expires_at = 周二 00:00:00`。

## 时间窗

活动按配置时区计算，默认使用系统配置里的 `timezone`，未配置时沿用项目默认 `Asia/Shanghai`。

每次执行时脚本计算“下一个周一”的窗口：

```text
starts_at  = 周一 00:00:00
expires_at = 周二 00:00:00
```

由于当前订阅有效性主要依赖 `status=active` 与 `expires_at > now`，不严格检查 `starts_at <= now`，如果在周日 23:59:59 提前写入，可能出现极短时间提前生效。因此脚本支持提前启动：如果 Dokploy 在周日 23:59 触发，脚本会 sleep 到周一 00:00:00 后再执行写入。

## Dokploy Schedule

如果 Dokploy 使用标准 5 段 cron：

```text
59 23 * * 0
```

命令示例：

```bash
/app/crazy-monday --group-name "疯狂星期一" --quota-usd 10 --execute
```

如果 Dokploy 支持 6 段带秒 cron，可配置：

```text
59 59 23 * * 0
```

仍建议保留脚本内部等待逻辑，避免不同 cron 实现或容器时间精度造成提前写入。

## 数据策略

目标分组必须满足：

- `name = 疯狂星期一`
- `status = active`
- `subscription_type = subscription`
- `daily_limit_usd = 10`

脚本启动时会校验这些条件。若分组不存在或不是订阅类型，直接失败，不自动创建分组，避免把错误配置静默带入生产。

发放对象：

- 默认仅处理 `status = active` 且未软删除用户。

订阅写入规则：

- 没有该分组订阅：创建一条 `user_subscriptions`。
- 已有该分组订阅且不是本周活动：刷新为本周一到周二，状态设为 `active`，用量清零。
- 已有该分组订阅且 notes 已包含本周活动标识：跳过。

活动标识格式：

```text
crazy-monday:YYYY-MM-DD
```

其中日期是本次活动周一日期，例如 `crazy-monday:2026-09-14`。

## 安全与可观测性

脚本需要输出：

- dry-run 或 execute 模式。
- 分组 ID、分组名、额度。
- 活动窗口 starts_at/expires_at。
- 扫描用户数、创建数、刷新数、跳过数、失败数。
- 每个失败用户的 ID 和原因。

脚本必须支持重复执行，不应重复叠加订阅有效期。

## 验证

最小验证范围：

```bash
cd backend
go test ./cmd/crazy-monday
```

上线前建议额外执行一次 dry-run：

```bash
/app/crazy-monday --group-name "疯狂星期一" --quota-usd 10
```

确认输出符合预期后，再在 Dokploy Schedule 中使用 `--execute`。
