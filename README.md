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

This project is at the very earliest stage.
The commands below make up some brain dumped version of the intended interface.
As of this writing, nothing is implemented yet.

## Commands

### 1. Scan

```
dupe-nukem scan --dir <dir> [--skip <names>] [--cache <file>]
```

Builds structure of `<dir>` and dumps it, along with all hashes, in JSON.
Optionally add file/directory names to skip (like '.git', '.stack-work', 'vendor', etc.).
Optionally add a reference file from a previous call to `scan` to use as hash cache.
The hashes of the scanned files will be looked up in this file as long as the file sizes match

### 2. Match

```
dupe-nukem match --source <dir-file> --targets <dir-files>
```

Search for subdirectories of source directory in target directories
(each of these directories represented by files output by invocations of `scan`).
Prints all matching files with their directory structure (relative to the subdirectory) preserved.

### 3. Validate (optional)

```
dupe-nukem validate --match <match-file>
```

Check that matches (made by hash comparison by `match`) are indeed identical.

Ideally, a "fixed" match file should be output.
But as this is expected to never happen, the command will just puke out any validation failure.

### 4. Diff

```
dupe-nukem diff --dir <dir-file> --match <match-file>
```

Lists all files in `<dir-file>` that were not matched in any of the target directories in the `match` call.

There should be options on what to do if files are matched with different names or directory structure.
I guess the latter is a matter of whether we match by files or directories?
