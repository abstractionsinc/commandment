# Invocation defaults

The host supplies `scout-7` as the current ship, but ordinary CLI input can override it.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Use the host default:

```sh
./alienctl status
```

```text
scout-7 status: nominal
```

Override it explicitly:

```sh
./alienctl status --ship voyager
```

```text
voyager status: nominal
```
