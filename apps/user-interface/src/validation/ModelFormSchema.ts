import { z } from 'zod';

/**
 * Name: string
 * Protocol: "h1 (HTTP/1.1 -- default)" | "h2" | "h2c"
 * BaseURL: string (validation regex url)
 * Enabled: boolean
 * ModelsServed: string[]
 * Weight: number
 * MaxConcurrent: number
 * KVCacheAwareRouting: boolean
 * MetricsURL: string (validate regex url)
 * ScrapeInterval(sec): number
 * MaxIdleConnectionsPerHost: number
 * IdleConnectionTimeout(sec): number
 * DialTimeout(sec): number
 * StreamStallTimeout(sec): number
 * ResponseHeaderTimeout(sec): number
 * failureThreshold: number
 * rollingWindow(sec): number
 * openBase(sec): number
 * openMax(sec): number
 * backoffFactor: number
 * halfOpenProbes: number
 * halfOpenSuccess: number
 * HealthCheckPath: string (regex validation route path?)
 * HealthInterval(sec): number
 * VerifyTLSCert: boolean
 * Description: string
 * Labels: string[] (key=value)
 */

export const ModelProtocols = [
  { label: 'h1 (HTTP/1.1 - default)', value: 'h1' },
  { label: 'h2', value: 'h2' },
  { label: 'h2c', value: 'h2c' },
] as const;

