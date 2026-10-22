# Default help

`alienctl` shows Commandment's built-in Help presentation for a realistic command, including groups, aliases, shortcuts, inherited flags, finite values, lifecycle guidance, sensitive input, and examples.

Build the CLI from this directory:

```sh
go build -o alienctl .
```

Inspect the application:

```sh
./alienctl --help
```

```text
Navigate expedition craft across nearby worlds.

Usage:
  alienctl [flags] <command>

Navigation:
  navigate  Navigate the expedition to a destination.
  travel    Navigate the expedition to a destination.
            [alias]
  home      Navigate directly to Earth.
            [shortcut]

Commands:
  help        Show help.
  completion  Generate shell completion source

Flags:
  --verbose  Show navigation detail.
  --help     Show help.
  -h         Show help.
  --version  Show version.

Examples:
Return the expedition home.
  alienctl home
```

Inspect the nested command:

```sh
./alienctl navigate --help
```

```text
Navigate the expedition to a destination.

Usage:
  alienctl navigate [flags] <destination>

Arguments:
  <destination>  World or station to navigate to.

Flags:
  --approach METHOD, --approach=METHOD
    Choose the navigation approach.
    METHOD:
      orbit - Enter a stable orbit.
      teleport - Use the legacy transporter.
        [deprecated 2: transporter navigation is being retired; use warp]
      warp - Use a controlled warp corridor.
      cruise (alias for orbit)
  --token[=TOKEN]
    Authenticate the navigation link.

Global flags:
  --verbose  Show navigation detail.
  --help     Show help.
  -h         Show help.

Examples:
Travel to Europa by warp corridor.
  alienctl navigate europa --approach=warp
```

The sensitive implicit token value is intentionally absent. The visible `home` shortcut can be inspected through either standard Help route:

```sh
./alienctl help home
./alienctl home --help
```

Both resolve to the same canonical `navigate` Help page shown above without applying the shortcut's fixed destination.
