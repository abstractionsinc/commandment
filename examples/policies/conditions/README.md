# Conditions

`--cloak-level` is available only when `alienctl approach` is operating in stealth mode.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Approach under cloak:

```sh
./alienctl approach --stealth --cloak-level 3 earth
```

```text
approaching earth under cloak level 3
```
