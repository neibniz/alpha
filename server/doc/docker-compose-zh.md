# 使用 Docker Compose 部署 Alpha Server

本文档提供一个面向 Linux 单机生产环境的基线：Alpha Server、PostgreSQL、Redis 和
单节点 ClickHouse 由同一个 Compose 项目管理，外部 TLS 由宿主机上的 Caddy、NGINX
或其他反向代理终止。

仓库示例文件：

- `server/docker-compose.yml`：完整服务栈。
- `server/etc/alpha.compose.example.yaml`：不含敏感值的 Server 配置。
- `server/.env.compose.example`：镜像、端口和资源限制示例。

复杂拓扑、多主机部署和自动化密钥管理应使用
[`deploy/README.md`](../../deploy/README.md) 中的 Ansible 方案。

## 部署结构

```text
Internet
  -> Caddy / NGINX :443
  -> Alpha Server 127.0.0.1:8080
  -> PostgreSQL 127.0.0.1:5432
  -> Redis      127.0.0.1:6379
  -> ClickHouse 127.0.0.1:9000
```

所有数据服务默认只发布到宿主机回环地址。Server 使用 Linux host network 并同样只
监听回环地址，因此不能绕过反向代理直接从公网访问。

## 前置条件

- Linux x86-64 或 ARM64。
- Docker Engine 和 Docker Compose v2 插件，使用 `docker compose` 命令。
- OpenSSL，用于生成随机密钥。
- 一个指向宿主机的 API 域名，以及已部署的 User/Admin 前端域名。

如果 GHCR 包不是公开的，先登录：

```bash
docker login ghcr.io
```

## 1. 准备配置

从仓库的 `server` 目录执行：

```bash
cp .env.compose.example .env
cp etc/alpha.compose.example.yaml etc/alpha.compose.yaml
install -d -m 0700 secrets
mkdir -p data/postgres data/redis data/clickhouse data/clickhouse-log
```

编辑 `.env`：

- 将 `ALPHA_SERVER_IMAGE` 固定到发布版本或镜像摘要；`latest` 仅适合快速验证。
- 如果默认端口已占用，修改相应的 `*_BIND`。

编辑 `etc/alpha.compose.yaml`：

- 将 `HTTP.AllowedOrigins` 改为真实 User/Admin HTTPS Origin，必须包含协议且不能带路径。
- 将 `Administrator.Email` 改为首次管理员邮箱。
- 如果修改了 PostgreSQL、Redis 或 ClickHouse 的宿主机端口，同时修改对应的 `Addr` 或
  `Host`。
- Server 端口冲突时修改 `Port`；反向代理上游地址也必须同步修改。

`HTTP.TrustedProxies` 只填写直接连接 Server 的代理。示例中反向代理运行在同一宿主机，
因此只信任 `127.0.0.1` 和 `::1`。不要填写 `0.0.0.0/0` 或 `::/0`。

## 2. 生成密钥

以下命令不会把密钥写入 Compose 或 YAML：

```bash
umask 077
openssl rand -hex 32 > secrets/jwt-access-secret
openssl rand -base64 32 > secrets/postgres-password
openssl rand -base64 32 > secrets/redis-password
openssl rand -base64 32 > secrets/clickhouse-password
openssl rand -hex 32 > secrets/node-secret
openssl rand -hex 32 > secrets/node-deploy-secret
openssl rand -hex 32 > secrets/forwarder-bootstrap-secret
openssl rand -base64 24 > secrets/admin-password

encryption_key="$(openssl rand -base64 32)"
index_key="$(openssl rand -base64 32)"
printf '{"active_key_id":"v1","encryption_keys":{"v1":"%s"},"index_key":"%s"}\n' \
  "$encryption_key" "$index_key" > secrets/privacy-keyring.json
unset encryption_key index_key
```

此时先将 `secrets/admin-password` 的值保存到密码管理器，再以 root 执行：

```bash
chown 65532:65532 secrets/*
chmod 0400 secrets/*
```

Compose 以 UID/GID `65532:65532` 的非 root 用户运行 Server，并将这些文件作为只读
secret 挂载。`0400` 同时满足 Server 的隐私 keyring 安全检查；PostgreSQL、Redis 和
ClickHouse 的 root 入口进程仍可读取各自密码并在启动时降权。不要改变文件所有者或
权限，也不要提交 `secrets` 目录。

管理员密码只在数据库没有任何用户时用于创建第一个账号；数据库已有用户后，修改此
文件不会重置管理员密码。

