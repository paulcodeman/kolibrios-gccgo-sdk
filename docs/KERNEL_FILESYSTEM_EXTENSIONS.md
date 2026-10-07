# SDK atomic filesystem creation

These extensions are native kernel sources in the sibling ColibriOS fork,
documented in its `kernel/trunk/docs/sysfuncs.txt`. They are also implemented by
`build-kolibri-socket-kernel.py
--exclusive-create` in an isolated copy of the kernel. They do not change
the original checkout or the behavior of existing operations. Consult the
original `kernel/trunk/docs/sysfuncs.txt` for the base syscall 70/80 ABI.

FAT12/16/32 advertise two additional operations in their filesystem table:

| Operation | Behavior | Parameters |
| --- | --- | --- |
| 14 | Create a file only when its name does not exist | Same request layout and buffer checks as operation 2 |
| 15 | Create a directory only when its name does not exist | Same request layout as operation 9 |

Upstream operations 11/12 implement symbolic links; 13 returns volume info.
Their numbers and behavior are retained. The early SDK qualification bundle
used private operations 11/12 with its own older kernel. That archived bundle
must remain paired with that kernel. Newly built programs use 14/15 and need
the updated kernel; they do not probe or call the obsolete private operations.

Both use the existing FAT partition mutex around name lookup and creation.
Either kind of existing entry returns filesystem status **17**, mapped by
the SDK to `fs.ErrExist` / `syscall.EEXIST`. The existing entry is untouched.
Normal filesystem errors retain their original values. Paths and encodings
follow the existing syscall 70/80 rules.

Older kernels and filesystems that do not advertise these operations return
status 2 (unsupported). The SDK reports an explicit unsupported operation
without falling back to a check followed by a destructive create/rewrite.

The builder also moves the partition's `createOption` update after acquiring
the FAT mutex, so a waiting creator cannot alter the current writer's mode.

## FAT filename case

`--fat-name-case` adapts the original `fat_get_name` routine to honor the
short-entry NT lowercase bits at byte 12 (base name `0x08`, extension `0x10`).
It preserves the original codepage conversion and case-insensitive lookup.
These fields agree with the upstream FatFs filename reader:
https://raw.githubusercontent.com/zephyrproject-rtos/fatfs/master/ff.c.

Creating a name containing lowercase ASCII uses the kernel's existing long
filename writer, preserving mixed spelling rather than returning an uppercase
short name later. Existing long-name and Unicode allocation code is retained.
No syscall number, request layout or ABI entrypoint changes for this fix.

`original-fat-case-native` verifies lower, mixed, uppercase and Russian names,
case-insensitive reads, rename, glob and removal. The original CLI's `grep`,
`glob` and `ls` additionally pass with `original-cli-native-shell-search-case`.
