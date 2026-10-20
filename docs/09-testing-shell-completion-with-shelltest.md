# Testing Shell Completion with shelltest

Use `shelltest` when the fact under test belongs to a real supported shell/editor integration rather than Commandment's semantic completion result alone.

```text
app.Complete / clitest.Complete
    -> semantic completion is correct

shelltest
    -> generated integration works in the real shell/editor
```

Use the semantic boundary first, and move to `shelltest` only when quoting, editor behavior, generated adapter source, protocol transport, or the shell runtime itself is part of the claim.

## Prove semantic completion first

Use `clitest.Complete` to check the completion target, completion candidates, native delegation, completion matching and ranking, partial state, or structured diagnostics. If the semantic result is wrong, the shell adapter is not the first owner of the bug.

When semantic completion is correct and the remaining question is what a supported shell/editor actually does, use `shelltest`.

## A Target describes one real-shell integration

A `shelltest.Target[T]` binds a Commandment shell completion provider to the runtime, setup, and interpretation required for a real shell.

```go
runtimeConfig := shelltest.Runtime{
	Executable:   bash,
	VersionArgs:  []string{"--version"},
	ParseVersion: parseBashVersion,
}

target := shelltest.Target[string]{
	Provider:  cli.Shell.Bash,
	Runtime:   runtimeConfig,
	Prepare:   prepareBash,
	Interpret: interpretBash,
}
```

`Runtime` starts and identifies the shell, while `Provider` owns Commandment's shell-completion generation. `Prepare` configures the callback-scoped shell environment, and `Interpret` turns the settled editor observation into the stable result the test cares about.

GoDoc owns exact target generics and lifetime contracts.

## Setup is callback-scoped

The setup capability passed to `Prepare` belongs to one target preparation. Configure the required executable/integration files, extra files, environment, and arguments there, then let the callback return.

Do not retain setup capabilities or use them to mutate an already-running session. Keep setup limited to what the integration proof actually needs rather than recreating a user's full shell configuration.

## Complete drives the real editor path

`shelltest.Complete` uses the generated integration, starts the target shell, drives editor state, waits for the interaction to settle, and returns the interpreted result.

```go
input := shelltest.Input{BeforeCursor: "alienctl re"}
screen := shelltest.Complete(t, app, target, input)
```

The shell participates in this proof, so quoting, shell syntax, editor behavior, generated adapter source, and the bridge back to Commandment are inside the boundary.

The [shell completion scenario](../examples/completion/shell/) contains the canonical runnable test shape.

## Generation target and runtime version are different

A shell completion generation target controls the syntax/version of emitted integration source. A runtime is the actual shell process used for proof.

They normally align through the detected runtime version, but `shelltest` also supports an explicit generation target when compatibility with a particular emitted syntax is the subject. Do not substitute a convenient runtime version for the generation target you intend to support.

## Interpret stable shell evidence

Interpret the smallest stable user-visible fact you need from the settled shell/editor observation. Avoid coupling tests to incidental terminal control sequences, timing, or internal bridge traffic when editor behavior is the real contract.

## Let shelltest own process lifecycle

Real-shell tests own subprocesses, PTYs, generated integration files, bridge communication, and cleanup. Do not start competing shell processes around `shelltest`.

Skipping is appropriate only when a required supported shell/runtime is genuinely unavailable in the current environment. A skip must not hide a semantic or integration failure. GoDoc and the repository's supported verification environment own exact platform prerequisites.

## Keep clitest and shelltest separate

Use `clitest` for semantic completion facts such as targets, candidate identity, native delegation, matching/ranking, partial state, and diagnostics. Use `shelltest` for generated integration loading, real editor behavior, quoting/protocol behavior, and supported generation-target/runtime compatibility.

Ordinary process tests own process behavior unrelated to shell completion.

See [Operations and Presentation](07-operations-and-presentation.md) for the semantic completion model and [Troubleshooting](11-troubleshooting.md) for mismatches between the semantic and real-shell boundaries.
