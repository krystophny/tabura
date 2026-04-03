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
  tmpdir="$(mktemp -d -t slopshell-dist-test-XXXXXX)"
  trap "rm -rf '$tmpdir'" EXIT

  checksums="${tmpdir}/checksums.txt"
  output_dir="${tmpdir}/out"

  cat > "${checksums}" <<'EOF'
1111111111111111111111111111111111111111111111111111111111111111  slopshell_1.2.3_linux_amd64.tar.gz
2222222222222222222222222222222222222222222222222222222222222222  slopshell_1.2.3_linux_arm64.tar.gz
3333333333333333333333333333333333333333333333333333333333333333  slopshell_1.2.3_darwin_amd64.tar.gz
4444444444444444444444444444444444444444444444444444444444444444  slopshell_1.2.3_darwin_arm64.tar.gz
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  slopshell_1.2.3_windows_amd64.zip
bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  slopshell_1.2.3_windows_arm64.zip
EOF

  "${ROOT_DIR}/scripts/generate-package-manager-artifacts.sh" \
    --version v1.2.3 \
    --checksums "${checksums}" \
    --output-dir "${output_dir}"

  assert_contains "${output_dir}/homebrew/Formula/slopshell.rb" 'version "1.2.3"'
  assert_contains "${output_dir}/homebrew/Formula/slopshell.rb" 'slopshell_1.2.3_linux_amd64.tar.gz'
  assert_contains "${output_dir}/homebrew/Formula/slopshell.rb" 'sha256 "1111111111111111111111111111111111111111111111111111111111111111"'
  assert_contains "${output_dir}/homebrew/Formula/slopshell.rb" "Run 'slopshell server' or use the full installer:"

  assert_contains "${output_dir}/aur/PKGBUILD" 'pkgver=1.2.3'
  assert_contains "${output_dir}/aur/PKGBUILD" 'source_x86_64=("https://github.com/sloppy-org/slopshell/releases/download/v1.2.3/slopshell_1.2.3_linux_amd64.tar.gz")'
  assert_contains "${output_dir}/aur/PKGBUILD" "sha256sums_aarch64=('2222222222222222222222222222222222222222222222222222222222222222')"
  assert_contains "${output_dir}/aur/PKGBUILD" "voxtype: speech-to-text sidecar"

  assert_contains "${output_dir}/winget/manifests/s/sloppy-org/slopshell/1.2.3/sloppy-org.slopshell.yaml" 'PackageVersion: 1.2.3'
  assert_contains "${output_dir}/winget/manifests/s/sloppy-org/slopshell/1.2.3/sloppy-org.slopshell.installer.yaml" 'InstallerUrl: https://github.com/sloppy-org/slopshell/releases/download/v1.2.3/slopshell_1.2.3_windows_amd64.zip'
  assert_contains "${output_dir}/winget/manifests/s/sloppy-org/slopshell/1.2.3/sloppy-org.slopshell.installer.yaml" 'InstallerSha256: AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA'
  assert_contains "${output_dir}/winget/manifests/s/sloppy-org/slopshell/1.2.3/sloppy-org.slopshell.installer.yaml" 'InstallerSha256: BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB'

  echo "distribution artifact tests passed"
}

main "$@"
