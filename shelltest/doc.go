// Package shelltest proves generated Commandment shell completion through a real
// shell editor. Use it when the contract under test includes generated shell
// integration, shell syntax, editor behavior, or another fact that cli.Complete
// and clitest.Complete cannot prove without a real shell process.
//
// A Target combines a Commandment shell-completion provider with Runtime process
// discovery, callback-scoped Setup preparation, and observation interpretation.
// Complete drives one editor interaction and returns the interpreted result.
// CompleteForGenerationTarget is the narrower variant for a test that must select
// an explicit shell completion generation target rather than using the target's
// detected runtime version.
//
// Setup and Observation values supplied to callbacks are scoped to those callbacks
// and must not be retained. Runtime process ownership, PTY/editor interaction,
// generated integration files, settlement, and cleanup remain package concerns;
// application completion semantics remain owned by package cli. Use clitest for
// semantic completion expectations that do not require a real shell.
package shelltest
