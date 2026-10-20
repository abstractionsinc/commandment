package cli

func generateBashCompletionScript(request shellCompletionScriptRequest) string {
	identifier := shellCompletionIdentifier(request.ApplicationName)
	return renderShellCompletionScript(bashCompletionScript,
		"@@ID@@", identifier,
		"@@APP@@", shellSingleQuoted(request.ApplicationName),
		"@@ACTIVATION@@", shellSingleQuoted(request.ActivationEnvironment),
		"@@ACTIVATION_VALUE@@", request.ActivationValue,
	)
}

const bashCompletionScript = `# Completion integration for @@APP@@.
# Generated code: source this file from Bash 3.2 or newer.
__completion_@@ID@@_app=@@APP@@
__completion_@@ID@@_activation=@@ACTIVATION@@

__completion_@@ID@@_record_escape() {
    local value=$1 out='' char hex code i
    local LC_ALL=C
    for ((i = 0; i < ${#value}; i++)); do
        char=${value:i:1}
        case $char in
            $'\\') out+='\\\\' ;;
            $'\t') out+='\\t' ;;
            $'\n') out+='\\n' ;;
            $'\r') out+='\\r' ;;
            *)
                printf -v code '%d' "'$char"
                if ((code >= 32 && code <= 126)); then
                    out+=$char
                else
                    printf -v hex '%02X' "$code"
                    out+="\\x$hex"
                fi
                ;;
        esac
    done
    printf '%s' "$out"
}

__completion_@@ID@@_record_decode() {
    printf '%b' "$1"
}

__completion_@@ID@@_request() {
    local shell_version=$1 cursor_argument=$2 cursor_byte=$3
    shift 3
    local index=0 argument
    printf 'records\n'
    printf 'response-encoding\trecords\n'
    printf 'shell\tbash\t%s\n' "$(__completion_@@ID@@_record_escape "$shell_version")"
    printf 'input\tuncertain\t0\t%d\t%d\tobserved-word-bytes\n' "$cursor_argument" "$cursor_byte"
    for argument in "$@"; do
        printf 'word\t%d\t%s\tlexical\n' "$index" "$(__completion_@@ID@@_record_escape "$argument")"
        ((index++))
    done
    printf 'matching\tprefix\n'
    printf 'native-capability\tpaths\n'
    printf 'native-capability\tfiles\n'
    printf 'native-capability\tdirectories\n'
    printf 'native-capability\texecutables\n'
    printf 'native-capability\tusers\n'
    printf 'native-capability\thosts\n'
    printf 'native-capability\tenvironment-variables\n'
    printf 'end\n'
}

# Capture only Bash's own programmable-completion observation. COMP_WORDS is
# Readline/editor state, not execution argv: COMP_WORDBREAKS may split one
# eventual argument into multiple observed words. Never reconstruct argv by
# parsing COMP_LINE; preserve value and topology uncertainty explicitly.
__completion_@@ID@@_context() {
    local -i cword=${COMP_CWORD:-0}
    ((cword >= 1 && cword < ${#COMP_WORDS[@]})) || return 1

    __completion_@@ID@@_args=("${COMP_WORDS[@]:1}")
    __completion_@@ID@@_cursor_argument=$((cword - 1))
    __completion_@@ID@@_current=${COMP_WORDS[cword]-}

    # The completion function's second positional parameter is the current
    # Readline word. It is the strongest byte-coordinate fact Bash supplies at
    # this boundary without decoding COMP_LINE ourselves.
    local current_prefix=${1-}
    local LC_ALL=C
    __completion_@@ID@@_cursor_byte=${#current_prefix}
    ((__completion_@@ID@@_cursor_byte <= ${#__completion_@@ID@@_current})) || return 1
    __completion_@@ID@@_readline_start_byte=0
}

__completion_@@ID@@_native() {
    local kind=$1 seed=$2 prefix=$3 suffix=$4 candidate
    if ((BASH_VERSINFO[0] > 5 || (BASH_VERSINFO[0] == 5 && BASH_VERSINFO[1] >= 3))); then
        local -a generated=()
        case $kind in
            paths|files) compgen -V generated -f -- "$seed" ;;
            directories) compgen -V generated -d -- "$seed" ;;
            executables) compgen -V generated -c -- "$seed" ;;
            users) compgen -V generated -u -- "$seed" ;;
            hosts) compgen -V generated -A hostname -- "$seed" ;;
            environment-variables) compgen -V generated -v -- "$seed" ;;
            *) return 0 ;;
        esac
        for candidate in "${generated[@]}"; do
            COMPREPLY+=("$prefix$candidate$suffix")
        done
        return 0
    fi

    # Before Bash 5.3 compgen can only publish through newline-framed stdout.
    # Preserve the historical native behavior for ordinary values; 5.3+ uses
    # the lossless array path above so newline-bearing names are never split.
    case $kind in
        paths|files)
            while IFS= read -r candidate; do COMPREPLY+=("$prefix$candidate$suffix"); done < <(compgen -f -- "$seed")
            ;;
        directories)
            while IFS= read -r candidate; do COMPREPLY+=("$prefix$candidate$suffix"); done < <(compgen -d -- "$seed")
            ;;
        executables)
            while IFS= read -r candidate; do COMPREPLY+=("$prefix$candidate$suffix"); done < <(compgen -c -- "$seed")
            ;;
        users)
            while IFS= read -r candidate; do COMPREPLY+=("$prefix$candidate$suffix"); done < <(compgen -u -- "$seed")
            ;;
        hosts)
            while IFS= read -r candidate; do COMPREPLY+=("$prefix$candidate$suffix"); done < <(compgen -A hostname -- "$seed")
            ;;
        environment-variables)
            while IFS= read -r candidate; do COMPREPLY+=("$prefix$candidate$suffix"); done < <(compgen -v -- "$seed")
            ;;
    esac
}

__completion_@@ID@@_complete() {
    __completion_@@ID@@_context "${2-}" || return 0
    local -a completion_args=("${__completion_@@ID@@_args[@]}")
    local cursor_argument=$__completion_@@ID@@_cursor_argument
    local cursor_byte=$__completion_@@ID@@_cursor_byte
    local readline_start_byte=$__completion_@@ID@@_readline_start_byte
    local current=$__completion_@@ID@@_current
    local LC_ALL=C

    local response status
    response=$(
        __completion_@@ID@@_request "$BASH_VERSION" "$cursor_argument" "$cursor_byte" "${completion_args[@]}" |
            env "$__completion_@@ID@@_activation=@@ACTIVATION_VALUE@@" \
                "$__completion_@@ID@@_app" 2>/dev/null
    )
    status=$?
    ((status == 0)) || return 0

    local failed=true replace_argument=-1 replace_start=0 replace_end=0 result_spacing=default
    local -a item_value item_replacement_present item_replacement item_replacement_count item_spacing native_kind
    local line tag
    local -a fields
    local item_index replacement_index native_index
    local saw_end=false

    while IFS= read -r line; do
        local split=${line//$'\t'/$'\034'}
        IFS=$'\034' read -r -a fields <<< "$split"
        tag=${fields[0]-}
        case $tag in
            result)
                failed=${fields[1]-true}
                ;;
            replace)
                replace_argument=${fields[1]:--1}
                replace_start=${fields[2]:-0}
                replace_end=${fields[3]:-0}
                ;;
            item)
                item_index=${fields[1]-}
                [[ $item_index =~ ^[0-9]+$ ]] || continue
                item_value[item_index]=$(__completion_@@ID@@_record_decode "${fields[2]-}")
                item_replacement_present[item_index]=${fields[3]-false}
                item_spacing[item_index]=${fields[4]-default}
                ;;
            item-replacement)
                item_index=${fields[1]-}
                replacement_index=${fields[2]-}
                [[ $item_index =~ ^[0-9]+$ && $replacement_index =~ ^[0-9]+$ ]] || continue
                if [[ -z ${item_replacement_count[item_index]+x} ]]; then
                    item_replacement_count[item_index]=0
                fi
                ((item_replacement_count[item_index]++))
                if ((replacement_index == 0)); then
                    item_replacement[item_index]=$(__completion_@@ID@@_record_decode "${fields[3]-}")
                fi
                ;;
            spacing)
                result_spacing=${fields[1]-default}
                ;;
            native)
                native_index=${fields[1]-}
                [[ $native_index =~ ^[0-9]+$ ]] || continue
                native_kind[native_index]=${fields[2]-}
                ;;
            end)
                saw_end=true
                ;;
        esac
    done <<< "$response"

    # A semantic completion failure may carry a valid partial result. The process
    # transport succeeded, so project any canonical items that were returned.
    [[ $saw_end == true ]] || return 0
    [[ $replace_argument == "$cursor_argument" ]] || return 0
    ((replace_start >= readline_start_byte && replace_end >= replace_start && replace_end <= cursor_byte && cursor_byte <= ${#current})) || return 0

    # Readline replaces only its current fragment up to COMP_POINT and keeps the
    # suffix after the cursor. Recompose from that fragment boundary; content
    # before it (for example --flag= when = is in COMP_WORDBREAKS) remains in
    # the line and must not be duplicated in COMPREPLY.
    local prefix=${current:readline_start_byte:replace_start-readline_start_byte}
    local suffix=${current:replace_end:cursor_byte-replace_end}
    local value spacing index count spacing_class='' current_class
    local require_nospace=false spacing_conflict=false
    ((cursor_byte < ${#current})) && require_nospace=true
    COMPREPLY=()
    for ((index = 0; index < ${#item_value[@]}; index++)); do
        [[ -n ${item_value[index]+x} ]] || continue
        value=${item_value[index]}
        if [[ ${item_replacement_present[index]-false} == true ]]; then
            count=${item_replacement_count[index]-0}
            ((count == 1)) || continue
            value=${item_replacement[index]-}
        fi
        spacing=${item_spacing[index]-default}
        [[ $spacing == default ]] && spacing=$result_spacing
        case $spacing in
            no-space) current_class=no-space ;;
            append-space) current_class=space ;;
            default) current_class=default ;;
            *) continue ;;
        esac
        # Bash's compopt spacing mode applies to the whole completion result, not
        # individual COMPREPLY entries. Mixed explicit spacing is unrepresentable.
        if [[ -n $spacing_class && $current_class != default && $spacing_class != default && $current_class != "$spacing_class" ]]; then
            spacing_conflict=true
            break
        fi
        [[ $current_class != default ]] && spacing_class=$current_class
        if ((cursor_byte < ${#current})) && [[ $spacing == append-space ]]; then
            spacing_conflict=true
            break
        fi
        [[ $spacing == no-space ]] && require_nospace=true
        COMPREPLY+=("$prefix$value$suffix")
    done
    if [[ $spacing_conflict == true ]]; then
        COMPREPLY=()
        return 0
    fi

    local kind
    local native_seed=${current:replace_start:cursor_byte-replace_start}
    if ((${#native_kind[@]} > 0)); then
        require_nospace=true
    fi
    for kind in "${native_kind[@]}"; do
        __completion_@@ID@@_native "$kind" "$native_seed" "$prefix" "$suffix"
    done
    if [[ $require_nospace == true ]] && ((BASH_VERSINFO[0] >= 4)); then
        compopt -o nospace 2>/dev/null || true
    fi
    if ((BASH_VERSINFO[0] > 4 || (BASH_VERSINFO[0] == 4 && BASH_VERSINFO[1] >= 4))); then
        compopt -o nosort 2>/dev/null || true
    fi
    return 0
}

# Bash 3.2 is the practical floor. Newer completion options are enabled only
# when the running shell actually provides them. Bash 5.3's fullquote mode is
# strictly stronger for arbitrary candidate values; older supported releases
# retain Bash's native programmable-completion quoting behavior.
if ((BASH_VERSINFO[0] > 5 || (BASH_VERSINFO[0] == 5 && BASH_VERSINFO[1] >= 3))); then
    complete -o fullquote -F __completion_@@ID@@_complete -- @@APP@@
else
    complete -F __completion_@@ID@@_complete -- @@APP@@
fi
`
