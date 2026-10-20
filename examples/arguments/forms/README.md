# Argument forms

`alienctl navigate` accepts either a world name or three coordinates as its destination.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Navigate by world:

```sh
./alienctl navigate mars
```

```text
navigating to mars
```

Or navigate by coordinates:

```sh
./alienctl navigate 12 8 4
```

```text
navigating to coordinates 12,8,4
```
