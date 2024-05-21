# 第一阶段: 构建阶段
FROM golang:1.20-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建 Go 应用程序
RUN go build -o myapp .

# 第二阶段: 运行阶段
FROM alpine:latest

# 设置工作目录
WORKDIR /root/

# 复制从构建阶段生成的可执行文件到运行阶段
COPY --from=builder /app/myapp .

# 复制静态文件到运行阶段
COPY --from=builder /app/static ./static

# 公开服务端口 (如果需要)
EXPOSE 8800

# 运行应用程序
CMD ["./myapp"]
