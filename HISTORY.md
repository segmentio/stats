# History

### Unreleased

- Adds support for datadog client configuration using environment variables.
  `STATSD_HOST`, `STATSD_UDP_PORT` and `STATSD_SOCKET` can now be used to
  configure the datadog client.

### v5.4.0 (February 21, 2025)

- Fix a regression in configured buffer size for the datadog client. Versions
5.0.0 to 5.3.1 would ignore any configured buffer size and use the default value
of 1024. This could lead to smaller than expected writes and contention for the
file handle.

### v5.3.1 (January 2, 2025)

- Fix version parsing logic.

### v5.3.0 (December 19, 2024)

- Add `debugstats` package that can easily print metrics to stdout.

- Fix file handle leak in procstats in the DelayMetrics collector.

### v5.2.0

- `go_version.value` and `stats_version.value` will be emitted the first
time any metrics are sent from an Engine. Disable this behavior by setting
`GoVersionReportingEnabled = false` in code, or setting environment variable
`STATS_DISABLE_GO_VERSION_REPORTING=true`.

### v5.1.0

Add support for publishing stats to Unix datagram sockets (UDS).

### v5.0.0 (Released on September 11, 2024)

In the `httpstats` package, replace misspelled `http_req_content_endoing`
and `http_res_content_endoing` with `http_req_content_encoding` and
`http_res_content_encoding`, respectively. This is a breaking change; any
dashboards or queries that filter on this tag must be updated.
