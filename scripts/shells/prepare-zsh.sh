#!/bin/sh
set -eu

if [ "$#" -eq 0 ]; then
    echo "usage: $0 VERSION..." >&2
    exit 2
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH= cd -- "$script_dir/../.." && pwd -P)
family_root="$repo_root/bin/shells/zsh"
work_root=$(mktemp -d "${TMPDIR:-/tmp}/commandment-zsh.XXXXXX")
trap 'rm -rf -- "$work_root"' EXIT HUP INT TERM
mkdir -p "$family_root"

probe() {
    executable=$1
    marker=$2
    [ -x "$executable" ] || return 1
    "$executable" --version 2>&1 | grep -F "$marker" >/dev/null
}

download() {
    url=$1
    sha256=$2
    archive=$3
    curl -fsSL "$url" -o "$archive"
    printf '%s  %s\n' "$sha256" "$archive" | sha256sum -c - >/dev/null
}

ensure_zsh() {
    version=$1
    url=$2
    sha256=$3
    archive_kind=$4
    standard=$5
    marker=$6
    legacy=$7
    final="$family_root/$version"
    executable="$final/bin/zsh"
    if probe "$executable" "$marker"; then
        echo "zsh $version: existing" >&2
        printf '%s\n' "$executable"
        return
    fi

    echo "zsh $version: building" >&2
    work="$work_root/zsh-$version"
    source="$work/source"
    install="$work/install"
    archive="$work/source.$archive_kind"
    mkdir -p "$source" "$install"
    download "$url" "$sha256" "$archive"
    case "$archive_kind" in
        tar.gz) tar -xzf "$archive" -C "$source" --strip-components=1 ;;
        tar.xz) tar -xJf "$archive" -C "$source" --strip-components=1 ;;
        *) echo "zsh $version: unsupported archive kind" >&2; exit 1 ;;
    esac
    (
        cd "$source"
        if [ "$legacy" = 1 ]; then
            CPPFLAGS="-D_POSIX_C_SOURCE=200112L -D_XOPEN_SOURCE=600" CFLAGS="-std=$standard" ./configure --prefix="$install" >&2
            make CPPFLAGS="-D_POSIX_C_SOURCE=200112L -D_XOPEN_SOURCE=600" CFLAGS="-std=$standard" >&2
            make CPPFLAGS="-D_POSIX_C_SOURCE=200112L -D_XOPEN_SOURCE=600" CFLAGS="-std=$standard" install >&2
        else
            CFLAGS="-std=$standard" ./configure --prefix="$install" >&2
            make >&2
            make install >&2
        fi
    )
    probe "$install/bin/zsh" "$marker" || { echo "zsh $version: sanity probe failed" >&2; exit 1; }
    rm -rf -- "$final"
    mv -- "$install" "$final"
    printf '%s\n' "$executable"
}

for version in "$@"; do
    case "$version" in
        4.0.1)
            ensure_zsh 4.0.1 https://www.zsh.org/pub/old/zsh-4.0.1.tar.gz 86d71aff300075ca36a7f7ba6646275413736f52fdc6083fb33ca2ee07276c93 tar.gz gnu89 "zsh 4.0.1" 1
            ;;
        5.9.2)
            ensure_zsh 5.9.2 https://www.zsh.org/pub/zsh-5.9.2.tar.xz 36fa734374b44783582cec09bcd67822e2f992c779ec1624ab5596df078d2f81 tar.xz gnu17 "zsh 5.9.2" 0
            ;;
        *)
            echo "zsh $version: unsupported completion runtime version" >&2
            exit 2
            ;;
    esac
done
