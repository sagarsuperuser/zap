# :zap: zap

<div align="center">

Blazing fast, structured, leveled logging in Go.

![Zap logo](assets/logo.png)

[![GoDoc][doc-img]][doc] [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov]

</div>

## Installation

`go get -u go.uber.org/zap`

Note that zap only supports the two most recent minor versions of Go.

## Quick Start

In contexts where performance is nice, but not critical, use the
`SugaredLogger`. It's 4-10x faster than other structured logging
packages and includes both structured and `printf`-style APIs.

```go
logger, _ := zap.NewProduction()
defer logger.Sync() // flushes buffer, if any
sugar := logger.Sugar()
sugar.Infow("failed to fetch URL",
  // Structured context as loosely typed key-value pairs.
  "url", url,
  "attempt", 3,
  "backoff", time.Second,
)
sugar.Infof("Failed to fetch URL: %s", url)
```

When performance and type safety are critical, use the `Logger`. It's even
faster than the `SugaredLogger` and allocates far less, but it only supports
structured logging.

```go
logger, _ := zap.NewProduction()
defer logger.Sync()
logger.Info("failed to fetch URL",
  // Structured context as strongly typed Field values.
  zap.String("url", url),
  zap.Int("attempt", 3),
  zap.Duration("backoff", time.Second),
)
```

### Example: Logging structured HTTP request/response using TEXT format

```go
// ----- request -----
type HTTPRequest struct {
    Method string
    URL    string
}

func (r HTTPRequest) MarshalLogObject(enc zapcore.ObjectEncoder) error {
    enc.AddString("method", r.Method)
    enc.AddString("url", r.URL)
    return nil
}

// ----- response -----
type HTTPResponse struct {
    Status int
    Bytes  int
}

func (r HTTPResponse) MarshalLogObject(enc zapcore.ObjectEncoder) error {
    enc.AddInt("status", r.Status)
    enc.AddInt("bytes", r.Bytes)
    return nil
}

// ----- http group combining request + response -----
type HTTP struct {
    Req  HTTPRequest
    Resp HTTPResponse
}

func (h HTTP) MarshalLogObject(enc zapcore.ObjectEncoder) error {
    if err := enc.AddObject("request", h.Req); err != nil {
        return err
    }
    if err := enc.AddObject("response", h.Resp); err != nil {
        return err
    }
    return nil
}

func main() {
    logger, _ := zap.NewProductionText()
    defer logger.Sync()

    httpFields := HTTP{
        Req:  HTTPRequest{Method: "GET", URL: "/api/v1/users"},
        Resp: HTTPResponse{Status: 200, Bytes: 1234},
    }

    logger.Info("served", zap.Object("http", httpFields))
}
```

Log Output


```console
"level"="info" "ts"=1758088355.259905 "caller"="zap_test/main.go:57" "msg"="served" "http.request.method"="GET" "http.request.url"="/api/v1/users" "http.response.status"=200 "http.response.bytes"=1234
```

## Performance

For applications that log in the hot path, reflection-based serialization and
string formatting are prohibitively expensive — they're CPU-intensive
and make many small allocations. Put differently, using `encoding/json` and
`fmt.Fprintf` to log tons of `interface{}`s makes your application slow.

Zap takes a different approach. It includes a reflection-free, zero-allocation
JSON encoder, and the base `Logger` strives to avoid serialization overhead
and allocations wherever possible. By building the high-level `SugaredLogger`
on that foundation, zap lets users *choose* when they need to count every
allocation and when they'd prefer a more familiar, loosely typed API.

