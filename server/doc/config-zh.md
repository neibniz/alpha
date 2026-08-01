# Alpha 启动配置

Alpha 使用 YAML 启动配置，默认路径为 `./etc/alpha.yaml`。可以通过
`alpha run --config /path/to/alpha.yaml` 指定其他文件。

配置加载器会自动填充默认值。生产配置只需声明连接信息和确实需要覆盖的行为，
不应复制全部默认参数。

## 最小配置

```yaml
HTTP:
  AllowedOrigins:
    - "https://user.example.com"
    - "https://admin.example.com"
  TrustedProxies:
    - "127.0.0.1"
    - "::1"

Postgres:
  Addr: "127.0.0.1:5432"
  Username: "alpha"
  Dbname: "alpha"

Redis:
  Host: "127.0.0.1:6379"

Administrator:
  Email: "admin@example.com"
```

默认监听 `0.0.0.0:8080`，运行模式为生产模式，JWT 有效期为 7 天。仅在需要变更时
增加 `Host`、`Port`、`Debug` 或 `JwtAuth.AccessExpire`。

## 客户端 IP 解析

Alpha 会在请求日志和业务处理之前统一解析一次客户端 IP。登录、注册、OAuth、密码
重置、退出、订单审计、订阅访问和转发进程注册均使用同一个解析结果。

- `TrustedProxies` 只能填写会直接连接 Server 的反向代理，禁止使用 `0.0.0.0/0` 或
  `::/0`。
- `X-Forwarded-For` 会从右向左跳过可信代理节点；该链不存在时再使用 `X-Real-IP`。
- 部署在 Cloudflare、Vercel 或其他主机上的 User BFF 会将入口观察到的地址覆盖写入
  `X-Alpha-Client-IP`，不会透传浏览器提交的同名值。
- BFF 头不存在时会读取 Cloudflare 的 `CF-Connecting-IP`。应确保源站只能通过
  Cloudflare 入口访问，避免调用方绕过入口直接提交该请求头。

BFF 传递不需要证书或共享密钥。生产环境应让 Server 监听在反向代理之后的私有地址，
且每层入口必须覆盖来源头。这里得到的是第一层入口观察到的公网地址，NAT 或 VPN 后的
设备真实地址无法由 HTTP 服务获取。

## 可选组件

启用 ClickHouse 访问日志和流量明细存储：

```yaml
ClickHouse:
  Enabled: true
  Addr: "127.0.0.1:9000"
  Database: "alpha_analytics"
  Username: "alpha"
```

启用隐私字段加密：

```yaml
Privacy:
  Enabled: true
  KeyFile: "/run/secrets/alpha_privacy_keyring"
```

容器内日志需要写入挂载目录时，只覆盖以下字段：

```yaml
Logger:
  Mode: file
  Path: /app/log
  Level: info
```

## 密钥注入

生产环境应将敏感值放入环境变量或只读文件，不要写入 `alpha.yaml`：

| 配置项 | 环境变量 |
|---|---|
| `JwtAuth.AccessSecret` | `ALPHA_JWT_ACCESS_SECRET` |
| `Postgres.Password` | `ALPHA_POSTGRES_PASSWORD` |
| `Redis.Pass` | `ALPHA_REDIS_PASSWORD` |
| `ClickHouse.Password` | `ALPHA_CLICKHOUSE_PASSWORD` |
| `Node.NodeSecret` | `ALPHA_NODE_SECRET` |
| `Node.NodeDeploySecret` | `ALPHA_NODE_DEPLOY_SECRET` |
| `Forwarder.BootstrapSecret` | `ALPHA_FORWARDER_BOOTSTRAP_SECRET` |
| `Administrator.Password` | `ALPHA_ADMIN_PASSWORD` |

每个变量都支持对应的 `_FILE` 形式，例如：

```bash
ALPHA_POSTGRES_PASSWORD_FILE=/run/secrets/postgres-password
ALPHA_ADMIN_PASSWORD_FILE=/run/secrets/admin-password
```

同一密钥不能同时设置直接变量和 `_FILE` 变量；密钥文件不存在或为空时服务会拒绝启动。
为兼容旧部署，YAML 中的敏感字段仍可读取，但不建议继续使用。

## 部署注意事项

- 普通配置和密钥挂载应使用 `0600`。Docker Compose 示例将 secret 文件设置为
  `65532:65532`、模式 `0400`，使非 root Server 可读取且不允许组或其他用户访问。
- JWT 和节点密钥至少使用 32 个随机字符。
- PostgreSQL、Redis 和 ClickHouse 密码至少使用 20 个随机字符。
- 不要在日志、Compose 输出或版本控制中记录密钥内容。
- Ansible 部署会生成精简的 `alpha.yaml`，并通过 `_FILE` 变量挂载密钥。
