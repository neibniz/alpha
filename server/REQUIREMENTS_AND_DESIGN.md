# Alpha Server 需求与设计文档

## 一、项目概述

Alpha Server 是 Alpha 开源代理面板的后端服务组件，基于 Go 语言开发。Alpha 是一个纯净、专业、完美的开源代理面板工具，旨在为学习和实际使用提供理想的代理服务管理解决方案。

### 1.1 项目定位

- 提供代理服务的用户管理、订阅管理、节点管理、支付集成、订单管理等核心后端能力
- 为 Alpha Web（前端）、Alpha User Web（用户端）、Alpha Admin Web（管理端）提供 RESTful API
- 支持多协议代理服务（Shadowsocks、V2Ray、Trojan 等）
- 坚持隐私优先原则，不收集用户日志

### 1.2 技术栈

| 层级 | 技术选型 |
|------|---------|
| 语言 | Go 1.23+ |
| Web 框架 | Gin |
| ORM | GORM（MySQL 驱动） |
| 数据库 | MySQL |
| 缓存/Session | Redis |
| 任务队列 | Asynq（基于 Redis） |
| 定时调度 | Asynq Scheduler |
| API 文档 | Swagger / go-zero goctl |
| 链路追踪 | OpenTelemetry + Jaeger |
| 日志 | Zap |
| CLI | Cobra |
| 认证 | JWT |
| 构建 | Makefile + Docker + GoReleaser |

---

## 二、功能需求

### 2.1 系统模块总览

```
Alpha Server
├── 认证授权模块 (Auth)
├── 用户管理模块 (User)
├── 订阅管理模块 (Subscribe)
├── 节点管理模块 (Server/Node)
├── 订单管理模块 (Order)
├── 支付管理模块 (Payment)
├── 工单系统模块 (Ticket)
├── 公告管理模块 (Announcement)
├── 文档管理模块 (Document)
├── 广告管理模块 (Ads)
├── 优惠券模块 (Coupon)
├── 邮件模块 (Email)
├── 系统配置模块 (System)
├── 日志审计模块 (Log)
├── 控制台模块 (Console)
├── 工具箱模块 (Tool)
├── 订阅适配器模块 (Adapter)
├── 异步任务队列 (Queue)
└── 定时任务调度 (Scheduler)
```

### 2.2 功能需求详细说明

#### 2.2.1 认证授权模块 (Auth)

| 功能 | 描述 |
|------|------|
| 用户注册 | 支持邮箱注册、GitHub OAuth第三方注册 |
| 用户登录 | 支持邮箱密码登录、GitHub OAuth登录 |
| 密码重置 | 支持邮箱验证码重置 |
| JWT令牌 | 基于 JWT 的无状态认证，AccessToken 默认7天有效期 |
| Session管理 | 基于 Redis 的用户会话管理 |
| 权限控制 | 区分普通用户(Public)和管理员(Admin)角色 |

#### 2.2.2 用户管理模块 (User)

| 功能 | 描述 |
|------|------|
| 用户CRUD | 管理员创建/查看/编辑/删除用户 |
| 批量操作 | 批量删除用户 |
| 个人信息 | 用户查看/修改个人信息（密码、通知设置、绑定方式） |
| 认证方式管理 | 绑定/解绑邮箱、GitHub OAuth |
| 订阅管理 | 管理员为用户分配/修改/删除订阅 |
| 设备管理 | 查看/踢下线用户设备 |
| 邀请/推广 | 查询推广用户列表及数量、佣金日志、佣金提现 |
| 余额管理 | 查看余额变动日志 |
| OAuth绑定 | GitHub OAuth绑定与解绑 |

#### 2.2.3 订阅管理模块 (Subscribe)

