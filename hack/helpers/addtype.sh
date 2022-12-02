#!/usr/bin/env bash

# Copyright 2022 The Crossplane Authors.
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

# Please set ProviderNameLower & ProviderNameUpper environment variables before running this script.
# See: https://github.com/crossplane/terrajet/blob/main/docs/generating-a-provider.md
set -euo pipefail

APIVERSION="${APIVERSION:-v1alpha1}"
echo "Adding type ${KIND} to group ${GROUP} with version ${APIVERSION}"

export GROUP
export KIND
export APIVERSION
export PROVIDER

kind_lower=$(echo "${KIND}" | tr "[:upper:]" "[:lower:]")
group_lower=$(echo "${GROUP}" | tr "[:upper:]" "[:lower:]")

gomplate < "hack/helpers/apis/GROUP_LOWER/APIVERSION/KIND_LOWER_types.go.tmpl" > "apis/${group_lower}/${APIVERSION}/${kind_lower}_types.go"

mkdir -p "internal/controller/${kind_lower}"
gomplate < "hack/helpers/controller/KIND_LOWER/KIND_LOWER.go.tmpl" > "internal/controller/${kind_lower}/${kind_lower}.go"
gomplate < "hack/helpers/controller/KIND_LOWER/KIND_LOWER_test.go.tmpl" > "internal/controller/${kind_lower}/${kind_lower}_test.go"



