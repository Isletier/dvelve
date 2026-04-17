## DVELVE

NOTE: Current implementation is very WIP (I really suck at Python). DO NOT USE IT unless you want to participate in its development.

This is a client implementation(and a fork) of original [delve debugger](https://github.com/go-delve/delve) and an implementation of [DVAP(Debug view adapter protocol)](https://github.com/Isletier/DVAP).

Please read the original documentation first for basic use-cases.

## dvelve usage:

```
shell$ gdb
gdb$ source ./DVAP_gdb_server.py
```

Or alternatively, just add the same line to the .gdbinit file in your $HOME directory; this will launch the server every time gdb starts:

```
source ./DVAP_gdb_server.py
```

## Neovim client:

https://github.com/Isletier/nvim/tree/dev

## About the concept

REPLs are cool, but no matter how well their UI is implemented, they suck at one particular thing: displaying text and, as a consequence, execution flow.

On the other side of the board, you have DAP. It was meant to solve the editor*debugger integration issue but ended up requiring same amount of configuration variations while significantly reducing debugger features that are language/debugger dependent.

So, I think I have a solution: let the debugger's native UI fully define the current state of the debugging session and its launch, while the editor just stays a passive observer of the current state of things—in particular, threads, breakpoints, and their positions.

I'm pretty sure this minimalistic interface could conform to any possible combination of editor/debugger/language, require zero non-UI configuration from the client side, and is generally more idiomatic for text interfaces and editors than the DAP IDE-like approach. All you need to do to start observing the debug session is just pass an endpoint to the DVAP server.

## for VS code/DAP victims like me:

## References

