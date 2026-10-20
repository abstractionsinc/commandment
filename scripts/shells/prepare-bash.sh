#!/bin/sh
set -eu

if [ "$#" -eq 0 ]; then
    echo "usage: $0 VERSION..." >&2
    exit 2
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH= cd -- "$script_dir/../.." && pwd -P)
family_root="$repo_root/bin/shells/bash"
work_root=$(mktemp -d "${TMPDIR:-/tmp}/commandment-bash.XXXXXX")
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

ensure_archive() {
    version=$1
    url=$2
    sha256=$3
    standard=$4
    marker=$5
    final="$family_root/$version"
    executable="$final/bin/bash"
    if probe "$executable" "$marker"; then
        echo "bash $version: existing" >&2
        printf '%s\n' "$executable"
        return
    fi

    echo "bash $version: building" >&2
    work="$work_root/bash-$version"
    source="$work/source"
    install="$work/install"
    archive="$work/source.tar.gz"
    mkdir -p "$source" "$install"
    download "$url" "$sha256" "$archive"
    tar -xzf "$archive" -C "$source" --strip-components=1
    (
        cd "$source"
        CFLAGS="-std=$standard" CFLAGS_FOR_BUILD="-std=$standard" ./configure --prefix="$install" >&2
        make CFLAGS="-std=$standard" CFLAGS_FOR_BUILD="-std=$standard" >&2
        make CFLAGS="-std=$standard" CFLAGS_FOR_BUILD="-std=$standard" install >&2
    )
    probe "$install/bin/bash" "$marker" || { echo "bash $version: sanity probe failed" >&2; exit 1; }
    rm -rf -- "$final"
    mv -- "$install" "$final"
    printf '%s\n' "$executable"
}

ensure_git() {
    version=4.0.38
    commit=89a92869e56aba4e4cab2d639c00a86f0545c862
    final="$family_root/$version"
    executable="$final/bin/bash"
    if probe "$executable" "version 4.0.38"; then
        echo "bash $version: existing" >&2
        printf '%s\n' "$executable"
        return
    fi

    echo "bash $version: building" >&2
    work="$work_root/bash-$version"
    source="$work/source"
    install="$work/install"
    mkdir -p "$work" "$install"
    git clone --quiet --no-checkout https://git.savannah.gnu.org/git/bash.git "$source"
    git -C "$source" checkout --quiet --detach "$commit"
    actual=$(git -C "$source" rev-parse HEAD)
    [ "$actual" = "$commit" ] || { echo "bash $version: source commit mismatch: $actual" >&2; exit 1; }
    (
        cd "$source"
        CFLAGS=-std=gnu89 CFLAGS_FOR_BUILD=-std=gnu89 ./configure --prefix="$install" >&2
        make CFLAGS=-std=gnu89 CFLAGS_FOR_BUILD=-std=gnu89 >&2
        make CFLAGS=-std=gnu89 CFLAGS_FOR_BUILD=-std=gnu89 install >&2
    )
    probe "$install/bin/bash" "version 4.0.38" || { echo "bash $version: sanity probe failed" >&2; exit 1; }
    rm -rf -- "$final"
    mv -- "$install" "$final"
    printf '%s\n' "$executable"
}

for version in "$@"; do
    case "$version" in
        3.2.57)
            ensure_archive 3.2.57 https://ftp.gnu.org/gnu/bash/bash-3.2.57.tar.gz 3fa9daf85ebf35068f090ce51283ddeeb3c75eb5bc70b1a4a7cb05868bfe06a4 gnu89 "version 3.2.57"
            ;;
        4.0.38)
            ensure_git
            ;;
        4.4.0)
            ensure_archive 4.4.0 https://ftp.gnu.org/gnu/bash/bash-4.4.tar.gz d86b3392c1202e8ff5a423b302e6284db7f8f435ea9f39b5b1b20fd3ac36dfcb gnu89 "version 4.4"
            ;;
        5.3.0)
            ensure_archive 5.3.0 https://ftp.gnu.org/gnu/bash/bash-5.3.tar.gz 0d5cd86965f869a26cf64f4b71be7b96f90a3ba8b3d74e27e8e9d9d5550f31ba gnu17 "version 5.3"
            ;;
        *)
            echo "bash $version: unsupported completion runtime version" >&2
            exit 2
            ;;
    esac
done
