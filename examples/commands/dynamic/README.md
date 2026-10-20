# Command provider

`alienctl outpost` discovers different outpost commands for the selected planetary system.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Contact a Jovian outpost:

```sh
./alienctl outpost --system jovian europa
```

```text
contacting europa outpost
```

Select a different system and a different discovered command:

```sh
./alienctl outpost --system saturnian titan
```

```text
contacting titan outpost
```
