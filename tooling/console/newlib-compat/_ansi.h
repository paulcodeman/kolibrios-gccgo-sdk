/* Compile unchanged portable newlib string sources with the SDK host GCC. */
#ifndef KOLIBRI_NEWLIB_ANSI_H
#define KOLIBRI_NEWLIB_ANSI_H
#define _PTR void *
#define _CONST const
#define _AND ,
#define _DEFUN(name, argument_names, declarations) name(declarations)
#endif
