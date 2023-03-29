# ChatGPT api 服务
## 特性
1. 适配所有能用"自定义OpenAI域名"的客户端或网页
2. 不需要把OpenAI的API key放在客户端或网页上，使得API key不会被盗用
3. 支持用户管理功能，为每个用户分配key
4. 统计每个用户的API调用次数

## 部署
1. 创建配置文件config.yaml，配置以下内容：
```yaml
GinPort: 8080
OpenAIKey: "sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
DBName: "chat.db"
```

2. 创建一个空的数据库文件chat.db:
```bash
touch chat.db
```

3. 运行docker:
```bash
docker run --name=chatapi -d \
  --restart=unless-stopped -p 8080:8080 \
  -v /root/chat/config.yaml:/web/config.yaml \
  -v /root/chat/chat.db:/web/chat.db \
  libli/chat:latest
```
把上面命令中的/root/chat替换为你的配置文件和数据库文件所在的目录。

4. 创建用户：
确保已经安装sqlite3客户端，然后运行：
```bash
sqlite3 chat.db
sqlite> insert into users (username, token) VALUES ('***', '****');
```

5. 如果需要支持https协议，使用nginx反向代理即可。参考如下配置：
```nginx
server {
    listen       80;
    listen       [::]:80;
    server_name  api.exapmle.com;
    return       301 https://$host$request_uri;
}

# Settings https.
server {
    listen       443 ssl http2;
    listen       [::]:443 ssl http2;
    server_name  api.exapmle.com;

    ssl_certificate             "/etc/pki/nginx/api.exapmle.com.crt";
    ssl_certificate_key         "/etc/pki/nginx/private/api.exapmle.com.key";
    ssl_session_cache           shared:SSL:1m;
    ssl_session_timeout         10m;
    ssl_ciphers                 HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers   on;

    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    location / {
        proxy_pass       http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

## 一些用法
### OpenCat客户端
OpenCat虽然有了团队版，但是划在收费版功能。可以直接利用自定义OpenAI域名的功能达到团队使用。
1. 在自己的服务器部署本服务，OpenAI Key只在自己服务器上保存，不会泄露。
2. 在服务器分配用户key，自己可以随便配key。
3. 在OpenCat客户端中，设置自定义OpenAI域名为自己的服务器地址，例如：https://api.exapmle.com
4. 在OpenCat客户端中，设置API key为自己在服务器上创建的key。

## 开源协议
MIT，随便拿去用，记得多帮我宣传宣传。

如果觉得帮助到你了，欢迎请[我喝一杯咖啡](https://github.com/libli/buy-me-a-coffee) ☕️。