# Native Go shell

`gosh` runs mvdan/sh's upstream parser, expansion and interpreter on KolibriOS.
It is available to any application that needs a shell; its build uses the shared
SDK sources and does not depend on OpenCode's source tree or vendor cache.

From the SDK root in WSL:

```sh
make -C apps/tools/gosh GO="$PWD/tooling/gccgo/gccgo-kolibri" FAST_PKG=1
```

The interpreter supports POSIX shell syntax and some Bash features. It does not
provide all GNU Bash features. External commands need native executables.