export const ModelSchema = z
  .object({
    name: z
      .string()
      .min(3, 'Backend name must be at least 3 characters.')
      .max(64, 'Backend name must be at most 64 characters.')
      .regex(
        /^[a-zA-Z0-9._-]+$/,
        'Use only letters, numbers, and . _ - (no spaces).',
      )
      .meta({
        id: 'name',
        title: 'Name',
        description: 'Used as the backend label in metrics, logs, and traces.',
      }),
    protocol: z
      .enum(ModelProtocols.map((p) => p.value), { error: 'Please select a protocol.' })
      .default('h1')
      .meta({
        id: 'protocol',
        title: 'Protocol',
        description:
          'Wire protocol for the upstream. vLLM/uvicorn speak h1; use h2/h2c only behind a proxy that supports it.',
      }),
    baseUrl: z
      .url({ error: 'Enter a valid URL, e.g. http://vllm-3.vllm.svc:8000.' })
      .meta({
        id: 'base_url',
        title: 'Base URL',
        description:
          'Scheme, host, and port of the backend server, e.g. http://vllm-3.vllm.svc:8000.',
      }),
    enabled: z.boolean().default(false).meta({
      id: 'enabled',
      title: 'Enabled',
      description:
        'Off removes it from rotation without deleting the configuration.',
    }),
    modelsServed: z
      .array(z.string().min(1, 'Model id cannot be empty.'))
      .min(1, 'Backend must serve at least 1 model.')
      .meta({
        id: 'models_served',
        title: 'Models served',
        description:
          'Model ids this backend can serve. Requests for these ids are eligible to route here (capability filter).',
      }),
    weight: z
      .int({ error: 'Weight is required.' })
      .min(1, 'Weight must be at least 1.')
      .max(1000, 'Weight must be at most 1000.')
      .default(100)
      .meta({
        id: 'weight',
        title: 'Weight',
        description:
          'Relative capacity hint for power-of-two-choices (P2C) scoring. Higher gets proportionally more traffic.',
      }),
    maxConcurrent: z
      .int('Max concurrent must be a whole number.')
      .min(0, 'Max concurrent cannot be negative.')
      .default(1)
      .meta({
        id: 'max_concurrent',
        title: 'Max concurrent',
        description:
          'Per-backend in-flight request ceiling. 0 means unlimited.',
      }),
    kvCacheAwareRouting: z.boolean().default(false).meta({
      id: 'kv_cache_aware_routing',
      title: 'KV-cache-aware routing',
      description: 'Scrape vLLM /metrics to bias routing by KV-cache pressure.',
    }),
    metricsUrl: z.url({ error: 'Enter a valid metrics URL.' }).optional().meta({
      id: 'metrics_url',
      title: 'Metrics URL',
      description:
        'Prometheus metrics endpoint to scrape. Required when KV-cache-aware routing is enabled.',
    }),
    scrapeInterval: z
      .int('Scrape interval must be a whole number of seconds.')
      .min(1, 'Scrape interval must be at least 1 second.')
      .max(120, 'Scrape interval must be at most 120 seconds.')
      .default(15)
      .meta({
        id: 'scrape_interval',
        title: 'Scrape interval (seconds)',
        description: 'How often to scrape the metrics endpoint.',
      }),
    maxIdleConnectionsPerHost: z
      .int('Must be a whole number.')
      .min(1, 'Backend must be able to maintain at least 1 connection.')
      .default(32)
      .meta({
        id: 'max_idle_connections_per_host',
        title: 'Max idle connections / host',
        description:
          'Size of the idle keep-alive connection pool kept per backend host.',
      }),
    idleConnectionTimeout: z
      .int('Must be a whole number of seconds.')
      .min(30, 'Idle connection timeout must be at least 30 seconds.')
      .default(90)
      .meta({
        id: 'idle_connection_timeout',
        title: 'Idle conn timeout (seconds)',
        description:
          'How long an idle keep-alive connection is kept before being closed.',
      }),
    dialTimeout: z
      .int('Must be a whole number of seconds.')
      .min(1, 'Dial timeout must be at least 1 second.')
      .default(5)
      .meta({
        id: 'dial_timeout',
        title: 'Dial timeout (seconds)',
        description:
          'Maximum time to establish a new TCP connection to the backend.',
      }),
    streamStallTimeout: z
      .int('Must be a whole number of seconds.')
      .min(1, 'Stream-stall timeout must be at least 1 second.')
      .default(30)
      .meta({
        id: 'stream_stall_timeout',
        title: 'Stream-stall timeout (seconds)',
        description:
          'Abort a streaming response if no bytes are received for this long.',
      }),
    responseHeaderTimeout: z
      .int('Must be a whole number of seconds.')
      .min(1, 'Response-header timeout must be at least 1 second.')
      .default(30)
      .meta({
        id: 'response_header_timeout',
        title: 'Response-header timeout (seconds)',
        description:
          "Maximum time to wait for the backend's response headers after the request is sent.",
      }),
    failureThreshold: z
      .int('Failure threshold must be a whole number.')
      .min(1, 'Failure threshold must be at least 1.')
      .default(5)
      .meta({
        id: 'failure_threshold',
        title: 'Failure threshold',
        description:
          'Number of failures within the rolling window before the circuit breaker opens.',
      }),
    rollingWindow: z
      .int('Must be a whole number of seconds.')
      .min(1, 'Rolling window must be at least 1 second.')
      .default(10)
      .meta({
        id: 'rolling_window',
        title: 'Rolling window (seconds)',
        description: 'Sliding window over which failures are counted.',
      }),
    openBase: z
      .int('Must be a whole number of seconds.')
      .min(1, 'Open base must be at least 1 second.')
      .default(1)
      .meta({
        id: 'open_base',
        title: 'Open base (seconds)',
        description:
          'Initial cool-down before the breaker first transitions to half-open.',
      }),
    openMax: z
      .int('Must be a whole number of seconds.')
      .min(1, 'Open max must be at least 1 second.')
      .default(30)
      .meta({
        id: 'open_max',
        title: 'Open max (seconds)',
        description:
          'Upper bound on the breaker cool-down after repeated failures.',
      }),
    backoffFactor: z
      .number({ error: 'Backoff factor is required.' })
      .min(1, 'Backoff factor must be at least 1.')
      .default(2)
      .meta({
        id: 'backoff_factor',
        title: 'Backoff factor',
        description:
          'Multiplier applied to the open duration on each consecutive trip (exponential backoff).',
      }),
    halfOpenProbes: z
      .int('Must be a whole number.')
      .min(1, 'There must be at least 1 half-open probe.')
      .default(2)
      .meta({
        id: 'half_open_probes',
        title: 'Half-open probes',
        description:
          'Number of trial requests allowed while the breaker is half-open.',
      }),
    halfOpenSuccesses: z
      .int('Must be a whole number.')
      .min(1, 'There must be at least 1 half-open success.')
      .default(2)
      .meta({
        id: 'half_open_successes',
        title: 'Half-open successes',
        description: 'Successful probes required to close the breaker again.',
      }),
    healthCheckPath: z
      .string()
      .regex(
        /^\/([a-zA-Z0-9\-._~%]+(?:\/[a-zA-Z0-9\-._~%]+)*)?\/?$/,
        'Must be an absolute path starting with /, e.g. /health.',
      )
      .default('/health')
      .meta({
        id: 'health_check_path',
        title: 'Health check path',
        description: 'Path polled to determine backend health, e.g. /health.',
      }),
    healthInterval: z
      .int('Must be a whole number of seconds.')
      .min(1, 'Health interval must be at least 1 second.')
      .default(10)
      .meta({
        id: 'health_interval',
        title: 'Health interval (seconds)',
        description: 'How often to poll the health check path.',
      }),
    verifyTlsCert: z.boolean().default(false).meta({
      id: 'verify_tls_cert',
      title: 'Verify TLS certificate',
      description:
        "Verify the backend's TLS certificate. Applies only when the base URL is https.",
    }),
    description: z
      .string()
      .max(280, 'Description must be at most 280 characters.')
      .optional()
      .meta({
        id: 'description',
        title: 'Description',
        description: 'Optional free-form notes about this backend.',
      }),
    labels: z
      .array(
        z
          .string()
          .regex(/^[^=\s]+=[^=]*$/, 'Each label must be in key=value format.'),
      )
      .default([])
      .meta({
        id: 'labels',
        title: 'Labels',
        description:
          'Optional key=value labels for filtering and grouping backends.',
      }),
  })
  .superRefine((val, ctx) => {
    if (val.kvCacheAwareRouting && !val.metricsUrl) {
      ctx.addIssue({
        code: 'custom',
        path: ['metricsUrl'],
        message:
          'Metrics URL is required when KV-cache-aware routing is enabled.',
      });
    }
    if (val.openMax < val.openBase) {
      ctx.addIssue({
        code: 'custom',
        path: ['openMax'],
        message: 'Open max must be greater than or equal to open base.',
      });
    }
    if (val.halfOpenSuccesses > val.halfOpenProbes) {
      ctx.addIssue({
        code: 'custom',
        path: ['halfOpenSuccesses'],
        message: 'Half-open successes cannot exceed half-open probes.',
      });
    }
  });


export type ModelFormData = z.infer<typeof ModelSchema>;