exturl
======

URLs for Go on steroids.

Usage example:

```go
import (
    "github.com/tliron/exturl"
)

func ReadAll(url string, format string) ([]byte, error) {
    context := exturl.NewContext()
    defer context.Release()

    if url_, err = exturl.NewURL(url, context); err == nil {
        if reader, err := url_.Open(); err == nil {
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

file:
-----

http: and https:
----------------

zip:
----

git:
----

docker:
-------

internal:
---------
