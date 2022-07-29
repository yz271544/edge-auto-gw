简体中文 | [English](./README.md)

# EdgeMesh

## 介绍
该服务作为`kubeedge::edgemesh::edgemesh-gateway`的辅助功能，通过增加标签（Label）的方式，自动地为边缘服务开通指定的暴露端口。
### 背景
采用edgemesh-gateway为指定的边缘应用服务对外暴露端口，需要创建对应的Gateway/DestinatonRule/VirtualService资源，因为其实现过程参考了Istio的资源定义;
然而我们并不想人工或者在应用中实现这些资源的调用，所以想办法简单的实现了自动创建这些资源的程序:edge-auto-gw。
### 优势
无状态的运行在云端，只需要1个实例，其会listwath所有的service资源，占用资源较小，轻量可漂移。
### 关键功能
只需要为kubernetes中需要在边缘暴露的容器服务，设置一对儿标签，服务即可创建对应的gw/dr/vs资源。
标签规则：
  - `kubeedge.io/edgemesh-gateway-ports: 9090-41131.1883-31883`
    以`.`分隔端口暴露组，每组中`-`之前为容器中端口，之后为暴露到边缘服务器上的端口;
  - `kubeedge.io/edgemesh-gateway-protocols: HTTP.TCP`
    以`.`分隔端口暴露的协议，与上面ports形成对应关系，目前支持HTTP和TCP协议名称;
```yaml
apiVersion: v1
kind: Service
metadata:
  creationTimestamp: "2022-05-17T03:33:30Z"
  labels:
    kubeedge.io/edgemesh-gateway-ports: 9090-41131.1883-31883
    kubeedge.io/edgemesh-gateway-protocols: HTTP.TCP
    name: edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
  name: edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
  namespace: tenant-iaiexta-env-kndwyph
  resourceVersion: "71873074"
  selfLink: /api/v1/namespaces/tenant-iaiexta-env-kndwyph/services/edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
  uid: f6171772-f519-4291-ad3f-09edd3f62017
spec:
  clusterIP: 10.247.184.161
  ports:
  - name: http-port
    port: 9090
    protocol: TCP
    targetPort: 9090
  selector:
    kubeedge.io/edgemesh-gateway-ports: 9090-41131.1883-31883
    kubeedge.io/edgemesh-gateway-protocols: HTTP.TCP
    name: edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
  sessionAffinity: None
  type: ClusterIP
```
## 架构
运行在云端，与kubeedge一起部署在同一个namespace中，并listwatch所有的service，当发现存在指定标签的服务，则开始创建gw/dr/vs资源。
```shell
$ kubectl get all -nkubeedge -owide |grep edge-auto-gw
deployment.apps/edge-auto-gw       1/1     1            1           26d   edge-auto-gw       docker.gridsumdissector.com/kubeedge/edge-auto-gw:v0.1.3                    k8s-app=kubeedge,kubeedge=edge-auto-gw
replicaset.apps/edge-auto-gw-6d97f75d8c      1         1         1       26d   edge-auto-gw       docker.gridsumdissector.com/kubeedge/edge-auto-gw:v0.1.3                    k8s-app=kubeedge,kubeedge=edge-auto-gw,pod-template-hash=6d97f75d8c
pod/edge-auto-gw-6d97f75d8c-8gfhg      1/1     Running   0          26d   172.16.1.164    10.201.82.139   <none>           <none>
```
## 指南

### 预备知识
可参考kubeedge::edgemesh::edgemesh-gateway的相关文档
### 文档
[EdgeMesh-Gateway](https://edgemesh.netlify.app/guide/edge-gateway.html)
### 安装
安装比较容易：
```shell
# git clone to local
cd edge-auto-gw
kubectl apply -f build/kubernetes
```
### 样例
```yaml
apiVersion: v1
kind: Service
metadata:
  creationTimestamp: "2022-05-17T03:33:30Z"
  labels:
    kubeedge.io/edgemesh-gateway-ports: 9090-41131
    kubeedge.io/edgemesh-gateway-protocols: HTTP
    name: edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
  name: edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
  namespace: tenant-iaiexta-env-kndwyph
  resourceVersion: "71873074"
  selfLink: /api/v1/namespaces/tenant-iaiexta-env-kndwyph/services/edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
  uid: f6171772-f519-4291-ad3f-09edd3f62017
spec:
  clusterIP: 10.247.184.161
  ports:
  - name: http-port
    port: 9090
    protocol: TCP
    targetPort: 9090
  selector:
    kubeedge.io/edgemesh-gateway-ports: 9090-41131
    kubeedge.io/edgemesh-gateway-protocols: HTTP
    name: edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
  sessionAffinity: None
  type: ClusterIP


apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  creationTimestamp: "2022-05-17T03:33:30Z"
  generation: 1
  name: edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
  namespace: tenant-iaiexta-env-kndwyph
  resourceVersion: "71873078"
  selfLink: /apis/networking.istio.io/v1alpha3/namespaces/tenant-iaiexta-env-kndwyph/gateways/edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
  uid: 73b60177-0edb-4cae-9c8b-697a3d57f5dc
spec:
  selector:
    kubeedge: edgemesh-gateway
  servers:
  - hosts:
    - '*'
    port:
      name: http-0
      number: 41131
      protocol: HTTP


apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  creationTimestamp: "2022-05-17T03:33:30Z"
  generation: 1
  name: edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
  namespace: tenant-iaiexta-env-kndwyph
  resourceVersion: "71873075"
  selfLink: /apis/networking.istio.io/v1alpha3/namespaces/tenant-iaiexta-env-kndwyph/destinationrules/edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
  uid: ceb2ae59-a54b-4bd0-83ae-be83257b92ca
spec:
  host: edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
  trafficPolicy:
    loadBalancer:
      simple: RANDOM


apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  creationTimestamp: "2022-05-17T03:33:30Z"
  generation: 1
  name: edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
  namespace: tenant-iaiexta-env-kndwyph
  resourceVersion: "71873077"
  selfLink: /apis/networking.istio.io/v1alpha3/namespaces/tenant-iaiexta-env-kndwyph/virtualservices/edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
  uid: 30fb4246-4f2b-4461-93a0-94481615686d
spec:
  gateways:
  - edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
  hosts:
  - '*'
  http:
  - match:
    - uri:
        prefix: /
    route:
    - destination:
        host: edge-data-access-296ef850-c78c-11ec-b0cf-e5b8216fb2ae
        port:
          number: 9090
  tcp: []

```

## 联系方式
huzhengyang@gridsum.com
## 贡献
