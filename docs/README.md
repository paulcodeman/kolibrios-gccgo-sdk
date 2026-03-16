# Documentation

This directory contains build and porting notes for the KolibriOS GCCGO SDK.

## Console readline (KolibriOS)

The Kolibri-native `gopkg.in/readline.v1` implementation (used by `apps/otto` and any REPLs that depend on it) supports:

- `Enter`: submit line
- `Backspace` and `Delete`: edit line
- `Left`/`Right` arrows, `Home`, `End`: cursor navigation
- `Up`/`Down`: history navigation
- `Tab`: completion (common prefix insertion or list display)
- `Ctrl+C` or `Esc`: interrupt (returns `readline.ErrInterrupt`)
- `Ctrl+D`: EOF when the line is empty

Behavior:

- Uses `Console.Getch2()` on the active console when available
- Falls back to buffered stdin otherwise
