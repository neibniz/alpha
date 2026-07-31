### 安装说明
#### 前置系统要求
- PostgreSQL 12+
- Redis 6.0+ (推荐使用7.0)

#### 二进制安装
1.确定系统架构，并下载对应的二进制文件

下载地址：`https://github.com/neibniz/alpha/releases`

示例说明：系统：Linux amd64，用户：root，当前目录：/root

- 下载二进制文件

```shell
$ wget https://github.com/neibniz/alpha/releases/download/server%2Fv1.0.0/alpha-server-linux-amd64.tar.gz
```

- 解压二进制文件

```shell
$ tar -zxvf alpha-server-linux-amd64.tar.gz
```

- 进入解压后的目录

```shell
$ cd alpha-server-linux-amd64
```

- 赋予二进制文件执行权限
    
```shell
$ chmod +x alpha-server
```

- 创建 systemd 服务文件

```shell
$ cat > /etc/systemd/system/alpha.service <<EOF
[Unit]
Description=ALPHA Server
After=network.target

[Service]
ExecStart=/root/alpha-server-linux-amd64/alpha-server
Restart=always
User=root
WorkingDirectory=/root/alpha-server-linux-amd64

[Install]
WantedBy=multi-user.target
EOF
```

- 重新加载 systemd 服务
    
```shell
$ systemctl daemon-reload
```
- 启动服务
    
```shell
$ systemctl start alpha
```
##### 其他说明
1. 安装路径：二进制文件将解压到 /root/alpha-server-linux-amd64 目录下
2. systemd 服务：
    - 服务名称：alpha
    - 服务配置文件：/etc/systemd/system/alpha.service
    - 服务启动命令：systemctl start alpha
    - 服务停止命令：systemctl stop alpha
    - 服务重启命令：systemctl restart alpha
    - 服务状态命令：systemctl status alpha
    - 服务开机自启：systemctl enable alpha
3. 设置开机自启可通过以下命令开机自启
    ```shell
    $ systemctl enable alpha
    ```
4. 服务日志：服务日志默认输出到 /root/alpha-server-linux-amd64/alpha.log 文件中
5. 可通过 `journalctl -u alpha -f` 查看服务日志
6. 当配置文件为空或者不存在的情况下，服务会使用默认配置启动，配置文件路径为：`./etc/alpha.yaml`，
请通过`http://服务器地址:8080/init` 初始化系统配置

#### NGINX 反向代理配置

以下是反向代理配置示例，将 `alpha` 服务代理到你的 API 域名下

```nginx
server {
    listen 80;
    server_name alpha.dev;

    location / {
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header REMOTE-HOST $remote_addr;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_http_version 1.1;
        
        add_header X-Cache $upstream_cache_status;
        
       #Set Nginx Cache
       
        set $static_filezbsQiET1 0;
        if ( $uri ~* "\.(gif|png|jpg|css|js|woff|woff2)$" )
        {
            set $static_filezbsQiET1 1;
            expires 1m;
            }
        if ( $static_filezbsQiET1 = 0 )
        {
            add_header Cache-Control no-cache;
        }
    }
}
```
如果使用cloudflare代理服务，需要获取到用户真实访问IP。请在Nginx配置文件中Http段落中加入:

- 需要依赖：**ngx_http_realip_module**模块， 使用nginx -V命令查看nginx是否已经编译该模块，没有的话需要自己编译。


```nginx
    # cloudflare Start
    set_real_ip_from 0.0.0.0/0;
    real_ip_header  X-Forwarded-For;
    real_ip_recursive on;
    # cloudflare END
```
