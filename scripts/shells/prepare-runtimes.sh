#!/bin/sh
set -eu

if [ "$#" -ne 0 ]; then
    echo "usage: $0" >&2
    exit 2
fi

case "${GOHOSTOS:-}" in
    linux)
        ;;
    '')
        echo "GOHOSTOS is required" >&2
        exit 2
        ;;
    *)
        echo "shell runtime preparation is unsupported on $GOHOSTOS" >&2
        exit 2
        ;;
esac

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
matrix="$script_dir/completion-matrix.tsv"

bash_versions=
elvish_versions=
fish_versions=
nushell_versions=
powershell_versions=
zsh_versions=

while IFS="$(printf '\t')" read -r family version host executable; do
    case "$family" in
        ''|'#'*) continue ;;
    esac
    [ -n "$version" ] && [ -n "$host" ] && [ -n "$executable" ] || {
        echo "malformed completion matrix row: $family $version $host $executable" >&2
        exit 2
    }
    [ "$host" = "$GOHOSTOS" ] || continue
    case "$family" in
        bash) bash_versions="$bash_versions $version" ;;
        elvish) elvish_versions="$elvish_versions $version" ;;
        fish) fish_versions="$fish_versions $version" ;;
        nushell) nushell_versions="$nushell_versions $version" ;;
        powershell) powershell_versions="$powershell_versions $version" ;;
        zsh) zsh_versions="$zsh_versions $version" ;;
        *)
            echo "unknown completion matrix family: $family" >&2
            exit 2
            ;;
    esac
done < "$matrix"

prepare_family() {
    family=$1
    versions=$2
    [ -n "$versions" ] || return 0
    # Matrix versions are numeric tokens without whitespace or shell metacharacters.
    "$script_dir/prepare-$family.sh" $versions
}

prepare_family bash "$bash_versions"
prepare_family elvish "$elvish_versions"
prepare_family fish "$fish_versions"
prepare_family nushell "$nushell_versions"
prepare_family powershell "$powershell_versions"
prepare_family zsh "$zsh_versions"
