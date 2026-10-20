# Counting flags

`alienctl scan` counts repeated `-v` flags as a verbosity level.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Increase verbosity by repeating the flag:

```sh
./alienctl scan -v -v -v mars
```

```text
scanning mars with verbosity 3
```
