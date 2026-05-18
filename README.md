## DVELVE

NOTE: Current implementation is still very WIP.

This is a fork of the [delve debugger](https://github.com/go-delve/delve) with an implementation of [DVAP (Debug View Adapter Protocol)](https://github.com/Isletier/DVAP) added to it.

Please read the original delve documentation first for basic use-cases.

## Installation

Requires Go 1.24 or later. The binary installs as `dvlv`.

```
shell$ go install github.com/Isletier/dvelve/dvlv@dvelve
```

Verify:

```
shell$ dvlv version
Dvelve Debugger (based on Delve)
Version: 1.26.1
...

shell$ dvlv debug --help | grep dvap
      --dvap string    Address for the DVAP SSE server (e.g. 127.0.0.1:56789). ...
```

## Usage

The only difference from vanilla delve is the `--dvap` flag, which takes the address of the SSE server dvelve will broadcast state to:

```
shell$ dvlv debug --dvap 127.0.0.1:56789
shell$ dvlv exec ./mybinary --dvap 127.0.0.1:56789
shell$ dvlv attach <pid> --dvap 127.0.0.1:56789
```

The flag is available on all subcommands that launch a local terminal session. It has no effect in headless mode.

Remote debugging scenarious are still possible - you can run debugger in `--headless` mode on your server and another debugger in `--connect` mod on your local machine. The `--connect` mod is still works with `--dvap` flag so you can freely combine them, also the the `--headless` debugger on your server could easiely be original delve, if you have strict policies on what should be installed on the server.

Once started, any client that connects to `http://127.0.0.1:56789/events` will receive the current debugger state as an SSE stream. The state is broadcast after every execution step or breakpoint change.

## DVAP commands

Once debugger started you can use following helper comamnds right inside the debugger REPL:

```
dvap start [addr]  start (or restart) the server
                   addr defaults to the previously used address or 127.0.0.0:56789
dvap stop          stop the server (debug session continues)
dvap show          print server status and listen address
dvap set <addr>    change the default address (takes effect on next start)
dvap help          print this message`)
```

## Neovim client

https://github.com/Isletier/nvim/tree/dev

## If you're coming from an IDE

Compile with debug symbols — pass `-gcflags="all=-N -l"` to disable optimizations:

```
shell$ go build -gcflags="all=-N -l" -o myprogram .
shell$ dvlv exec ./myprogram --dvap 127.0.0.1:56789
```

Or let delve build for you, which disables optimizations automatically:

```
shell$ dvlv debug --dvap 127.0.0.1:56789
```

Set a breakpoint:

```
(dvlv) b main.go:20
(dvlv) b main.MyFunction
```

Navigate execution:

```
(dvlv) continue   (or: c)     — resume
(dvlv) step       (or: s)     — step into
(dvlv) next       (or: n)     — step over
(dvlv) stepout    (or: so)    — step out
```

Inspect state:

```
(dvlv) goroutines            — list all goroutines
(dvlv) goroutine <id>        — switch to a goroutine
(dvlv) locals                — print local variables
(dvlv) p <expr>              — evaluate an expression
(dvlv) stack                 — print stack trace
```

How to use the editor and debugger at the same time: split your terminal with tmux, your desktop environment. The editor observes the session passively — you only ever type into the debugger.

## References

https://github.com/go-delve/delve/tree/master/Documentation

https://github.com/go-delve/delve/blob/master/Documentation/usage/dlv.md
