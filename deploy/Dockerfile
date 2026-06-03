# 第 1 阶段：只负责把 Go 后端编译成 Linux 可执行文件，编译工具不会进入最终镜像。
# 使用 Go 官方 Alpine 镜像作为编译环境；版本与 server/go.mod 中的 go 1.24.0 对齐。
FROM golang:1.24-alpine AS build-server

# 设置编译阶段的工作目录，后面的 COPY、RUN 都基于这个目录执行。
WORKDIR /build/server

# 安装 git；部分 Go 依赖在下载时可能需要 git 拉取源码。
RUN apk add --no-cache git

# 强制开启 Go Modules，避免受镜像默认 Go 环境影响。
ENV GO111MODULE=on

# 配置 Go 依赖代理；先走国内代理，失败时再直连源站。
ENV GOPROXY=https://goproxy.cn,direct

# 关闭 CGO，让编译产物尽量成为静态二进制，方便放进 Alpine 运行。
ENV CGO_ENABLED=0

# 接收 Docker BuildKit 自动注入的目标平台参数，避免在 ARM/AMD64 环境之间硬编码架构。
ARG TARGETOS
ARG TARGETARCH

# 先只复制依赖清单；业务代码变更时可以复用 go mod download 的缓存。
COPY server/go.mod server/go.sum ./

# 下载 Go 依赖；单独一层缓存可以加快后续 Docker 构建速度。
RUN go mod download

# 复制完整 server 源码到编译目录，准备正式编译。
COPY server/ ./

# 按目标平台编译后端二进制；-trimpath 去掉本机路径，-s -w 减小产物体积。
RUN GOOS="${TARGETOS:-linux}" GOARCH="${TARGETARCH:-amd64}" go build -trimpath -ldflags="-s -w" -o /build/server/server ./cmd/main.go

# 第 2 阶段：只放运行所需文件，最终镜像会比完整 Go 编译环境小很多。
# 使用 Alpine 作为运行环境，镜像轻量，适合只运行一个 Go 服务加 nginx。
FROM alpine:3.22

# 设置容器内应用根目录，后续运行脚本也从这里启动。
WORKDIR /opt/webtrailai

# 设置默认字符集，避免中文日志或响应出现编码问题。
ENV LANG=C.UTF-8

# 设置容器默认时区，方便日志时间和本地排查习惯一致。
ENV TZ=Asia/Shanghai

# 设置容器默认配置文件路径；部署时把宿主机目录挂载到 /opt/webtrailai/config 即可外部管理配置。
ENV WEBTRAIL_CONFIG_PATH=/opt/webtrailai/config/config.yaml

# 安装运行时依赖：bash 执行 entrypoint，ca-certificates 支持 HTTPS，nginx 做 80 端口反代，tzdata 提供时区数据。
RUN apk add --no-cache bash ca-certificates nginx tzdata \
    && rm -f /etc/nginx/http.d/default.conf

# 复制容器启动脚本，负责同时启动 Go 后端和 nginx。
COPY deploy/entrypoint.sh /opt/webtrailai/entrypoint.sh

# 覆盖 Alpine nginx 默认站点配置，使用本项目自己的反向代理规则。
COPY deploy/default.conf /etc/nginx/http.d/default.conf

# 复制脱敏配置模板到非挂载目录；避免宿主机挂载 /opt/webtrailai/config 时把模板遮住。
COPY deploy/config.yaml.template /opt/webtrailai/defaults/config.yaml.template

# 从编译阶段复制 Go 二进制到最终镜像，最终镜像不包含 Go 编译工具链。
COPY --from=build-server /build/server/server /opt/webtrailai/server/server

# 创建运行期需要的配置、数据和日志目录，并给启动脚本添加可执行权限。
RUN mkdir -p /opt/webtrailai/config /opt/webtrailai/defaults /opt/webtrailai/server/filedb /opt/webtrailai/server/logs \
    && chmod +x /opt/webtrailai/entrypoint.sh

# 声明配置、数据和日志目录为 Docker volume，提醒部署时应做持久化挂载。
VOLUME ["/opt/webtrailai/config", "/opt/webtrailai/server/filedb", "/opt/webtrailai/server/logs"]

# 声明容器对外提供 80 端口；实际宿主机映射由 docker run -p 控制。
EXPOSE 80

# 定期访问 nginx 根路径检查容器健康状态；失败多次后 Docker 会标记 unhealthy。
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD wget -q -O /dev/null http://127.0.0.1/ || exit 1

# 容器启动时执行 entrypoint 脚本，由脚本负责启动和管理内部进程。
ENTRYPOINT ["/bin/bash", "./entrypoint.sh"]
