# Lexical aliases

`alienctl scan` also accepts `survey` as another name for the same command.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Both names select the same operation:

```sh
./alienctl scan mars
./alienctl survey mars
```

```text
scanning mars
scanning mars
```