| 功能 | 描述 |
|------|------|
| 订阅计划CRUD | 创建/编辑/删除/排序订阅计划 |
| 订阅组管理 | 创建/编辑/删除订阅组，支持批量操作 |
| 订阅配置 | 设置订阅路径、泛域名、UserAgent限制 |
| 订阅令牌 | 重置单个/全部订阅令牌 |
| 订阅输出 | 根据客户端协议输出对应的代理配置（支持多格式） |
| 流量管理 | 流量统计、流量重置 |
| 用户订阅 | 用户查看可用订阅、订阅节点信息 |

#### 2.2.4 节点管理模块 (Server/Node)

| 功能 | 描述 |
|------|------|
| 服务器管理 | 创建/编辑/删除/排序服务器 |
| 节点管理 | 创建/编辑/删除/排序节点、启用/停用节点 |
| Xray 模板绑定 | 将 inbound/outbound/dns/routing 模板绑定到服务器 |
| 节点标签 | 查询所有节点标签 |
| 节点配置拉取 | xray-agent 拉取服务器最终 Xray 配置 |
| 用户与流量同步 | xray-agent 拉取有效用户、上报流量数据和服务器状态 |
| 节点密钥验证 | 基于 NodeSecret 的节点身份验证 |

#### 2.2.5 订单管理模块 (Order)

| 功能 | 描述 |
|------|------|
| 订单创建 | 管理员创建订单、用户预创建订单 |
| 订单查询 | 订单详情、订单列表查询 |
| 订单状态 | 更新订单状态、关闭订单 |
| 购买场景 | 新购订阅、续费订阅、重置流量、余额充值 |
| 门户购买 | 公开门户订阅购买流程 |

#### 2.2.6 支付管理模块 (Payment)

| 功能 | 描述 |
|------|------|
| 支付方式管理 | 创建/编辑/删除支付方式 |
| 支付平台 | 支持支付宝(Alipay)、Stripe等多种支付网关 |
| 可用支付方式 | 查询用户可用的支付方式列表 |

#### 2.2.7 工单系统模块 (Ticket)

| 功能 | 描述 |
|------|------|
| 工单CRUD | 用户创建工单、管理员查看工单列表 |
| 工单跟进 | 创建工单跟进/回复 |
| 状态管理 | 更新工单状态 |
| 工单统计 | 控制台查询待回复工单数量 |

#### 2.2.8 公告管理模块 (Announcement)

| 功能 | 描述 |
|------|------|
| 公告CRUD | 管理员创建/编辑/删除公告 |
| 公告展示 | 用户端公告列表查询 |

#### 2.2.9 文档管理模块 (Document)

| 功能 | 描述 |
|------|------|
| 文档CRUD | 创建/编辑/删除文档、批量删除 |
| 文档展示 | 文档详情、文档列表查询 |

#### 2.2.10 广告管理模块 (Ads)

| 功能 | 描述 |
|------|------|
| 广告CRUD | 创建/编辑/删除/查询广告 |

#### 2.2.11 优惠券模块 (Coupon)

| 功能 | 描述 |
|------|------|
| 优惠券CRUD | 创建/编辑/删除/批量删除/查询优惠券 |

#### 2.2.12 邮件模块

| 功能 | 描述 |
|------|------|
| 邮件发送 | 支持SMTP邮件发送，可配置多平台（支持验证码、通知等模板） |
| 验证码 | 邮件验证码发送与校验，可配置过期时间、频率限制 |
| 认证方式配置 | 管理员配置邮箱与GitHub OAuth认证方式 |
| 测试发送 | 管理员测试邮件发送功能 |

#### 2.2.13 系统配置模块 (System)

| 功能 | 描述 |
|------|------|
| 站点配置 | 站点名称/描述/Logo/关键词/自定义HTML/自定义数据 |
| 节点配置 | 节点密钥/拉取间隔/推送间隔/流量上报阈值/IP策略/DNS/出口规则 |
| 订阅配置 | 订阅路径/泛域名/UA限制 |
| 注册配置 | 注册开关/试用设置/IP注册限制 |
| 邀请配置 | 强制邀请/推荐佣金比例/仅首购 |
| 验证配置 | Turnstile人机验证/HCaptcha |
| 验证码配置 | 过期时间/频率限制/发送间隔 |
| 货币配置 | 货币单位/符号/汇率API密钥 |
| Telegram配置 | Bot Token/WebHook/通知开关 |
| 隐私政策 | 隐私政策内容配置 |
| 服务条款 | 服务条款内容配置 |
| 节点倍率 | 设置/预览节点流量倍率 |

