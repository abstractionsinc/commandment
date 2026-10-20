# Interrupts

`alienctl listen` routes a real Ctrl-C through Commandment's signal policy and shuts down successfully on a Unix-like host.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Start listening:

```sh
./alienctl listen
```

```text
listening for signals
```

Press Ctrl-C. The application handles the interrupt and prints:

```text
signal received
```
