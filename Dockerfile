# 使用官方 Golang 镜像作为构建阶段的基础镜像
FROM golang:1.20-alpine AS builder

# 设置工作目录
WORKDIR /app

# 将 Go Modules 相关文件复制到工作目录
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 将应用程序源代码复制到工作目录
COPY . .

# 构建应用程序
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# 使用轻量级的 Alpine 作为最终的基础镜像
FROM alpine:3.14

# 设置工作目录
WORKDIR /app

# 从构建阶段复制构建好的二进制文件到最终镜像中
COPY --from=builder /app/main .

# 暴露端口
ENV PORT 8080
EXPOSE 8080

# 运行应用程序
CMD ["./main"]