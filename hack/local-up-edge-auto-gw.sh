#!/bin/bash

# Copyright 2021 The KubeEdge Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Developers can run `hack/local-up-edge-auto-gw.sh` to setup up a local environment:
# 1. a local k8s cluster with a master node.
# 2. a kubeedge node.
# 3. our edge-auto-gw.

# It does:
# 1. build the edge-auto-gw image.
# 2. use kind install a k8s cluster
# 3. use keadm install kubeedge
# 4. prepare our k8s env.
# 5. config edge-auto-gw config and start edge-auto-gw.
# 6. add cleanup.

set -o errexit
set -o nounset
set -o pipefail

# ENABLE_DAEMON will
ENABLE_DAEMON=${ENABLE_DAEMON:-false}
EDGE_AUTO_GW_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

cd "$EDGE_AUTO_GW_ROOT"

NO_CLEANUP=${NO_CLEANUP:-false}

IMAGE_TAG=localup

CLUSTER_NAME=test
MASTER_NODENAME=${CLUSTER_NAME}-control-plane
HOST_IP=`hostname -I | awk '{print $1}'`
EDGE_NODENAME=edge-node
KUBEEDGE_VERSION=1.8.2
NAMESPACE=kubeedge
LOG_DIR=${LOG_DIR:-"/tmp"}
TIMEOUT=${TIMEOUT:-120}s

if [[ "${CLUSTER_NAME}x" == "x" ]];then
    CLUSTER_NAME="test"
fi

export CLUSTER_CONTEXT="--name ${CLUSTER_NAME}"


TMP_DIR="$(realpath local-up-tmp)"

get_kubeedge_pid() {
  ps -e -o pid,comm,args |
   grep -F "$TMP_DIR" |
   # match executable name and print the pid
   awk -v bin="${1:-edgecore}" 'NF=$2==bin'
}


# spin up cluster with kind command
function kind_up_cluster {
  echo "Running kind: [kind create cluster ${CLUSTER_CONTEXT} --image kindest/node:v1.18.2]"
  kind create cluster ${CLUSTER_CONTEXT}  --image kindest/node:v1.18.2
  add_cleanup '
    echo "Running kind: [kind delete cluster ${CLUSTER_CONTEXT}]"
    kind delete cluster ${CLUSTER_CONTEXT}
  '
}


function check_control_plane_ready {
  echo "wait the control-plane ready..."
  kubectl wait --for=condition=Ready node/${CLUSTER_NAME}-control-plane --timeout=${TIMEOUT}
  MASTER_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' test-control-plane`
}

function check_node_ready {
  echo "wait the $1 node ready"
  kubectl wait --for=condition=Ready node/${1} --timeout=${TIMEOUT}
}

localup_kubeedge() {
  set -x
  # init cloudcore
  add_cleanup 'rm `ls /etc/kubeedge | grep -v "kubeedge"` -rf'
  add_cleanup 'sudo -E keadm reset --force --kube-config=${KUBECONFIG}'
  sudo -E keadm init --advertise-address=${HOST_IP} --kubeedge-version=${KUBEEDGE_VERSION} --kube-config=${KUBECONFIG}

  # ensure tokensecret is generated
  for ((i=1;i<20;i++)) ; do
      sleep 3
      kubectl get secret -n kubeedge| grep -q tokensecret && break
  done

  # join edgenode
  sleep 5
  add_cleanup 'sudo keadm reset --force --kube-config=${KUBECONFIG}'
  add_cleanup 'sudo rm /etc/systemd/system/edgecore.service'
  token=$(sudo keadm gettoken --kube-config=${KUBECONFIG})
  echo $token

  # turn off edge-auto-gw and turn on list-watch featuren and resart edgeocre
  export CHECK_EDGECORE_ENVIRONMENT="false"
  sudo -E keadm join --cloudcore-ipport=${HOST_IP}:10000 --kubeedge-version=${KUBEEDGE_VERSION} --token=${token} --edgenode-name=${EDGE_NODENAME}

  EDGE_BIN=/usr/local/bin/edgecore
  EDGE_CONFIGFILE=/etc/kubeedge/config/edgecore.yaml
  EDGECORE_LOG=${LOG_DIR}/edgecore.log
  sudo sed -i '$a\  edge-auto-gw:\n    enable: false\n'  ${EDGE_CONFIGFILE}

  ps -aux | grep edgecore

  sudo pkill edgecore
  nohup sudo -E ${EDGE_BIN} --config=${EDGE_CONFIGFILE} > "${EDGECORE_LOG}" 2>&1 &
  EDGECORE_PID=$!
  sleep 15
  ps -aux | grep edgecore
  check_node_ready ${EDGE_NODENAME}
}

