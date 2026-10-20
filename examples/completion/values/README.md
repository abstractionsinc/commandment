# Value completion

`alienctl-complete` asks Commandment for completion candidates matching partial CLI input.

Build the utility from this directory:

```sh
go build -o alienctl-complete .
```

Complete a partial world name:

```sh
./alienctl-complete visit eu
```

```text
europa
```
