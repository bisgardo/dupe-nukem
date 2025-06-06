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
relative to a previous backup (or scan).

Attempts are made to present the results in the aggregated form that makes the most sense:
If all files in some directory are present in some other,
it just reports that the directories match - not that each individual file does.
If they don't match exactly, it will report the differences if they're relevant.

The tool is designed with the following decentralized workflow in mind:

1. Run command `scan` on the directories (or archives) of interest.
   This is the only command that's expected to run locally (but with remote file systems like SSHFS might not even have to).
2. Match the result files of `scan` runs with command `match` to compare the scanned directories.
   The compared directories may be different (possibly on different hosts) or one directory scanned at different times.
3. Analyze, visualize, and act on the conclusions.

## Status

This project is at a very early stage:
Only the `scan` command (of regular directories) has been implemented.

The commands listed below make up an approximate subset of the envisioned interface
to give a rough idea of what remains to be done.

### Windows

A few tests related to symlinking are disabled on Windows because elevated privileges are required to create symlinks.
Other tests fail unless the repository is mounted on an NTFS formatted drive (which for instance a VirtualBox shared folder might not be).

## Install

Clone the source:

```shell
git clone https://github.com/bisgardo/dupe-nukem.git
cd dupe-nukem
```

Build the source directly:

```shell
make build
```

A simple dockerfile is also provided to allow building in a tightly controlled environment. For example:

```shell
docker build -t dupe-nukem --build-arg=build_image=golang:1.15-buster --build-arg=base_image=debian:buster-slim --pull .
docker run --rm --volume=<volume-mounts> dupe-nukem <args>
```

Due to the file-based nature of this tool, running it in Docker is quite a hassle with volume mounts.
For this reason, the intended purpose of this is building with different versions,
either for testing or extracting the binary for use outside of Docker.

## Commands

### 1. Scan

```shell
dupe-nukem scan --dir <dir> [--skip <expr>] [--cache <file>]
```

Builds structure of directory `<dir>` and dumps it, along with all sizes, modification times, and hashes (in JSON).

A skip expression `<expr>` may be used to make the command skip
certain files and directories like `.git`, `.stack-work`, `vendor`, `node_modules`, `.DS_Store`, etc.
The skip expression may either specify these names literally as a comma-separated list
or point to a file `<f>` that contains a name for each non-empty line using the expression `@<f>`.

The result file `<file>` of a previous `scan` may be provided for use as a "cache"
for hashes of files that didn't change since that previous run:
As long as the size and modification time of any given file being scanned matches what's in the cache file,
then the hash is simply read from that file.
As a sanity check, the root name (which, as mentioned below, is an absolute path) of the cache
must match that of the root (with any symlinks evaluated).
If the filename ends with `.gz`, then the file is automatically decompressed.

The root directory "name" in the JSON output is the absolute path of `<dir>`.
The other commands are likely going to provide ways of understanding what a path from one context (scan)
means in others (matching, validating, etc.) as different actions may happen on different hosts.

The command is intended to also be able to scan archive files,
but this feature is not yet implemented.

### 2. Match

*This command is not yet implemented.*

```shell
dupe-nukem match --source <dir-file> --targets <dir-files>
```

Search for subdirectories of source directory in target directories
(each of these directories represented by files output by invocations of `scan`).
Prints all matching files with their directory structure (relative to the subdirectory) preserved.

This command is not yet implemented.

### 3. Validate (optional)

*This command is not yet implemented.*

```shell
dupe-nukem validate --match <match-file>
```

Check that matches (made with hash comparison by `match`) are indeed identical.

Ideally, a "fixed" match file should be output.
But as this is expected to never happen, the command will just puke out any validation failure.

This command is not yet implemented.

### 4. Diff

*This command is not yet implemented.*

```shell
dupe-nukem diff --dir <dir-file> --match <match-file>
```

Lists all files in `<dir-file>` that were not matched in any of the target directories in the `match` call.

There should be options on what to do if files are matched with different names or directory structure.
I guess the latter is a matter of whether we match by files or directories?

This command is not yet implemented.
