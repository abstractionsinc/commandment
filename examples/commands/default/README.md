# Default relationship

`alienctl fleet` defaults to `status`, while other fleet commands remain available normally.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Run the parent directly:

```sh
./alienctl fleet
```

```text
fleet status: 3 ships ready
```

The explicit child produces the same result:

```sh
./alienctl fleet status
```

```text
fleet status: 3 ships ready
```

Another child is still selected explicitly:

```sh
./alienctl fleet launch scout-7
```

```text
launching scout-7
```
