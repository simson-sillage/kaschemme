# Kaschemme

Simple proxy for Redis Sentinel HA cluster (or TCP). Use at your own risk.

## Modes

Currently, we support two modes:

- `redis`: Fetches Redis masters from sentinels and then proxies the request.
- `tcp`: Just proxies a TCP endpoint.

## Usage

```
Usage:
  kaschemme [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  redis       run kaschemme in redis mode
  tcp         run kaschemme in tcp streaming mode
  version     prints the version

Flags:
  -h, --help   help for kaschemme
```

### TCP

```
Usage:
  kaschemme tcp [flags]

Flags:
  -h, --help                    help for tcp
      --local-address string    local address to bind to       [REQUIRED]
      --remote-address string   remote address to foward to    [REQUIRED]
```

Example:

`kaschemme tcp --local-address :8080 --remote-address 192.168.3.100:80`

Forwards any traffic on `localhost` port `8080` to the remote server `192.168.3.100` port `80`. Starts logging to stdout.

### Redis

```
Usage:
  kaschemme redis [flags]

Flags:
  -h, --help                   help for redis
      --local-address string   local address to bind to    [REQUIRED]
      --redis-conf string      path to redis config        [REQUIRED]
```

Example usage:

`kaschemme redis --local-address :6379 --redis-conf redis-config.yaml`

Forwards any traffic on `localhost` port `6379` to the Redis Sentinel HA cluster specified in the `redis-config.yaml`. Traffic is always forwarded to the Redis master, there is no introspection. The current Redis master is determined by quering the Redis sentinels specified.

Example `redis-config.yaml`:

```yaml
 # List all your sentinels here.
sentinels:
- 192.168.1.100:26379
- 192.168.1.101:26379
- 192.168.1.102:26379
# Provide your sentinel `requirepass`.
sentinel_pass: mypass
# How long to wait for an answer from a sentinel, in seconds.
sentinel_timeout: 2  # Seconds. Defaults to 2 seconds.
master_name: master
# How long to cache the current master, in seconds.
master_cache_timeout: 6  # Seconds. Defaults to 6 seconds.
```

## Build

Just clone the repo and run `go build .` inside.

## Motivation

You have an app, e.g. Label Studio, that takes only a Redis endpoint, but you are running a Redis Sentinel HA cluster. Send the app traffic to _kaschemme_ and let _kaschemme_ talk to the Redis Sentinel cluster.

## Roadmap

Maybe implement the same for PostgreSQL (see Label Studio)
