# 豆包空间

轻量级、单用户的个人工作台，目标运行在 macOS，也支持在 Ubuntu 上开发和交叉编译。

## 技术栈

- 前端：Vue 3、Vite、Element Plus、Vue Router
- 后端：Go、Gin
- 数据库：SQLite（纯 Go 驱动，避免 CGO 影响跨平台构建）
- 发布：前端静态资源通过 `go:embed` 嵌入 Go 二进制

## 目录结构

```text
cmd/doubao-space/       Go 程序入口
internal/config/        配置和数据目录解析
internal/database/      SQLite 初始化和迁移
internal/httpapi/       HTTP 路由和 API
web/frontend/           Vue 前端源码
web/dist/               前端构建产物（发布时嵌入二进制）
```

## 开发

需要 Go、Node.js 和 npm。启动后端：

```bash
go run ./cmd/doubao-space --dev --addr 127.0.0.1:8081
```

另开一个终端启动前端：

```bash
cd web/frontend
npm ci
npm run dev
```

前端开发服务器会把 `/api` 请求代理到 `http://127.0.0.1:8081`。

也可以使用 Make：

```bash
make dev-backend
make dev-frontend
```

任务模块 API 已接入本地 SQLite，主要接口包括：

- `GET /api/tasks`：任务树，支持 `include_completed`、`search`、`status`、`priority`、`tag` 和 `sort` 查询参数
- `POST /api/tasks`：创建顶层任务或子任务
- `GET/PUT/DELETE /api/tasks/:id`：查看、编辑和永久删除任务子树
- `POST /api/tasks/:id/complete`：完成任务
- `POST /api/tasks/:id/move`：移动任务并重新校验层级和期限规则
- `POST/GET/DELETE /api/tasks/:id/attachments`：上传、下载和删除本地附件（单个文件最大 25 MB）
- `GET/POST /api/tags`：读取和创建标签

豆包日历的食谱功能已接入本地 SQLite，主要接口包括：

- `GET /api/recipes`：读取当前周、今天和全部食物安排
- `POST /api/recipes/foods`：新增食物并设置每周日期
- `PUT/DELETE /api/recipes/foods/:id`：编辑或删除食物及其周安排

豆包日程支持按间隔天数循环提醒，主要接口包括：

- `GET /api/schedules`：读取今天提醒、逾期状态和下一次提醒日期
- `POST /api/schedules`：设置日程名称、开始日期和间隔天数
- `GET/PUT/DELETE /api/schedules/:id`：查看、编辑或删除周期日程
- `POST /api/schedules/:id/complete`：完成日程，并从完成当天重新计算下一次提醒

个人计划支持管理需要周期打卡的事情，主要接口包括：

- `GET /api/plans`：读取今天提醒、逾期状态和下一次计划日期
- `POST /api/plans`：设置计划名称、开始日期和间隔天数
- `GET/PUT/DELETE /api/plans/:id`：查看、编辑或删除周期计划
- `POST /api/plans/:id/complete`：完成计划，并从完成当天重新计算下一次提醒

## 构建

构建前端并生成当前平台二进制：

```bash
make build
```

从 Ubuntu 构建 Apple Silicon 版本：

```bash
make build-darwin-arm64
```

构建产物在 `build/`。正式分发时还需要在 macOS 环境完成 `.app` 打包、签名和公证。

## 数据目录

运行时数据不放在源码目录或 `.app` 内部：

- 开发模式 `--dev`：默认使用项目根目录下的 `data/`
- 生产模式 macOS：默认使用系统用户配置目录下的 `DoubaoSpace/`
- 可通过 `--data-dir` 或 `DOUBAO_DATA_DIR` 覆盖

当前数据目录包含：

```text
data/
├── doubao.db       SQLite 数据库
├── backups/        数据库备份
└── uploads/        用户上传文件
```
