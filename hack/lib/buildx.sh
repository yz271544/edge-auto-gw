#!/usr/bin/env bash

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

# update the code from https://github.com/kubeedge/sedna/blob/fa6df45c30b1ff934dba07fa0c59f3f64a943d2e/hack/lib/buildx.sh

set -o errexit
set -o nounset
set -o pipefail

edge_auto_gw::buildx::prepare_env() {
  # Check whether buildx exists.
  if ! docker buildx >/dev/null 2>&1; then
    echo "ERROR: docker buildx not available. Docker 19.03 or higher is required with experimental features enabled" >&2
    exit 1
  fi

  # Use tonistiigi/binfmt that is to enable an execution of different multi-architecture containers
  docker run --privileged --rm tonistiigi/binfmt --install all

  # Create a new builder which gives access to the new multi-architecture features.
  builder_instance="edge-auto-gw-buildx"
  if ! docker buildx inspect $builder_instance >/dev/null 2>&1; then
    docker buildx create --use --name $builder_instance --driver docker-container
  fi
  docker buildx use $builder_instance
}

edge_auto_gw::buildx:generate-dockerfile() {
  dockerfile=${1}
  echo "dockerfile:${dockerfile}"
  sed "/AS builder/s/FROM/FROM --platform=\$BUILDPLATFORM/g" ${dockerfile}
}


edge_auto_gw::buildx::build-multi-platform-images() {
  edge_auto_gw::buildx::prepare_env

  echo "EDGE_AUTO_GW_OUTPUT_IMAGEPATH:${EDGE_AUTO_GW_OUTPUT_IMAGEPATH}"
  mkdir -p ${EDGE_AUTO_GW_OUTPUT_IMAGEPATH}
  arch_array=(${PLATFORMS//,/ })

  temp_dockerfile=${EDGE_AUTO_GW_OUTPUT_IMAGEPATH}/buildx_dockerfile
  echo "temp_dockerfile:${temp_dockerfile}"
  for component in ${COMPONENTS[@]}; do
    echo "building ${PLATFORMS} image for edge-auto-gw-${component}"

    edge_auto_gw::buildx:generate-dockerfile build/${component}/Dockerfile > ${temp_dockerfile}

    for arch in ${arch_array[@]}; do
      tag_name=${IMAGE_REPO}/${component}:${IMAGE_TAG}-${arch////-}
      echo "building ${arch} image for ${component} and the image tag name is ${tag_name}"

      docker buildx build -o type=docker \
        --build-arg "GO_LDFLAGS=${GO_LDFLAGS}" \
        --platform ${arch} \
        -t ${tag_name} \
        -f ${temp_dockerfile} .
      done
  done

  rm ${temp_dockerfile}
}
