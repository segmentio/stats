# History

### v5.1.0

Add support for publishing stats to Unix datagram sockets (UDS).

### v5.0.0 (Released on September 11, 2024)

In the `httpstats` package, replace misspelled `http_req_content_endoing`
and `http_res_content_endoing` with `http_req_content_encoding` and
`http_res_content_encoding`, respectively. This is a breaking change; any
dashboards or queries that filter on this tag must be updated.
