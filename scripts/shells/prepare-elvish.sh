#!/bin/sh
set -eu

if [ "$#" -eq 0 ]; then
    echo "usage: $0 VERSION..." >&2
    exit 2
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH= cd -- "$script_dir/../.." && pwd -P)
family_root="$repo_root/bin/shells/elvish"
work_root=$(mktemp -d "${TMPDIR:-/tmp}/commandment-elvish.XXXXXX")
trap 'rm -rf -- "$work_root"' EXIT HUP INT TERM
mkdir -p "$family_root"

probe() {
    executable=$1
    marker=$2
    [ -x "$executable" ] || return 1
    "$executable" -version 2>&1 | grep -F "$marker" >/dev/null
}

ensure_elvish() {
    version=$1
    commit=$2
    ldflags=$3
    final="$family_root/$version"
    executable="$final/elvish"
    if probe "$executable" "$version"; then
        echo "elvish $version: existing" >&2
        printf '%s\n' "$executable"
        return
    fi

    echo "elvish $version: building" >&2
    work="$work_root/elvish-$version"
    gopath="$work/gopath"
    source="$gopath/src/github.com/elves/elvish"
    install="$work/install"
    mkdir -p "$(dirname "$source")" "$install"
    git clone --quiet --no-checkout https://github.com/elves/elvish.git "$source"
    git -C "$source" checkout --quiet --detach "$commit"
    actual=$(git -C "$source" rev-parse HEAD)
    [ "$actual" = "$commit" ] || { echo "elvish $version: source commit mismatch: $actual" >&2; exit 1; }
    (
        cd "$source"
        GO111MODULE=off GOPATH="$gopath" go build -o "$install/elvish" -ldflags "$ldflags" . >&2
    )
    probe "$install/elvish" "$version" || { echo "elvish $version: sanity probe failed" >&2; exit 1; }
    rm -rf -- "$final"
    mv -- "$install" "$final"
    printf '%s\n' "$executable"
}

for version in "$@"; do
    case "$version" in
        0.11.0)
            ensure_elvish 0.11.0 6331c1515ba41515c74f0f8f24be10e74bf3acbe '-X github.com/elves/elvish/build.Version=v0.11.0'
            ;;
        0.12.0)
            ensure_elvish 0.12.0 99687f337442f07c0dd6959103b34c450300dce7 '-X github.com/elves/elvish/buildinfo.Version=v0.12.0'
            ;;
        *)
            echo "elvish $version: unsupported completion runtime version" >&2
            exit 2
            ;;
    esac
done
