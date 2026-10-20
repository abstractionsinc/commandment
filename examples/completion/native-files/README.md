# Native file completion

`alienctl-complete` shows Commandment delegating file suggestions to the completion client.

Build the utility from this directory:

```sh
go build -o alienctl-complete .
```

Complete the manifest value:

```sh
./alienctl-complete upload --manifest ""
```

```text
native completion: files
```