#### 2.2.14 日志审计模块 (Log)

| 功能 | 描述 |
|------|------|
| 余额日志 | 按条件筛选余额变动日志 |
| 佣金日志 | 按条件筛选佣金日志 |
| 邮件日志 | 按条件筛选邮件发送日志 |
| 赠金日志 | 按条件筛选赠金日志 |
| 登录日志 | 按条件筛选登录日志 |
| 消息日志 | 查看消息通知日志 |
| 短信日志 | 按条件筛选短信日志 |
| 注册日志 | 按条件筛选注册日志 |
| 流量日志 | 服务器流量、用户订阅流量、流量详情 |
| 日志清理 | 自动清理配置，默认保留7天 |

#### 2.2.15 控制台模块 (Console)

| 功能 | 描述 |
|------|------|
| 收入统计 | 查询收入统计数据 |
| 服务概览 | 查询服务器总体数据 |
| 用户统计 | 查询用户统计数据 |
| 工单概览 | 查询待回复工单数量 |

#### 2.2.16 工具箱模块 (Tool)

| 功能 | 描述 |
|------|------|
| IP查询 | 查询IP归属地信息 |
| 系统日志 | 获取系统运行日志 |
| 系统重启 | 通过API重启服务 |
| 版本查询 | 获取当前服务版本信息 |

#### 2.2.17 公共接口模块 (Common)

| 功能 | 描述 |
|------|------|
| 广告展示 | 客户端获取广告 |
| 验证码 | 发送/校验邮件和短信验证码 |
| 站点配置 | 获取站点全局配置、隐私政策、服务条款 |
| 站点统计 | 获取站点统计数据 |
| 心跳检测 | 服务健康检查 |
| 客户端信息 | 获取客户端配置信息 |

---

## 三、系统架构设计

### 3.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      Alpha Server                          │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐  │
│  │  HTTP    │  │  Queue   │  │Scheduler │  │  Initialize │  │
│  │  Server  │  │  Server  │  │  Server  │  │   System    │  │
│  │  (Gin)   │  │ (Asynq)  │  │ (Asynq)  │  │  (Config)   │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └──────┬──────┘  │
│       │              │              │               │         │
│  ┌────┴──────────────┴──────────────┴───────────────┴──────┐ │
│  │                  Service Context (SVC)                   │ │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────────────┐│ │
│  │  │  Config  │ │  Redis  │ │  MySQL  │ │   Model Layer   ││ │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────────────┘│ │
│  └──────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 服务启动流程

```
main()
  └── cmd.Execute()
       └── run()
            ├── 1. 检查配置文件是否存在
            ├── 2. 初始化配置（首次运行进入 Web 引导）
            ├── 3. 加载配置到 Config 结构体
            ├── 4. 设置 Gin 运行模式（Debug/Release）
            ├── 5. 初始化日志系统 (Zap)
            ├── 6. 创建 ServiceContext（含 DB/Redis/Clients）
            ├── 7. 创建服务组 (ServiceGroup)
            │    ├── internal.NewService(ctx)    → HTTP Server
            │    ├── queue.NewService(ctx)       → Asynq Consumer
            │    └── scheduler.NewService(ctx)   → Asynq Scheduler
            └── 8. 启动所有服务 & 等待退出信号
```

### 3.3 HTTP Server 初始化流程

