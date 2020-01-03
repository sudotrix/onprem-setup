#!/bin/bash

# Copyright 2019 Istio Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

fail() {
  echo "$@" 1>&2
  exit 1
}

API_TMP="$(mktemp -d -u)"

trap 'rm -rf "${API_TMP}"' EXIT

SCRIPTPATH="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOTDIR=$(dirname "${SCRIPTPATH}")
cd "${ROOTDIR}"

# using the pseudo version we have in go.mod file. e.g. v.0.0.0-<timestamp>-<SHA>
SHA=$(grep "istio.io/api" go.mod | sed 's/[[:blank:]]istio\.io\/api v0\.0\.0-[[:digit:]]*-//g')

if [ -z "${SHA}" ]; then
  fail "Unable to retrieve the commit SHA of istio/api from go.mod file. Not updating the CRD file. Please make sure istio/api exists in the Go module.";
fi

mkdir -p "${API_TMP}"
cd "${API_TMP}"
git init -q && git fetch "https://github.com/istio/api" -q && git merge "${SHA}" -q
if [ ! -f "${API_TMP}/kubernetes/customresourcedefinitions.gen.yaml" ]; then
  echo "Generated Custom Resource Definitions file does not exist in the commit SHA. Not updating the CRD file."
  exit
fi
rm -f "${ROOTDIR}/install/kubernetes/helm/istio-init/files/crd-all.gen.yaml"
cp "${API_TMP}/kubernetes/customresourcedefinitions.gen.yaml" "${ROOTDIR}/install/kubernetes/helm/istio-init/files/crd-all.gen.yaml"
