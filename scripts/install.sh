#!/bin/sh
#
# install.sh — dflow installer for Linux and macOS.
#
# Downloads the latest dflow release (or a pinned version) from GitHub,
# verifies it against the published SHA-256 checksums, and installs the
# binary into a user-writable directory. It never uses sudo.
#
# Usage:
#   curl -fsSL https://github.com/yepizrene-devoost/dflow/releases/latest/download/install.sh | sh
#   ./scripts/install.sh --help
#
# Environment:
#   DFLOW_VERSION      Pin a version instead of the latest release.
#                      Accepts both "0.2.0" and "v0.2.0".
#   DFLOW_INSTALL_DIR  Install directory. Default: $HOME/.local/bin
#
# Options:
#   -h, --help         Show this help and exit.
#
set -eu

REPO="yepizrene-devoost/dflow"
PROJECT="dflow"
BINARY="dflow"
RELEASES_URL="https://github.com/${REPO}/releases"
LATEST_URL="https://github.com/${REPO}/releases/latest"
API_LATEST_URL="https://api.github.com/repos/${REPO}/releases/latest"

TMP_DIR=""
STAGE_FILE=""
DOWNLOADER=""
SHA_TOOL=""
OS=""
ARCH=""
TAG=""
VERSION=""
INSTALL_DIR=""

# --- Output helpers ---------------------------------------------------------

info() {
    printf '  %s\n' "$*"
}

warn() {
    printf '⚠️  %s\n' "$*" >&2
}

fail() {
    printf '❌ %s\n' "$*" >&2
    exit 1
}

usage() {
    cat <<'EOF'
dflow installer — Linux and macOS

Usage:
  curl -fsSL https://github.com/yepizrene-devoost/dflow/releases/latest/download/install.sh | sh
  ./install.sh [--help]

The installer downloads the release archive and its SHA-256 checksums,
verifies the archive before touching your system, and installs the dflow
binary without sudo.

Environment variables:
  DFLOW_VERSION      Pin a version instead of the latest release.
                     Accepts both "0.2.0" and "v0.2.0".
  DFLOW_INSTALL_DIR  Where to install the binary.
                     Default: $HOME/.local/bin

Options:
  -h, --help         Show this help and exit.
EOF
}

# --- Cleanup ----------------------------------------------------------------

cleanup() {
    if [ -n "$STAGE_FILE" ] && [ -e "$STAGE_FILE" ]; then
        rm -f "$STAGE_FILE"
    fi
    if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
        rm -rf "$TMP_DIR"
    fi
}

trap cleanup EXIT
trap 'exit 1' INT TERM HUP

# --- Environment checks -----------------------------------------------------

detect_downloader() {
    if command -v curl >/dev/null 2>&1; then
        DOWNLOADER="curl"
    elif command -v wget >/dev/null 2>&1; then
        DOWNLOADER="wget"
    else
        fail "I need curl or wget to download dflow, and found neither. Please install one and try again."
    fi
}

detect_platform() {
    _uname_s="$(uname -s)"
    _uname_m="$(uname -m)"

    case "$_uname_s" in
        Linux)
            OS="Linux"
            ;;
        Darwin)
            OS="Darwin"
            ;;
        MINGW*|MSYS*|CYGWIN*|Windows_NT)
            fail "This installer is for Linux and macOS. On Windows, use the PowerShell installer (install.ps1) instead."
            ;;
        *)
            fail "Unsupported operating system: ${_uname_s}. dflow ships prebuilt binaries for Linux and macOS only."
            ;;
    esac

    case "$_uname_m" in
        x86_64|amd64)
            ARCH="x86_64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            fail "Unsupported architecture: ${_uname_m}. dflow ships x86_64 and arm64 builds only."
            ;;
    esac
}

