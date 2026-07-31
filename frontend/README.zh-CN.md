# Alpha 前端

Alpha 前端采用 Bun workspace，包含两个生产应用：

| 路径 | 运行方式 | 用途 |
| --- | --- | --- |
| `apps/admin` | Vite 静态构建 | 管理端 |
| `apps/user-next` | Bun 运行 Next.js standalone | 用户端 |

共享 UI 和 TypeScript 配置位于 `packages/`。

## 本地开发

安装 `package.json` 中声明的 Bun 版本，然后在当前目录执行：

```bash
bun install --frozen-lockfile
bun run dev
bun run check
bun run test
bun run build
```

仅启动一个应用：

```bash
bun --filter alpha-admin-web dev
bun --filter alpha-user-next dev
```

## 发布

```bash
bash scripts/package-release.sh
```

脚本会在 `release/` 中生成 `alpha-admin-web.tar.gz` 和
`alpha-user-next.tar.gz`。`frontend/v*` 标签会触发 GitHub Actions 发布相同产物。

两个应用也可以分别构建为独立镜像：

```bash
docker compose -f compose.yml build
```

生产部署由 [`../deploy`](../deploy/README.md) 管理。
镜像名称、运行时变量、Vercel 和 Cloudflare 部署方式请参阅
[`DEPLOYMENT.md`](DEPLOYMENT.md)。
