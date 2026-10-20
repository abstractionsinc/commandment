# Dynamic choices

`alienctl visit` changes the valid moons according to the selected planetary system.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Visit Europa in the Jovian system:

```sh
./alienctl visit --system jovian europa
```

```text
visiting europa in the jovian system
```