```
internal.NewService(ctx)
  └── initServer(svc)
       ├── 1. 执行系统初始化 (数据库迁移、种子数据)
       ├── 2. 创建 Gin Engine
       ├── 3. 配置 RemoteIPHeaders (代理 IP 透传)
       ├── 4. 初始化 Redis Session Store
       ├── 5. 注册中间件
       │    ├── Session 中间件
       │    ├── Trace 中间件 (OpenTelemetry)
       │    ├── Logger 中间件
       │    ├── CORS 中间件
       │    └── Recovery 中间件
       ├── 6. 注册路由
       │    ├── RegisterHandlers       → /v1/admin/*, /v1/auth/*, /v1/common/*, /v1/public/*, /v1/server/*
       │    ├── RegisterSubscribeHandlers  → 订阅配置下发
       │    ├── RegisterTelegramHandlers   → Telegram WebHook
       │    └── RegisterNotifyHandlers     → WebSocket 通知
       └── 7. 启动 HTTP Server (支持 TLS)
```

### 3.4 分层架构

```
┌──────────────────────────────────────────────────────┐
│                   Handler Layer                       │
│  (HTTP 请求处理、参数解析、响应格式化)                    │
├──────────────────────────────────────────────────────┤
│                    Logic Layer                        │
│  (业务逻辑编排、事务管理、权限校验)                        │
├──────────────────────────────────────────────────────┤
│                    Model Layer                        │
│  (数据访问对象、GORM 模型定义、软删除支持)                 │
├──────────────────────────────────────────────────────┤
│                  Middleware Layer                     │
│  (认证、授权、CORS、日志、链路追踪、泛域名)                │
├──────────────────────────────────────────────────────┤
│                  Service Context                      │
│  (Config / Redis / DB / Model Clients 统一持有)       │
└──────────────────────────────────────────────────────┘
```

### 3.5 路由分组设计

| 路由前缀 | 目标用户 | 中间件 | 说明 |
|---------|---------|--------|------|
| `/v1/admin/*` | 管理员 | AuthMiddleware + AdminCheck | 管理后台接口 |
| `/v1/auth/*` | 未认证用户 | 无 | 注册/登录/密码重置 |
| `/v1/auth/oauth/*` | 未认证用户 | 无 | OAuth登录回调 |
| `/v1/common/*` | 所有用户 | 无 | 公共接口 |
| `/v1/public/*` | 已认证用户 | AuthMiddleware | 用户前台接口 |
| `/v1/server/*` | 代理节点 | ServerMiddleware | 节点通信接口 |
| `/v2/server/*` | 代理节点 | SecretKey | Xray 配置和有效用户拉取 |
| `/v1/subscribe/*` | 客户端 | 无（UA/PanDomain限制） | 订阅配置下发 |
| Telegram WebHook | Telegram Bot | Token验证 | Telegram Bot回调 |
| WebSocket通知 | 已连接用户 | - | 实时消息推送 |

### 3.6 中间件体系

| 中间件 | 功能 |
|--------|------|
| AuthMiddleware | JWT Token解析、会话验证、用户身份注入、管理员权限校验 |
| ServerMiddleware | 节点密钥(secret_key)验证 |
| CorsMiddleware | 跨域请求处理 |
| TraceMiddleware | OpenTelemetry 链路追踪 |
| LoggerMiddleware | 请求日志记录 |
| PanDomainMiddleware | 泛域名订阅 Token 校验 |
| NotifyMiddleware | 通知功能辅助 |

---

## 四、数据流设计

### 4.1 用户请求流程

```
Client → Nginx/负载均衡 → Gin Router → Middleware Chain
  → Handler（参数绑定）
    → Logic（业务逻辑）
      → Model（数据持久层）
        → MySQL / Redis
      ← Model
    ← Logic
  ← Handler（响应格式化）
← Client
```

### 4.2 订单支付流程

```
用户发起购买 → PreCreateOrder（预创建订单，计算金额）
  → 选择支付方式 → 调用支付网关（支付宝/Stripe）
    → 支付回调 → ActivateOrder（激活订单，分配订阅）
      → 创建订阅记录 → 分配流量 → 发送通知
```

