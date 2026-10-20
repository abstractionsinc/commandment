# Custom response loader

`alienctl hail` loads response-file content from a host-provided source instead of the filesystem.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Load the logical `crew` response source:

```sh
./alienctl hail @crew
```

```text
hail Zorblax
```
