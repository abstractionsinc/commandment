# Application errors

`alienctl land` refuses a protected destination and maps that application error to exit status `7`.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Try to land on protected Earth:

```sh
./alienctl land earth
```

The error is written to stderr:

```text
landing denied: earth is protected
```

The command exits with status `7`.
