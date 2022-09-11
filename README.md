# dupe-nukem

Tool for identifying duplicate data across multiple directories or archive files,
potentially residing on different disks that may not be online at the same time.

Contrary to what the name might imply,
dupe-nukem will not be seen going nuts, blowing up every redundant byte in sight
in a spin-looping digital rampage (sorry I guess).
It's actually entirely well-mannered and doesn't even know how to do destructive changes at all;
it only reports changes that could be made (amongst other things)
and leaves it up to the nearest human to decide what to do with the information.
So to actually nuke the dupes,
you need to run the list that dupe-nukem reports through e.g. `rm`.

Examples of the kinds of questions that dupe-nukem can answer are:

- Which files in some directory are already present elsewhere (and where)?
- Which preserve-worthy parts of some harddisk are *not* yet properly backed up?
- Which other directories contain *any* files from a given directory?
  Do they, in combination, contain all the files?

It may also be used to investigate how files have moved around (including renaming)
relative to a previous backup.

Attempts are made to present the results in the aggregated form that makes the most sense:
If all files in some directory are present in some other,
it just reports that the directories match - not that each individual file does.
If they don't match exactly, it will report the differences if they're relevant.

## Status

This project is at a very early stage:
Only the `scan` command (of regular directories) has been implemented.

The commands listed below make up an approximate subset of the envisioned interface
to give a rough idea of what should be done.

### Windows

A few tests related to symlinking are disabled on Windows because elevated privileges are required to create symlinks.
Other tests fail unless the repository is mounted on an NTFS formatted drive (which for instance a VirtualBox shared folder might not be).

## Install

Instructions will be here as soon as there's something to install!

## Commands

### 1. Scan

```
dupe-nukem scan --dir <dir> [--skip <expr>] [--cache <file>]
```

Builds structure of `<dir>` and dumps it, along with all hashes, in JSON.

A skip expression `<expr>` may be used to make the command skip
certain files and directories like '.git', '.stack-work', 'vendor', 'node_modules', '.DS_Store', etc.
The skip expression may either specify these names literally as a comma-separated list
or point to a file `<f>` that contains a name for each non-empty line using the expression `@<f>`.

A reference file `<file>` from a previous call to `scan` may be provided to use as
a cache for file hashes.
The hashes of scanned files will be looked up in this file as long as the file sizes match.
If the filename ends with `.gz`, it will automatically be gzip decompressed.

The root dir in the JSON output is the basename of `<dir>`.
The commands below will have a way of mapping this root dir back to the concrete location
in the filesystem (or URI in general).
The reason for this behavior is that the scanning may be performed in one context (e.g. locally)
and validation etc. in another (e.g. remotely or mounted on a different path).
It could make sense to keep the (absolute) path of the scanned directory in the file
while still being able to remap by need in later commands, so this decision is not final.

The command is intended to also be able to scan archive files,
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

Check that matches (made with hash comparison by `match`) are indeed identical.

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

This command is not yet implemented.
