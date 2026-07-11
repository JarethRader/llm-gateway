# llm-gateway

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.1.0](https://img.shields.io/badge/AppVersion-0.1.0-informational?style=flat-square)

Kubernetes-native LLM inference gateway

**Homepage:** <https://github.com/jarethrader/llm-gateway>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Jareth Rader | <jarethrader@gmail.com> |  |

## Source Code

* <https://github.com/jarethrader/llm-gateway>

## Requirements

Kubernetes: `>=1.25.0-0`

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| autoscaling.enable | bool | `false` | Enable horizontal pod autoscaling |
| autoscaling.maxReplicas | int | `6` | Maximum number of replicas |
| autoscaling.minReplicas | int | `2` | Minimum number of replicas |
| autoscaling.targetCPUUtilizationPercentage | int | `70` | Target CPU utilization percentage |
| config | object | `{"admission":{"enabled":true,"max_concurrent":64,"max_queue":256,"max_retry_after":"30s","token_weighted":false},"app":{"addr":"0.0.0.0","environment":"PROD","log_level":"INFO","name":"gateway","port":":8848","version":"0.1.0"},"auth":{"api_url":"","interval":"60s","key_table_path":"/secrets/keys.json","mode":"static"},"backend_discovery":{"api_url":"","backends":[],"interval":"60s","mode":"static"},"circuit":{"backoff_factor":2,"buckets":10,"enabled":true,"failure_ratio":0.5,"failure_statuses":[500,502,503,504,429],"half_open_max":3,"half_open_success":3,"min_requests":20,"open_base":"5s","open_max":"60s","window":"10s"},"connection_pool":{"dial_timeout":"2s","h2_ping_timeout":"5s","h2_read_idle_timeout":"15s","idle_conn_timeout":"90s","max_conns_per_host":0,"max_idle_conns":512,"max_idle_conns_per_host":64,"read_buffer_size":65536,"tcp_keep_alive":"30s","tls_handshake_timeout":"3s","write_buffer_size":65536},"load_balancer":{"ewma_alpha":0.2,"max_in_flight_per_backend":256,"scrape":{"cache_ttl":"1s","enabled":true,"interval":"2s","timeout":"1s"},"weights":{"cache":0.8,"in_flight":0.5,"latency":1,"ref_ttft_ms":200}},"namespace":"llm-gateway","owner":"","proxy":{"max_retries":2},"rate_limit":{"default_key":{"burst":100,"rate_per_sec":50},"default_model":{"burst":1000,"rate_per_sec":500},"enabled":true,"global":{"burst":4000,"rate_per_sec":2000},"idle_ttl":"10m","max_retry_after":"60s","per_key":{},"per_model":{},"sweep_interval":"30s","token_weighted":false},"shutdown_wait_sec":30,"sse_streaming":{"flush_every_write":true,"frame_aware":true,"heartbeat_interval":"5s","idle_timeout":"30s","max_body_bytes":4194304},"telemetry":{"enabled":true,"endpoint":"","insecure":false,"sample_ratio":1}}` | Application configuration (rendered as configmap) |
| config.admission.enabled | bool | `true` | Enable admission control |
| config.admission.max_concurrent | int | `64` | Maximum concurrent requests |
| config.admission.max_queue | int | `256` | Maximum queued requests |
| config.admission.max_retry_after | string | `"30s"` | Maximum Retry-After for admission control |
| config.admission.token_weighted | bool | `false` | Use token-weighted admission |
| config.app.addr | string | `"0.0.0.0"` | Bind address |
| config.app.environment | string | `"PROD"` | Deployment environment |
| config.app.log_level | string | `"INFO"` | Log level (DEBUG, INFO, WARN, ERROR) |
| config.app.name | string | `"gateway"` | Application name |
| config.app.port | string | `":8848"` | Bind port |
| config.app.version | string | `"0.1.0"` | Application version |
| config.auth.api_url | string | `""` | API URL for dynamic key fetching |
| config.auth.interval | string | `"60s"` | Dynamic key refresh interval |
| config.auth.key_table_path | string | `"/secrets/keys.json"` | Path to API keys file |
| config.auth.mode | string | `"static"` | Authentication mode (static, dynamic) |
| config.backend_discovery.api_url | string | `""` | API URL for dynamic backend discovery |
| config.backend_discovery.backends | list | `[]` | Static backend list |
| config.backend_discovery.interval | string | `"60s"` | Backend discovery refresh interval |
| config.backend_discovery.mode | string | `"static"` | Backend discovery mode (static, dynamic) |
| config.circuit.backoff_factor | float | `2` | Exponential backoff factor |
| config.circuit.buckets | int | `10` | Number of buckets in the window |
| config.circuit.enabled | bool | `true` | Enable circuit breaker |
| config.circuit.failure_ratio | float | `0.5` | Failure ratio threshold to open circuit |
| config.circuit.failure_statuses | list | `[500,502,503,504,429]` | HTTP statuses considered failures |
| config.circuit.half_open_max | int | `3` | Max requests in half-open state |
| config.circuit.half_open_success | int | `3` | Successes needed to close circuit from half-open |
| config.circuit.min_requests | int | `20` | Minimum requests before circuit evaluation |
| config.circuit.open_base | string | `"5s"` | Base open state duration |
| config.circuit.open_max | string | `"60s"` | Maximum open state duration |
| config.circuit.window | string | `"10s"` | Circuit evaluation window |
| config.connection_pool.dial_timeout | string | `"2s"` | TCP dial timeout |
| config.connection_pool.h2_ping_timeout | string | `"5s"` | HTTP/2 ping timeout |
| config.connection_pool.h2_read_idle_timeout | string | `"15s"` | HTTP/2 read idle timeout |
| config.connection_pool.idle_conn_timeout | string | `"90s"` | Idle connection timeout |
| config.connection_pool.max_conns_per_host | int | `0` | Maximum connections per host (0 = unlimited) |
| config.connection_pool.max_idle_conns | int | `512` | Maximum idle connections in pool |
| config.connection_pool.max_idle_conns_per_host | int | `64` | Maximum idle connections per host |
| config.connection_pool.read_buffer_size | int | `65536` | TCP read buffer size in bytes |
| config.connection_pool.tcp_keep_alive | string | `"30s"` | TCP keep-alive interval |
| config.connection_pool.tls_handshake_timeout | string | `"3s"` | TLS handshake timeout |
| config.connection_pool.write_buffer_size | int | `65536` | TCP write buffer size in bytes |
| config.load_balancer.ewma_alpha | float | `0.2` | EWMA smoothing factor |
| config.load_balancer.max_in_flight_per_backend | int | `256` | Maximum in-flight requests per backend |
| config.load_balancer.scrape.cache_ttl | string | `"1s"` | Scrape cache TTL |
| config.load_balancer.scrape.enabled | bool | `true` | Enable backend metrics scraping |
| config.load_balancer.scrape.interval | string | `"2s"` | Scrape interval |
| config.load_balancer.scrape.timeout | string | `"1s"` | Scrape timeout |
| config.load_balancer.weights.cache | float | `0.8` | Weight for cache hit rate |
| config.load_balancer.weights.in_flight | float | `0.5` | Weight for in-flight requests |
| config.load_balancer.weights.latency | float | `1` | Weight for latency in load balancing decision |
| config.load_balancer.weights.ref_ttft_ms | int | `200` | Reference TTFT in milliseconds |
| config.namespace | string | `"llm-gateway"` | Kubernetes namespace for the gateway |
| config.owner | string | `""` | Owner identifier for this deployment |
| config.proxy.max_retries | int | `2` | Maximum number of retries for proxied requests |
| config.rate_limit.default_key.burst | int | `100` | Default per-key burst limit |
| config.rate_limit.default_key.rate_per_sec | int | `50` | Default per-key rate limit (requests/sec) |
| config.rate_limit.default_model.burst | int | `1000` | Default per-model burst limit |
| config.rate_limit.default_model.rate_per_sec | int | `500` | Default per-model rate limit (requests/sec) |
| config.rate_limit.enabled | bool | `true` | Enable rate limiting |
| config.rate_limit.global.burst | int | `4000` | Global burst limit |
| config.rate_limit.global.rate_per_sec | int | `2000` | Global rate limit (requests/sec) |
| config.rate_limit.idle_ttl | string | `"10m"` | Idle rate limit entry TTL |
| config.rate_limit.max_retry_after | string | `"60s"` | Maximum Retry-After header value |
| config.rate_limit.per_key | object | `{}` | Per-key rate limits override |
| config.rate_limit.per_model | object | `{}` | Per-model rate limits override |
| config.rate_limit.sweep_interval | string | `"30s"` | Rate limit state sweep interval |
| config.rate_limit.token_weighted | bool | `false` | Use token-weighted rate limiting |
| config.shutdown_wait_sec | int | `30` | Graceful shutdown wait time in seconds |
| config.sse_streaming.flush_every_write | bool | `true` | Flush response on every write |
| config.sse_streaming.frame_aware | bool | `true` | Enable frame-aware streaming |
| config.sse_streaming.heartbeat_interval | string | `"5s"` | SSE heartbeat interval |
| config.sse_streaming.idle_timeout | string | `"30s"` | SSE stream idle timeout |
| config.sse_streaming.max_body_bytes | int | `4194304` | Maximum body size in bytes |
| config.telemetry.enabled | bool | `true` | Enable telemetry collection |
| config.telemetry.endpoint | string | `""` | OTLP endpoint for telemetry |
| config.telemetry.insecure | bool | `false` | Allow insecure (non-TLS) telemetry connection |
| config.telemetry.sample_ratio | float | `1` | Telemetry sampling ratio (0.0-1.0) |
| deployment.annotations | object | `{}` | Annotations for the deployment |
| fullnameOverride | string | `""` | String to fully override llm-gateway.fullname template |
| gateway | object | `{"annotations":{},"enabled":true,"gateway":{"gateway":"default","gatewayClassName":"","sectionName":"http"},"hosts":null,"paths":null,"tls":{"secretName":""}}` | Gateway API configuration (alternative to Ingress) |
| gateway.annotations | object | `{}` | Annotations for the Gateway resource |
| gateway.enabled | bool | `true` | Enable Gateway resource |
| gateway.gateway.gateway | string | `"default"` | Gateway name |
| gateway.gateway.gatewayClassName | string | `""` | GatewayClass name to reference |
| gateway.gateway.sectionName | string | `"http"` | Gateway section name |
| gateway.hosts | string | `nil` | Gateway hosts configuration |
| gateway.paths | string | `nil` | Gateway path routing rules |
| gateway.tls.secretName | string | `""` | TLS secret name for the Gateway |
| image | object | `{"digest":"","pullPolicy":"IfNotPresent","repository":"ghcr.io/jarethrader/llm-gateway","tag":""}` | Container image configuration |
| image.digest | string | `""` | Image digest |
| image.pullPolicy | string | `"IfNotPresent"` | Image pull policy |
| image.repository | string | `"ghcr.io/jarethrader/llm-gateway"` | Image repository |
| image.tag | string | `""` | Image tag (defaults to Chart.appVersion) |
| imagePullSecrets | list | `[]` | Docker registry secrets for pulling images |
| ingress.className | string | `""` | Ingress className |
| ingress.enabled | bool | `false` | Enable ingress resource |
| ingress.hosts | string | `nil` | Ingress hosts configuration |
| ingress.tls | list | `[]` | Ingress TLS configuration |
| metrics.serviceMonitor.additionalLabels | object | `{}` | Additional labels for ServiceMonitor |
| metrics.serviceMonitor.enabled | bool | `false` | Create ServiceMonitor for Prometheus Operator |
| metrics.serviceMonitor.interval | string | `"15s"` | Scrape interval |
| metrics.serviceMonitor.scrapeTimeout | string | `"10s"` | Scrape timeout |
| nameOverride | string | `""` | String to partially override llm-gateway.fullname template |
| networkPolicy.allowDNS | bool | `true` | Allow DNS egress |
| networkPolicy.allowIngress | bool | `true` | Allow ingress traffic |
| networkPolicy.egressTo | list | `[]` | Egress traffic rules |
| networkPolicy.enabled | bool | `true` | Enable NetworkPolicy |
| networkPolicy.ingressFrom | list | `[]` | Ingress traffic rules |
| podDisruptionBudget.enabled | bool | `true` | Enable PodDisruptionBudget |
| podDisruptionBudget.minAvailable | int | `1` | Minimum available pods during disruption |
| rbac.configMapNames | list | `[]` | ConfigMap names to grant RBAC access to |
| rbac.create | bool | `true` | Create RBAC resources |
| replicaCount | int | `1` | Number of replicas to deploy |
| resources.limits | object | `{"memory":"512Mi"}` | Kubernetes resource limits (CPU omitted to avoid CFS throttling) |
| resources.requests | object | `{"cpu":"100m","memory":"128Mi"}` | Kubernetes resource requests |
| secret.data | object | `{}` | Inline secret data |
| secret.existingSecret | string | `""` | Name of existing Secret to use for configuration |
| securityContext.podSecurityContext.fsGroup | int | `65532` | Filesystem group ID |
| securityContext.podSecurityContext.runAsGroup | int | `65532` | Container group ID |
| securityContext.podSecurityContext.runAsNonRoot | bool | `true` | Run container as non-root |
| securityContext.podSecurityContext.runAsUser | int | `65532` | Container user ID |
| securityContext.podSecurityContext.seccompProfile.type | string | `"RuntimeDefault"` | Seccomp profile type |
| securityContext.securityContext.allowPrivilegeEscalation | bool | `false` | Allow privilege escalation |
| securityContext.securityContext.capabilities.drop | list | `["ALL"]` | Linux capabilities to drop |
| securityContext.securityContext.readOnlyRootFilesystem | bool | `true` | Mount root filesystem as read-only |
| service.port | int | `8080` | Service port |
| service.type | string | `"ClusterIP"` | Kubernetes service type |
| serviceAccount.annotations | object | `{}` | Annotations for the service account |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