detect_sha_tool() {
    case "$OS" in
        Linux)
            if command -v sha256sum >/dev/null 2>&1; then
                SHA_TOOL="sha256sum"
            elif command -v shasum >/dev/null 2>&1; then
                SHA_TOOL="shasum"
            else
                fail "I need sha256sum (or shasum) to verify the download, and found neither."
            fi
            ;;
        Darwin)
            if command -v shasum >/dev/null 2>&1; then
                SHA_TOOL="shasum"
            elif command -v sha256sum >/dev/null 2>&1; then
                SHA_TOOL="sha256sum"
            else
                fail "I need shasum (or sha256sum) to verify the download, and found neither."
            fi
            ;;
    esac
}

resolve_install_dir() {
    if [ -n "${DFLOW_INSTALL_DIR:-}" ]; then
        INSTALL_DIR="$DFLOW_INSTALL_DIR"
    elif [ -n "${HOME:-}" ]; then
        INSTALL_DIR="$HOME/.local/bin"
    else
        fail "HOME is not set and DFLOW_INSTALL_DIR is empty. Set DFLOW_INSTALL_DIR to pick an install directory."
    fi

    # Keep it absolute so the PATH check below is meaningful.
    case "$INSTALL_DIR" in
        /*) ;;
        *) INSTALL_DIR="$(pwd)/$INSTALL_DIR" ;;
    esac
}

# --- Download helpers -------------------------------------------------------

download() {
    _dl_url="$1"
    _dl_out="$2"

    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fsSL -o "$_dl_out" "$_dl_url"
    else
        wget -qO "$_dl_out" "$_dl_url"
    fi
}

download_to_stdout() {
    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fsSL "$1"
    else
        wget -qO- "$1"
    fi
}

normalize_tag() {
    case "$1" in
        v*) printf '%s' "$1" ;;
        *) printf 'v%s' "$1" ;;
    esac
}

resolve_latest_tag() {
    if [ "$DOWNLOADER" = "curl" ]; then
        _tag_url="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "$LATEST_URL")" || return 1
        printf '%s' "${_tag_url##*/}"
        return 0
    fi

    # wget: read the final Location header of the /releases/latest redirect.
    _tag_url="$(wget -qS --spider "$LATEST_URL" 2>&1 | sed -n 's/[[:space:]]*[Ll]ocation:[[:space:]]*//p' | tail -n 1 | tr -d '\r')"

    if [ -z "$_tag_url" ]; then
        # Fallback: ask the GitHub API, which also works behind proxies.
        _tag_url="$(download_to_stdout "$API_LATEST_URL" 2>/dev/null | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
    fi

    printf '%s' "${_tag_url##*/}"
}

compute_sha256() {
    if [ "$SHA_TOOL" = "sha256sum" ]; then
        sha256sum "$1" | awk '{print $1}'
    else
        shasum -a 256 "$1" | awk '{print $1}'
    fi
}

# --- Main -------------------------------------------------------------------

parse_args() {
    for _arg in "$@"; do
        case "$_arg" in
            -h|--help)
                usage
                exit 0
                ;;
            *)
                fail "Unknown option: ${_arg} (try --help)"
                ;;
        esac
    done
}

