# :zap: zap [![GoDoc][doc-img]][doc] [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov]

Blazing fast, structured, leveled logging in Go.

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

See the [documentation][doc] and [FAQ](FAQ.md) for more details.

## Performance

For applications that log in the hot path, reflection-based serialization and
string formatting are prohibitively expensive &mdash; they're CPU-intensive
and make many small allocations. Put differently, using `encoding/json` and
`fmt.Fprintf` to log tons of `interface{}`s makes your application slow.

Zap takes a different approach. It includes a reflection-free, zero-allocation
JSON encoder, and the base `Logger` strives to avoid serialization overhead
and allocations wherever possible. By building the high-level `SugaredLogger`
on that foundation, zap lets users *choose* when they need to count every
allocation and when they'd prefer a more familiar, loosely typed API.

As measured by its own [benchmarking suite][], not only is zap more performant
than comparable structured logging packages &mdash; it's also faster than the
standard library. Like all benchmarks, take these with a grain of salt.<sup
id="anchor-versions">[1](#footnote-versions)</sup>

Log a message and 10 fields:

| Package | Time | Objects Allocated |
| :--- | :---: | :---: |
| :zap: zap | 1241 ns/op | 5 allocs/op |
| :zap: zap (sugared) | 2154 ns/op | 11 allocs/op |
| zerolog | 4905 ns/op | 76 allocs/op |
| lion | 8694 ns/op | 105 allocs/op |
| go-kit | 5702 ns/op | 105 allocs/op |
| logrus | 35654 ns/op | 125 allocs/op |
| log15 | 38019 ns/op | 122 allocs/op |
| apex/log | 32085 ns/op | 115 allocs/op |

Log a message with a logger that already has 10 fields of context:

| Package | Time | Objects Allocated |
| :--- | :---: | :---: |
| :zap: zap | 160 ns/op | 0 allocs/op |
| :zap: zap (sugared) | 250 ns/op | 2 allocs/op |
| zerolog | 103 ns/op | 0 allocs/op |
| lion | 2399 ns/op | 34 allocs/op |
| go-kit | 6021 ns/op | 103 allocs/op |
| logrus | 28798 ns/op | 113 allocs/op |
| log15 | 21603 ns/op | 73 allocs/op |
| apex/log | 28604 ns/op | 104 allocs/op |

Log a static string, without any context or `printf`-style templating:

| Package | Time | Objects Allocated |
| :--- | :---: | :---: |
| :zap: zap | 138 ns/op | 0 allocs/op |
| :zap: zap (sugared) | 232 ns/op | 2 allocs/op |
| zerolog | 98 ns/op | 0 allocs/op |
| standard library | 489 ns/op | 2 allocs/op |
| go-kit | 353 ns/op | 11 allocs/op |
| lion | 563 ns/op | 8 allocs/op |
| logrus | 3467 ns/op | 24 allocs/op |
| apex/log | 2240 ns/op | 10 allocs/op |
| log15 | 4012 ns/op | 23 allocs/op |

## Development Status: Stable

All APIs are finalized, and no breaking changes will be made in the 1.x series
of releases. Users of semver-aware dependency management systems should pin
zap to `^1`.

## Contributing

We encourage and support an active, healthy community of contributors &mdash;
including you! Details are in the [contribution guide](CONTRIBUTING.md) and
the [code of conduct](CODE_OF_CONDUCT.md). The zap maintainers keep an eye on
issues and pull requests, but you can also report any negative conduct to
oss-conduct@uber.com. That email list is a private, safe space; even the zap
maintainers don't have access, so don't hesitate to hold us to a high
standard.

<hr>

Released under the [MIT License](LICENSE.txt).

<sup id="footnote-versions">1</sup> In particular, keep in mind that we may be
benchmarking against slightly older versions of other packages. Versions are
pinned in zap's [glide.lock][] file. [↩](#anchor-versions)

[doc-img]: https://godoc.org/go.uber.org/zap?status.svg
[doc]: https://godoc.org/go.uber.org/zap
[ci-img]: https://travis-ci.com/uber-go/zap.svg?branch=master
[ci]: https://travis-ci.com/uber-go/zap
[cov-img]: https://codecov.io/gh/uber-go/zap/branch/master/graph/badge.svg
[cov]: https://codecov.io/gh/uber-go/zap
[benchmarking suite]: https://github.com/uber-go/zap/tree/master/benchmarks
[glide.lock]: https://github.com/uber-go/zap/blob/master/glide.lock
