# Advisories

`alienctl approach` warns about a reckless approach without preventing execution, and `--quiet` suppresses that notice.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Approach recklessly:

```sh
./alienctl approach --reckless earth
```

The warning is written to stderr:

```text
warning: reckless approach enabled
```

The command still runs and writes its normal result to stdout:

```text
approaching earth
```

Suppress the warning:

```sh
./alienctl approach --reckless --quiet earth
```

```text
approaching earth
```
