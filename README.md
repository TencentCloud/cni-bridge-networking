# cni-bridge-networking

`cni-bridge-networking` 一个用于部署 `bridge` 和 `loopback` cni 插件的工具，用于提供类似 `kubenet` 的功能。工具会 watch kubernetes 的 `node` 对象并根据 `spec.podCIDR` 配置相应的 conf。

## 编译

### 编译二进制文件
将此项目 clone 到 GOPATH 下，假设 GOPATH 为 /root/go

```
mkdir -p /root/go/src/github.com/tencentcloud/
git clone https://github.com/tencentcloud/cni-bridge-networking.git /root/go/src/github.com/tencentcloud/cni-bridge-networking
cd /root/go/src/github.com/tencentcloud/cni-bridge-networking
go build -v
```

### 打包 Docker Image (需要 Docker 17.05 或者更高版本)

```
docker build -f Dockerfile.multistage -t bridged:0.0.1 .
```

## 运行

```
kubectl create -f https://raw.githubusercontent.com/TencentCloud/cni-bridge-networking/master/deploy/deploy.yaml
```