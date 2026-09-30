# VSITE · 视频浏览网站

兼容 PC 与手机（H5）的视频浏览网站。首页为瀑布流视频浏览与播放，支持点赞、评论、分享；点击博主可查看其全部作品；下拉无限加载，首屏随机展示。配套后台支持博主 / 视频的增删改查与内容管理。

## 技术栈

- 后端：Golang 1.25 + Gin + `database/sql` + go-sql-driver/mysql
- 前端：原生 HTML/CSS/JS（无构建步骤），自适应 PC / 手机
- 数据库：MySQL 8.0

## 目录结构

```
myvsite/
├── main.go                  # 入口
├── internal/
│   ├── config/              # 配置加载（命令行 / 环境变量）
│   ├── model/               # 数据模型、建表、数据访问
│   ├── handler/             # 前台 / 后台接口
│   ├── middleware/          # 后台鉴权中间件
│   └── router/              # 路由注册
├── web/
│   ├── index.html           # 前台（H5）
│   ├── admin.html           # 后台管理页
│   └── static/{css,js}/     # 静态资源
├── scripts/
│   ├── init_db.sql          # 数据库与账号初始化
│   └── test_api.sh          # 接口端到端测试
└── Makefile
```

## 环境准备

### Go 1.25

本项目使用 Go 1.25。若本机默认 `go` 不是 1.25，请使用 SDK 路径：

```bash
export PATH=$HOME/sdk/go1.25.0/go/bin:$PATH
go version   # 应显示 go1.25.0
```

> 已在本项目环境安装到 `/home/gjy/sdk/go1.25.0`，对应的 `go1.25.0` 包装脚本在 `$HOME/go/bin/go1.25.0`。

### 数据库

交付约定：MySQL 地址 `localhost:3306`，数据库 `vsite`，账号 `admin` / 密码 `!QAZ2wsx`。

由具有建库权限的账号执行一次初始化（创建库、创建 `admin` 账号并授权）：

```bash
sudo mysql < scripts/init_db.sql
```

## 运行

```bash
# 1) 初始化数据库（见上）
# 2) 编译并启动（首次启动自动建表 + 写入默认管理员）
make run
# 或自定义参数：
./bin/vsite -addr :8080 -db-host localhost -db-port 3306 -db-user admin -db-pass '!QAZ2wsx' -db-name vsite
```

- 前台首页： http://localhost:8080/
- 后台管理： http://localhost:8080/admin

启动时若库为空，会自动写入默认管理员账号 **ADMIN / 123456**（首次登录强制修改密码）。

可选：写入示例数据（便于体验瀑布流与随机展示）：

```bash
./bin/vsite -seed
```

## 后台说明

- 默认账号：`ADMIN` / `123456`，首次登录必须修改密码（新密码至少 6 位，且不可与原密码相同）。
- 登录后进入后台：
  - **概览**：视频 / 博主 / 播放 / 点赞 / 评论统计。
  - **博主管理**：新增（账号 + 描述 + 头像 + 状态）、编辑、删除。
  - **视频管理**：按博主新增视频（视频链接、封面、描述、状态），支持上传封面 / 视频文件；可编辑、删除、查看评论。
  - **评论管理**：按视频查看并删除评论。

### 主要 API

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/videos` | 视频列表，支持 `cursor`（随机下拉）、`blogger_id`、`keyword`、`client_id` |
| GET | `/api/videos/:id` | 视频详情（含最新评论） |
| POST | `/api/videos/:id/view` | 播放计数 +1 |
| POST | `/api/videos/:id/like` | 点赞 / 取消点赞 `{client_id}` |
| GET/POST | `/api/videos/:id/comments` | 评论列表 / 发表评论 |
| GET | `/api/bloggers` | 博主列表 |
| GET | `/api/bloggers/:id` | 博主详情 |
| POST | `/api/admin/login` | 登录 `{username,password}` |
| POST | `/api/admin/password` | 修改密码（首次登录强制） |
| GET/POST/PUT/DELETE | `/api/admin/bloggers` | 博主管理（需登录） |
| GET/POST/PUT/DELETE | `/api/admin/videos` | 视频管理（需登录） |
| POST | `/api/admin/upload` | 文件上传（需登录） |

> 后台接口需携带请求头 `X-Admin-Token: <token>`，由登录接口返回。

## 测试

启动服务后执行接口端到端测试：

```bash
bash scripts/test_api.sh
```

测试覆盖：登录 → 强制改密 → 重建登录 → 博主增删 → 视频增删 → 点赞 / 评论 → 前台列表 → 页面可访问。

## 配置项

所有配置均可通过命令行参数或环境变量覆盖（环境变量优先级：命令行 > 环境变量 > 默认值）：

| 参数 | 环境变量 | 默认 |
| --- | --- | --- |
| `-addr` | `ADDR` | `:8080` |
| `-db-host` | `DB_HOST` | `localhost` |
| `-db-port` | `DB_PORT` | `3306` |
| `-db-user` | `DB_USER` | `admin` |
| `-db-pass` | `DB_PASS` | `!QAZ2wsx` |
| `-db-name` | `DB_NAME` | `vsite` |
| `-admin-user` | `ADMIN_USER` | `ADMIN` |
| `-admin-pass` | `ADMIN_PASS` | `123456` |
| `-web` | `WEB_DIR` | `./web` |
| `-upload` | `UPLOAD_DIR` | `./web/static/uploads` |
