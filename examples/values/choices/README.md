# Value choices

`alienctl visit` restricts its approach to known choices and supplies a default.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Use the default approach:

```sh
./alienctl visit europa
```

```text
visiting europa by orbit
```

Or choose another approach:

```sh
./alienctl visit --approach land europa
```

```text
visiting europa by land
```
