#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

bash $SCRIPT_ROOT/vendor/k8s.io/code-generator/generate-groups.sh "client,informer,lister" \
  github.com/kuda-io/kuda/pkg/generated \
  github.com/kuda-io/kuda/pkg/api \
  "data:v1alpha1" \
  --go-header-file hack/boilerplate.go.txt