隐私 keyring 必须长期备份。丢失当前或历史加密密钥后，已加密的用户隐私数据无法恢复。

## 3. 校验并启动

```bash
docker compose config --quiet
docker compose pull
docker compose up -d
docker compose ps
```

Server 会等待 PostgreSQL、Redis 和 ClickHouse 健康后启动。首次启动会自动运行当前版本
的 PostgreSQL 与 ClickHouse 迁移，并在 PostgreSQL 没有用户时创建管理员。迁移创建的
表数量随版本和功能变化，不应预期固定数量。

检查日志和心跳：

```bash
docker compose logs --tail=200 server
curl -fsS http://127.0.0.1:8080/v1/common/heartbeat
```

心跳返回 HTTP 200 才表示 Server 已可接收请求。依赖状态可单独检查：

```bash
docker compose exec postgres pg_isready -U alpha -d alpha
docker compose exec redis sh -ec 'redis-cli --no-auth-warning -a "$(cat /run/secrets/redis-password)" ping'
docker compose exec clickhouse sh -ec 'clickhouse-client --user alpha --password "$(cat /run/secrets/clickhouse-password)" --query "SELECT 1"'
```

## 4. 配置 HTTPS 反向代理

Caddy 最小示例：

```caddyfile
api.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

NGINX 或其他代理必须覆盖而不是信任客户端提交的 `X-Forwarded-For`、`X-Real-IP` 等来源
头。Server 会根据 `HTTP.TrustedProxies` 解析登录、注册、OAuth、订阅访问和审计日志的
客户端 IP。

部署后从外部确认：

```bash
curl -fsS https://api.example.com/v1/common/heartbeat
```

## 5. 更新与回滚

更新前先备份 PostgreSQL、配置和密钥，然后修改 `.env` 中的固定镜像版本：

```bash
docker compose pull server
docker compose up -d --no-deps --force-recreate server
curl -fsS http://127.0.0.1:8080/v1/common/heartbeat
```

Server 数据库迁移是前向执行的。回滚到旧镜像前必须确认旧版本兼容当前数据库结构；不能
确认时，应恢复升级前的 PostgreSQL 备份，而不是只替换镜像。

## 6. 备份

至少备份以下内容：

- PostgreSQL：订单、用户、订阅和全部核心业务数据。
- `secrets/`：特别是隐私 keyring、JWT 和节点密钥。
- `etc/alpha.compose.yaml` 与 `.env`。
- ClickHouse 数据：访问日志、原始流量明细和转发连接明细。
- Redis AOF：缓存、会话和尚未处理的异步任务；它不能替代 PostgreSQL 备份。

PostgreSQL 逻辑备份示例：

```bash
install -d -m 0700 backup
docker compose exec -T postgres pg_dump -U alpha -d alpha -Fc \
  > "backup/alpha-$(date +%Y%m%d-%H%M%S).dump"
```

ClickHouse 和 Redis 建议使用文件系统快照或平台备份工具，并定期在隔离环境验证恢复流程。

## 7. 使用外部数据服务

使用托管或独立 PostgreSQL、Redis、ClickHouse 时：

1. 从 Compose 中移除对应服务和 Server 的 `depends_on` 条目。
2. 在 `alpha.compose.yaml` 中配置可从宿主机访问的地址、用户名、数据库和 TLS 参数。
3. 保留 Server 的 `ALPHA_*_PASSWORD_FILE`，在相应 secret 文件中填写外部服务密码。
4. Redis 必须指向可写主节点；Server 会使用 Redis DB 0 和 DB 5。
5. 不要为了远程连接把数据库端口公开到整个互联网，应使用私网或严格防火墙规则。

## 常见问题

**端口已占用**

先使用 `ss -lntp` 查看占用。修改 `.env` 中的数据服务绑定后，也要修改
`alpha.compose.yaml` 中的连接端口。Server 端口只在 YAML 中配置。

**Server 不断重启**

查看 `docker compose logs server`。常见原因包括密钥文件为空、JWT/节点密钥不足 32
字节、管理员密码少于 12 个字符、数据服务密码不一致或隐私 keyring 格式错误。

**前端出现 CORS 错误**

`AllowedOrigins` 必须与浏览器地址栏中的 Origin 完全一致，包括 `https` 和端口，但不能
包含路径或末尾斜杠。修改配置后执行：

```bash
docker compose up -d --no-deps --force-recreate server
```

**订阅链接使用了错误域名**

在 Admin 的订阅配置中填写面向用户的订阅域名。该值只填写域名，不带协议、端口、路径
或通配符。
