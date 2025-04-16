FROM golang:1.16-alpine

WORKDIR /app

# 安装必要的工具
RUN apk add --no-cache curl

# 复制源代码
COPY . .

# 编译应用
RUN go build -o server

# 暴露端口
EXPOSE 8001 8002 8003 9999

# 设置入口点
ENTRYPOINT ["/app/server"]