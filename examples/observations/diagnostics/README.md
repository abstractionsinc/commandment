# Diagnostics

`alienctl-diagnose` prints the structured diagnostic code for an invalid `alienctl` invocation.

Build the utility from this directory:

```sh
go build -o alienctl-diagnose .
```

Analyze an invalid contact invocation:

```sh
./alienctl-diagnose contact
```

```text
diagnostic: rule-violation
```
