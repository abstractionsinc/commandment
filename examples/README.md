# Examples

Commandment examples are grouped by the CLI problem they teach. Each scenario is an independent fictional application or developer utility unless its README says otherwise.

If you are building your first Commandment application, start with [Getting Started](../docs/01-getting-started.md). For a focused example, choose the collection that matches what you are trying to build.

- [Arguments](arguments/) - positional input, alternate forms, and pass-through command lines.
- [Commands](commands/) - nesting, lexical aliases, command shortcuts, default relationships, and command-provider-produced commands.
- [Flags](flags/) - alternate flag spellings and repeated counting flags.
- [Values](values/) - static and invocation-dependent choices.
- [Host context](host-context/) - trusted facts and overridable invocation defaults supplied by the host.
- [Policies](policies/) - conditional declarations, advisories, and notice suppression.
- [Completion](completion/) - value completion, completion delegation, and shell completion integration.
- [Execution](execution/) - middleware and application-owned error behavior.
- [Presentation](presentation/) - custom help, localization, and version output.
- [Observations](observations/) - Inspect, Schema, Reference, and structured diagnostics for developer tools.
- [Response files](response-files/) - filesystem and host-provided response sources.
- [Signals](signals/) - process interrupts and custom typed signal sources.

Every scenario README gives commands relative to that scenario directory. The complete program and focused test are beside it in `main.go` and `main_test.go`.
