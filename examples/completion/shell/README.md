# Shell completion

This example installs Commandment's generated Bash completion for a real `alienctl` executable. It requires Bash on a Unix-like host.

Build the CLI from this directory and make it available to the shell:

```sh
go build -o alienctl .
export PATH="$PWD:$PATH"
source <(alienctl completion bash)
```

At an interactive Bash prompt, type:

```text
alienctl sc<TAB>
```

The command completes to `scan`.
