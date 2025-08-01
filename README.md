# Titan VM Management System




Titan VM is a distributed virtual machine management system with client-server architecture, supporting both KVM and Multipass backends.

## 🚀 Features

- **Centralized Management** - Manage VMs across multiple machines from a single server
- **Multi-backend Support** - Works with both KVM and Multipass
- **Secure Communication** - WebSocket-based protocol with authentication
- **Lightweight** - Minimal resource footprint on client machines

## 📦 Components

| Component | Description | Location |
|-----------|-------------|----------|
| `vmc` | Client agent installed on user machines | `/vmc` |
| `vms` | Management server | `/vms` |
| `vmadm` | Administration tools | `/vmadm` |

## ⚙️ Installation

### Prerequisites
- Go 1.16+
- libvirt (for KVM support)
- Multipass (for Multipass support)

### Build
```bash
git clone https://github.com/zscboy/titan-vm.git

cd titan-vm/
go build ./vmc
go build ./vms

