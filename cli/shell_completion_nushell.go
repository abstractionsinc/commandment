package cli

import (
	"strconv"
	"strings"
)

const (
	nushellLegacyMinor072 = 72
	nushellLegacyMinor102 = 102
	nushellLegacyMinor108 = 108
	nushellLegacyMinor113 = 113
	nushellLegacyMinor115 = 115
)

type nushellGenerationBand uint8

const (
	nushellGenerationLegacyFloor072 nushellGenerationBand = iota
	nushellGenerationLegacyFloor
	nushellGenerationLegacyNull
	nushellGenerationPerCommand
	nushellGenerationDescription
	nushellGenerationCurrent
)

func nushellGenerationBandForTarget(target string) (nushellGenerationBand, error) {
	if target == "" {
		return nushellGenerationCurrent, nil
	}
	parts := strings.Split(target, ".")
	if len(parts) < shellVersionComponentMinimum {
		return 0, &shellCompletionError{
			kind:   errInvalidShellCompletionGeneration,
			detail: "invalid Nushell completion generation target version " + strconv.Quote(target),
		}
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil || major < 0 {
		return 0, &shellCompletionError{
			kind:   errInvalidShellCompletionGeneration,
			detail: "invalid Nushell completion generation target version " + strconv.Quote(target),
		}
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil || minor < 0 {
		return 0, &shellCompletionError{
			kind:   errInvalidShellCompletionGeneration,
			detail: "invalid Nushell completion generation target version " + strconv.Quote(target),
		}
	}
	if major > 0 {
		return nushellGenerationCurrent, nil
	}
	switch {
	case minor < nushellLegacyMinor072:
		return 0, &shellCompletionError{
			kind:   errUnsupportedShellCompletionTarget,
			detail: "Nushell completion generation target version " + strconv.Quote(target),
		}
	case minor == nushellLegacyMinor072:
		return nushellGenerationLegacyFloor072, nil
	case minor < nushellLegacyMinor102:
		return nushellGenerationLegacyFloor, nil
	case minor < nushellLegacyMinor108:
		return nushellGenerationLegacyNull, nil
	case minor < nushellLegacyMinor113:
		return nushellGenerationPerCommand, nil
	case minor < nushellLegacyMinor115:
		return nushellGenerationDescription, nil
	default:
		return nushellGenerationCurrent, nil
	}
}

func generateNushellCompletionScript(request shellCompletionScriptRequest) (string, error) {
	identifier := shellCompletionIdentifier(request.ApplicationName)
	band, err := nushellGenerationBandForTarget(request.TargetVersion)
	if err != nil {
		return "", err
	}

	template := nushellCompletionScript
	transportInvocation := "^$__completion_" + identifier + "_app"
	modernTransportInvocation := "^" + nushellDoubleQuoted(request.ApplicationName)
	replacements := []string{
		"@@ID@@", identifier,
		"@@APP@@", nushellDoubleQuoted(request.ApplicationName),
		"@@MODERN_TRANSPORT_INVOCATION@@", modernTransportInvocation,
		"@@ACTIVATION@@", nushellDoubleQuoted(request.ActivationEnvironment),
		"@@ACTIVATION_VALUE@@", nushellDoubleQuoted(request.ActivationValue),
	}
	switch band {
	case nushellGenerationLegacyFloor072, nushellGenerationLegacyFloor:
		template = nushellLegacyCompletionScript
		if band == nushellGenerationLegacyFloor072 {
			template = strings.Replace(template, nushellLegacyOwnerRegistration, nushell072OwnerRegistration, 1)
			template = strings.Replace(
				template,
				"    let prefix = ($current | str substring@@UTF8_BYTES_FLAG@@ 0..<($replace_start))",
				"    let prefix = (if $replace_start == 0 { \"\" } else { $current | str substring 0..<($replace_start) })",
				1,
			)
		}
		legacyFilter := ""
		if band == nushellGenerationLegacyFloor072 {
			legacyFilter = " and ($replace_start != 0 or $replace_end != $cursor_byte or ($value | str starts-with $current))"
		}
		replacements = append(replacements,
			"@@BAND@@", "0.72 through 0.101",
			"@@FAILURE_RESULT@@", "[]",
			"@@LIST_TYPE@@", "list",
			"@@UTF8_BYTES_FLAG@@", "",
			"@@RECORD_ESCAPE_BODY@@", nushell072RecordEscapeBody,
			"@@TRANSPORT_INVOCATION@@", transportInvocation,
			"@@MATCH_MODE@@", "client",
			"@@LEGACY_FILTER@@", legacyFilter,
		)
	case nushellGenerationLegacyNull:
		template = nushellLegacyCompletionScript
		template = strings.Replace(template, nushellLegacyOwnerRegistration, nushell0102OwnerRegistration, 1)
		replacements = append(replacements,
			"@@BAND@@", "0.102 through 0.107",
			"@@FAILURE_RESULT@@", "null",
			"@@LIST_TYPE@@", "list<string>",
			"@@UTF8_BYTES_FLAG@@", " --utf-8-bytes",
			"@@RECORD_ESCAPE_BODY@@", nushellLegacyRecordEscapeBody,
			"@@TRANSPORT_INVOCATION@@", "^$__completion_"+identifier+"_app",
			"@@MATCH_MODE@@", "prefix",
			"@@LEGACY_FILTER@@", "",
		)
	case nushellGenerationPerCommand, nushellGenerationDescription, nushellGenerationCurrent:
	}
	return renderShellCompletionScript(template, replacements...), nil
}

const nushellLegacyRecordEscapeBody = `    let hex = ($value | encode hex)
    mut out = ""
    let length = ($hex | str length)
    if $length == 0 { return $out }
    for offset in (0..2..($length - 2)) {
        let pair = ($hex | str substring $offset..($offset + 1))
        let byte = ($pair | into int --radix 16)
        $out = ($out + (match $byte {
            0x5c => "\\\\"
            0x09 => "\t"
            0x0a => "\n"
            0x0d => "\r"
            _ if $byte >= 0x20 and $byte <= 0x7e => ($pair | decode hex | decode utf-8)
            _ => ("\\x" + $pair)
        }))
    }
    $out`

const nushell072RecordEscapeBody = `    let bytes = ($value | encode utf-8)
    mut out = ""
    let length = ($bytes | bytes length)
    if $length == 0 { return $out }
    let hex = "000102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F202122232425262728292A2B2C2D2E2F303132333435363738393A3B3C3D3E3F404142434445464748494A4B4C4D4E4F505152535455565758595A5B5C5D5E5F606162636465666768696A6B6C6D6E6F707172737475767778797A7B7C7D7E7F808182838485868788898A8B8C8D8E8F909192939495969798999A9B9C9D9E9FA0A1A2A3A4A5A6A7A8A9AAABACADAEAFB0B1B2B3B4B5B6B7B8B9BABBBCBDBEBFC0C1C2C3C4C5C6C7C8C9CACBCCCDCECFD0D1D2D3D4D5D6D7D8D9DADBDCDDDEDFE0E1E2E3E4E5E6E7E8E9EAEBECEDEEEFF0F1F2F3F4F5F6F7F8F9FAFBFCFDFEFF"
    for offset in 0..($length - 1) {
        let one = ($bytes | bytes at [$offset ($offset + 1)])
        let byte = ($one | into int)
        let escaped = (if $byte == 0x5c {
            "\\\\"
        } else if $byte == 0x09 {
            "\\t"
        } else if $byte == 0x0a {
            "\\n"
        } else if $byte == 0x0d {
            "\\r"
        } else if $byte >= 0x20 and $byte <= 0x7e {
            $one | decode utf-8
        } else {
            let pair_start = ($byte * 2)
            let pair = ($hex | str substring $pair_start..($pair_start + 2))
            "\\x" + $pair
        })
        $out = ($out + $escaped)
    }
    $out`

const nushellLegacyOwnerRegistration = `let __completion_@@ID@@_prior_external_completer = $env.config.completions.external.completer
let __completion_@@ID@@_external_completer = {|spans|
    if (not ($spans | is-empty)) and (($spans | first) == $__completion_@@ID@@_app) {
        return (__completion_@@ID@@_complete_legacy $spans)
    }
    if $__completion_@@ID@@_prior_external_completer == null {
        return null
    }
    do $__completion_@@ID@@_prior_external_completer $spans
}
$env.config.completions.external.completer = $__completion_@@ID@@_external_completer`

const nushell0102OwnerRegistration = `let __completion_@@ID@@_prior_external_completer = $env.config.completions.external.completer
let __completion_@@ID@@_external_completer = {|spans|
    if (not ($spans | is-empty)) and (($spans | first) == $__completion_@@ID@@_app) {
        __completion_@@ID@@_complete_legacy $spans
    } else if $__completion_@@ID@@_prior_external_completer == null {
        null
    } else {
        do $__completion_@@ID@@_prior_external_completer $spans
    }
}
$env.config.completions.external.completer = $__completion_@@ID@@_external_completer`

const nushell072OwnerRegistration = `let __completion_@@ID@@_prior_config = $env.config
let __completion_@@ID@@_prior_completions = $__completion_@@ID@@_prior_config.completions
let __completion_@@ID@@_prior_external = $__completion_@@ID@@_prior_completions.external
let __completion_@@ID@@_prior_external_completer = $__completion_@@ID@@_prior_external.completer
let __completion_@@ID@@_external_completer = {|spans|
    if (not ($spans | is-empty)) and ($spans.0 == $__completion_@@ID@@_app) {
        __completion_@@ID@@_complete_legacy $spans
    } else if $__completion_@@ID@@_prior_external_completer == null {
        null
    } else {
        do $__completion_@@ID@@_prior_external_completer $spans
    }
}
let __completion_@@ID@@_new_external = ($__completion_@@ID@@_prior_external | merge {completer: $__completion_@@ID@@_external_completer})
let __completion_@@ID@@_new_completions = ($__completion_@@ID@@_prior_completions | merge {external: $__completion_@@ID@@_new_external})
let-env config = ($__completion_@@ID@@_prior_config | merge {completions: $__completion_@@ID@@_new_completions})`

const nushellLegacyCompletionScript = `# Completion integration for @@APP@@.
# Generated code: source this file from Nushell @@BAND@@.
let __completion_@@ID@@_app = @@APP@@

def __completion_@@ID@@_record_escape [value: string] {
@@RECORD_ESCAPE_BODY@@
}

def __completion_@@ID@@_request [shell_version: string, arguments: @@LIST_TYPE@@, cursor_argument: int, cursor_byte: int] {
    mut lines = ["records" "response-encoding\tjson"]
    $lines = ($lines | append $"shell\tnushell\t(__completion_@@ID@@_record_escape $shell_version)")
    $lines = ($lines | append $"input\tuncertain\t0\t($cursor_argument)\t($cursor_byte)\tobserved-word-bytes")
    mut index = 0
    for item in $arguments {
        $lines = ($lines | append $"word\t($index)\t(__completion_@@ID@@_record_escape $item)\tlexical")
        $index = ($index + 1)
    }
    $lines = ($lines | append "matching\t@@MATCH_MODE@@")
    $lines = ($lines | append "capability\tdescriptions")
    $lines = ($lines | append "end")
    ($lines | str join (char newline)) + (char newline)
}

def __completion_@@ID@@_complete_legacy [spans: @@LIST_TYPE@@] {
    if ($spans | is-empty) or (($spans | first) != $__completion_@@ID@@_app) { return null }
    let shell_version = (version | get version)
    mut arguments = ($spans | skip 1)
    if ($arguments | is-empty) { $arguments = [""] }
    if (($arguments | last) == " ") {
        $arguments = ($arguments | drop 1 | append "")
    }
    let cursor_argument = (($arguments | length) - 1)
    let current = ($arguments | last)
    let cursor_byte = ($current | str length@@UTF8_BYTES_FLAG@@)
    let request = (__completion_@@ID@@_request $shell_version $arguments $cursor_argument $cursor_byte)
    let process = (with-env { @@ACTIVATION@@: @@ACTIVATION_VALUE@@ } {
        $request | @@TRANSPORT_INVOCATION@@ | complete
    })
    if $process.exit_code != 0 { return @@FAILURE_RESULT@@ }
    let parsed = (try { $process.stdout | from json } catch { return @@FAILURE_RESULT@@ })
    let replace = (try { $parsed.replace } catch { return @@FAILURE_RESULT@@ })
    let replace_argument = (try { $replace.argument } catch { return @@FAILURE_RESULT@@ })
    let replace_start = (try { $replace.start } catch { return @@FAILURE_RESULT@@ })
    let replace_end = (try { $replace.end } catch { return @@FAILURE_RESULT@@ })
    if $replace_argument != $cursor_argument { return @@FAILURE_RESULT@@ }
    if $replace_start < 0 or $replace_end < $replace_start or $replace_end > $cursor_byte { return @@FAILURE_RESULT@@ }
    let result_spacing = (try { $parsed.spacing } catch { return @@FAILURE_RESULT@@ })
    let items = (try { $parsed.items } catch { return @@FAILURE_RESULT@@ })
    let prefix = ($current | str substring@@UTF8_BYTES_FLAG@@ 0..<($replace_start))
    let suffix = ($current | str substring@@UTF8_BYTES_FLAG@@ ($replace_end)..)
    mut completions = []
    for item in $items {
        mut value = (try { $item.value } catch { return @@FAILURE_RESULT@@ })
        let replacement = (try { $item.replacement } catch { null })
        let replacement_valid = if $replacement == null { true } else { ($replacement | length) == 1 }
        if $replacement != null and $replacement_valid {
            $value = ($replacement | first)
        }
        let item_spacing = (try { $item.spacing } catch { return @@FAILURE_RESULT@@ })
        let spacing = if $item_spacing == "default" { $result_spacing } else { $item_spacing }
        if $replacement_valid and $spacing == "default"@@LEGACY_FILTER@@ {
            $value = $prefix + $value + $suffix
            let item_description = (try { $item.description } catch { "" })
            let description = if ($item_description | is-empty) { null } else { $item_description }
            $completions = ($completions | append { value: $value, description: $description })
        }
    }
    $completions | each {|item| $item }
}

let __completion_@@ID@@_prior_external_completer = $env.config.completions.external.completer
let __completion_@@ID@@_external_completer = {|spans|
    if (not ($spans | is-empty)) and (($spans | first) == $__completion_@@ID@@_app) {
        return (__completion_@@ID@@_complete_legacy $spans)
    }
    if $__completion_@@ID@@_prior_external_completer == null {
        return null
    }
    do $__completion_@@ID@@_prior_external_completer $spans
}
$env.config.completions.external.completer = $__completion_@@ID@@_external_completer
`

const nushellCompletionScript = `# Completion integration for @@APP@@.
# Generated code: source this file from Nushell 0.108 or newer.
# The canonical process request remains the Completion Records codec. Nushell
# decodes the canonical JSON response so structured candidate text never becomes
# shell source or an ad-hoc line protocol.
def __completion_@@ID@@_record_escape [value: string] {
    let hex = ($value | encode hex)
    mut out = ""
    let length = ($hex | str length)
    if $length == 0 { return $out }
    for offset in (0..2..($length - 2)) {
        let pair = ($hex | str substring $offset..($offset + 1))
        let byte = ($pair | into int --radix 16)
        $out = ($out + (match $byte {
            0x5c => "\\\\"
            0x09 => "\\t"
            0x0a => "\\n"
            0x0d => "\\r"
            _ if $byte >= 0x20 and $byte <= 0x7e => ($pair | decode hex | decode utf-8)
            _ => ("\\x" + $pair)
        }))
    }
    $out
}

def __completion_@@ID@@_request [shell_version: string, arguments: list<string>, cursor_argument: int, cursor_byte: int] {
    mut lines = ["records" "response-encoding\tjson"]
    $lines = ($lines | append $"shell\tnushell\t(__completion_@@ID@@_record_escape $shell_version)")
    $lines = ($lines | append $"input\tprefix-exact\t($cursor_argument)\t($cursor_argument)` +
	`\t($cursor_byte)\tobserved-word-bytes")
    for item in ($arguments | enumerate) {
        let fidelity = if $item.index < $cursor_argument { "exact" } else { "lexical" }
        $lines = ($lines | append $"word\t($item.index)\t(__completion_@@ID@@_record_escape $item.item)\t($fidelity)")
    }
    $lines = ($lines | append "matching\tvalue-prefix")
    $lines = ($lines | append "capability\tdescriptions")
    $lines = ($lines | append "end")
    ($lines | str join (char newline)) + (char newline)
}

def __completion_@@ID@@_complete [spans: list<string>] {
    try {
        if ($spans | is-empty) or (($spans | first) != @@APP@@) { error make {msg: "completion completion owner mismatch"} }
        let shell_version = (version | get version)
        mut arguments = ($spans | skip 1)
        if ($arguments | is-empty) { $arguments = [""] }
        if (($arguments | last) == " ") {
            $arguments = ($arguments | drop 1 | append "")
        }
        let cursor_argument = (($arguments | length) - 1)
        let current = ($arguments | last)
        let cursor_byte = ($current | str length --utf-8-bytes)
        let request = (__completion_@@ID@@_request $shell_version $arguments $cursor_argument $cursor_byte)

        let process = (with-env { @@ACTIVATION@@: @@ACTIVATION_VALUE@@ } {
            $request | @@MODERN_TRANSPORT_INVOCATION@@ | complete
        })
        if $process.exit_code != 0 { error make {msg: "completion completion transport failed"} }
        let parsed = ($process.stdout | from json)
        # A semantic completion failure may carry a valid partial result.
        let replace = $parsed.replace
        let replace_argument = $replace.argument
        let replace_start = $replace.start
        let replace_end = $replace.end
        if $replace_argument != $cursor_argument { error make {msg: "completion completion replacement argument mismatch"} }
        if $replace_start < 0 or $replace_end < $replace_start or $replace_end > $cursor_byte { error make {msg: "completion completion replacement range invalid"} }
        let result_spacing = $parsed.spacing
        let items = $parsed.items

        let prefix = ($current | str substring --utf-8-bytes 0..<($replace_start))
        let suffix = ($current | str substring --utf-8-bytes ($replace_end)..)
        mut completions = []
        for item in $items {
            mut value = $item.value
            let replacement = (try { $item.replacement } catch { null })
            if $replacement != null {
                if ($replacement | length) != 1 { continue }
                $value = ($replacement | first)
            }
            let item_spacing = $item.spacing
            let spacing = if $item_spacing == "default" { $result_spacing } else { $item_spacing }
            # Nushell's custom-completion record has no per-candidate spacing/suffix
            # control. Preserve only client-default spacing rather than changing an
            # explicit Completion append/no-space request.
            if $spacing != "default" { continue }
            $value = $prefix + $value + $suffix
            let item_description = (try { $item.description } catch { "" })
            let description = if ($item_description | is-empty) { null } else { $item_description }
            $completions = ($completions | append { value: $value, description: $description })
        }

        $completions
    } catch {
        null
    }
}
@complete __completion_@@ID@@_complete
extern @@APP@@ [...completion_args: string]
`
