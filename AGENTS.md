# AGENTS.md

本文件为参与 Sub2API 项目的编码代理提供项目级上下文。内容以项目信息和协作约定为主，避免把临时环境细节写入长期说明。

## 项目概览

- Sub2API 是一个用于订阅额度分发和多模型转发的 AI API Gateway 项目。
- 后端位于 `backend/`，主要技术栈为 Go、Gin、Ent ORM，并包含服务层、仓储层、HTTP 处理层、插件 API、迁移与生成代码。
- 前端位于 `frontend/`，主要技术栈为 Vue 3、Vite、TypeScript、Pinia、Vue Router、Tailwind CSS。
- 部署、容器和生产运维相关文件位于 `deploy/`、根目录 Dockerfile、GoReleaser 配置等位置。
- 功能设计、变更说明和阶段性方案可能位于 `openspec/changes/`。
- 项目内的 `skills/` 主要是本仓库配套的代理技能和说明，除非任务明确要求，不要把它当作生产应用代码修改。

## 目录说明

后端重点目录：

- `backend/cmd/server/`：服务入口、依赖注入和 Wire 生成文件。
- `backend/internal/handler/`：HTTP/API 处理层。
- `backend/internal/service/`：核心业务逻辑、上游模型适配、额度、计费、调度、支付等服务。
- `backend/internal/repository/`：数据访问与持久化相关实现。
- `backend/internal/config/`：运行时配置。
- `backend/ent/schema/`：Ent Schema 定义。
- `backend/ent/`：Ent 生成代码，修改 Schema 后需要同步生成。
- `backend/pkg/pluginapi/`：插件 API、protobuf 和运行时接口。
- `backend/resources/`：模型价格、上下文窗口等资源数据。

前端重点目录：

- `frontend/src/api/`：前端 API 客户端。
- `frontend/src/views/`：路由级页面。
- `frontend/src/components/`：通用组件和业务组件。
- `frontend/src/features/`：按功能聚合的前端模块。
- `frontend/src/composables/`：Vue 组合式函数。
- `frontend/src/stores/`：Pinia 状态管理。
- `frontend/src/i18n/`：国际化文案。
- `frontend/src/router/`：路由配置。

### 用户端订阅隐藏策略

- 当前产品线刻意在用户端隐藏“订阅”概念，只对用户展示“充值”。
- 用户端侧边栏隐藏入口位于 `frontend/src/components/layout/AppSidebar.vue`：`buildSelfNavItems` 不展示 `/subscriptions`，`/purchase` 文案使用 `nav.buySubscription`，中文为“充值”。
- 用户充值页位于 `frontend/src/views/user/PaymentView.vue`：默认不展示充值/订阅切换 tab，只展示充值；固定充值金额为 `10, 20, 50, 100, 200, 400, 800`。
- 充值到账映射同时存在于前端预览和后端实际入账：前端见 `rechargeCreditByPaymentAmount`，后端见 `backend/internal/service/payment_amounts.go` 的 `fixedBalanceRechargeCredits`。
- 如以后要恢复用户端订阅入口，需要同时恢复侧边栏 `/subscriptions`、购买页订阅 tab/套餐展示入口、相关 i18n 文案，并检查 `frontend/src/views/user/__tests__/PaymentView.spec.ts` 与 `frontend/src/components/layout/__tests__/AppSidebar.spec.ts` 的约束。
- 管理员端订阅管理仍保留，不属于该隐藏策略的范围。

## 开发约定

- 搜索文件和文本优先使用 `rg` 或 `rg --files`。
- 修改范围应聚焦在当前任务相关模块，不要顺手重构无关代码。
- 前端依赖管理使用 `pnpm`，不要引入 npm lockfile。
- 修改 `frontend/package.json` 时，同步更新 `frontend/pnpm-lock.yaml`。
- 后端模块路径为 `github.com/Wei-Shaw/sub2api`。
- Go 版本在 `backend/go.mod` 中固定为 `1.27.0`；变更 Go 版本时，需要同步检查 CI 和 Docker 构建配置。
- 修改 `backend/ent/schema/*.go` 后，需要重新生成并提交 `backend/ent/` 相关变化。
- 修改依赖注入、构造函数或服务装配时，注意同步 `backend/cmd/server/` 下的 Wire 生成结果。
- 给接口新增方法时，要同时补齐相关测试 stub、mock 和 fake 实现。
- 涉及 OpenAI 兼容、Anthropic 兼容、网关转发、额度、计费、支付、账号、代理、审计等逻辑时，优先保持现有 API 合约和审计能力。
- 不要提交密钥、Token、Cookie、真实账号数据、数据库导出或其他敏感信息。

## Git 分支策略

本项目推荐按以下思路维护分支：

```text
main       只同步开源 upstream，尽量不放私有改动
release    生产部署分支，Dokploy 只部署它
feature/*  功能开发分支，完成后合入 release
```

推荐规则：

```text
GitHub Sync fork -> main
main 定期 merge -> release
feature/* 完成 -> release
Dokploy 部署 -> release
```

也就是说，`main` 更像“上游镜像”，`release` 才是实际产品线。

日常同步上游：

```bash
git checkout release
git pull origin release
git merge origin/main
git push origin release
```

开发自己的功能：

```bash
git checkout release
git pull origin release
git checkout -b feature/my-feature
# 开发完成后
git checkout release
git merge feature/my-feature
git push origin release
```

Dokploy 部署分支应设置为：

```text
release
```

这样做的好处：

```text
main 坏了不影响生产
upstream 更新先进入 main
release 可控吸收更新
feature 不污染 main
```

## 常用验证

后端常用命令：

```bash
cd backend
go test ./...
go test -tags=unit ./...
go test -tags=integration ./...
golangci-lint run ./...
go generate ./ent
go generate ./cmd/server
```

前端常用命令：

```bash
cd frontend
pnpm install
pnpm run lint:check
pnpm run typecheck
pnpm run test:run
pnpm run build
```

根目录常用命令：

```bash
make build
make test
make test-backend
make test-frontend
make test-frontend-critical
```

验证时按变更范围选择最小必要命令；涉及跨模块、接口合约、计费、支付、调度、鉴权或数据迁移时，应扩大测试范围。

