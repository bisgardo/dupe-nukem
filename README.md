# dupe-nukem

Tool for identifying duplicate data across multiple directories or archive files,
potentially residing on different disks that might not be online at the same time.

Contrary to what the name might imply, the main purpose of the tool is not to eliminate duplicates automatically,
but to identify, for example, which preserve-worthy parts of some harddisk are not yet properly backed up.
It may also be used to figure out how files have moved around relative to a previous backup.

The tool itself never does destructive changes but only outputs changes that could be made for human review.
So while it can list files that can be safely deleted,
this list needs to be passed through e.g. `rm` to actually delete the files.

## Status

This project is at a very early stage:
As of this writing, only the `scan` command (of regular directories) has been implemented.

## Commands

### 1. Scan

```
dupe-nukem scan --dir <dir> [--skip <expr>] [--cache <file>]
```

Builds structure of `<dir>` and dumps it, along with all hashes, in JSON.

A skip expression `<expr>` may be used to make the scanning skip
certain files and directories like '.git', '.stack-work', 'vendor', 'node_modules', '.DS_Store', etc.
The skip expression may either specify these names literally as a comma-separated list
or point to a file `<f>` that contains a name for each non-empty line using the expression `@<f>`.

A reference file `<file>` from a previous call to `scan` may be provided to use as
a cache for file hashes.
The hashes of scanned files will be looked up in this file as long as the file sizes match.

This command is intended to also be able to scan archive files,
but this feature is not yet implemented.

### 2. Match

*This command is not yet implemented.*

```
dupe-nukem match --source <dir-file> --targets <dir-files>
```

Search for subdirectories of source directory in target directories
(each of these directories represented by files output by invocations of `scan`).
Prints all matching files with their directory structure (relative to the subdirectory) preserved.

This command is not yet implemented.

### 3. Validate (optional)

*This command is not yet implemented.*

```
dupe-nukem validate --match <match-file>
```

Check that matches (made by hash comparison by `match`) are indeed identical.

Ideally, a "fixed" match file should be output.
But as this is expected to never happen, the command will just puke out any validation failure.

This command is not yet implemented.

### 4. Diff

*This command is not yet implemented.*

```
dupe-nukem diff --dir <dir-file> --match <match-file>
```

Lists all files in `<dir-file>` that were not matched in any of the target directories in the `match` call.

There should be options on what to do if files are matched with different names or directory structure.
I guess the latter is a matter of whether we match by files or directories?
