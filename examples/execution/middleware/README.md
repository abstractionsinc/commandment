# Middleware

`alienctl scan` uses middleware to run work before and after its handler.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Scan Mars:

```sh
./alienctl scan mars
```

```text
preparing sensors
scanning mars
recording scan
```
