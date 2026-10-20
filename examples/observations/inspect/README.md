# Inspect

`alienctl-inspect` explains a specific invocation without running its command handler.

Build the utility from this directory:

```sh
go build -o alienctl-inspect .
```

Inspect a scan invocation:

```sh
./alienctl-inspect scan --range 5 mars
```

```text
command: scan
range: 5
world: mars
```
