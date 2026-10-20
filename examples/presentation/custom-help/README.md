# Custom help

`alienctl` renders help itself using the structured `cli.HelpPage` Commandment supplies.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Ask for scan help:

```sh
./alienctl help scan
```

```text
ALIENCTL scan
scan a nearby world
```
