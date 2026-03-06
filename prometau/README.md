# prometau

## 目录结构

```
.
├── build/              # 构建输出目录
├── cmd/                # 主应用程序入口
│   └── prometau/       # 应用主程序
├── configs/            # 配置文件目录
├── deployments/        # 部署配置 (Docker, K8s等)
├── docs/               # 设计文档、方案文档
├── internal/           # 私有应用代码
│   ├── api/            # API处理器/控制器
│   ├── model/          # 数据模型
│   ├── repository/     # 数据访问层
│   └── service/        # 业务逻辑层
├── pkg/                # 可被外部使用的公共库
├── scripts/            # 脚本文件 (构建、安装、分析等)
├── test/               # 测试数据和测试辅助工具
└── go.mod              # Go模块定义
```