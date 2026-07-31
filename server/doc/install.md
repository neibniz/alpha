### Installation Instructions

#### Prerequisites
- PostgreSQL 12+
- Redis 6.0+ (recommended: 7.0)

#### Binary Installation

1. Determine your system architecture and download the corresponding binary file.

Download URL: `https://github.com/neibniz/alpha/releases`

Example setup: OS: Linux amd64, User: root, Current directory: `/root`

- Download the binary file:

```shell
$ wget https://github.com/neibniz/alpha/releases/download/server%2Fv1.0.0/alpha-server-linux-amd64.tar.gz
```

- Extract the binary file:

```shell
$ tar -zxvf alpha-server-linux-amd64.tar.gz
```

- Navigate to the extracted directory:

```shell
$ cd alpha-server-linux-amd64
```

- Grant execution permissions to the binary:

```shell
$ chmod +x alpha
```

- Create a systemd service file:

```shell
$ cat > /etc/systemd/system/alpha.service <<EOF
[Unit]
Description=ALPHA Server
After=network.target

[Service]
ExecStart=/root/alpha-server-linux-amd64/alpha
Restart=always
User=root
WorkingDirectory=/root/alpha-server-linux-amd64

[Install]
WantedBy=multi-user.target
EOF
```

- Reload the systemd service configuration:

```shell
$ systemctl daemon-reload
```
- Start the service:

```shell
$ systemctl start alpha
```

#### Additional Notes

1. Installation Path: The binary files will be extracted to /root/alpha-server-linux-amd64.

2. systemd Service:
   - Service Name: alpha
   
   - Service Configuration File: /etc/systemd/system/alpha.service
   
   - Service Commands:
   
   - Start: systemctl start alpha
   
   - Stop: systemctl stop alpha
   
   - Restart: systemctl restart alpha
   
   - Status: systemctl status alpha
   
   - Enable on Boot: systemctl enable alpha

3. Enable Auto-start: Use the following command to enable the service on boot:
    ```shell
    $ systemctl enable alpha
    ```
4. Service Logs: By default, logs are output to `/root/alpha-server-linux-amd64/alpha.log`.

5. You can view service logs using: `journalctl -u alpha -f`
6. If the configuration file is missing or empty, the service will start with default settings. The configuration file path is `./etc/alpha.yaml`. Access `http://<server_address>:8080/init` to **initialize the system configuration**.

#### NGINX Reverse Proxy Configuration

Below is an example configuration to proxy the alpha service to your API domain:

```nginx
server {
    listen 80;
    server_name alpha.dev;

    location / {
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        # Overwrite caller-supplied forwarding metadata.
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header REMOTE-HOST $remote_addr;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_http_version 1.1;
        
        add_header X-Cache $upstream_cache_status;
        
        # Set Nginx Cache
        set $static_file_cache 0;
        if ($uri ~* "\.(gif|png|jpg|css|js|woff|woff2)$") {
            set $static_file_cache 1;
            expires 1m;
        }
        if ($static_file_cache = 0) {
            add_header Cache-Control no-cache;
        }
    }
}
```

When Cloudflare is the public entry, it supplies `CF-Connecting-IP` and NGINX
passes it through to Alpha. Keep the origin behind Cloudflare and set
`HTTP.TrustedProxies` to the NGINX address that directly connects to Alpha,
normally `127.0.0.1` or `::1`.
