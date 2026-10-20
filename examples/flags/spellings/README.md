# Flag spellings

`alienctl scan` gives the range option both long and short spellings.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Use either spelling:

```sh
./alienctl scan --range 5 mars
./alienctl scan -r 5 mars
```

```text
scanning mars at range 5
scanning mars at range 5
```
