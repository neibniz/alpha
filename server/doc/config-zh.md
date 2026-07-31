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
| `Administrator.Password` | `ALPHA_ADMIN_PASSWORD` |

每个变量都支持对应的 `_FILE` 形式，例如：

```bash
ALPHA_POSTGRES_PASSWORD_FILE=/run/secrets/postgres-password
ALPHA_ADMIN_PASSWORD_FILE=/run/secrets/admin-password
```

同一密钥不能同时设置直接变量和 `_FILE` 变量；密钥文件不存在或为空时服务会拒绝启动。
为兼容旧部署，YAML 中的敏感字段仍可读取，但不建议继续使用。

## 部署注意事项

- 配置文件和密钥文件权限应为 `0600`。
- JWT 和节点密钥至少使用 32 个随机字符。
- PostgreSQL、Redis 和 ClickHouse 密码至少使用 20 个随机字符。
- 不要在日志、Compose 输出或版本控制中记录密钥内容。
- Ansible 部署会生成精简的 `alpha.yaml`，并通过 `_FILE` 变量挂载密钥。
