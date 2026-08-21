# Jiuin Nginx gateway

These are templates, not a replacement for the BaoTa-generated `jiuin.cn`
site configuration. Substitute every `__PLACEHOLDER__`, validate with
`nginx -t`, then include only the indicated fragment. Do not expose the Go
port or the `bkgapi` gateway on a public interface.

1. Put `http/jiuin-upstreams.conf.template` in an include evaluated inside
   Nginx's `http {}` context.
2. Add the locations from `sites/jiuin.cn.locations.conf.template` to the
   existing BaoTa server block for `jiuin.cn`; retain its static-file rules,
   certificate directives, logging, and PHP management directives.
3. Install `sites/bkgapi.jiuin.cn.conf.template` as a separate server block.
   It listens only on `127.0.0.1` and its own local TLS port. `jiuin.cn`
   talks to it with SNI `bkgapi.jiuin.cn`; browsers cannot reach it.
4. Place the two files from `snippets/` in the Nginx snippets include path.

`bkgapi.jiuin.cn` needs a certificate whose SAN covers that hostname when the
main site uses the HTTPS hop shown here. Obtain it with the deployment's
actual ACME/DNS procedure and set `proxy_ssl_trusted_certificate` to the CA
bundle that validates it. A certificate for `jiuin.cn` alone is not assumed to
cover `bkgapi.jiuin.cn`.

The primary FastCGI route has no HTTP upstream, so Nginx cannot use a single
`proxy_pass upstream` for both PHP-FPM and Go. The internal `bkgapi` server
uses `fastcgi_intercept_errors` plus a named HTTP location for Go. GET/HEAD
routes fail over on connection errors, timeout, invalid FastCGI headers, and
500/502/503/504. The exact upload location only permits its controlled replay
when `Idempotency-Key` is present; all other writes remain on PHP and are not
retried by Nginx.