### 4.3 节点通信流程

```
xray-agent → GET /v2/server/{server_id}/xray-config?secret_key=xxx（拉取最终 Xray 配置）
xray-agent → GET /v2/server/{server_id}/xray-users?secret_key=xxx（拉取有效用户）
xray-agent → POST /v1/server/status（上报服务器状态）
xray-agent → POST /v1/server/push（上报用户流量）
```

### 4.4 订阅下发流程

```
客户端（代理软件）→ GET {SubscribePath}?token=xxx
  → PanDomain 校验（可选）
    → UserAgent 校验（可选）
      → 查询订阅信息 → 解析节点配置 → 适配客户端模板
        → 输出代理配置（Base64/明文）
```

### 4.5 异步任务队列

```
┌──────────────────┐
│  Asynq Client    │ ← HTTP Handler 中创建任务
│  (Redis DB:5)    │
└────────┬─────────┘
         │ Enqueue
┌────────▼─────────┐
│   Redis Queue    │
└────────┬─────────┘
         │ Dequeue
┌────────▼─────────┐
│  Asynq Server    │ ← Queue Service (Concurrency: 20)
│  (Redis DB:5)    │
└────────┬─────────┘
         │ Process
┌────────▼─────────┐
│   Task Handler   │
│  ├─ SendEmail    │
│  ├─ CloseOrder   │
│  ├─ ActivateOrder│
│  ├─ TrafficStats │
│  └─ Subscription │
└──────────────────┘
```

---

## 五、数据库设计概要

### 5.1 核心数据模型

| 模型 | 说明 | 主要字段 |
|------|------|---------|
| User | 用户 | 密码/余额/佣金/是否管理员/邀请人 |
| UserAuthMethod | 用户认证方式 | 用户ID/认证类型(邮箱/GitHub OAuth) |
| Subscribe | 订阅计划 | 名称/描述/周期/价格/流量限制/设备限制/协议/节点关联 |
| SubscribeGroup | 订阅组 | 名称/描述/关联的订阅列表 |
| UserSubscribe | 用户订阅 | 用户ID/订阅ID/开始时间/到期时间/已用流量/令牌/状态 |
| Server | 服务器 | 名称/地址/地区/模板绑定/实时状态/排序 |
| Node | 节点 | 名称/地址/inbound alias/端口/标签/关联服务器 |
| Order | 订单 | 用户ID/金额/支付方式/订单类型/状态/交易号 |
| PaymentMethod | 支付方式 | 名称/平台/配置/状态 |
| Ticket | 工单 | 用户ID/标题/内容/状态/分类 |
| TicketFollow | 工单跟进 | 工单ID/用户ID/内容 |
| Announcement | 公告 | 标题/内容/类型/是否置顶 |
| Document | 文档 | 标题/内容/分类/排序 |
| Coupon | 优惠券 | 名称/类型/面值/使用门槛/有效期 |
| Ads | 广告 | 名称/图片/链接/位置/状态 |
| Client | 客户端 | 名称/UA/UserAgent |
| Log（多张表） | 各类日志 | 余额/佣金/邮件/短信/登录/注册/流量等 |

### 5.2 ORM 特性

- 使用 GORM 作为 ORM 框架
- 支持软删除（`gorm.io/plugin/soft_delete`）
- 自动迁移（`initialize/migrate`）
- 支持 MySQL 驱动

---

## 六、配置管理设计

### 6.1 配置文件结构

配置文件为 YAML 格式（`etc/alpha.yaml`），运行时通过 `--config` 参数指定。

### 6.2 配置项分类

