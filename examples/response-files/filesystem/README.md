# Filesystem response files

`alienctl hail` accepts ordinary CLI input from a response file on disk.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

`crew.args` contains the name `Zorblax`. Use it as input:

```sh
./alienctl hail @crew.args
```

```text
hail Zorblax
```
