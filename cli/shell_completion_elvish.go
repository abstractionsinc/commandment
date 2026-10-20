package cli

import (
	"strconv"
	"strings"
)

const (
	shellVersionComponentMinimum = 2
	elvishLegacyMinor011         = 11
	elvishLegacyMinor012         = 12
)

type elvishGenerationCapability uint8

const (
	elvishGenerationCurrent elvishGenerationCapability = iota
	elvishGeneration011
	elvishGeneration012
)

func elvishGenerationCapabilityForTarget(target string) (elvishGenerationCapability, error) {
	if target == "" {
		return elvishGenerationCurrent, nil
	}
	parts := strings.Split(target, ".")
	if len(parts) < shellVersionComponentMinimum {
		return 0, &shellCompletionError{
			kind:   errInvalidShellCompletionGeneration,
			detail: "invalid Elvish completion generation target version " + strconv.Quote(target),
		}
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil || major < 0 {
		return 0, &shellCompletionError{
			kind:   errInvalidShellCompletionGeneration,
			detail: "invalid Elvish completion generation target version " + strconv.Quote(target),
		}
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil || minor < 0 {
		return 0, &shellCompletionError{
			kind:   errInvalidShellCompletionGeneration,
			detail: "invalid Elvish completion generation target version " + strconv.Quote(target),
		}
	}
	if major == 0 && minor < elvishLegacyMinor011 {
		return 0, &shellCompletionError{
			kind:   errUnsupportedShellCompletionTarget,
			detail: "Elvish completion generation target version " + strconv.Quote(target),
		}
	}
	if major == 0 && minor == elvishLegacyMinor011 {
		return elvishGeneration011, nil
	}
	if major == 0 && minor == elvishLegacyMinor012 {
		return elvishGeneration012, nil
	}
	return elvishGenerationCurrent, nil
}

func generateElvishCompletionScript(request shellCompletionScriptRequest) (string, error) {
	identifier := shellCompletionIdentifier(request.ApplicationName)
	capability, err := elvishGenerationCapabilityForTarget(request.TargetVersion)
	if err != nil {
		return "", err
	}
	template := elvishCompletionScript
	shellVersion := "$version"
	switch capability {
	case elvishGenerationCurrent:
	case elvishGeneration011:
		template = elvish011CompletionScript
		shellVersion = elvishSingleQuoted(request.TargetVersion)
	case elvishGeneration012:
		template = elvish012CompletionScript
		shellVersion = elvishSingleQuoted(request.TargetVersion)
	}
	return renderShellCompletionScript(template,
		"@@ID@@", identifier,
		"@@APP@@", elvishSingleQuoted(request.ApplicationName),
		"@@ACTIVATION@@", elvishSingleQuoted(request.ActivationEnvironment),
		"@@ACTIVATION_IDENT@@", request.ActivationEnvironment,
		"@@ACTIVATION_VALUE@@", elvishSingleQuoted(request.ActivationValue),
		"@@SHELL_VERSION@@", shellVersion,
	), nil
}

const elvish011CompletionScript = `# Completion integration for @@APP@@.
# Generated code: source this file from Elvish 0.11.
__completion_@@ID@@_app = @@APP@@
fn __completion_@@ID@@_append_byte [out byte]{
    hex = [0 1 2 3 4 5 6 7 8 9 A B C D E F]
    high = (/ (- $byte (% $byte 16)) 16)
    low = (% $byte 16)
    joins '' [$out '\x' $hex[$high] $hex[$low]]
}

fn __completion_@@ID@@_record_escape [value]{
    if (==s $value '') { put ''; return }
    out = ''
    for codepoint [(ord $value)] {
        if (<= $codepoint 0x7f) {
            out = (__completion_@@ID@@_append_byte $out $codepoint)
        } elif (<= $codepoint 0x7ff) {
            quotient = (/ (- $codepoint (% $codepoint 64)) 64)
            out = (__completion_@@ID@@_append_byte $out (+ 0xc0 $quotient))
            out = (__completion_@@ID@@_append_byte $out (+ 0x80 (% $codepoint 64)))
        } elif (<= $codepoint 0xffff) {
            quotient1 = (/ (- $codepoint (% $codepoint 64)) 64)
            quotient2 = (/ (- $quotient1 (% $quotient1 64)) 64)
            out = (__completion_@@ID@@_append_byte $out (+ 0xe0 $quotient2))
            out = (__completion_@@ID@@_append_byte $out (+ 0x80 (% $quotient1 64)))
            out = (__completion_@@ID@@_append_byte $out (+ 0x80 (% $codepoint 64)))
        } else {
            quotient1 = (/ (- $codepoint (% $codepoint 64)) 64)
            quotient2 = (/ (- $quotient1 (% $quotient1 64)) 64)
            quotient3 = (/ (- $quotient2 (% $quotient2 64)) 64)
            out = (__completion_@@ID@@_append_byte $out (+ 0xf0 $quotient3))
            out = (__completion_@@ID@@_append_byte $out (+ 0x80 (% $quotient2 64)))
            out = (__completion_@@ID@@_append_byte $out (+ 0x80 (% $quotient1 64)))
            out = (__completion_@@ID@@_append_byte $out (+ 0x80 (% $codepoint 64)))
        }
    }
    put $out
}

fn __completion_@@ID@@_request [shell-version cursor-argument cursor-byte @arguments]{
    printf 'records\n'
    printf 'response-encoding\tjson\n'
    printf 'shell\telvish\t%s\n' (__completion_@@ID@@_record_escape $shell-version)
    printf 'input\tuncertain\t0\t%d\t%d\tobserved-word-bytes\n' $cursor-argument $cursor-byte
    index = 0
    for argument $arguments {
        printf 'word\t%d\t%s\texpanded\n' $index (__completion_@@ID@@_record_escape $argument)
        index = (+ $index 1)
    }
    printf 'matching\tclient\n'
    printf 'capability\tdescriptions\n'
    printf 'end\n'
}

fn __completion_@@ID@@_bytes [value]{
    if (==s $value '') { put 0; return }
    size = 0
    for codepoint [(ord $value)] {
        if (<= $codepoint 0x7f) { size = (+ $size 1) } elif (<= $codepoint 0x7ff) { size = (+ $size 2) } elif (<= $codepoint 0xffff) { size = (+ $size 3) } else { size = (+ $size 4) }
    }
    put $size
}

fn __completion_@@ID@@_complete [@all-arguments]{
    if (< (count $all-arguments) 2) { return }
    arguments = $all-arguments[1:]
    cursor-argument = (- (count $arguments) 1)
    current = $arguments[-1]
    cursor-byte = (__completion_@@ID@@_bytes $current)

    response = [&]
    try {
        request = (__completion_@@ID@@_request @@SHELL_VERSION@@ $cursor-argument $cursor-byte $@arguments | slurp)
        response = (printf '%s' $request | E:@@ACTIVATION_IDENT@@=@@ACTIVATION_VALUE@@` +
	` (external $__completion_@@ID@@_app)  2>/dev/null | from-json)
    } except _ {
        return
    }

    replace = $response[replace]
    if (!= $replace[argument] $cursor-argument) { return }
    if (or (!= $replace[start] 0) (!= $replace[end] $cursor-byte)) { return }

    for item $response[items] {
        value = $item[value]
        if (has-key $item replacement) {
            if (!= (count $item[replacement]) 1) { continue }
            value = $item[replacement][0]
        }
        display-suffix = ''
        if (and (has-key $item description) (!=s $item[description] '')) {` +
	` display-suffix = (joins '' [' - ' $item[description]]) }
        spacing = $item[spacing]
        if (==s $spacing default) { spacing = $response[spacing] }
        code-suffix = ''
        if (!=s $spacing no-space) { code-suffix = ' ' }
        edit:complex-candidate &display-suffix=$display-suffix &code-suffix=$code-suffix $value
    }
}

if (has-key $edit:arg-completer $__completion_@@ID@@_app) {
    fail (printf 'completion: completion already registered for %s' $__completion_@@ID@@_app)
}
edit:arg-completer[$__completion_@@ID@@_app] = $__completion_@@ID@@_complete~
`

const elvish012CompletionScript = `# Completion integration for @@APP@@.
# Generated code: source this file from Elvish 0.12.
__completion_@@ID@@_app = @@APP@@
fn __completion_@@ID@@_append_byte [out byte]{
    joins '' [$out (printf '\\x%02X' $byte | slurp)]
}

fn __completion_@@ID@@_record_escape [value]{
    if (==s $value '') { put ''; return }
    out = ''
    for codepoint [(ord $value)] {
        if (<= $codepoint 0x7f) {
            out = (__completion_@@ID@@_append_byte $out $codepoint)
        } elif (<= $codepoint 0x7ff) {
            quotient = (/ (- $codepoint (% $codepoint 64)) 64)
            out = (__completion_@@ID@@_append_byte $out (+ 0xc0 $quotient))
            out = (__completion_@@ID@@_append_byte $out (+ 0x80 (% $codepoint 64)))
        } elif (<= $codepoint 0xffff) {
            quotient1 = (/ (- $codepoint (% $codepoint 64)) 64)
            quotient2 = (/ (- $quotient1 (% $quotient1 64)) 64)
            out = (__completion_@@ID@@_append_byte $out (+ 0xe0 $quotient2))
            out = (__completion_@@ID@@_append_byte $out (+ 0x80 (% $quotient1 64)))
            out = (__completion_@@ID@@_append_byte $out (+ 0x80 (% $codepoint 64)))
        } else {
            quotient1 = (/ (- $codepoint (% $codepoint 64)) 64)
            quotient2 = (/ (- $quotient1 (% $quotient1 64)) 64)
            quotient3 = (/ (- $quotient2 (% $quotient2 64)) 64)
            out = (__completion_@@ID@@_append_byte $out (+ 0xf0 $quotient3))
            out = (__completion_@@ID@@_append_byte $out (+ 0x80 (% $quotient2 64)))
            out = (__completion_@@ID@@_append_byte $out (+ 0x80 (% $quotient1 64)))
            out = (__completion_@@ID@@_append_byte $out (+ 0x80 (% $codepoint 64)))
        }
    }
    put $out
}

fn __completion_@@ID@@_request [shell-version cursor-argument cursor-byte @arguments]{
    printf 'records\n'
    printf 'response-encoding\tjson\n'
    printf 'shell\telvish\t%s\n' (__completion_@@ID@@_record_escape $shell-version)
    printf 'input\tuncertain\t0\t%d\t%d\tobserved-word-bytes\n' $cursor-argument $cursor-byte
    index = 0
    for argument $arguments {
        printf 'word\t%d\t%s\texpanded\n' $index (__completion_@@ID@@_record_escape $argument)
        index = (+ $index 1)
    }
    printf 'matching\tclient\n'
    printf 'capability\tdescriptions\n'
    printf 'end\n'
}

fn __completion_@@ID@@_bytes [value]{
    if (==s $value '') { put 0; return }
    size = 0
    for codepoint [(ord $value)] {
        if (<= $codepoint 0x7f) { size = (+ $size 1) } elif (<= $codepoint 0x7ff) { size = (+ $size 2) } elif (<= $codepoint 0xffff) { size = (+ $size 3) } else { size = (+ $size 4) }
    }
    put $size
}

fn __completion_@@ID@@_complete [@all-arguments]{
    if (< (count $all-arguments) 2) { return }
    arguments = $all-arguments[1:]
    cursor-argument = (- (count $arguments) 1)
    current = $arguments[-1]
    cursor-byte = (__completion_@@ID@@_bytes $current)

    response = [&]
    try {
        external~ = (external $__completion_@@ID@@_app)
        response = (__completion_@@ID@@_request @@SHELL_VERSION@@ $cursor-argument $cursor-byte $@arguments |` +
	` E:@@ACTIVATION_IDENT@@=@@ACTIVATION_VALUE@@ $external~  2>/dev/null | from-json)
    } except _ {
        return
    }

    replace = $response[replace]
    if (!= $replace[argument] $cursor-argument) { return }
    if (or (!= $replace[start] 0) (!= $replace[end] $cursor-byte)) { return }

    for item $response[items] {
        value = $item[value]
        if (has-key $item replacement) {
            if (!= (count $item[replacement]) 1) { continue }
            value = $item[replacement][0]
        }
        display-suffix = ''
        if (and (has-key $item description) (!=s $item[description] '')) {` +
	` display-suffix = (joins '' [' - ' $item[description]]) }
        spacing = $item[spacing]
        if (==s $spacing default) { spacing = $response[spacing] }
        code-suffix = ''
        if (!=s $spacing no-space) { code-suffix = ' ' }
        edit:complex-candidate &display-suffix=$display-suffix &code-suffix=$code-suffix $value
    }
}

if (has-key $edit:completion:arg-completer $__completion_@@ID@@_app) {
    fail (printf 'completion: completion already registered for %s' $__completion_@@ID@@_app)
}
edit:completion:arg-completer[$__completion_@@ID@@_app] = $__completion_@@ID@@_complete~
`

const elvishCompletionScript = `# Completion integration for @@APP@@.
# Generated code: source this file from current Elvish.
use str

var __completion_@@ID@@_app = @@APP@@
var __completion_@@ID@@_activation = @@ACTIVATION@@
var __completion_@@ID@@_external~ = (external $__completion_@@ID@@_app)

fn __completion_@@ID@@_record_escape {|value|
    var out = ''
    for byte [(str:to-utf8-bytes $value)] {
        var encoded = ''
        if (== $byte 0x5c) {
            set encoded = '\\\\'
        } elif (== $byte 0x09) {
            set encoded = '\\t'
        } elif (== $byte 0x0a) {
            set encoded = '\\n'
        } elif (== $byte 0x0d) {
            set encoded = '\\r'
        } elif (and (>= $byte 0x20) (<= $byte 0x7e)) {
            set encoded = (str:from-utf8-bytes $byte)
        } else {
            set encoded = (printf '\\x%02X' $byte | slurp)
        }
        set out = (str:join '' [$out $encoded])
    }
    put $out
}

fn __completion_@@ID@@_request {|shell-version cursor-argument cursor-byte @arguments|
    printf "records\n"
    printf "response-encoding\tjson\n"
    printf "shell\telvish\t%s\n" (__completion_@@ID@@_record_escape $shell-version)
    printf "input\tuncertain\t0\t%d\t%d\tobserved-word-bytes\n" $cursor-argument $cursor-byte
    var index = 0
    for argument $arguments {
        printf "word\t%d\t%s\tlexical\n" $index (__completion_@@ID@@_record_escape $argument)
        set index = (+ $index 1)
    }
    printf "matching\tclient\n"
    printf "capability\tdescriptions\n"
    printf "end\n"
}

fn __completion_@@ID@@_bytes {|value|
    str:to-utf8-bytes $value | count
}

fn __completion_@@ID@@_complete {|@all-arguments|
    # Elvish calls argument completers with the command name followed by all
    # arguments, retaining the final empty argument at a completion boundary.
    if (< (count $all-arguments) 2) { return }
    var arguments = $all-arguments[1..]
    var cursor-argument = (- (count $arguments) 1)
    var current = $arguments[-1]
    var cursor-byte = (__completion_@@ID@@_bytes $current)

    var response = [&]
    var had-activation = (has-env $__completion_@@ID@@_activation)
    var old-activation = ''
    if $had-activation { set old-activation = (get-env $__completion_@@ID@@_activation) }
    set-env $__completion_@@ID@@_activation @@ACTIVATION_VALUE@@
    try {
        set response = (__completion_@@ID@@_request @@SHELL_VERSION@@ $cursor-argument $cursor-byte $@arguments |
            $__completion_@@ID@@_external~  2>/dev/null |
            from-json)
    } catch _ {
        return
    } finally {
        if $had-activation {
            set-env $__completion_@@ID@@_activation $old-activation
        } else {
            unset-env $__completion_@@ID@@_activation
        }
    }

    # A semantic completion failure may carry a valid partial result.
    var replace = $response[replace]
    if (!= $replace[argument] $cursor-argument) { return }
    # Elvish exposes decoded arguments but no byte-range replacement primitive.
    # Completion replacement offsets are UTF-8 bytes, so only exact replacement
    # of the current argument is representable without corrupting non-ASCII text.
    if (or (!= $replace[start] 0) (!= $replace[end] $cursor-byte)) { return }

    for item $response[items] {
        var value = $item[value]
        if (has-key $item replacement) {
            if (!= (count $item[replacement]) 1) { continue }
            set value = $item[replacement][0]
        }
        var display = $item[value]
        if (has-key $item display_text) { set display = $item[display_text] }
        if (and (has-key $item description) (!=s $item[description] '')) {
            set display = (str:join '' [$display ' - ' $item[description]])
        }
        var spacing = $item[spacing]
        if (==s $spacing default) { set spacing = $response[spacing] }
        var code-suffix = ''
        if (!=s $spacing no-space) { set code-suffix = ' ' }
        edit:complex-candidate $value &display=$display &code-suffix=$code-suffix
    }
}

set edit:completion:arg-completer[$__completion_@@ID@@_app] = $__completion_@@ID@@_complete~
`