| 配置分类 | 说明 |
|---------|------|
| Server | Host/Port/Debug/TLS |
| JwtAuth | JWT密钥/过期时间 |
| Logger | 日志级别/格式/输出路径 |
| MySQL | 数据库连接配置 |
| Redis | 缓存连接配置 |
| Site | 站点名称/描述/Logo/关键词/CustomHTML/CustomData |
| Node | 节点密钥/拉取间隔/推送间隔/流量上报阈值/IP策略/DNS/出口规则 |
| Email | 邮件平台/模板/通知开关/域名后缀限制 |
| Mobile | 短信平台/白名单 |
| Device | 设备验证开关/安全密钥 |
| Verify | Turnstile人机验证/HCaptcha |
| VerifyCode | 验证码过期时间/频率限制 |
| Register | 注册开关/试用设置/IP限制 |
| Subscribe | 订阅路径/泛域名/UA限制 |
| Invite | 强制邀请/佣金比例/仅首购 |
| Telegram | Bot配置/WebHook |
| Log | 自动清理/保留天数 |
| Currency | 货币单位/符号/汇率API密钥 |
| Administrator | 默认管理员邮箱/密码 |

### 6.3 配置加载流程

1. 检查配置文件是否存在，不存在则进入 Web 初始化引导
2. 支持环境变量注入（`ALPHA_DB`、`ALPHA_REDIS`）
3. 首次运行自动生成 JWT AccessSecret（UUID）
4. 使用 `conf.MustLoad()` 加载 YAML 并映射到 Config 结构体

---

## 七、关键技术设计

### 7.1 适配器模式（Adapter Pattern）

订阅配置采用适配器模式，针对不同客户端（Clash、Shadowrocket、V2Ray 等）生成对应格式的代理配置文件。

```
Adapter
├── ClientTemplate (客户端模板内容)
├── SiteName (站点名称)
├── SubscribeName (订阅名称)
├── OutputFormat (输出格式: base64/plain)
├── UserInfo (用户信息)
├── Params (其他参数)
└── Proxies (代理节点列表)
    └── Proxy (节点代理信息: 协议/地址/端口/加密/传输等)
```

### 7.2 服务组模式（ServiceGroup）

所有服务（HTTP/Queue/Scheduler）通过 ServiceGroup 统一管理生命周期：

```
ServiceGroup
├── Add(Service)    ← 注册服务
├── Start()         ← 并发启动所有服务
└── Stop()          ← 优雅关闭所有服务
```

### 7.3 订阅下发设计

```
请求 → PanDomain校验 → UA校验 → 查询用户订阅
  → 获取关联节点列表 → Adapter 适配客户端模板
    → Proxies 协议转换 → 输出配置（文本/Base64）
      → 响应头 subscription-userinfo（流量信息）
```

### 7.4 设备记录与限额

- 保留设备列表、在线状态和踢下线功能
- 按订阅配置限制同时使用的设备数量
- 设备标识仅用于使用记录和限额，不作为登录认证方式

### 7.5 节点通信安全

- 节点与服务器通过预共享密钥（`NodeSecret`）认证
- 节点定期拉取配置（`NodePullInterval`）
- 节点定期上报状态和流量（`NodePushInterval`）
- 支持流量上报阈值（`TrafficReportThreshold`）

---

## 八、部署架构

### 8.1 推荐部署拓扑

```
                    ┌──────────────┐
                    │   Nginx/CDN  │
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │ Alpha Server│
                    │   (Gin)      │
                    └──┬───┬───┬───┘
                       │   │   │
              ┌────────┘   │   └────────┐
              ▼            ▼            ▼
        ┌─────────┐ ┌──────────┐ ┌──────────┐
        │  MySQL  │ │  Redis   │ │ 节点集群  │
        └─────────┘ └──────────┘ └──────────┘
```

### 8.2 部署方式

| 方式 | 说明 |
|------|------|
| 源码编译 | `make linux-amd64` 等，支持多平台交叉编译 |
| Docker | `docker buildx build` 构建镜像，单容器运行 |
| Docker Compose | 编排 MySQL + Redis + Server 一起运行 |
| Docker Hub | 官方镜像: `alpha/alpha-server:latest` |