As measured by its own [benchmarking suite][], not only is zap more performant
than comparable structured logging packages — it's also faster than the
standard library. Like all benchmarks, take these with a grain of salt.<sup  
id="anchor-versions">[1](#footnote-versions)</sup>

### Log a message and 10 fields

| Package              | Time (ns/op) | Time % vs Zap | Objects Allocated |
| -------------------- | ------------ | ------------- | ----------------- |
| ⚡ **zap** (json)     | **656**      | +0%           | 5 allocs/op       |
| ⚡ zap (text)         | 852          | +30%          | 5 allocs/op       |
| ⚡ zap (sugared json) | 854          | +30%          | 10 allocs/op      |
| ⚡ zap (sugared text) | 1071         | +63%          | 10 allocs/op      |
| zerolog              | **318**      | -52%          | 1 alloc/op        |
| go-kit               | 1989         | +203%         | 56 allocs/op      |
| slog                 | 2122         | +224%         | 41 allocs/op      |
| slog (LogAttrs)      | 2161         | +229%         | 40 allocs/op      |
| apex/log             | 10762        | +1539%        | 64 allocs/op      |
| log15                | 12049        | +1736%        | 73 allocs/op      |
| logrus               | 12721        | +1840%        | 84 allocs/op      |
| klog/v2 textlogger   | 2776         | +323%         | 45 allocs/op      |

### Log a message with a logger that already has 10 fields of context

| Package              | Time (ns/op) | Time % vs Zap | Objects Allocated |
| -------------------- | ------------ | ------------- | ----------------- |
| ⚡ **zap**  (json)    | **69.7**     | +0%           | 0 allocs/op       |
| ⚡ zap (text)         | 110.4        | +58%          | 0 allocs/op       |
| ⚡ zap (sugared json) | 91.2         | +31%          | 1 alloc/op        |
| ⚡ zap (sugared text) | 121.9        | +75%          | 1 alloc/op        |
| zerolog              | **43.5**     | -38%          | 0 allocs/op       |
| go-kit               | 2639         | +3683%        | 55 allocs/op      |
| slog                 | 201.9        | +190%         | 0 allocs/op       |
| slog (LogAttrs)      | 205.1        | +194%         | 0 allocs/op       |
| apex/log             | 9556         | +13698%       | 52 allocs/op      |
| log15                | 8424         | +11982%       | 67 allocs/op      |
| logrus               | 11104        | +15828%       | 69 allocs/op      |

### Log a static string, without any context or printf-style templating

| Package              | Time (ns/op) | Time % vs Zap | Objects Allocated |
| -------------------- | ------------ | ------------- | ----------------- |
| ⚡ **zap** (json)     | **68.2**     | +0%           | 0 allocs/op       |
| ⚡ zap (text)         | 99.6         | +46%          | 0 allocs/op       |
| ⚡ zap (sugared json) | 88.4         | +30%          | 1 alloc/op        |
| ⚡ zap (sugared text) | 121.7        | +79%          | 1 alloc/op        |
| zerolog              | **35.1**     | -49%          | 0 allocs/op       |
| go-kit               | 195.0        | +186%         | 8 allocs/op       |
| slog                 | 198.1        | +191%         | 0 allocs/op       |
| slog (LogAttrs)      | 204.2        | +199%         | 0 allocs/op       |
| apex/log             | 672.6        | +887%         | 4 allocs/op       |
| log15                | 1676         | +2360%        | 17 allocs/op      |
| logrus               | 1155         | +1595%        | 20 allocs/op      |
| standard library     | 136.1        | +100%         | 1 alloc/op        |

## Development Status: Stable

All APIs are finalized, and no breaking changes will be made in the 1.x series
of releases. Users of semver-aware dependency management systems should pin
zap to `^1`.

## Contributing

We encourage and support an active, healthy community of contributors —
including you! Details are in the [contribution guide](CONTRIBUTING.md) and
the [code of conduct](CODE_OF_CONDUCT.md). The zap maintainers keep an eye on
issues and pull requests, but you can also report any negative conduct to
[oss-conduct@uber.com](mailto:oss-conduct@uber.com). That email list is a private, safe space; even the zap
maintainers don't have access, so don't hesitate to hold us to a high
standard.

<hr>

Released under the [MIT License](LICENSE).

<sup id="footnote-versions">1</sup> In particular, keep in mind that we may be
benchmarking against slightly older versions of other packages. Versions are
pinned in the [benchmarks/go.mod][] file. [↩](#anchor-versions)

[doc-img]: https://pkg.go.dev/badge/go.uber.org/zap
[doc]: https://pkg.go.dev/go.uber.org/zap
[ci-img]: https://github.com/uber-go/zap/actions/workflows/go.yml/badge.svg
[ci]: https://github.com/uber-go/zap/actions/workflows/go.yml
[cov-img]: https://codecov.io/gh/uber-go/zap/branch/master/graph/badge.svg
[cov]: https://codecov.io/gh/uber-go/zap
[benchmarking suite]: https://github.com/uber-go/zap/tree/master/benchmarks
[benchmarks/go.mod]: https://github.com/uber-go/zap/blob/master/benchmarks/go.mod
