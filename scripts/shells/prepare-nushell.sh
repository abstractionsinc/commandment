#!/bin/sh
set -eu

if [ "$#" -eq 0 ]; then
    echo "usage: $0 VERSION..." >&2
    exit 2
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH= cd -- "$script_dir/../.." && pwd -P)
family_root="$repo_root/bin/shells/nushell"
work_root=$(mktemp -d "${TMPDIR:-/tmp}/commandment-nushell.XXXXXX")
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

ensure_git() {
    version=0.72.0
    commit=8fa965118c2e5a0beaa528d07eaf9b33d96779cb
    final="$family_root/$version"
    executable="$final/nu"
    if probe "$executable" "$version"; then
        echo "nushell $version: existing" >&2
        printf '%s\n' "$executable"
        return
    fi

    echo "nushell $version: building" >&2
    work="$work_root/nushell-$version"
    source="$work/source"
    install="$work/install"
    mkdir -p "$work" "$install"
    git clone --quiet --no-checkout https://github.com/nushell/nushell.git "$source"
    git -C "$source" checkout --quiet --detach "$commit"
    actual=$(git -C "$source" rev-parse HEAD)
    [ "$actual" = "$commit" ] || { echo "nushell $version: source commit mismatch: $actual" >&2; exit 1; }
    (
        cd "$source"
        cargo build --release --locked --bin nu >&2
    )
    cp "$source/target/release/nu" "$install/nu"
    chmod +x "$install/nu"
    probe "$install/nu" "$version" || { echo "nushell $version: sanity probe failed" >&2; exit 1; }
    rm -rf -- "$final"
    mv -- "$install" "$final"
    printf '%s\n' "$executable"
}

ensure_archive() {
    version=$1
    url=$2
    sha256=$3
    final="$family_root/$version"
    executable="$final/nu"
    if probe "$executable" "$version"; then
        echo "nushell $version: existing" >&2
        printf '%s\n' "$executable"
        return
    fi

    echo "nushell $version: installing" >&2
    work="$work_root/nushell-$version"
    unpack="$work/unpack"
    install="$work/install"
    archive="$work/nushell.tar.gz"
    mkdir -p "$unpack" "$install"
    download "$url" "$sha256" "$archive"
    tar -xzf "$archive" -C "$unpack"
    candidate=$(find "$unpack" -type f -name nu -print | head -n 1)
    [ -n "$candidate" ] || { echo "nushell $version: archive contained no nu executable" >&2; exit 1; }
    cp "$candidate" "$install/nu"
    chmod +x "$install/nu"
    probe "$install/nu" "$version" || { echo "nushell $version: sanity probe failed" >&2; exit 1; }
    rm -rf -- "$final"
    mv -- "$install" "$final"
    printf '%s\n' "$executable"
}

for version in "$@"; do
    case "$version" in
        0.72.0)
            ensure_git
            ;;
        0.102.0)
            ensure_archive 0.102.0 https://github.com/nushell/nushell/releases/download/0.102.0/nu-0.102.0-x86_64-unknown-linux-gnu.tar.gz 8facc8575dc5cc1406d5f00625faf40556da986f8932e90c8f891241df7275a3
            ;;
        0.108.0)
            ensure_archive 0.108.0 https://github.com/nushell/nushell/releases/download/0.108.0/nu-0.108.0-x86_64-unknown-linux-gnu.tar.gz f0daaa0256b0f7dd5f4d9925b4d5522d8de5e947ed9ad64a43526e5cffeb05df
            ;;
        0.113.0)
            ensure_archive 0.113.0 https://github.com/nushell/nushell/releases/download/0.113.0/nu-0.113.0-x86_64-unknown-linux-gnu.tar.gz 68c0f84254b23ed86d1582797d32f73db0969fa39297b3cc339656e634663468
            ;;
        0.115.1)
            ensure_archive 0.115.1 https://github.com/nushell/nushell/releases/download/0.115.1/nu-0.115.1-x86_64-unknown-linux-gnu.tar.gz d11d825241f6504a3617c535fa725a9dd6d009c86d7b19fb3168b47635b9d8b0
            ;;
        *)
            echo "nushell $version: unsupported completion runtime version" >&2
            exit 2
            ;;
    esac
done