### 8.3 支持平台

- Linux: 386, amd64, arm64, armv5-v7, mips, riscv64, loong64
- Windows: 386, amd64, arm64, armv7
- macOS: amd64, arm64
- FreeBSD: amd64, arm64

---

## 九、项目目录结构

```
server/
├── alpha.go              # 程序入口
├── cmd/                   # CLI 命令
│   ├── root.go            # 根命令
│   ├── run.go             # 启动服务命令
│   └── version.go         # 版本命令
├── adapter/               # 订阅适配器（客户端配置生成）
│   ├── adapter.go         # 适配器核心
│   ├── client.go          # 客户端配置结构
│   └── utils.go           # 工具函数
├── initialize/            # 系统初始化
│   ├── init.go            # 初始化入口
│   ├── config.go          # Web 引导初始化
│   ├── migrate/           # 数据库迁移
│   └── *.go               # 各模块初始化（种子数据）
├── internal/              # 内部模块
│   ├── config/            # 配置定义
│   ├── handler/           # HTTP 处理器
│   │   ├── routes.go      # 路由注册
│   │   ├── subscribe.go   # 订阅处理器
│   │   ├── admin/         # 管理端处理器
│   │   ├── auth/          # 认证处理器
│   │   ├── common/        # 公共处理器
│   │   ├── public/        # 用户端处理器
│   │   ├── server/        # 节点通信处理器
│   │   ├── notify/        # 通知处理器
│   │   └── telegram/      # Telegram处理器
│   ├── logic/             # 业务逻辑
│   ├── middleware/        # 中间件
│   ├── model/             # 数据模型（GORM）
│   ├── svc/               # 服务上下文
│   ├── types/             # 类型定义
│   ├── report/            # 模块注册上报
│   └── trace/             # 链路追踪配置
├── pkg/                   # 公共工具库
│   ├── aes/               # AES 加密
│   ├── calculateMonths/   # 月数计算
│   ├── conf/              # 配置加载
│   ├── deduction/         # 流量扣减
│   ├── email/             # 邮件服务
│   ├── fs/                # 文件系统
│   ├── ip/                # IP 处理
│   ├── jwt/               # JWT 工具
│   ├── logger/            # 日志
│   ├── orm/               # ORM 工具
│   ├── proc/              # 进程管理
│   ├── random/            # 随机数
│   ├── result/            # 响应格式化
│   ├── service/           # 服务组
│   ├── threading/         # 并发工具
│   ├── timex/             # 时间工具
│   ├── tool/              # 通用工具集
│   ├── trace/             # 链路追踪
│   └── xerr/              # 错误处理
├── queue/                 # 异步任务队列
│   ├── queue.go           # 队列服务
│   ├── handler/           # 任务处理器注册
│   ├── logic/             # 任务处理逻辑（email/order/traffic/subscription/task）
│   └── types/             # 任务类型定义
├── scheduler/             # 定时任务调度
│   └── scheduler.go       # 定时任务注册（订阅检查/流量统计/汇率更新）
├── etc/                   # 配置文件
├── doc/                   # 文档
└── script/                # 构建脚本
```

---

## 十、非功能性需求

| 需求项 | 描述 |
|--------|------|
| 性能 | 使用 Gin 高性能框架，Asynq 支持并发20处理任务 |
| 安全 | JWT认证 + NodeSecret节点认证 + 设备指纹 + Turnstile人机验证 |
| 可扩展 | 适配器模式支持多客户端，支付平台可插拔扩展 |
| 可观测 | OpenTelemetry 链路追踪 + Zap 结构化日志 |
| 可部署 | 支持多平台交叉编译 + Docker/Docker Compose |
| 隐私 | 不收集用户日志，加密传输，支持 TLS |
| 高可用 | Redis Session 共享，MySQL 数据持久化，服务优雅关闭 |
| 国际化 | 支持中英文文档，货币多币种支持 |
