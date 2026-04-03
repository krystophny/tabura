#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Fq "$pattern" "$file"; then
    echo "assertion failed: expected '$pattern' in $file" >&2
    exit 1
  fi
}

main() {
  local tmpdir checksums output_dir
  tmpdir="$(mktemp -d -t sloppad-dist-test-XXXXXX)"
  trap "rm -rf '$tmpdir'" EXIT

  checksums="${tmpdir}/checksums.txt"
  output_dir="${tmpdir}/out"

  cat > "${checksums}" <<'EOF'
1111111111111111111111111111111111111111111111111111111111111111  sloppad_1.2.3_linux_amd64.tar.gz
2222222222222222222222222222222222222222222222222222222222222222  sloppad_1.2.3_linux_arm64.tar.gz
3333333333333333333333333333333333333333333333333333333333333333  sloppad_1.2.3_darwin_amd64.tar.gz
4444444444444444444444444444444444444444444444444444444444444444  sloppad_1.2.3_darwin_arm64.tar.gz
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  sloppad_1.2.3_windows_amd64.zip
bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  sloppad_1.2.3_windows_arm64.zip
EOF

  "${ROOT_DIR}/scripts/generate-package-manager-artifacts.sh" \
    --version v1.2.3 \
    --checksums "${checksums}" \
    --output-dir "${output_dir}"

  assert_contains "${output_dir}/homebrew/Formula/sloppad.rb" 'version "1.2.3"'
  assert_contains "${output_dir}/homebrew/Formula/sloppad.rb" 'sloppad_1.2.3_linux_amd64.tar.gz'
  assert_contains "${output_dir}/homebrew/Formula/sloppad.rb" 'sha256 "1111111111111111111111111111111111111111111111111111111111111111"'
  assert_contains "${output_dir}/homebrew/Formula/sloppad.rb" "Run 'sloppad server' or use the full installer:"

  assert_contains "${output_dir}/aur/PKGBUILD" 'pkgver=1.2.3'
  assert_contains "${output_dir}/aur/PKGBUILD" 'source_x86_64=("https://github.com/krystophny/sloppad/releases/download/v1.2.3/sloppad_1.2.3_linux_amd64.tar.gz")'
  assert_contains "${output_dir}/aur/PKGBUILD" "sha256sums_aarch64=('2222222222222222222222222222222222222222222222222222222222222222')"
  assert_contains "${output_dir}/aur/PKGBUILD" "voxtype: speech-to-text sidecar"

  assert_contains "${output_dir}/winget/manifests/k/krystophny/sloppad/1.2.3/krystophny.sloppad.yaml" 'PackageVersion: 1.2.3'
  assert_contains "${output_dir}/winget/manifests/k/krystophny/sloppad/1.2.3/krystophny.sloppad.installer.yaml" 'InstallerUrl: https://github.com/krystophny/sloppad/releases/download/v1.2.3/sloppad_1.2.3_windows_amd64.zip'
  assert_contains "${output_dir}/winget/manifests/k/krystophny/sloppad/1.2.3/krystophny.sloppad.installer.yaml" 'InstallerSha256: AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA'
  assert_contains "${output_dir}/winget/manifests/k/krystophny/sloppad/1.2.3/krystophny.sloppad.installer.yaml" 'InstallerSha256: BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB'

  echo "distribution artifact tests passed"
}

main "$@"
