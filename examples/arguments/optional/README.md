# Optional arguments

`alienctl status` accepts an optional ship name. Without one, it reports the mothership.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Use the default subject:

```sh
./alienctl status
```

```text
mothership status: nominal
```

Or name a ship:

```sh
./alienctl status scout-7
```

```text
scout-7 status: nominal
```