build_component_image() {
  local bin
  for bin; do
    echo "building $bin"
    make -C "${EDGE_AUTO_GW_ROOT}" ${bin}image IMAGE_TAG=$IMAGE_TAG
    eval ${bin^^}_IMAGE="'kubeedge/edge-auto-gw-${bin}:${IMAGE_TAG}'"
  done
  # no clean up for images
}


load_images_to_master() {
  local image
  for image in $SERVER_IMAGE; do
    kind load --name $CLUSTER_NAME docker-image $image
  done
}

prepare_k8s_env() {
  kind get kubeconfig --name $CLUSTER_NAME > $TMP_DIR/kubeconfig
  export KUBECONFIG=$(realpath $TMP_DIR/kubeconfig)
  # prepare our k8s environment

}

start_edge_auto_gw() {
  local edge_auto_gw_deploy_name=edge-auto-gw

  add_cleanup "
  kubectl delete --timeout=5s deployment edge-auto-gw -n kubeedge
  "
  # create rbac
  kubectl create -f- <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    k8s-app: kubeedge
    kubeedge: edge-auto-gw
  name: edge-auto-gw
  namespace: kubeedge
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: edge-auto-gw
  labels:
    k8s-app: kubeedge
    kubeedge: edge-auto-gw
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list", "watch", "create", "update"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: edge-auto-gw
  labels:
    k8s-app: kubeedge
    kubeedge: edge-auto-gw
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: edge-auto-gw
subjects:
  - kind: ServiceAccount
    name: edge-auto-gw
    namespace: kubeedge
EOF

  # create configmap
  kubectl create -f- <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: edge-auto-gw
  namespace: kubeedge
  labels:
    k8s-app: kubeedge
    kubeedge: edge-auto-gw
data:
  edge-auto-gw.yaml: |
    modules:
      tunnel:
        enable: true
        publicIP: ${MASTER_IP}
        enableSecurity: true
        ACL:
          httpServer: https://${HOST_IP}:10002
EOF

  # create deployment

  kubectl create -f- <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    k8s-app: kubeedge
    kubeedge: edge-auto-gw
  name: ${edge_auto_gw_deploy_name}
  namespace: kubeedge
spec:
  selector:
    matchLabels:
      k8s-app: kubeedge
      kubeedge: edge-auto-gw
  template:
    metadata:
      labels:
        k8s-app: kubeedge
        kubeedge: edge-auto-gw
    spec:
      hostNetwork: true
      #use label to selector node
      nodeName: ${MASTER_NODENAME}
      containers:
      - name: edge-auto-gw
        image: ${SERVER_IMAGE}
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            cpu: 200m
            memory: 1Gi
          requests:
            cpu: 100m
            memory: 512Mi
        volumeMounts:
          - name: conf
            mountPath: /etc/kubeedge/config
      restartPolicy: Always
      serviceAccountName: edge-auto-gw
      volumes:
        - name: conf
          configMap:
            name: edge-auto-gw-cfg
        - name: ca-server-token
          secret:
            secretName: tokensecret
EOF

echo "wait the edge-auto-gw pod ready"
kubectl wait --timeout=${TIMEOUT} --for=condition=Ready pod -l kubeedge=edge-auto-gw -n kubeedge
}

