#!/usr/bin/env bash
set -euo pipefail
# Official release SHA256, retained and independently reviewed with this source.
version=1.9.8
sha=186e0145f5e5f2eb97cbd785bc78f21bae4ef15119349f6ad4fa535b83b10df8
[[ $(uname -s) == Linux && $(uname -m) == x86_64 ]]
: "${RUNNER_TEMP:?}" "${GITHUB_PATH:?}"
dir=$(mktemp -d "$RUNNER_TEMP/terraform-verified.XXXXXX")
curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 "https://releases.hashicorp.com/terraform/${version}/terraform_${version}_linux_amd64.zip" -o "$dir/terraform.zip"
printf '%s  %s\n' "$sha" "$dir/terraform.zip" | sha256sum --check --strict
unzip -q "$dir/terraform.zip" -d "$dir"
printf '%s\n' "$dir" >> "$GITHUB_PATH"
