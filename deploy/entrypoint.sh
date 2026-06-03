#!/usr/bin/env bash

# 开启严格模式：命令失败、未定义变量、管道失败时尽早暴露问题。
set -Eeuo pipefail

# 应用根目录；Dockerfile 中 WORKDIR 也设置成了这个路径。
APP_DIR="/opt/webtrailai"

# 后端服务目录；配置文件、二进制、filedb、logs 都放在这个目录下。
SERVER_DIR="${APP_DIR}/server"

# 配置文件默认放在容器外置目录，部署时挂载 /opt/webtrailai/config 即可直接查看和修改。
CONFIG_FILE="${WEBTRAIL_CONFIG_PATH:-${APP_DIR}/config/config.yaml}"

# 镜像内只保留脱敏模板；首次启动发现外置配置不存在时，用模板初始化一份。
CONFIG_TEMPLATE="${APP_DIR}/defaults/config.yaml.template"

# 确保配置目录存在；挂载空目录或自定义 WEBTRAIL_CONFIG_PATH 时也能自动补齐。
CONFIG_DIR="$(dirname "${CONFIG_FILE}")"
mkdir -p "${CONFIG_DIR}"

# 如果宿主机还没有配置文件，自动从镜像模板生成，避免用户必须先手动复制文件。
if [[ ! -f "${CONFIG_FILE}" ]]; then
    if [[ ! -f "${CONFIG_TEMPLATE}" ]]; then
        echo "启动失败：配置文件 ${CONFIG_FILE} 不存在，且模板 ${CONFIG_TEMPLATE} 不存在。" >&2
        exit 1
    fi
    cp "${CONFIG_TEMPLATE}" "${CONFIG_FILE}"
    chmod 600 "${CONFIG_FILE}"
    echo "已初始化外置配置文件：${CONFIG_FILE}"
fi

# 配置文件必须可读，否则 Go 服务启动时只会 panic，不利于定位部署问题。
if [[ ! -r "${CONFIG_FILE}" ]]; then
    echo "启动失败：配置文件不可读：${CONFIG_FILE}" >&2
    exit 1
fi

# 确保文件数据库和日志目录存在；挂载空 volume 或覆盖目录变量时也能自动补齐。
mkdir -p "${WEBTRAIL_DB_FILEDIR:-${SERVER_DIR}/filedb}" "${WEBTRAIL_LOG_PATH:-${SERVER_DIR}/logs}"

# 切到后端目录运行，这样 ./config.yaml、./filedb 等相对路径都能正确解析。
cd "${SERVER_DIR}"

# 后台启动 Go 后端；--conf 指向外置配置文件路径。
./server --conf "${CONFIG_FILE}" &

# 记录 Go 后端进程号，后面用于等待和优雅停止。
server_pid=$!

# 前台模式启动 nginx，但放到后台运行，方便脚本同时监控 Go 后端和 nginx。
nginx -g "daemon off;" &

# 记录 nginx 进程号，后面用于等待和优雅停止。
nginx_pid=$!

# 定义统一清理函数：任一进程退出或收到停止信号时，把两个进程都停掉。
stop_all() {
    # 给 Go 后端和 nginx 发送 TERM 信号；进程已退出时忽略错误。
    kill -TERM "${server_pid}" "${nginx_pid}" 2>/dev/null || true
    # 等待两个进程真正退出；已退出或不存在时忽略错误，避免清理阶段报错。
    wait "${server_pid}" "${nginx_pid}" 2>/dev/null || true
}

# 当容器收到 Ctrl+C 或 docker stop 的信号时，执行 stop_all 做清理。
trap stop_all INT TERM

# 打印启动成功信息和两个进程号，方便 docker logs 排查。
echo "webtrailai all started, config=${CONFIG_FILE}, server_pid=${server_pid}, nginx_pid=${nginx_pid}"

# 临时关闭 set -e；wait -n 在进程异常退出时会返回非 0，需要手动接住退出码。
set +e

# 等待 Go 后端或 nginx 任意一个进程退出；哪个先退出都说明容器不应继续假装健康。
wait -n "${server_pid}" "${nginx_pid}"

# 保存先退出进程的退出码，后面作为整个容器的退出码。
exit_code=$?

# 恢复严格错误模式，避免后续命令静默失败。
set -e

# 清理另一个仍在运行的进程，避免容器退出前留下孤儿进程。
stop_all

# 使用真实退出码结束脚本，让 Docker/部署平台能看到失败原因。
exit "${exit_code}"
