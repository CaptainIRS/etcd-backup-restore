#!/usr/bin/env bash
#
# SPDX-FileCopyrightText: Contributors to the Gardener project
#
# SPDX-License-Identifier: Apache-2.0
set -o errexit
set -o nounset
set -o pipefail

KUBECONFIG=$1

case "$(docker info --format '{{.Architecture}}')" in
  arm64|aarch64) rpm_arch=aarch64 ;;
  amd64|x86_64) rpm_arch=x86_64 ;;
  *) echo 'Unsupported AWS Emulator architecture' >&2; exit 1 ;;
esac

docker build --build-arg RPM_ARCH="${rpm_arch}" \
  -t aws-emulator:latest ./hack/e2e-test/infrastructure/aws-emulator
kind load docker-image aws-emulator:latest --name etcdbr-e2e

kubectl --kubeconfig="${KUBECONFIG}" apply -f ./hack/e2e-test/infrastructure/aws-emulator/aws-emulator.yaml
kubectl --kubeconfig="${KUBECONFIG}" -n default wait --for=condition=Available deployment/aws-emulator --timeout=240s || {
  kubectl --kubeconfig="${KUBECONFIG}" -n default logs deployment/aws-emulator
  exit 1
}
