# Pass-through arguments

`alienctl relay` forwards everything after `--` to a rover unchanged.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Relay a command:

```sh
./alienctl relay rover-7 -- echo hello earth
```

```text
relaying to rover-7: echo hello earth
```
