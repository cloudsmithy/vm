# VirtPanel

> 轻量级 KVM 虚拟机管理面板，基于 Go + Vue 3，开箱即用。

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vue.js&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-yellow)

## 特性

- 🖥️ **虚拟机全生命周期** — 创建 / 启动 / 关机 / 重启 / 暂停 / 克隆 / 删除 / 重命名 / 导入
- 🎯 **系统预设** — Linux / Windows / 兼容模式，自动配置芯片组、CPU、时钟、磁盘总线、网卡
- 🖱️ **VNC 控制台** — 浏览器内 noVNC，支持 Ctrl+Alt+Del
- 💾 **磁盘管理** — 热挂载/卸载磁盘，ISO 挂载/弹出
- 🌐 **网络管理** — NAT / 桥接 / macvtap，网卡热添加/移除
- 📸 **快照** — 创建 / 恢复 / 删除 / 恢复到新虚拟机
- 🗄️ **存储** — 存储池和存储卷管理
- 📊 **仪表盘** — 主机 CPU / 内存 / 磁盘 / 负载概览，虚拟机实时 CPU 和内存使用率
- ⚡ **批量操作** — 批量启动 / 关机 / 强制关机 / 删除
- 📤 **ISO 管理** — 多文件并行上传，独立进度显示，支持取消

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go + Gin + go-libvirt |
| 前端 | Vue 3 + TypeScript + Arco Design + noVNC |
| 虚拟化 | KVM / QEMU / libvirt |
| 引导方式 | BIOS (SeaBIOS)，仅支持 x86_64 |

## 环境要求

- Linux x86_64 主机，支持硬件虚拟化（Intel VT-x / AMD-V）
- Go 1.22+（编译后端）
- Node.js 18+、pnpm（编译前端）

## 安装依赖

### Debian / Ubuntu

```bash
apt update

# 安装 QEMU、libvirt、磁盘工具
apt install -y qemu-kvm qemu-utils libvirt-daemon-system virtinst

# 启动 libvirt 相关服务
systemctl start libvirtd
systemctl start virtlogd

# 如果 systemctl 不可用（如容器环境），手动启动守护进程
libvirtd -d
virtlogd -d

# 确认 KVM 设备存在
ls -la /dev/kvm
```

### KVM 权限

如果创建虚拟机时报 `Permission denied` 访问 `/dev/kvm`，需要确保 libvirt 的 QEMU 进程有权限：

```bash
# 方案一：将 libvirt-qemu 用户加入 kvm 组
usermod -aG kvm libvirt-qemu
systemctl restart libvirtd

# 方案二：直接开放权限（快速但不推荐用于生产）
chmod 666 /dev/kvm
```

### 验证环境

```bash
# 确认 KVM 可用
virsh version

# 确认所有依赖命令
qemu-img --version   # 创建磁盘
virt-clone --version  # 克隆虚拟机
ip -V                 # 网桥管理

# 确认 libvirt socket 存在
ls /var/run/libvirt/libvirt-sock
ls /run/libvirt/virtlogd-sock
```

## 快速开始

```bash
# 后端
cd backend
go build -o virtpanel ./cmd/main.go
./virtpanel   # 监听 :8080

# 前端（开发）
cd frontend
pnpm install
pnpm dev   # 监听 :5173，自动代理 /api 和 /ws 到后端

# 前端（生产构建）
pnpm build  # 输出到 dist/，用 nginx 反代即可
```

## 项目结构

```
virtpanel/
├── backend/
│   ├── cmd/main.go              # 入口
│   ├── internal/
│   │   ├── handler/             # HTTP 路由处理
│   │   ├── service/             # libvirt 业务逻辑
│   │   └── model/               # 数据模型
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── api/                 # API 请求封装
│   │   ├── views/               # 页面组件
│   │   ├── layout/              # 布局
│   │   ├── router/              # 路由
│   │   └── styles/              # 全局样式
│   ├── package.json
│   └── vite.config.ts
└── .gitignore
```

## Nginx 部署

```nginx
server {
    listen 80;

    location / {
        root /path/to/frontend/dist;
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
    }

    location /ws/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/host/info | 主机信息 |
| GET | /api/vms | 虚拟机列表 |
| POST | /api/vms | 创建虚拟机 |
| POST | /api/vms/:name/start | 启动 |
| POST | /api/vms/:name/shutdown | 关机 |
| POST | /api/vms/:name/destroy | 强制关机 |
| DELETE | /api/vms/:name | 删除 |
| GET | /api/vms/:name/detail | 虚拟机详情 |
| POST | /api/vms/:name/iso | 挂载 ISO |
| POST | /api/vms/:name/clone | 克隆 |
| POST | /api/vms/:name/rename | 重命名 |
| POST | /api/vms/import | 导入 |
| POST | /api/vms/batch | 批量操作 |
| GET | /ws/vnc/:name | VNC WebSocket |

完整路由见 `backend/cmd/main.go`。

## 常见问题

| 错误 | 原因 | 解决 |
|------|------|------|
| `dial unix /var/run/libvirt/libvirt-sock: no such file` | libvirtd 未启动 | `systemctl start libvirtd` 或 `libvirtd -d` |
| `connect socket to '/run/libvirt/virtlogd-sock': No such file` | virtlogd 未启动 | `systemctl start virtlogd` 或 `virtlogd -d` |
| `clone failed:` (空错误) | virt-clone 未安装 | `apt install -y virtinst` |
| `create disk failed:` (空错误) | qemu-img 未安装 | `apt install -y qemu-utils` |
| `failed to initialize kvm: Permission denied` | /dev/kvm 权限不足 | `chmod 666 /dev/kvm` 或将用户加入 kvm 组 |

## License

MIT
