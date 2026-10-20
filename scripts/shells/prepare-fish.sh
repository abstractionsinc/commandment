#!/bin/sh
set -eu

if [ "$#" -eq 0 ]; then
    echo "usage: $0 VERSION..." >&2
    exit 2
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH= cd -- "$script_dir/../.." && pwd -P)
family_root="$repo_root/bin/shells/fish"
work_root=$(mktemp -d "${TMPDIR:-/tmp}/commandment-fish.XXXXXX")
trap 'rm -rf -- "$work_root"' EXIT HUP INT TERM
mkdir -p "$family_root"

probe() {
    executable=$1
    marker=$2
    [ -x "$executable" ] || return 1
    "$executable" --version 2>&1 | grep -F "$marker" >/dev/null
}

probe_fish4_runtime() {
    executable=$1
    probe "$executable" "fish, version 4.0.0" || return 1
    output=$(HOME=/nonexistent XDG_CONFIG_HOME=/nonexistent "$executable" -c 'printf "%s\n" commandment-fish-runtime-ready' 2>&1) || return 1
    printf '%s\n' "$output" | grep -Fx 'commandment-fish-runtime-ready' >/dev/null || return 1
    if printf '%s\n' "$output" | grep -F 'cannot find its asset files' >/dev/null; then
        return 1
    fi
}

download() {
    url=$1
    sha256=$2
    archive=$3
    curl -fsSL "$url" -o "$archive"
    printf '%s  %s\n' "$sha256" "$archive" | sha256sum -c - >/dev/null
}

ensure_fish3() {
    version=3.0.0
    final="$family_root/$version"
    executable="$final/bin/fish"
    if probe "$executable" "fish, version 3.0.0"; then
        echo "fish $version: existing" >&2
        printf '%s\n' "$executable"
        return
    fi

    echo "fish $version: building" >&2
    work="$work_root/fish-$version"
    source="$work/source"
    install="$work/install"
    archive="$work/source.tar.gz"
    mkdir -p "$source" "$install"
    download https://github.com/fish-shell/fish-shell/releases/download/3.0.0/fish-3.0.0.tar.gz ea9dd3614bb0346829ce7319437c6a93e3e1dfde3b7f6a469b543b0d2c68f2cf "$archive"
    tar -xzf "$archive" -C "$source" --strip-components=1
    (
        cd "$source"
        if [ ! -x ./configure ] && [ -x ./autogen.sh ]; then
            ./autogen.sh >&2
        fi
        # Fish 3.0's _sys_errs probe must link unoptimized so missing legacy libc symbols stay observable.
        CXXFLAGS=-O0 ./configure --prefix="$install" >&2
        make >&2
        make install >&2
    )
    probe "$install/bin/fish" "fish, version 3.0.0" || { echo "fish $version: sanity probe failed" >&2; exit 1; }
    rm -rf -- "$final"
    mv -- "$install" "$final"
    printf '%s\n' "$executable"
}

ensure_fish4() {
    version=4.0.0
    final="$family_root/$version"
    executable="$final/bin/fish"
    if probe_fish4_runtime "$executable"; then
        echo "fish $version: existing" >&2
        printf '%s\n' "$executable"
        return
    fi

    echo "fish $version: building" >&2
    work="$work_root/fish-$version"
    source="$work/source"
    install="$work/install"
    archive="$work/source.tar.xz"
    mkdir -p "$source" "$install"
    download https://github.com/fish-shell/fish-shell/releases/download/4.0.0/fish-4.0.0.tar.xz 2fda5bd970357064d8d4c896e08285ba59965ca2a8c4829ca8a82bf3b89c69f3 "$archive"
    tar -xJf "$archive" -C "$source" --strip-components=1
    (
        cd "$source"
        cargo build --release --locked >&2
        ./target/release/fish --install="$install" >&2
    )
    probe_fish4_runtime "$install/bin/fish" || { echo "fish $version: staged runtime probe failed" >&2; exit 1; }
    rm -rf -- "$final"
    mv -- "$install" "$final"
    probe_fish4_runtime "$executable" || { echo "fish $version: relocated runtime probe failed" >&2; exit 1; }
    printf '%s\n' "$executable"
}

for version in "$@"; do
    case "$version" in
        3.0.0) ensure_fish3 ;;
        4.0.0) ensure_fish4 ;;
        *)
            echo "fish $version: unsupported completion runtime version" >&2
            exit 2
            ;;
    esac
done
