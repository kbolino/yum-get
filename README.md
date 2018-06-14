# yum-get

A tool for obtaining packages as RPM files from Yum repositories, usable from
non-Red Hat and even non-Linux systems.

## Current features

- List the packages contained in a repository
- Download a package by (name, ver, rel) tuple

## Possible future goals

The following are roughly ordered by usefulness/ease of implementation.

- Handle multiple arch values
- Implement `rpmvercmp` to remove the need to specify ver and rel
- Implement requires/provides logic to download dependencies
- Allow multiple repositories to be used
- Read the standard yum repo file format (/etc/yum.repos.d) to obtain
  repository URLs
- Verify downloads using:
    - checksums
    - GPG signatures
- Allow searching within the filelists metadata, to locate packages by
  contained file(s)

## Definite non-goals

- Installing packages
- Replacing `yum` or `dnf`
- Handling RPM internals not exposed in the yum repo metadata
- Email
