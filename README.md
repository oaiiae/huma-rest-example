# Huma Rest Example

This is a toy project for me to test and explore. Most of the value here is not in the code itself but more in the various configuration for managing and operating it.

## Why HUMA?

- Generates OpenAPI from the actual code and embeds a swagger for it
- Documentation is embedded with the code making required changes obvious
- Makes use of generics to use idiomatic functions as HTTP handlers.
  ```go
  func Handle[I, O any](context.Context, I) (O, error) { ... }
  ```
- Handles errors and marshalling consistently

## Metrics

The API expose those metrics:
```
build_info{goversion,title,version,revision,created}
http_requests_in_flight{method,path}
http_request_duration_seconds_bucket{method,path,status,le}
http_request_duration_seconds_sum{method,path,status}
http_request_duration_seconds_count{method,path,status}
http_requests_total{method,path,status}
process_*
```

This allow for request rate, error rate, concurrency, latency percentiles, averages...

## CI / CD

### Testing & Linting

There is basic github actions for triggering test and lint runs. The linter (golangci) configuration is pretty aggressive but most of the time it was for the better. Almost never been in the way.

### Releasing

Making use of Goreleaser to streamline the process. The configuration is pretty basic but it probably fits lots of real world use cases. It builds the project according to a OS / Arch matrix, build a docker image, pushes it to a registry & creates a changelog.

### Updating

Basic Renovate configuration, making use of the Mend-hosted bot.
