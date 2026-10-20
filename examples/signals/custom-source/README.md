# Custom signal sources

`alienctl watch` receives an alarm from a custom in-process signal source instead of the operating system.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Watch for the custom alarm:

```sh
./alienctl watch
```

```text
received alarm
```
