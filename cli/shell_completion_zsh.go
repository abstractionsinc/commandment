package cli

func generateZshCompletionScript(request shellCompletionScriptRequest) string {
	identifier := shellCompletionIdentifier(request.ApplicationName)
	return renderShellCompletionScript(zshCompletionScript,
		"@@ID@@", identifier,
		"@@APP@@", shellSingleQuoted(request.ApplicationName),
		"@@ACTIVATION@@", shellSingleQuoted(request.ActivationEnvironment),
		"@@ACTIVATION_VALUE@@", request.ActivationValue,
	)
}

const zshCompletionScript = `# Completion integration for @@APP@@.
# Generated code: source this file from Zsh 4.0.1 or newer.
typeset -g __completion_@@ID@@_app=@@APP@@
typeset -g __completion_@@ID@@_activation=@@ACTIVATION@@

__completion_@@ID@@_record_escape() {
    local value out char hex code high low
    local hex_digits
    value=$1
    out=''
    hex_digits=0123456789ABCDEF
    local i
    local LC_ALL
    LC_ALL=C
    for ((i = 1; i <= ${#value}; i++)); do
        char=${value[i]}
        case $char in
            $'\\') out=$out'\\' ;;
            $'\t') out=$out'\t' ;;
            $'\n') out=$out'\n' ;;
            $'\r') out=$out'\r' ;;
            *)
                code=$(( #char ))
                if ((code >= 32 && code <= 126)); then
                    out=$out$char
                else
                    high=$((code / 16 + 1))
                    low=$((code % 16 + 1))
                    hex=${hex_digits[high]}${hex_digits[low]}
                    out="${out}\\x$hex"
                fi
                ;;
        esac
    done
    REPLY=$out
}

__completion_@@ID@@_record_decode() {
    local value out char escape hex code
    value=$1
    out=''
    local i
    local LC_ALL
    LC_ALL=C
    i=1
    while ((i <= ${#value})); do
        char=${value[i]}
        if [[ $char != $'\\' ]]; then
            out=$out$char
            i=$((i + 1))
            continue
        fi
        i=$((i + 1))
        ((i <= ${#value})) || return 1
        escape=${value[i]}
        case $escape in
            $'\\') out=$out$'\\' ;;
            t) out=$out$'\t' ;;
            n) out=$out$'\n' ;;
            r) out=$out$'\r' ;;
            x)
                ((i + 2 <= ${#value})) || return 1
                hex=${value[i + 1,i + 2]}
                [[ $hex == [0-9A-F][0-9A-F] ]] || return 1
                code=$((16#$hex))
                char=${(#)code}
                out=$out$char
                i=$((i + 2))
                ;;
            *) return 1 ;;
        esac
        i=$((i + 1))
    done
    REPLY=$out
}

__completion_@@ID@@_request() {
    local shell_version cursor_argument cursor_byte
    shell_version=$1
    cursor_argument=$2
    cursor_byte=$3
    shift 3
    local index
    index=0
    local argument escaped request_buffer
    request_buffer=$'records\nresponse-encoding\trecords\n'
    __completion_@@ID@@_record_escape "$shell_version"
    escaped=$REPLY
    request_buffer=$request_buffer"shell"$'\t'"zsh"$'\t'"$escaped"$'\n'
    request_buffer=$request_buffer"input"$'\t'"uncertain"$'\t'"0"$'\t'"$cursor_argument"` +
	`$'\t'"$cursor_byte"$'\t'"observed-word-bytes"$'\n'
    for argument in "$@"; do
        __completion_@@ID@@_record_escape "$argument"
        escaped=$REPLY
        request_buffer=$request_buffer"word"$'\t'"$index"$'\t'"$escaped"$'\t'"lexical"$'\n'
        index=$((index + 1))
    done
    request_buffer=$request_buffer$'matching\tprefix\n'
    request_buffer=$request_buffer$'capability\tdescriptions\n'
    request_buffer=$request_buffer$'capability\tgroups\n'
    request_buffer=$request_buffer$'end\n'
    REPLY=$request_buffer
}

__completion_@@ID@@_complete() {
    local version_major
    version_major=${ZSH_VERSION%%.*}
    [[ $version_major == <-> && $version_major -ge 4 ]] || return 1

    local cursor_argument
    cursor_argument=$((CURRENT - 2))
    [[ $cursor_argument -ge 0 ]] || return 1
    local -a completion_args
    completion_args=("${words[2,-1]}")
    local current
    current=${completion_args[cursor_argument + 1]-}
    local current_prefix current_suffix
    current_prefix=$PREFIX
    current_suffix=$SUFFIX
    local cursor_byte
    local LC_ALL
    LC_ALL=C
    cursor_byte=${#current_prefix}

    local request REPLY
    __completion_@@ID@@_request "$ZSH_VERSION" "$cursor_argument" "$cursor_byte" "${completion_args[@]}" || return 1
    request=$REPLY

    local failed saw_end
    failed=true
    saw_end=false
    local replace_argument replace_start replace_end
    replace_argument=-1
    replace_start=0
    replace_end=0
    local result_spacing line tag
    result_spacing=default
    local -a fields
    local -a item_value item_present item_replacement_present item_replacement item_replacement_count item_display item_description item_group item_spacing
    local -A group_label
    local item_index replacement_index slot

    local request_path response_path
    request_path=${TMPDIR:-/tmp}/completion-request-$$-$RANDOM
    response_path=${TMPDIR:-/tmp}/completion-response-$$-$RANDOM
    print -rn -- "$request" >| "$request_path" || return 1
    env "$__completion_@@ID@@_activation=@@ACTIVATION_VALUE@@" "$__completion_@@ID@@_app" \
        < "$request_path" >| "$response_path" 2>/dev/null || {
        rm -f -- "$request_path" "$response_path"
        return 1
    }
    rm -f -- "$request_path"
    while IFS= read -r line; do
        local split
        split=${line//$'\t'/$'\034'}
        IFS=$'\034' read -rA fields <<< "$split"
        tag=${fields[1]-}
        case $tag in
            result)
                failed=${fields[2]-true}
                ;;
            replace)
                replace_argument=${fields[2]:--1}
                replace_start=${fields[3]:-0}
                replace_end=${fields[4]:-0}
                ;;
            item)
                item_index=${fields[2]-}
                [[ $item_index == <-> ]] || continue
                slot=$((item_index + 1))
                __completion_@@ID@@_record_decode "${fields[3]-}" || return 1
                item_value[slot]=$REPLY
                item_present[slot]=true
                item_replacement_present[slot]=${fields[4]-false}
                item_spacing[slot]=${fields[5]-default}
                ;;
            item-display)
                item_index=${fields[2]-}
                [[ $item_index == <-> ]] || continue
                slot=$((item_index + 1))
                __completion_@@ID@@_record_decode "${fields[3]-}" || return 1
                item_display[slot]=$REPLY
                ;;
            item-description)
                item_index=${fields[2]-}
                [[ $item_index == <-> ]] || continue
                slot=$((item_index + 1))
                __completion_@@ID@@_record_decode "${fields[3]-}" || return 1
                item_description[slot]=$REPLY
                ;;
            item-group)
                item_index=${fields[2]-}
                [[ $item_index == <-> ]] || continue
                slot=$((item_index + 1))
                __completion_@@ID@@_record_decode "${fields[3]-}" || return 1
                item_group[slot]=$REPLY
                ;;
            item-replacement)
                item_index=${fields[2]-}
                replacement_index=${fields[3]-}
                [[ $item_index == <-> && $replacement_index == <-> ]] || continue
                slot=$((item_index + 1))
                item_replacement_count[slot]=$(( ${item_replacement_count[slot]:-0} + 1 ))
                if ((replacement_index == 0)); then
                    __completion_@@ID@@_record_decode "${fields[4]-}" || return 1
                    item_replacement[slot]=$REPLY
                fi
                ;;
            group)
                local group_id group_text
                __completion_@@ID@@_record_decode "${fields[3]-}" || return 1
                group_id=$REPLY
                __completion_@@ID@@_record_decode "${fields[4]-}" || return 1
                group_text=$REPLY
                [[ -n $group_id ]] && group_label[$group_id]=$group_text
                ;;
            spacing)
                result_spacing=${fields[2]-default}
                ;;
            end)
                saw_end=true
                ;;
        esac
    done < "$response_path"
    rm -f -- "$response_path"

    # A semantic completion failure may carry a valid partial result.
    [[ $saw_end == true ]] || return 1
    [[ $replace_argument -eq $cursor_argument ]] || return 1
    ((replace_start >= 0 && replace_end >= replace_start && replace_end <= ${#current})) || return 1

    local unchanged_prefix
    unchanged_prefix=''
    ((replace_start > 0)) && unchanged_prefix=${current[1,replace_start]}
    local unchanged_suffix
    unchanged_suffix=''
    ((replace_end < ${#current})) && unchanged_suffix=${current[replace_end + 1,-1]}
    local value display description group spacing suffix group_name explanation
    local -a displays
    local index count
    local added
    added=0

    for ((index = 1; index <= ${#item_value}; index++)); do
        [[ ${item_present[index]-false} == true ]] || continue
        value=${item_value[index]}
        if [[ ${item_replacement_present[index]-false} == true ]]; then
            count=${item_replacement_count[index]:-0}
            ((count == 1)) || continue
            value=${item_replacement[index]-}
        fi
        display=${item_display[index]-}
        [[ -n $display ]] || display=$value
        description=${item_description[index]-}
        [[ -n $description ]] && display="$display - $description"
        displays=("$display")
        group=${item_group[index]-}
        group_name=${group:-completion}
        group_name=${group_name//\%/%%}
        explanation=${group_label[$group]-}
        explanation=${explanation//\%/%%}
        spacing=${item_spacing[index]-default}
        [[ $spacing == default ]] && spacing=$result_spacing
        suffix=$unchanged_suffix
        [[ $spacing != no-space ]] && suffix="$suffix "

        if [[ -n $explanation ]]; then
            compadd -U -2 -V "$group_name" -X "$explanation" -d displays -i "$unchanged_prefix" -I "$suffix" -- "$value" && added=$((added + 1))
        else
            compadd -U -2 -V "$group_name" -d displays -i "$unchanged_prefix" -I "$suffix" -- "$value" && added=$((added + 1))
        fi
    done
    ((added > 0))
}

autoload -U compinit
if ! whence compdef >/dev/null 2>&1; then
    compinit || return 1
fi
compdef __completion_@@ID@@_complete @@APP@@
[[ ${_comps[$__completion_@@ID@@_app]-} == __completion_@@ID@@_complete ]] || return 1
return 0
`
