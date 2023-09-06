exturl
======

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Reference](https://pkg.go.dev/badge/github.com/tliron/exturl.svg)](https://pkg.go.dev/github.com/tliron/exturl)
[![Go Report Card](https://goreportcard.com/badge/github.com/tliron/exturl)](https://goreportcard.com/report/github.com/tliron/exturl)

URLs for Go on steroids.

Simply put, it allows you to get a Go `Reader` from a wide variety of URL types, including
specific entries in archives using a URL structure inspired by Java's
[JarURLConnection](https://docs.oracle.com/javase/8/docs/api/java/net/JarURLConnection.html).

Features
--------

Especially powerful is the ability to refer to entries in remote archives, e.g. a zip file
over http. Where possible exturl will stream the data (e.g. remote tarballs), but if filesystem
access is required (for remote zip, git repository clones, and Docker images) it will download
them to a temporary local location. The use of a shared context allows for optimization, e.g. a
remote zip file will not be downloaded again if it was already downloaded in the context.
Examples:

    tar:http://mysite.org/cloud.tar.gz!main.yaml
    git:https://github.com/tliron/puccini.git!examples/openstack/hello-world.yaml

Another powerful feature is support for relative URLs using common filesystem paths, including
usage of `..` and `.`. All URL types support this: file URLs, local and remote zip URLs, etc.
Use `url.Relative()`.

You can also ensure that a URL is valid (e.g. remote location is available, tarball entry
exists, etc.) before attempting to read from it (which may trigger a download) or passing it
to other parts of your program. To do so, use `NewValidURL()` instead of `NewURL()`.
`NewValidURL()` also supports relative URLs tested against a list of potential bases.
Compare with how the `PATH` environment variable is used by the OS to find commands.

The combination of relative URL and archive support allows for embedded relative URLs,
for example as imports or references in configuration files or parsed code. These trivially
will "always work" whatever the base URL is, whether it's a directory in the local filesystem,
a tarball, and even a remote tarball. Exturl will do the heavy lifting to make sure that
the content is readable.

Also supported are URLs for in-memory data using a special `internal:` scheme. This allows you
to have a unified API for accessing data, whether it's available externally or created
internally by your program.

Finally, there are tools for mocking and testing, e.g. a `MockURL` type that can mimic any
scheme with arbitrary data, and support for custom URL transformation functions, including
straightforward mapping of URLs to other URLs. For example, you can map a `http:` URL
to a `file:` or `internal:` URL.

Example
-------

```go
import (
    "context"
    "github.com/tliron/exturl"
)

func ReadAll(url string) ([]byte, error) {
    urlContext := exturl.NewContext()
    defer urlContext.Release()

    url_ := urlContext.NewURL(url)
    if reader, err := url_.Open(context.TODO()); err == nil {
        defer reader.Close()
        return io.ReadAll(reader)
    } else {
        return nil, err
    }
}
```

Supported URL Types
-------------------

### `http:` and `https:`

Uses standard Go access libraries (`net/http`). `Open()` is an HTTP GET verb.

### `file:`

An absolute path to the local filesystem.

The URL must begin with two slashes. If a hostname is present before the path it will
be ignored by exturl, so this:

    file://localhost/the/path

is equivalent to this:

    file:///the/path

Because the path must be absolute, it always begins with a slash. The consequence is that
most `file:` URLs beging with 3 slashes.

When compiled for Windows the URL path will be converted to a Windows path. The convention
is that backslashes become slashes and a first slash is added to make it absolute. So this
URL:

    file:///C:/Windows/win.ini

is equivalent to this path:

    C:\Windows\win.ini

Note that for security reasons relative file URLs *are not* tested against the current
working directory (`pwd`) by default. This is unlike OS services, such as Go's `os.Open()`.
If you do want to support the working directory then call `NewWorkingDirFileURL()` and add
it explicitly to the bases of `NewValidURL()`.

It is often desirable to accept input that is *either* a URL *or* a file path. For this
use case `NewAnyOrFileURL()` and `NewValidAnyOrFileURL()` are provided. If the argument
is not a parsable URL it will be treated as a file path and a `*FileURL` will be returned.

The functions' design may trip over a rare edge case for Windows. If there happens to be
a drive that has the same name as a supported URL scheme, e.g. "http", then callers would
have to provide a full file URL, otherwise it will be parsed as a URL of that scheme. E.g.
instead of:

    http:\Dir\file

you must use:

    file:///http:/Dir/file

### `tar:`

Entries in tarballs. `.tar` and `.tar.gz` (or `.tgz`) are supported. The archive URL
can be any full exturl URL *or* a local filesystem path. Examples:

    tar:http://mysite.org/cloud.tar.gz!path/to/main.yaml
    tar:file:///local/path/cloud.tar.gz!path/to/main.yaml
    tar:/local/path/cloud.tar.gz!path/to/main.yaml

Note that tarballs are serial containers optimized for streaming, meaning that, when
read, unwanted entries will be skipped until our entry is found, and then subsequent
entries will be ignored. This means that when accessing tarballs over the network the
tarball does *not* have to be downloaded in its entirety, unlike with zip (see below).

Gzip decompression uses [klauspost's pgzip library](https://github.com/klauspost/pgzip).

### `zip:`

Entries in zip files. The archive URL can be any full exturl URL *or* a local
filesystem path. Example:

    zip:http://mysite.org/cloud.tar.gz!path/to/main.yaml

Note that zip files require random access and thus must be on the local file system.
Consequently for remote zips the *entire* archive must be downloaded in order to access
one entry. For optimization, exturl will make sure to download it only once per context.

Uses [klauspost's compress library](https://github.com/klauspost/compress).

### `git:`

Files in git repositories. The repository URL is *not* an exturl URL, but rather must
be URLs supported by [go-git](https://github.com/go-git/go-git). Example:

    git:https://github.com/tliron/puccini.git!examples/openstack/hello-world.yaml

You can specify a reference (tag, branch tip, or commit hash) in the URL fragment, e.g.:

    git:https://github.com/tliron/puccini.git#main!examples/openstack/hello-world.yaml

Because we are only interested in reading files, not making commits, exturl will optimize
by performing shallow clones of *only* the requested reference.

### `docker:`

Images on OCI/Docker registries. The URL structure is
`docker://HOSTNAME/[NAMESPACE/]REPOSITORY[:TAG]`. The tag will default to "latest".
Example:

    docker://docker.io/tliron/prudence:latest

The `url.Open()` API will decode the first layer (a `.tar.gz`) it finds in the image.
The intended use case is using OCI registries to store arbitrary data. In the future
we may support more elaborate use cases.

Uses [go-containerregistry](https://github.com/google/go-containerregistry).

### `internal:`

Internal URLs can be stored globally so that all contexts are able to access them.

Supported APIs for global internal URLs are `RegisterInternalURL()` (which will fail if
the URL has already been registered), `DeregisterInternalURL()`, `UpdateInternalURL()`
(which will always succeed), `ReadToInternalURL()`, `ReadToInternalURLFromStdin()`.

It is also possible to create ad-hoc internal URLs using `NewInternalURL()` and then
`url.SetContent()`. These are *not* stored globally.

Content can be `[]byte` or an implementation of the `InternalURLProvider` interface.
Other types will be converted to strings and then to `[]byte`.

### Mock URLs

These are intended to be used for testing. They must be created explicitly via
`NewMockURL()` and can use any scheme. They are not created by `NewURL()`.
