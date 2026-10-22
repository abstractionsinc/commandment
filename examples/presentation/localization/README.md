# Localization

`alienctl` shows its scan description in French when `--locale fr` is selected.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Request French help:

```sh
./alienctl --locale fr help scan
```

```text
analyser les mondes proches

Usage:
  alienctl scan [flags]

Global flags:
  --help  Show help.
  -h      Show help.
```
