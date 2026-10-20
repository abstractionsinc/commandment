package cli

func generatePowerShellCompletionScript(request shellCompletionScriptRequest) string {
	identifier := shellCompletionIdentifier(request.ApplicationName)
	return renderShellCompletionScript(powerShellCompletionScript,
		"@@ID@@", identifier,
		"@@APP@@", powerShellSingleQuoted(request.ApplicationName),
		"@@ACTIVATION@@", powerShellSingleQuoted(request.ActivationEnvironment),
		"@@ACTIVATION_VALUE@@", powerShellSingleQuoted(request.ActivationValue),
		"@@BT@@", "`",
	)
}

const powerShellCompletionScript = `# Completion integration for @@APP@@.
# Generated code: dot-source this file from Windows PowerShell 5.1+ or PowerShell 7+.
$script:Completion_@@ID@@_App = @@APP@@
$script:Completion_@@ID@@_Activation = @@ACTIVATION@@

function Completion_@@ID@@_RecordEscape([string] $Value) {
    $builder = [System.Text.StringBuilder]::new()
    foreach ($byte in [System.Text.Encoding]::UTF8.GetBytes($Value)) {
        switch ($byte) {
            92 { [void] $builder.Append('\\') }
            9  { [void] $builder.Append('\t') }
            10 { [void] $builder.Append('\n') }
            13 { [void] $builder.Append('\r') }
            default {
                if ($byte -ge 0x20 -and $byte -le 0x7e) {
                    [void] $builder.Append([char] $byte)
                } else {
                    [void] $builder.AppendFormat('\x{0:X2}', $byte)
                }
            }
        }
    }
    $builder.ToString()
}

function Completion_@@ID@@_RecordDecode([string] $Value) {
    $bytes = [System.Collections.Generic.List[byte]]::new()
    for ($i = 0; $i -lt $Value.Length; $i++) {
        $c = $Value[$i]
        if ($c -ne '\') {
            if ([int] $c -gt 0x7f) { return $null }
            $bytes.Add([byte] $c)
            continue
        }
        $i++
        if ($i -ge $Value.Length) { return $null }
        switch ($Value[$i]) {
            '\' { $bytes.Add(92) }
            't'  { $bytes.Add(9) }
            'n'  { $bytes.Add(10) }
            'r'  { $bytes.Add(13) }
            'x' {
                if ($i + 2 -ge $Value.Length) { return $null }
                $hex = $Value.Substring($i + 1, 2)
                try { $bytes.Add([Convert]::ToByte($hex, 16)) } catch { return $null }
                $i += 2
            }
            default { return $null }
        }
    }
    try {
        [System.Text.UTF8Encoding]::new($false, $true).GetString($bytes.ToArray())
    } catch {
        return $null
    }
}

function Completion_@@ID@@_QuoteArgument([string] $Value) {
    # CompletionText is shell source. A single-quoted PowerShell string preserves
    # arbitrary argument data without evaluating interpolation or metacharacters.
    "'" + $Value.Replace("'", "''") + "'"
}

function Completion_@@ID@@_ResultType([string] $Kind) {
    switch ($Kind) {
        'command' { 'Command' }
        'command-alias' { 'Command' }
        'command-shortcut' { 'Command' }
        'action' { 'Command' }
        'executable' { 'Command' }
        'flag' { 'ParameterName' }
        'file' { 'ProviderItem' }
        'directory' { 'ProviderContainer' }
        'environment-variable' { 'Variable' }
        'namespace' { 'Namespace' }
        'keyword' { 'Keyword' }
        default { 'ParameterValue' }
    }
}

function Completion_@@ID@@_Complete {
    param($WordToComplete, $CommandAst, $CursorPosition)

    $shellVersion = $PSVersionTable.PSVersion.ToString()
    if ($PSVersionTable.PSVersion.Major -lt 5) { return }

    # Windows PowerShell 5.1 keeps the established one-word uncertain observation.
    # PowerShell 7.6+ additionally exposes CommandAst and the native cursor position.
    # Use only host-decoded literal command elements proven to end strictly before
    # the cursor-owned current contribution as an exact argv prefix. Elements at or
    # after the cursor frontier must not affect topology or context. Never synthesize
    # execution argv from source text or from an AST shape we cannot establish as a
    # literal argument.
    $argsList = [System.Collections.Generic.List[string]]::new()
    $fidelityList = [System.Collections.Generic.List[string]]::new()
    $topology = 'uncertain'
    $exactArgvPrefix = 0
    if ($PSVersionTable.PSVersion.Major -eq 7 -and $PSVersionTable.PSVersion.Minor -ge 6 -and
        $null -ne $CommandAst -and $CommandAst.CommandElements.Count -ge 2) {
        $literalPrefix = [System.Collections.Generic.List[string]]::new()
        $literal = $true
        for ($i = 1; $i -lt $CommandAst.CommandElements.Count; $i++) {
            $element = $CommandAst.CommandElements[$i]
            # EndOffset == CursorPosition is the cursor-owned contribution: PowerShell
            # reports it through WordToComplete below, so treating it as already exact
            # would duplicate the current word. Any later element belongs to suffix text
            # beyond the native cursor and cannot participate in this completion request.
            if ($element.Extent.StartOffset -ge $CursorPosition -or
                $element.Extent.EndOffset -ge $CursorPosition) {
                break
            }
            if ($element -isnot [System.Management.Automation.Language.StringConstantExpressionAst]) {
                $literal = $false
                break
            }
            $literalPrefix.Add([string] $element.Value)
        }
        if ($literal -and $literalPrefix.Count -gt 0) {
            foreach ($argument in $literalPrefix) {
                $argsList.Add($argument)
                $fidelityList.Add('exact')
            }
            $topology = 'prefix-exact'
            $exactArgvPrefix = $literalPrefix.Count
        }
    }
    $argsList.Add([string] $WordToComplete)
    $fidelityList.Add('lexical')
    $cursorArgument = $argsList.Count - 1
    $cursorByte = [System.Text.Encoding]::UTF8.GetByteCount([string] $WordToComplete)

    $lines = [System.Collections.Generic.List[string]]::new()
    $lines.Add("records")
    $lines.Add("response-encoding@@BT@@trecords")
    $lines.Add("shell@@BT@@tpowershell@@BT@@t$(Completion_@@ID@@_RecordEscape $shellVersion)")
    $lines.Add("input@@BT@@t$topology@@BT@@t$exactArgvPrefix@@BT@@t$cursorArgument` +
	`@@BT@@t$cursorByte@@BT@@tobserved-word-bytes")
    for ($i = 0; $i -lt $argsList.Count; $i++) {
        $lines.Add("word@@BT@@t$i@@BT@@t$(Completion_@@ID@@_RecordEscape $argsList[$i])@@BT@@t$($fidelityList[$i])")
    }
    $lines.Add("matching@@BT@@tprefix")
    $lines.Add("capability@@BT@@tdescriptions")
    $lines.Add("capability@@BT@@tcandidate-kinds")
    $lines.Add('end')
    # Native-command pipeline input contributes the terminating newline.
    # Do not embed a second one after the Records "end" terminator.
    $requestText = [string]::Join("@@BT@@n", $lines)

    $oldActivation = [Environment]::GetEnvironmentVariable($script:Completion_@@ID@@_Activation, 'Process')
    [Environment]::SetEnvironmentVariable($script:Completion_@@ID@@_Activation, @@ACTIVATION_VALUE@@, 'Process')
    try {
        $response = @($requestText | & $script:Completion_@@ID@@_App 2>$null)
        if ($LASTEXITCODE -ne 0) { return }
    } finally {
        [Environment]::SetEnvironmentVariable($script:Completion_@@ID@@_Activation, $oldActivation, 'Process')
    }

    $failed = $true
    $sawEnd = $false
    $replaceArgument = -1
    $replaceStart = 0
    $replaceEnd = 0
    $resultSpacing = 'default'
    $items = @{}
    $replacements = @{}

    foreach ($line in $response) {
        $fields = @(([string] $line).Split([char]9))
        if ($fields.Count -eq 0) { continue }
        switch ($fields[0]) {
            'result' { $failed = ($fields[1] -ne 'false') }
            'replace' {
                $replaceArgument = [int] $fields[1]
                $replaceStart = [int] $fields[2]
                $replaceEnd = [int] $fields[3]
            }
            'item' {
                $index = [int] $fields[1]
                $value = Completion_@@ID@@_RecordDecode $fields[2]
                if ($null -eq $value) { return }
                $items[$index] = @{
                    Value = $value
                    Replacement = ($fields[3] -eq 'true')
                    Display = ''
                    Description = ''
                    Kind = ''
                    Spacing = $fields[4]
                }
            }
            'item-display' {
                $index = [int] $fields[1]
                if (-not $items.ContainsKey($index)) { continue }
                $display = Completion_@@ID@@_RecordDecode $fields[2]
                if ($null -eq $display) { return }
                $items[$index].Display = $display
            }
            'item-description' {
                $index = [int] $fields[1]
                if (-not $items.ContainsKey($index)) { continue }
                $description = Completion_@@ID@@_RecordDecode $fields[2]
                if ($null -eq $description) { return }
                $items[$index].Description = $description
            }
            'item-kind' {
                $index = [int] $fields[1]
                if (-not $items.ContainsKey($index)) { continue }
                $items[$index].Kind = $fields[2]
            }
            'item-replacement' {
                $index = [int] $fields[1]
                if (-not $replacements.ContainsKey($index)) { $replacements[$index] = @() }
                $replacement = Completion_@@ID@@_RecordDecode $fields[3]
                if ($null -eq $replacement) { return }
                $replacements[$index] += ,$replacement
            }
            'spacing' { $resultSpacing = $fields[1] }
            'end' { $sawEnd = $true }
        }
    }

    # A semantic completion failure may carry a valid partial result.
    if (-not $sawEnd -or $replaceArgument -ne $cursorArgument) { return }
    $current = [string] $WordToComplete
    $currentBytes = [System.Text.Encoding]::UTF8.GetByteCount($current)
    if ($replaceStart -ne 0 -or $replaceEnd -ne $currentBytes) { return }

    # PowerShell 7.6 treats an empty-string native completer result as handled:
    # PSReadLine swallows the engine's invalid CompletionResult exception and does
    # not fall back to filename completion. PowerShell 7.7 preserves that user-
    # visible suppression with repaired internals. Keep Windows PowerShell 5.1
    # and unproven PowerShell 7.0-7.5 on their legacy no-object fallback path.
    # Only a fully successful semantic zero qualifies; locally unrepresentable
    # nonzero items remain nonzero because $items still records them here.
    if (-not $failed -and $items.Count -eq 0 -and
        $PSVersionTable.PSVersion.Major -eq 7 -and $PSVersionTable.PSVersion.Minor -ge 6) {
        return ''
    }

    foreach ($index in @($items.Keys | Sort-Object {[int] $_})) {
        $item = $items[$index]
        $value = [string] $item.Value
        if ($item.Replacement) {
            $replacement = @($replacements[$index])
            if ($replacement.Count -ne 1) { continue }
            $value = [string] $replacement[0]
        }
        $spacing = [string] $item.Spacing
        if ($spacing -eq 'default') { $spacing = $resultSpacing }
        $value = Completion_@@ID@@_QuoteArgument $value
        if ($spacing -ne 'no-space') { $value += ' ' }
        $display = [string] $item.Display
        if ([string]::IsNullOrEmpty($display)) { $display = [string] $item.Value }
        $description = [string] $item.Description
        if ([string]::IsNullOrEmpty($description)) { $description = $display }
        [System.Management.Automation.CompletionResult]::new(
            $value,
            $display,
            (Completion_@@ID@@_ResultType ([string] $item.Kind)),
            $description
        )
    }
}

Register-ArgumentCompleter -Native -CommandName @@APP@@ -ScriptBlock ${function:Completion_@@ID@@_Complete}
`