main() {
    parse_args "$@"
    detect_downloader
    detect_platform
    detect_sha_tool
    resolve_install_dir

    printf '\n📥 Installing dflow for %s/%s\n\n' "$OS" "$ARCH"
    info "🏠 Install directory: ${INSTALL_DIR}"

    if [ -n "${DFLOW_VERSION:-}" ]; then
        case "$DFLOW_VERSION" in
            */*|*..*)
                fail "Invalid DFLOW_VERSION: '${DFLOW_VERSION}'."
                ;;
        esac
        TAG="$(normalize_tag "$DFLOW_VERSION")"
        info "🔖 Pinned version: ${TAG}"
    else
        info "🔎 Looking up the latest release..."
        TAG="$(resolve_latest_tag)" || fail "Could not reach GitHub to find the latest release. Check your connection (or pin one with DFLOW_VERSION)."
        if [ -z "$TAG" ]; then
            fail "Could not resolve the latest release tag. Pin one with DFLOW_VERSION (for example DFLOW_VERSION=v0.2.0)."
        fi
        info "🔖 Latest version: ${TAG}"
    fi

    case "$TAG" in
        v[0-9]*)
            ;;
        *)
            fail "Unexpected release tag: '${TAG}'. Pin a valid version with DFLOW_VERSION (for example DFLOW_VERSION=v0.2.0)."
            ;;
    esac

    VERSION="${TAG#v}"
    ASSET="${PROJECT}_${OS}_${ARCH}.tar.gz"
    CHECKSUMS="${PROJECT}_${VERSION}_checksums.txt"
    BASE_URL="${RELEASES_URL}/download/${TAG}"

    TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/dflow-install.XXXXXX")" || fail "Could not create a temporary directory."

    printf '\n'
    info "⬇️  Downloading ${ASSET}"
    download "${BASE_URL}/${ASSET}" "${TMP_DIR}/${ASSET}" \
        || fail "Download failed: ${BASE_URL}/${ASSET}"
    info "⬇️  Downloading ${CHECKSUMS}"
    download "${BASE_URL}/${CHECKSUMS}" "${TMP_DIR}/${CHECKSUMS}" \
        || fail "Download failed: ${BASE_URL}/${CHECKSUMS}"

    info "🔐 Verifying checksum..."
    _expected="$(awk -v name="$ASSET" '$2 == name { print $1; exit }' "${TMP_DIR}/${CHECKSUMS}")"
    if [ -z "$_expected" ]; then
        fail "No checksum entry for ${ASSET} in ${CHECKSUMS}. The release looks incomplete, so I will not install it."
    fi

    _actual="$(compute_sha256 "${TMP_DIR}/${ASSET}")"
    if [ "$_expected" != "$_actual" ]; then
        fail "Checksum mismatch for ${ASSET}!
    expected: ${_expected}
    actual:   ${_actual}
  Refusing to install a download that does not match the published checksum."
    fi
    info "✅ Checksum verified."

    info "📦 Extracting archive..."
    tar -xzf "${TMP_DIR}/${ASSET}" -C "$TMP_DIR" \
        || fail "Could not extract ${ASSET}."
    if [ ! -f "${TMP_DIR}/${BINARY}" ]; then
        fail "The archive did not contain a '${BINARY}' binary."
    fi

    info "🚚 Installing to ${INSTALL_DIR}/${BINARY}..."
    mkdir -p "$INSTALL_DIR" || fail "Could not create ${INSTALL_DIR}."

    # Stage next to the target, then rename, so the swap is atomic and an
    # existing binary is never left half-written.
    STAGE_FILE="${INSTALL_DIR}/.${BINARY}.tmp.$$"
    cp "${TMP_DIR}/${BINARY}" "$STAGE_FILE" || fail "Could not write to ${INSTALL_DIR}."
    chmod 755 "$STAGE_FILE"
    mv -f "$STAGE_FILE" "${INSTALL_DIR}/${BINARY}" || fail "Could not replace ${INSTALL_DIR}/${BINARY}."
    STAGE_FILE=""

    printf '\n🎉 dflow %s installed at %s\n' "$VERSION" "${INSTALL_DIR}/${BINARY}"

    case ":${PATH:-}:" in
        *":${INSTALL_DIR}:"*)
            printf '👉 Run dflow version to confirm, then dflow --help to get started.\n'
            ;;
        *)
            printf '\n'
            warn "${INSTALL_DIR} is not on your PATH yet."
            printf '   Add it to your shell profile:\n\n'
            printf '     export PATH="%s:$PATH"\n' "$INSTALL_DIR"
            printf '\n   Then reload it. Common profiles:\n'
            printf '     bash   ~/.bashrc  (or ~/.bash_profile on macOS)\n'
            printf '     zsh    ~/.zshrc\n'
            printf '     fish   fish_add_path %s\n' "$INSTALL_DIR"
            printf '\n👉 Then run dflow version to confirm the install.\n'
            ;;
    esac
}

main "$@"
