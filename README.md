English | [简体中文](./README_zh.md)

# EdgeAutoGw

## Introduction
As an auxiliary function of `kubeedge::edgemesh::edgemesh-gateway`, this service automatically opens a specified exposed port for edge services by adding a label (Label).
### Background
Using edgemesh-gateway to expose the port for the specified edge application service, it is necessary to create the corresponding Gateway/DestinatonRule/VirtualService resource, because the implementation process refers to the resource definition of Istio;
However, we don't want to call these resources manually or in the application, so we tried to simply implement the program that automatically creates these resources: edge-auto-gw.
### Advantage
Running stateless in the cloud, only one instance is required, which will list all the service resources of Wath, occupying less resources, and it is lightweight and can be drifted.
### Key Features
You only need to set a pair of labels for the container services that need to be exposed at the edge in kubernetes, and the service can create corresponding gw/dr/vs resources.
Labeling rules:
   - `kubeedge.io/edgemesh-gateway-ports: 9090-41131.1883-31883`
     Separate port exposure groups with `.`, in each group `-` is the port in the container before the port, and then the port exposed to the edge server;
   - `kubeedge.io/edgemesh-gateway-protocols: HTTP.TCP`
     The protocols exposed by ports separated by `.` form a corresponding relationship with the above ports. Currently, HTTP and TCP protocol names are supported;
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
## Architecture
Running in the cloud, deployed in the same namespace with kubeedge, and listwatch all services, when a service with the specified tag is found, it starts to create gw/dr/vs resources.
```shell
$ kubectl get all -nkubeedge -owide |grep edge-auto-gw
deployment.apps/edge-auto-gw       1/1     1            1           26d   edge-auto-gw       docker.gridsumdissector.com/kubeedge/edge-auto-gw:v0.1.3                    k8s-app=kubeedge,kubeedge=edge-auto-gw
replicaset.apps/edge-auto-gw-6d97f75d8c      1         1         1       26d   edge-auto-gw       docker.gridsumdissector.com/kubeedge/edge-auto-gw:v0.1.3                    k8s-app=kubeedge,kubeedge=edge-auto-gw,pod-template-hash=6d97f75d8c
pod/edge-auto-gw-6d97f75d8c-8gfhg      1/1     Running   0          26d   172.16.1.164    10.201.82.139   <none>           <none>
```
## Guides

### Prerequisites
Please refer to the related documentation of `kubeedge::edgemesh::edgemesh-gateway`
### Documents
[EdgeMesh-Gateway](https://edgemesh.netlify.app/guide/edge-gateway.html)
### Installation
Installation is easier:
```shell
# git clone to local
cd edge-auto-gw
kubectl apply -f build/kubernetes
```
### Examples
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
## Contact
huzhengyang@gridsum.com
## Contributing
