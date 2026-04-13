# ini (Go)

Version: 0.6.0

A Go port of [@jsonic/ini](https://github.com/jsonicjs/ini), a
[Jsonic](https://github.com/jsonicjs/jsonic) syntax plugin that
parses INI format files into Go maps, with support for sections,
nested keys, arrays, multiline values, and inline comments.

## Install

```bash
go get github.com/jsonicjs/ini/go@latest
```

## Quick Example

```go
package main

import (
    "fmt"
    ini "github.com/jsonicjs/ini/go"
)

func main() {
    result, err := ini.Parse(`
[database]
host = localhost
port = 5432

[server.production]
debug = false
`)
    if err != nil {
        panic(err)
    }
    fmt.Println(result)
    // map[database:map[host:localhost port:5432] server:map[production:map[debug:false]]]
}
```

## Documentation

- [Go API reference](../doc/ini-go.md#reference)
- [Tutorials](../doc/ini-go.md)

## License

MIT. Copyright (c) Richard Rodger and other contributors.
