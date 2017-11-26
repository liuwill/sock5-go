# socks5-go

> golang socks5 server 使用golang实现的socks5服务器

## 启动方法

```shell
# 启动不需要验证的服务器
make server

# 启动需要验证的服务器
make server-auth

```

## 编程启动服务

安装`go get github.com/liuwill/sock5-go`

启动

```go
package main

import (
  "os"
  "sock5-go"
)

func main() {
  defaultPort := os.Getenv("DEFAULT_PORT")
  if len(defaultPort) <= 0 {
    defaultPort = "10008"
  }

  sock5Server, err := sock5.NewSocks5ServerConfigurable(":"+defaultPort, sock5.ServerConfiguration{
    Mode: "auth",
  })
  if err != nil {
    panic(err)
  }

  sock5Server.Start()
}
```