# kz

[![Tests](https://github.com/kzzan/kz/actions/workflows/test.yml/badge.svg)](https://github.com/kzzan/kz/actions/workflows/test.yml)
[![Lint](https://github.com/kzzan/kz/actions/workflows/lint.yml/badge.svg)](https://github.com/kzzan/kz/actions/workflows/lint.yml)
[![Release](https://github.com/kzzan/kz/actions/workflows/release.yml/badge.svg)](https://github.com/kzzan/kz/actions/workflows/release.yml)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

> 面向脚本和组合使用的 Go 脚手架工具：`init` 负责初始化项目，`generate` 负责生成单一组件或组件集合。

---

## 安装

```bash
go install github.com/kzzan/kz@latest
```

确保 `$GOPATH/bin` 已加入 `PATH`：

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

---

## 快速开始

```bash
# 1. 创建新项目目录并初始化
kz init myapp
cd myapp

# 2. 安装依赖
go mod tidy

# 3. 一键生成完整四层组件
kz generate all order
```

---

## 命令一览

### `kz init [directory]` — 初始化项目

```bash
# 在新目录中初始化
kz init myapp

# 在当前目录中初始化
kz init .
```

行为说明：

- 只负责创建项目骨架，不承担组件生成
- 默认要求目标目录为空；如需覆盖现有目录，显式传 `--force`
- 不传目录时，默认初始化当前目录

生成内容包括：

- 完整目录结构（handler / service / repository / model / cron / consumer / middleware）
- 基于 [samber/do v2](https://github.com/samber/do/v2) 的依赖注入入口
- PostgreSQL、Redis、消息队列相关基础包
- `user` 示例组件
- `Makefile`、`docker-compose.yml`、`.env.example`

---

### `kz generate all [name]` — 生成完整业务组件

```bash
kz generate all order
```

`generate` 子命令会自动向上查找项目根目录，所以在项目任意子目录执行都可以。

一条命令完成 8 个步骤：

| 步骤 | 生成文件 |
|------|---------|
| 1 | `internal/models/order.go` |
| 2 | `internal/repository/order.go` |
| 3 | `internal/service/order.go` |
| 4 | `internal/handler/order.go` |
| 5 | 注册到 `internal/repository/package.go` |
| 6 | 注册到 `internal/service/package.go` |
| 7 | 注册到 `internal/handler/package.go` |
| 8 | 追加路由到 `internal/server/routes.go` + `server.go` |

---

### `kz generate handler [name]` — 生成 Handler

```bash
kz generate handler order
```

- 生成 `internal/handler/order.go`
- 自动注册到 `internal/handler/package.go`
- 自动追加路由组到 `internal/server/routes.go`
- 自动追加字段和 `MustInvoke` 到 `internal/server/server.go`

---

### `kz generate service [name]` — 生成 Service

```bash
kz generate service order
```

- 生成 `internal/service/order.go`
- 自动注册到 `internal/service/package.go`

---

### `kz generate repo [name]` — 生成 Repository

```bash
kz generate repo order
```

- 生成 `internal/repository/order.go`
- 自动注册到 `internal/repository/package.go`

---

### `kz generate model [name]` — 生成 Model

```bash
kz generate model order
```

- 生成 `internal/models/order.go`
- 含 GORM 基础字段、`TableName()`、Response 转换方法

---

### `kz generate cron [name]` — 生成定时任务

```bash
kz generate cron cleanExpired
```

- 生成 `internal/cron/clean_expired.go`（默认 `@every 1m`）
- 自动创建或更新 `internal/cron/package.go`，注册 Job 到 Scheduler
- 自动注册 `cron.Package` 到 `internal/server/package.go`

启动调度器：
```go
scheduler := do.MustInvoke[*cron.Scheduler](injector)
scheduler.Start()
```

---

### `kz generate consumer [name]` — 生成消息队列消费者

```bash
kz generate consumer orderPaid
```

- 生成 `internal/consumer/order_paid.go`（含 Topic / Start / handle 方法）
- 自动创建或更新 `internal/consumer/package.go`，注册到 Manager
- 自动注册 `consumer.Package` 到 `internal/server/package.go`

启动消费者：
```go
manager := do.MustInvoke[*consumer.Manager](injector)
manager.Start(ctx)
```

---

### `kz generate middleware [name]` — 生成中间件

```bash
kz generate middleware rateLimit
```

- 生成 `internal/middleware/rate_limit.go` 空模板
- 手动在 `server.go` 中注册：
  ```go
  engine.Use(middleware.RateLimit(logger))
  ```

---

### 兼容别名

旧用法 `kz new ...` 仍可用，但建议迁移到新的命令语义：

- `kz new myapp` -> `kz init myapp`
- `kz new all order` -> `kz generate all order`
- `kz new handler order` -> `kz generate handler order`

---

## 项目结构

```
myapp/
├── main.go
├── Makefile
├── docker-compose.yml
├── .env.example
├── go.mod
└── internal/
    ├── handler/
    │   ├── package.go       ← 依赖注入注册
    │   └── user.go
    ├── service/
    │   ├── package.go
    │   └── user.go
    ├── repository/
    │   ├── package.go
    │   └── user.go
    ├── models/
    │   └── user.go
    ├── server/
    │   ├── server.go        ← HTTP 服务器
    │   ├── routes.go        ← 路由注册
    │   └── package.go       ← server 依赖注入
    ├── cron/
    │   └── package.go       ← 定时任务调度器
    ├── consumer/
    │   └── package.go       ← 消息队列 Manager
    └── middleware/
```

---

## 命名规则

kz 支持任意大小写风格输入，自动转换：

| 输入 | 生成文件 | 结构体名 |
|------|---------|---------|
| `order` | `order.go` | `Order` |
| `orderItem` | `order_item.go` | `OrderItem` |
| `OrderItem` | `order_item.go` | `OrderItem` |
| `order_item` | `order_item.go` | `OrderItem` |

---

## 版本管理

每次推送到 `main` 分支，GitHub Actions 自动：

1. 运行测试 & lint
2. 递增版本号并打 tag
3. 发布 GitHub Release

| commit message | 版本变化 |
|---------------|---------|
| 普通提交 | `v1.0.3` → `v1.0.4` |
| 含 `[minor]` | `v1.0.3` → `v1.1.0` |
| 含 `[major]` | `v1.0.3` → `v2.0.0` |

---

## License

MIT
