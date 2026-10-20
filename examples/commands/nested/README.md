# Nested commands

`alienctl fleet` groups operations that act on the fleet.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Check fleet status:

```sh
./alienctl fleet status
```

```text
fleet status: 3 ships ready
```

Launch a ship:

```sh
./alienctl fleet launch scout-7
```

```text
launching scout-7
```
