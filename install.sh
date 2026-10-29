#!/usr/bin/env bash
set -euo pipefail

REPO_OWNER="liuwentao"
REPO_NAME="xray-installer"
REPO_SLUG="${REPO_OWNER}/${REPO_NAME}"
INSTALL_PATH="/usr/local/bin/xray-installer"

usage() {
  cat <<'EOF'
xray-installer bootstrap script

Usage:
  install.sh install [--version vX.Y.Z] [--no-run] [-- xray-installer-args...]
  install.sh uninstall [-- xray-installer uninstall args...]
  install.sh update [--xray-version vX.Y.Z] [--dry-run]
  install.sh help

Examples:
  bash <(curl -fsSL https://raw.githubusercontent.com/liuwentao/xray-installer/main/install.sh) install
  bash <(curl -fsSL https://raw.githubusercontent.com/liuwentao/xray-installer/main/install.sh) install -- --dry-run
  bash <(curl -fsSL https://raw.githubusercontent.com/liuwentao/xray-installer/main/install.sh) uninstall
  bash <(curl -fsSL https://raw.githubusercontent.com/liuwentao/xray-installer/main/install.sh) update
EOF
}

log() {
  printf '%s\n' "$*"
}

fail() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

find_checksum_line() {
  local checksums_file="$1"
  local asset_name="$2"
  awk -v name="$asset_name" '$2 == name || $2 == "./" name {print; exit}' "$checksums_file"
}

detect_os() {
  local os
  os="$(uname -s)"
  case "$os" in
    Linux) echo "linux" ;;
    *) fail "unsupported operating system: $os (only Linux is supported)" ;;
  esac
}

detect_arch() {
  local arch
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) fail "unsupported architecture: $arch (supported: amd64, arm64)" ;;
  esac
}

sudo_prefix() {
  if [[ "${EUID}" -eq 0 ]]; then
    return 1
  fi
  if command -v sudo >/dev/null 2>&1; then
    return 0
  fi
  fail "please run as root or install sudo"
}

install_binary() {
  local version_ref="$1"
  local run_after_install="$2"
  shift 2
  local forward_args=("$@")

  need_cmd curl
  need_cmd tar
  need_cmd install

  local os arch asset_name release_base tmpdir checksum_url asset_url
  os="$(detect_os)"
  arch="$(detect_arch)"
  asset_name="${REPO_NAME}_${os}_${arch}.tar.gz"

  if [[ -n "$version_ref" ]]; then
    release_base="https://github.com/${REPO_SLUG}/releases/download/${version_ref}"
  else
    release_base="https://github.com/${REPO_SLUG}/releases/latest/download"
  fi

  checksum_url="${release_base}/checksums.txt"
  asset_url="${release_base}/${asset_name}"

  tmpdir="$(mktemp -d)"
  trap "rm -rf -- $(printf '%q' "$tmpdir")" EXIT

  log "==> Downloading ${asset_name}"
  if ! curl -fsSL "${asset_url}" -o "${tmpdir}/${asset_name}"; then
    fail "failed to download ${asset_url}. Ensure a GitHub Release exists for this project."
  fi

  log "==> Verifying checksum"
  if curl -fsSL "${checksum_url}" -o "${tmpdir}/checksums.txt"; then
    local checksum_line
    checksum_line="$(find_checksum_line "${tmpdir}/checksums.txt" "${asset_name}")"
    [[ -n "${checksum_line}" ]] || fail "checksum entry not found for ${asset_name}"

    if command -v sha256sum >/dev/null 2>&1; then
      (
        cd "$tmpdir"
        printf '%s\n' "${checksum_line}" | sha256sum -c -
      ) >/dev/null
    elif command -v shasum >/dev/null 2>&1; then
      local expected actual
      expected="$(printf '%s\n' "${checksum_line}" | awk '{print $1}')"
      actual="$(shasum -a 256 "${tmpdir}/${asset_name}" | awk '{print $1}')"
      [[ -n "$expected" && "$expected" == "$actual" ]] || fail "checksum verification failed for ${asset_name}"
    else
      fail "missing checksum tool: sha256sum or shasum"
    fi
  else
    fail "failed to download checksums from ${checksum_url}"
  fi

  log "==> Extracting package"
  tar -xzf "${tmpdir}/${asset_name}" -C "$tmpdir"
  [[ -f "${tmpdir}/${REPO_NAME}" ]] || fail "archive did not contain ${REPO_NAME}"

  log "==> Installing to ${INSTALL_PATH}"
  if sudo_prefix; then
    sudo install -m 0755 "${tmpdir}/${REPO_NAME}" "${INSTALL_PATH}"
  else
    install -m 0755 "${tmpdir}/${REPO_NAME}" "${INSTALL_PATH}"
  fi

  if [[ "$run_after_install" == "false" ]]; then
    log "Installed ${INSTALL_PATH}"
    return 0
  fi

  log "==> Launching xray-installer"
  if sudo_prefix; then
    sudo "${INSTALL_PATH}" "${forward_args[@]}"
  else
    "${INSTALL_PATH}" "${forward_args[@]}"
  fi
}

run_uninstall() {
  local forward_args=("$@")

  if [[ ! -x "${INSTALL_PATH}" ]]; then
    log "xray-installer is not installed at ${INSTALL_PATH}; nothing to do."
    return 0
  fi

  log "==> Running xray-installer uninstall"
  if sudo_prefix; then
    sudo "${INSTALL_PATH}" uninstall "${forward_args[@]}"
  else
    "${INSTALL_PATH}" uninstall "${forward_args[@]}"
  fi

  local is_dry_run="false"
  for arg in "${forward_args[@]}"; do
    if [[ "$arg" == "--dry-run" ]]; then
      is_dry_run="true"
      break
    fi
  done

  if [[ "$is_dry_run" == "true" ]]; then
    return 0
  fi

  log "==> Removing ${INSTALL_PATH}"
  if sudo_prefix; then
    sudo rm -f "${INSTALL_PATH}"
  else
    rm -f "${INSTALL_PATH}"
  fi
}

run_update() {
  [[ -x "${INSTALL_PATH}" ]] || fail "xray-installer is not installed at ${INSTALL_PATH}; install it first"
  if sudo_prefix; then
    sudo "${INSTALL_PATH}" update "$@"
  else
    "${INSTALL_PATH}" update "$@"
  fi
}

main() {
  local command="${1:-install}"
  shift || true

  case "$command" in
    install)
      local version_ref=""
      local run_after_install="true"
      local forward_args=()
      while [[ $# -gt 0 ]]; do
        case "$1" in
          --version)
            [[ $# -ge 2 ]] || fail "--version requires a value like v0.1.0"
            version_ref="$2"
            shift 2
            ;;
          --no-run)
            run_after_install="false"
            shift
            ;;
          --help|-h)
            usage
            exit 0
            ;;
          --)
            shift
            forward_args=("$@")
            break
            ;;
          *)
            forward_args+=("$1")
            shift
            ;;
        esac
      done
      install_binary "$version_ref" "$run_after_install" "${forward_args[@]}"
      ;;
    update)
      run_update "$@"
      ;;
    uninstall)
      run_uninstall "$@"
      ;;
    help|--help|-h)
      usage
      ;;
    *)
      fail "unknown command: ${command}"
      ;;
  esac
}

main "$@"
