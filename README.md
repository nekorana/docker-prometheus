# 题目要求

**基于Docker构建Prometheus+Grafana监控集群模型及技术实现**

- 搭建Prometheus+Grafana的全方位监控告警系统
  - 配置prometheus 的**动态、静态服务发现**
  - 实现对容器、物理节点、service、pod等**资源指标监控**
  - 在Grafana的**web界面展示**prometheus的监控指标。
- 研究docker与kubernetes容器编排环境下的安全监控。
- 研究Prometheus监控系统警告工具包以及Prometheus在Kubernetes集群下的部署。
- 研究Grafana度量分析可视化工具的使用，并添加prometheus收集的数据作为输入源。
- 完成Prometheus和Grafana在k8s或docker环境下的部署，并对本机服务器性能和集群状态进行监控

使用docker部署prometheus+grafana+alert manager，监控k8s中的容器并提供预警服务

# 初始准备

## 配置虚拟机环境

三台虚拟机均为centos7，一个master两个node

```bash
关闭防火墙：
systemctl stop firewalld
systemctl disable firewalld

关闭selinux：
sed -i 's/enforcing/disabled/' /etc/selinux/config  # 永久
setenforce 0  # 临时

关闭swap：
swapoff -a  # 临时
vim /etc/fstab  # 永久 注释掉swap那一行

设置主机名：
hostnamectl set-hostname <hostname>

在master添加hosts：
cat >> /etc/hosts << EOF
192.168.253.135 k8s-master
192.168.253.137 k8s-node01
192.168.253.136 k8s-node02
EOF

将桥接的IPv4流量传递到iptables的链：
cat > /etc/sysctl.d/k8s.conf << EOF
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
EOF
sysctl --system  # 生效

时间同步：
yum install ntpdate -y
ntpdate time.windows.com
```

## 安装docker

```bash
wget https://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo -O /etc/yum.repos.d/docker-ce.repo
yum install -y docker-ce-20.10.9-3.el7
systemctl enable docker && systemctl start docker
```

## 配置/etc/docker/daemon.json

不配置cgroupdriver会导致后面kubelet起不来

```bash
vim /etc/docker/daemon.json

{
  "registry-mirrors": ["https://registry.cn-hangzhou.aliyuncs.com"],
  "exec-opts": ["native.cgroupdriver=systemd"]
}

systemctl daemon-reload
systemctl restart docker
```

## 添加阿里云K8S源

```bash
cat <<EOF > /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://mirrors.aliyun.com/kubernetes/yum/repos/kubernetes-el7-x86_64/
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://mirrors.aliyun.com/kubernetes/yum/doc/yum-key.gpg https://mirrors.aliyun.com/kubernetes/yum/doc/rpm-package-key.gpg
EOF
```

## 安装k8s

```bash
yum install -y kubelet-1.23.6 kubeadm-1.23.6 kubectl-1.23.6
systemctl enable kubelet && systemctl start kubelet
```

## 初始化master

```sh
kubeadm init --apiserver-advertise-address=192.168.253.135 --image-repository registry.aliyuncs.com/google_containers --kubernetes-version v1.23.6 --service-cidr=10.96.0.0/12 --pod-network-cidr=10.244.0.0/16

mkdir -p $HOME/.kube

sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config

sudo chown $(id -u):$(id -g) $HOME/.kube/config
```

## 获取token与证书哈希

```bash
kubeadm token list

openssl x509 -pubkey -in /etc/kubernetes/pki/ca.crt | openssl rsa -pubin -outfrom der 2>/dev/null | openssl dgst -sha256 -hex | sed 's/^.* //'

// 或者直接输入以下命令
kubeadm token create --print-join-command
```

## 添加node到集群

```bash
kubeadm join 192.168.253.135:6443 --token <token> --discovery-token-ca-cert-hash <sha256:ca>
```

## 配置CNI网络插件

在master节点上：

```bash
mkdir /opt/k8s && cd /opt/k8s

wget https://docs.projectcalico.org/v3.25/manifests/calico.yaml --no-check-certificate

sed -i 's#docker.io/##g' calico.yaml

kubectl apply -f calico.yaml
```

## 部署k8s集群

```bash
kubectl create deployment nginx --image=nginx

kubectl expose deployment nginx --port=80 --type=NodePort
```

此时访问三个ip均可看到nginx的默认界面，即k8s集群搭建完毕

# Prometheus搭建

## 服务发现
编写admin-role.yaml配置文件
```yaml
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: admin
  annotations:
    rbac.authorization.kubernetes.io/autoupdate: "true"
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
subjects:
- kind: ServiceAccount
  name: admin
  namespace: kube-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: admin
  namespace: kube-system
  labels:
    kubernetes.io/cluster-service: "true"
    addonmanager.kubernetes.io/mode: Reconcile
```
创建admin用户并获取token
```bash
kubectl create -f admin-role.yaml

kubectl -n kube-system get secret|grep admin-token

kubectl -n kube-system describe secret <admin-name>
```
## grafana配置

由于系统是部署在docker container中，所以data source不能连接localhost

data source的ip应该设置为`http://prometheus:9090`

详情参考[grafana issue #46434](https://github.com/grafana/grafana/issues/46434)

## alert manager配置

用go语言简单写了个调用飞书webhook机器人的脚本，解析从alertmanager收到的告警消息，转化成larkReuqest格式发给飞书bot，实现prometheus系统对接飞书告警