write_nameserver() {
  sudo chown $USER:$USER /etc/resolv.conf
  cat >/etc/resolv.conf<<EOF
# This file is managed by man:systemd-resolved(8). Do not edit.
#
# This is a dynamic resolv.conf file for connecting local clients to the
+ sleep 5
# internal DNS stub resolver of systemd-resolved. This file lists all
# configured search domains.
#
# Run "resolvectl status" to see details about the uplink DNS servers
# currently in use.
#
# Third party programs must not access this file directly, but only through the
# symlink at /etc/resolv.conf. To manage man:resolv.conf(5) in a different way,
# replace this symlink by a static file or a different symlink.
#
# See man:systemd-resolved.service(8) for details about the supported modes of
# operation for /etc/resolv.conf.

nameserver 169.254.96.16
nameserver 127.0.0.53
options edns0 trust-ad
search ild0l4k5vsluppoevu2oqvhmda.cx.internal.cloudapp.net
EOF
}

declare -a CLEANUP_CMDS=()
add_cleanup() {
  CLEANUP_CMDS+=("$@")
}

cleanup() {
  if [[ "${NO_CLEANUP}" = true ]]; then
    echo "No clean up..."
    return
  fi

  set +o errexit

  echo "Cleaning up edge-auto-gw..."

  sudo rm -rf /etc/kubeedge /var/lib/kubeedge

  local idx=${#CLEANUP_CMDS[@]} cmd
  # reverse call cleanup
  for((;--idx>=0;)); do
    cmd=${CLEANUP_CMDS[idx]}
    echo "calling $cmd:"
    eval "$cmd"
  done

  set -o errexit
}

check_healthy() {
  true
}

debug_infos=""
add_debug_info() {
  debug_infos+="$@
"
}

check_prerequisites() {
  true
}

NO_COLOR='\033[0m'
RED='\033[0;31m'
GREEN='\033[0;32m'
green_text() {
  echo -ne "$GREEN$@$NO_COLOR"
}

red_text() {
  echo -ne "$RED$@$NO_COLOR"
}

label_node() {
  kubectl label nodes ${EDGE_NODENAME} lan=edge-lan-01
}

create_istio_crd() {
  echo "createing the istio crd..."
  kubectl apply -f ${EDGE_AUTO_GW_ROOT}/build/crds/istio/destinationrule-crd.yaml
  kubectl apply -f ${EDGE_AUTO_GW_ROOT}/build/crds/istio/gateway-crd.yaml
  kubectl apply -f ${EDGE_AUTO_GW_ROOT}/build/crds/istio/virtualservice-crd.yaml
}

do_up() {
  cleanup

  mkdir -p "$TMP_DIR"
  add_cleanup 'rm -rf "$TMP_DIR"'

  kind_up_cluster

  prepare_k8s_env

  check_control_plane_ready

  kubectl delete daemonset kindnet -n kube-system
  kubectl create ns kubeedge

  # here local up kubeedge before building our images, this could avoid our
  # images be removed since edgecore image gc would be triggered when high
  # image usage(>=80%), see https://github.com/kubeedge/sedna/issues/26 for
  # more details
  localup_kubeedge

  check_prerequisites

  create_istio_crd

  build_component_image agent server
  load_images_to_master

  start_edge_auto_gw

  label_node
}

do_up_fg() {

  do_up

  echo "Local cluster is $(green_text running).
  Currently local-up script only support foreground running.
  Press $(red_text Ctrl-C) to shut it down!
  You can use it with: kind export kubeconfig --name ${CLUSTER_NAME}
  $debug_infos
  "
  while check_healthy; do sleep 5; done
}

main() {

  if [ "${ENABLE_DAEMON}" = false ]; then
    trap cleanup EXIT
    trap clean ERR
    do_up_fg
  else  # DAEMON mode, for run e2e
    trap clean ERR
    trap clean INT
    do_up
  fi

}

main
