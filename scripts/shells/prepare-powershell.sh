#!/bin/sh
set -eu

if [ "$#" -eq 0 ]; then
    echo "usage: $0 VERSION..." >&2
    exit 2
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH= cd -- "$script_dir/../.." && pwd -P)
family_root="$repo_root/bin/shells/powershell"
work_root=$(mktemp -d "${TMPDIR:-/tmp}/commandment-powershell.XXXXXX")
trap 'rm -rf -- "$work_root"' EXIT HUP INT TERM
mkdir -p "$family_root"

probe() {
    executable=$1
    version=$2
    [ -x "$executable" ] || return 1
    "$executable" -NoLogo -NoProfile -NonInteractive -Command '$PSVersionTable.PSVersion.ToString()' 2>&1 | grep -F "$version" >/dev/null
}

ensure_powershell() {
    version=7.6.6
    final="$family_root/$version"
    executable="$final/pwsh"
    if probe "$executable" "$version"; then
        echo "powershell $version: existing" >&2
        printf '%s\n' "$executable"
        return
    fi

    echo "powershell $version: installing" >&2
    work="$work_root/powershell-$version"
    install="$work/install"
    archive="$work/powershell.tar.gz"
    mkdir -p "$install"
    curl -fsSL https://github.com/PowerShell/PowerShell/releases/download/v7.6.6/powershell-7.6.6-linux-x64.tar.gz -o "$archive"
    printf '%s  %s\n' ddbc4a2d113bbd46d283cfedcbcd117a70caefd7673f41f2b4e0000badf103bc "$archive" | sha256sum -c - >/dev/null
    tar -xzf "$archive" -C "$install"
    chmod +x "$install/pwsh"
    probe "$install/pwsh" "$version" || { echo "powershell $version: sanity probe failed" >&2; exit 1; }
    rm -rf -- "$final"
    mv -- "$install" "$final"
    printf '%s\n' "$executable"
}

for version in "$@"; do
    case "$version" in
        7.6.6) ensure_powershell ;;
        *)
            echo "powershell $version: unsupported prepared completion runtime version" >&2
            exit 2
            ;;
    esac
done
