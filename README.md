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
web/dist/               前端构建产物（已纳入版本控制，编译时嵌入二进制）
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
- `GET /api/tasks/import/template`：下载只含字段名的空白任务导入模板
- `POST /api/tasks/:id/import/preview`：上传 `.xlsx` 任务表并预览逐行校验与查重结果
- `POST /api/tasks/:id/import`：将预览通过的 Excel 行批量创建为当前任务的子任务
- `GET/PUT/DELETE /api/tasks/:id`：查看、编辑和永久删除任务子树
- `POST /api/tasks/:id/complete`：完成任务
- `POST/DELETE /api/tasks/:id/schedule`：提上日程或移出日程
- `POST /api/tasks/:id/move`：移动任务并重新校验层级和期限规则
- `POST/GET/DELETE /api/tasks/:id/attachments`：上传、下载和删除本地附件（单个文件最大 25 MB）
- `GET/POST /api/tags`：读取和创建标签

日历总览会实时聚合任务、豆包日程和个人计划，主要接口包括：

- `GET /api/calendar`：按 `start`、`end`、`sources` 和 `include_completed` 查询日历事件与范围汇总

豆包的食谱功能已接入本地 SQLite，主要接口包括：

- `GET /api/recipes`：读取当前周、今天和全部食物安排
- `PUT /api/recipes/order`：保存食物行和星期列的显示顺序
- `POST /api/recipes/foods`：新增食物并设置每周日期
- `PUT/DELETE /api/recipes/foods/:id`：编辑或删除食物及其周安排

豆包日程支持按间隔天数循环提醒，主要接口包括：

- `GET /api/schedules`：读取今天提醒、逾期状态和下一次提醒日期
- `POST /api/schedules`：设置日程名称、开始日期和间隔天数
- `GET/PUT/DELETE /api/schedules/:id`：查看、编辑或删除周期日程
- `POST /api/schedules/:id/complete`：完成日程，并从完成当天重新计算下一次提醒

豆包相册用于保存豆包的照片和视频，媒体文件存放在本地数据目录，主要接口包括：

- `GET /api/doubao/album`：读取相册记录和照片、视频统计
- `POST /api/doubao/album`：上传照片或视频，单个文件最大 100 MB，可同时提交备注
- `GET /api/doubao/album/:id/media`：读取相册媒体内容
- `GET/PUT/DELETE /api/doubao/album/:id`：查看、编辑备注或删除相册记录

个人计划支持管理需要周期打卡的事情，主要接口包括：

- `GET /api/plans`：读取今天提醒、逾期状态和下一次计划日期
- `POST /api/plans`：设置计划名称、开始日期和间隔天数
- `GET/PUT/DELETE /api/plans/:id`：查看、编辑或删除周期计划
- `POST /api/plans/:id/complete`：完成计划，并从完成当天重新计算下一次提醒

娱乐模块用于管理小说、电视剧和电影，主要接口包括：

- `GET /api/entertainment`：读取作品列表和汇总统计，支持 `type`、`status`、`search`、`rating`、`favorite` 和 `sort` 筛选
- `POST /api/entertainment`：新增作品，并记录类型、状态、进度、评分、标签和个人评价
- `GET/PUT/DELETE /api/entertainment/:id`：查看、编辑或删除作品

备忘录模块用于记录可长期整理的想法，支持富文本、归档和恢复，主要接口包括：

- `GET /api/memos`：读取备忘录列表和统计，支持 `archived`、`search` 筛选
- `POST /api/memos`：创建备忘录，内容以经过白名单清洗的 HTML 保存
- `GET/PUT/DELETE /api/memos/:id`：查看、编辑或删除备忘录
- `POST /api/memos/:id/archive`：归档备忘录
- `POST /api/memos/:id/restore`：恢复已归档备忘录

## 构建

构建前端并生成当前平台二进制：

```bash
make build
```

`web/dist/` 中的 HTML、JavaScript 和 CSS 会通过 `go:embed` 编译进程序，因此完整构建产物需要保留在源码中。`make build` 会先重新构建前端；`go test ./...` 还会检查入口 HTML 引用的资源是否都存在。

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
└── uploads/        用户上传文件（含任务附件和豆包相册）
```

启动时如果检测到待执行的数据库迁移，程序会先用 SQLite 一致性快照创建 `backups/doubao-migration-*.db`，再逐个以事务执行迁移。迁移成功后会在 `schema_migrations` 中记录文件名和 SHA-256；已执行的迁移如果文件名或内容发生变化，程序会拒绝启动，避免把已发布的迁移直接改写成不可预测的状态。

迁移失败时当前迁移事务会回滚，启动也会停止；修复新版本代码后重新启动即可重试。备份文件是迁移前的数据库副本，恢复前应先停止程序，并将需要恢复的备份复制为 `doubao.db`（同时清理同目录下旧的 `doubao.db-wal` 和 `doubao.db-shm` 文件）。
