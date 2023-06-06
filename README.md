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

    tar:http://mysite.org/cloud.tar.gz\!main.yaml
    git:https://github.com/tliron/puccini.git!examples/openstack/hello-world.yaml

Another powerful feature is support for relative URLs using common filesystem paths, including
usage of `..` and `.`. All URL types support this: file URLs, local and remote zip URLs, etc.
Use `url.Relative()`.

You can also ensure that a URL is valid (remote location is available) before attempting to
read from it (which may trigger a download) or passing it to other parts of your program. To
do so, use `NewValidURL()` instead of `NewURL()`.

Also supported are URLs for in-memory data using a special `internal:` scheme. This allows you
to have a unified API for accessing data, whether it's available externally or created
internally by your program.

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

    if url_, err = urlContext.NewURL(url); err == nil {
        if reader, err := url_.Open(context.TODO()); err == nil {
            defer reader.Close()
            return io.ReadAll(reader)
        } else {
            return nil, err
        }
    } else {
        return nil, err
    }
}
```

Supported Schemes
-----------------

### `file:`

A path to the local filesystem. This is the default URL type if no schema is provided.

The URL *must* begin with two slashes. If a hostname is present before the path it will
be ignored, so this:

    file://localhost/the/path

is equivalent to this:

    file:///the/path

Relative paths are supported, but only when no scheme is provided. In other words, the
`file:` scheme *requires* absolute paths. The consequence is that `file:` URLs usually
begin with *three* slashes because absolute paths also begin with a slash.

When compiled for Windows the URL path will be converted to a Windows path. So this:

    file:///C:/Windows/win.ini

will be treated as this path:

    C:\Windows\win.ini

A URL with no schema will be treated as a `file:` URL. Thus this:

    /the/path

is equivalent to this:

    file:///the/path

However, note that this will *not* work for absolute Windows paths because the initial
drive prefix (e.g. `C:`) would be interpreted as a schema. Thus, the most reliable ways
to create `file:` URLs are to either use the full URL format (`file:///...`) or to
call `NewFileURL()` with a path.

### `http:` and `https:`

Uses standard Go access libraries (`net/http`).

### `tar:`

Entries in tarballs. `.tar` and `.tar.gz` (or `.tgz`) are supported. Examples:

    tar:/local/path/cloud.tar.gz\!path/to/main.yaml
    tar:http://mysite.org/cloud.tar.gz\!path/to/main.yaml

Note that tarballs are serial containers optimized for streaming. That means that when
accessed the entries will be skipped until our entry is found and then subsequent entries
will be ignored. This means that when accessing tarballs over the network the tarball
does *not* have to be downloaded, unlike with zip (see below).

Gzip decompression uses [klauspost's pgzip library](https://github.com/klauspost/pgzip).

### `zip:`

Entries in zip files. Example:

    zip:http://mysite.org/cloud.tar.gz\!path/to/main.yaml

Note that zip files require random access and thus *must* be on the local file system.
Consequently for remote zips the *entire* archive must be downloaded in order to access
one entry. For optimization, exturl will make sure to download it only once per context.

Uses [klauspost's compress library](github.com/klauspost/compress/zip).

### `git:`

Files in git repositories. Example:

    git:https://github.com/tliron/puccini.git!examples/openstack/hello-world.yaml

You can specify a reference (tag, branch tip, or commit hash) in the URL fragment, e.g.:

    git:https://github.com/tliron/puccini.git#main!examples/openstack/hello-world.yaml

Because we are only interested in reading files, exturl will optimize by performing
shallow clones of only the requested reference.

Uses [go-git](https://github.com/go-git/go-git).

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

Internal URLs can ne stored globally so that all contexts will be able to access them.

Supported APIs for global internal URLs are `RegisterInternalURL()` (which will fail if
the URL has already been registered), `DeregisterInternalURL()`, `UpdateInternalURL()`
(which will always succeed), `ReadToInternalURL()`, `ReadToInternalURLFromStdin()`.

It is also possible to create ad-hoc internal URLs using `NewInternalURL()` and then
`SetContent()`. These are *not* stored globally.

Content can be `[]byte` or an implementation of the `InternalURLProvider` interface.
Other types will be converted to string and then to `[]byte`.
