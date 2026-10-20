# Command shortcuts

`alienctl home` is a convenience command for navigating to Earth.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Use the canonical command:

```sh
./alienctl navigate mars
```

```text
navigating to mars
```

Or take the shortcut home:

```sh
./alienctl home
```

```text
navigating to earth
```
