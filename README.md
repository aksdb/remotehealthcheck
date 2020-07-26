# RemoteHealthCheck

This service provides a simple health checker for remote services.
It is meant to be lightweight, so it can run on some cheap VPS or
a Raspberry Pi to monitor your websites and other services.

## Running

```
./RemoteHealthCheck -interval 10m -webListener :3000
```

... would run the health checker every 10 minutes and report the results
via HTTP endpoint listening on all interfaces on port 3000. 

## Example config

```yaml
- name: example.com
  type: group
  checks:
    - name: https
      type: tls
      address: example.com:443
    - name: smtps
      type: smtp
      address: example.com:587
    - name: imaps
      type: tls
      address: example.com:993
    - name: mumble
      type: tls
      address: example.com:64738
- name: Home
  type: tls
  address: home.dyndns.com:443
  insecure: true
```

## Available checks

All checks consist of at least a `name` and a `type`. For possible types,
see below.

### TLS

Checks the specified address by connecting using a TLS client. Optionally the
certificate can be insecure (for example to cope with self signed certs).

This check allows validating a wide range of TLS secured protocols to make
sure, the service is basically working.

| Option | Example | Description |
| --- | --- | --- |
| `address` | `something.com:1234` | Address to connect to. |
| `insecure` | `true` | If set to true, invalid certificates are allowed. | 

### SMTP

Checks the specified address by connecting via SMTP(S).

| Option | Example | Description |
| --- | --- | --- |
| `address` | `something.com:1234` | Address to connect to. |

### Group

Combines other checks to be executed. If at least one of these checks
fails, the whole group is considered to be in a failed state.

| Option | Description |
| --- | --- |
| `checks` | A list of checks to execute. |

## Reporters

The following channels are available for reporting health check
results.

### Log

Enabled by default, changes to a health check are logged to stdout. If
a check switched to a failed state, it is reported in level "warn", if
it switches to healthy, it is reported on "info".

### Web

The web reporter can be enabled by specifying a listenAddress. It exposes
two endpoints.

`/health` will simply answer with status 200 if all health checks are
currently passing, and 503 if at least one failed. This can be used as
kind of canary endpoint to be polled from SmartPhone apps like
"nock nock".

`/` renders a (very simplistic) HTML page detailing the status of the
individual checks. This can be used to proactively check what is failing
and why.

## FAQ

* Why?
  
  I needed a simple, lightweight service to watch over my servers and
  tell me if a crucial process has died or a SSL cert has expired.
  
* Why no push notifications?

  I would need an app for that which I so far was too lazy to develop.
  Another downside of "pushing" changes is, that I don't get notified
  if the *RemoteHealthCheck* itself dies. So by using a polling based
  mechanism, I would also notice if the health checker itself has gone
  down.
  
* Why no chat integration?

  Basically the same as the question before. If reports are sent to
  telegram, slack, etc. I wouldn't notice if the service just doesn't
  send anything anymore. So it's hard to distinguish between broken
  healthcheck and healthy system.