#!/usr/bin/env bash
#
# SPDX-FileCopyrightText: Contributors to the Gardener project
#
# SPDX-License-Identifier: Apache-2.0

# Check if all required AWS environment variables are set
if [[ -z "${AWS_ACCESS_KEY_ID:-}" || -z "${AWS_SECRET_ACCESS_KEY:-}" || -z "${AWS_DEFAULT_REGION:-}" ]]; then
  if [[ -n "${AWS_ACCESS_KEY_ID:-}" || -n "${AWS_SECRET_ACCESS_KEY:-}" || -n "${AWS_DEFAULT_REGION:-}" ]]; then
    echo "Error: Partial AWS environment variables set. Please set all or none of AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, and AWS_DEFAULT_REGION."
    exit 1
  fi
fi

export AWS_APPLICATION_CREDENTIALS="/tmp/.aws"

rm -rf /tmp/.aws
mkdir -p /tmp/.aws

if [[ -z "${AWS_ACCESS_KEY_ID:-}" ]]; then
  export AWS_ENDPOINT_OVERRIDE="https://aws-emulator.default:9297"
  export AWS_ENDPOINT_URL_S3="https://localhost:9297"
  export EMULATOR_URL="https://localhost:9297"
  export AWS_ACCESS_KEY_ID="ACCESSKEYAWSUSER"
  export AWS_SECRET_ACCESS_KEY="sEcreTKey"
  export AWS_DEFAULT_REGION="us-east-2"
  echo -n "true" > "${AWS_APPLICATION_CREDENTIALS}/s3ForcePathStyle"
  echo -n "true" > "${AWS_APPLICATION_CREDENTIALS}/insecureSkipVerify"
fi

echo -n "${AWS_ACCESS_KEY_ID}"     > "${AWS_APPLICATION_CREDENTIALS}/accessKeyID"
echo -n "${AWS_SECRET_ACCESS_KEY}" > "${AWS_APPLICATION_CREDENTIALS}/secretAccessKey"
echo -n "${AWS_DEFAULT_REGION}"    > "${AWS_APPLICATION_CREDENTIALS}/region"
