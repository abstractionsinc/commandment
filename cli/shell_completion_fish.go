package cli

func generateFishCompletionScript(request shellCompletionScriptRequest) string {
	identifier := shellCompletionIdentifier(request.ApplicationName)
	return renderShellCompletionScript(fishCompletionScript,
		"@@ID@@", identifier,
		"@@APP@@", fishSingleQuoted(request.ApplicationName),
		"@@ACTIVATION@@", fishSingleQuoted(request.ActivationEnvironment),
		"@@ACTIVATION_VALUE@@", request.ActivationValue,
	)
}

const fishCompletionScript = `# Completion integration for @@APP@@.
# Generated code: source this file from Fish 3.x or newer.
set -g __completion_@@ID@@_app @@APP@@
set -g __completion_@@ID@@_activation @@ACTIVATION@@

function __completion_@@ID@@_record_escape --argument-names value
    for byte in (printf '%s' "$value" | od -An -v -t x1 | string split ' ' | string match -r '[0-9a-fA-F]{2}')
        set -l upper (string upper "$byte")
        switch $upper
            case 5C
                printf '\\\\'
            case 09
                printf '\\t'
            case 0A
                printf '\\n'
            case 0D
                printf '\\r'
            case '*'
                set -l decimal (math "0x$upper")
                if test $decimal -ge 32 -a $decimal -le 126
                    printf '%b' "\\x$upper"
                else
                    printf '\\x%s' "$upper"
                end
        end
    end
end

function __completion_@@ID@@_record_decode --argument-names value
    printf '%b' "$value"
end

function __completion_@@ID@@_request --argument-names shell_version cursor_argument cursor_byte
    # --argument-names aliases the leading positional values but leaves them in
    # argv. Copy only the completion words explicitly so framing cannot depend on
    # mutating Fish's special argv variable across supported floors.
    set -l arguments $argv[4..-1]
    printf 'records\n'
    printf 'response-encoding\trecords\n'
    printf 'shell\tfish\t%s\n' (__completion_@@ID@@_record_escape "$shell_version")
    printf 'input\tuncertain\t0\t%d\t%d\tobserved-word-bytes\n' $cursor_argument $cursor_byte
    set -l index 0
    set -l fidelity lexical
    set -l major (string split -m 1 . $shell_version)[1]
    if string match -qr '^[0-9]+$' -- $major; and test $major -ge 4
        set fidelity expanded
    end
    for argument in $arguments
        set -l escaped (__completion_@@ID@@_record_escape "$argument")
        printf 'word\t%d\t%s\t%s\n' $index "$escaped" $fidelity
        set index (math $index + 1)
    end
    printf 'matching\tclient\n'
    printf 'capability\tdescriptions\n'
    printf 'end\n'
end

function __completion_@@ID@@_complete
    set -l major (string split -m 1 . $version)[1]
    string match -qr '^[0-9]+$' -- $major; or return 0
    test $major -ge 3; or return 0

    # Fish 4 exposes argument-expanded tokens without evaluating command
    # substitutions; Fish 3 exposes tokenized string arguments. Capture both
    # the complete process and the completed arguments before the cursor so
    # Completion receives the full decoded argument vector and cursor owner.
    set -l prior
    set -l all
    if test $major -ge 4
        set prior (commandline -cpx)
        set all (commandline -px)
    else
        set prior (commandline -cpo)
        set all (commandline -po)
    end
    test (count $prior) -ge 1; or return 0
    test (count $all) -ge 1; or return 0
    set -e prior[1]
    set -e all[1]

    # commandline's completion callback can run with the cursor inside a token,
    # but its candidate API cannot express Completion's arbitrary byte-range
    # replacement against a decoded mid-token argument. Fail closed there.
    set -l raw_current (commandline -t)
    set -l raw_prefix (commandline -ct)
    test "$raw_current" = "$raw_prefix"; or return 0

    set -l cursor_argument (count $prior)
    set -l current ''
    set -l completion_args
    set -l current_index (math $cursor_argument + 1)
    if test -n "$raw_current"
        set -q all[$current_index]; or return 0
        if test $major -ge 4
            # Fish 4 can expand one source token to multiple arguments. The
            # source cursor has a truthful expanded owner only when the current
            # token expands to exactly one argument and that argument agrees
            # with the corresponding slot in the expanded process.
            set -l expanded_current (commandline --current-token --tokens-expanded)
            test (count $expanded_current) -eq 1; or return 0
            test "$expanded_current[1]" = "$all[$current_index]"; or return 0
            set current $expanded_current[1]
        else
            set current $all[$current_index]
        end
        set completion_args $all
    else
        # An empty cursor-owned argument is not present in the tokenized process;
        # insert it between the completed prefix and any arguments after cursor.
        set completion_args $prior ''
        if test (count $all) -ge $current_index
            set -a completion_args $all[$current_index..-1]
        end
    end
    set -l cursor_byte (printf '%s' "$current" | wc -c | string trim)

    set -l response (
        __completion_@@ID@@_request $version $cursor_argument $cursor_byte $completion_args |
            env "$__completion_@@ID@@_activation=@@ACTIVATION_VALUE@@" \
                "$__completion_@@ID@@_app" 2>/dev/null
    )
    or return 0

    set -l failed true
    set -l saw_end false
    set -l replace_argument -1
    set -l replace_start 0
    set -l replace_end 0
    set -l item_value
    set -l item_description
    set -l item_replacement_present
    set -l item_replacement
    set -l item_replacement_count
    set -l item_spacing
    set -l result_spacing default

    for line in $response
        set -l fields (string split \t -- "$line")
        set -l tag $fields[1]
        switch $tag
            case result
                set failed $fields[2]
            case replace
                set replace_argument $fields[2]
                set replace_start $fields[3]
                set replace_end $fields[4]
            case item
                set -l slot (math $fields[2] + 1)
                set item_value[$slot] (__completion_@@ID@@_record_decode "$fields[3]")
                set item_replacement_present[$slot] $fields[4]
                set item_description[$slot] ''
                set item_spacing[$slot] $fields[5]
            case item-description
                set -l slot (math $fields[2] + 1)
                set item_description[$slot] (__completion_@@ID@@_record_decode "$fields[3]")
            case spacing
                set result_spacing $fields[2]
            case item-replacement
                set -l slot (math $fields[2] + 1)
                set item_replacement_count[$slot] (math 0$item_replacement_count[$slot] + 1)
                if test $fields[3] -eq 0
                    set item_replacement[$slot] (__completion_@@ID@@_record_decode "$fields[4]")
                end
            case end
                set saw_end true
        end
    end

    # A semantic completion failure may carry a valid partial result.
    test "$saw_end" = true; or return 0
    test "$replace_argument" -eq "$cursor_argument"; or return 0
    # Fish owns quoting and insertion. Its completion API does not expose a
    # general in-token byte-range replacement primitive, so only the exact
    # current-token range can be represented truthfully here.
    set -l current_bytes (printf '%s' "$current" | wc -c | string trim)
    test "$replace_start" -eq 0 -a "$replace_end" -eq "$current_bytes"; or return 0

    for slot in (seq (count $item_value))
        set -q item_value[$slot]; or continue
        set -l value $item_value[$slot]
        if test "$item_replacement_present[$slot]" = true
            test "0$item_replacement_count[$slot]" -eq 1; or continue
            set value $item_replacement[$slot]
        end
        # Fish's dynamic completion wire is newline-delimited with tab-delimited
        # descriptions. Values containing those delimiters are not representable
        # on this API surface; omit them rather than changing candidate text.
        string match -qr '[\t\r\n]' -- "$value"; and continue

        # Fish owns spacing after accepted dynamic candidates and exposes no
        # per-candidate suffix control here. Only Completion's client-default
        # spacing is therefore representable without silently changing intent.
        set -l spacing $item_spacing[$slot]
        test "$spacing" = default; and set spacing $result_spacing
        test "$spacing" = default; or continue

        set -l description $item_description[$slot]
        if string match -qr '[\t\r\n]' -- "$description"
            set description ''
        end
        if test -n "$description"
            printf '%s\t%s\n' "$value" "$description"
        else
            printf '%s\n' "$value"
        end
    end
end

complete -c @@APP@@ -f -k -a '(__completion_@@ID@@_complete)'
`
