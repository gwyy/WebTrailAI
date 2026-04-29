# 项目说明

- 这是一个 golang 项目，使用 gin 来做的一个 http 服务后端项目
- 启动文件在 cmd/main.go
- 项目目录都在 internal 目录里面
- internal/config 是项目配置文件
- internal/ctrl   是 controller 目录 处理 http 请求
- internal/logger 是项目日志代码，使用 zap 三方包
- internal/model  是项目模型文件 定义各种模型，项目不使用传统数据库，使用文本数据库
- internal/router  是项目路由目录
- internal/service 是项目逻辑处理目录
- 依赖的第三方组件在 pkg 目录下面
- 使用第三方包 scribble 来做文本数据库

# 开发规范

- 优先复用已有组件
- 不要随意新增依赖



# 注意事项

