const BASE_URL = 'http://localhost:9000';

type backend_list_resp = {
  backends: {
    id: number;
    name: string;
    protocol: string;
    base_url: string;
    enabled: true;
    models_served: string[];
    weight: number;
    max_concurrent: number;
  }[];
};

export async function GetBackendList(): Promise<SparseBackend[]> {
  const resp = await fetch(`${BASE_URL}/api/v1/backend`, {
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  });

  if (!resp.ok) {
    throw new Error("Couldn't retrieve data from server. Please try again.");
  }

  const body: backend_list_resp = await resp.json();

  return body['backends'].map(
    (backend) =>
      ({
        id: backend.id,
        name: backend.name,
        baseUrl: backend.base_url,
        modelsServed: backend.models_served,
        weight: backend.weight,
        inFlight: '',
        total: backend.max_concurrent,
        breakerState: 'closed',
        enabled: backend.enabled,
      }) as unknown as SparseBackend,
  );
}

type backend_resp = {
  backend: {
    id: number;
    name: string;
    protocol: 'h1' | 'h2c' | 'h2';
    base_url: string;
    enabled: boolean;
    models_served: string[];
    weight: number;
    max_concurrent: number;
    kv_cache_aware_routing: boolean;
    metrics_url: string;
    scrape_interval: number;
    max_idle_connections_per_host: number;
    idle_connection_timeout: number;
    dial_timeout: number;
    stream_stall_timeout: number;
    response_header_timeout: number;
    failure_threshold: number;
    rolling_window: number;
    open_base: number;
    open_max: number;
    backoff_factor: number;
    half_open_probes: number;
    half_open_successes: number;
    health_check_path: string;
    health_interval: number;
    verify_tls_cert: boolean;
    description: string;
    labels: string[];
    created_at: string;
    updated_at: string;
  };
};

export async function GetBackend(id: number): Promise<Backend> {
  const resp = await fetch(`${BASE_URL}/api/v1/backend/${id}`, {
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  });

  if (!resp.ok) {
    throw new Error("Couldn't retrieve data from server. Please try again.");
  }

  const body: backend_resp = await resp.json();

  return {
    id: body.backend.id,
    name: body.backend.name,
    protocol: body.backend.protocol,
    baseUrl: body.backend.base_url,
    enabled: body.backend.enabled,
    modelsServed: body.backend.models_served,
    weight: body.backend.weight,
    maxConcurrent: body.backend.max_concurrent,
    kvCacheAwareRouting: body.backend.kv_cache_aware_routing,
    metricsUrl: body.backend.metrics_url,
    scrapeInterval: body.backend.scrape_interval,
    maxIdleConnectionsPerHost: body.backend.max_idle_connections_per_host,
    idleConnectionTimeout: body.backend.idle_connection_timeout,
    dialTimeout: body.backend.dial_timeout,
    streamStallTimeout: body.backend.stream_stall_timeout,
    responseHeaderTimeout: body.backend.response_header_timeout,
    failureThreshold: body.backend.failure_threshold,
    rollingWindow: body.backend.rolling_window,
    openBase: body.backend.open_base,
    openMax: body.backend.open_max,
    backoffFactor: body.backend.backoff_factor,
    halfOpenProbes: body.backend.half_open_probes,
    halfOpenSuccesses: body.backend.half_open_successes,
    healthCheckPath: body.backend.health_check_path,
    healthInterval: body.backend.health_interval,
    verifyTlsCert: body.backend.verify_tls_cert,
    description: body.backend.description,
    labels: body.backend.labels,
    createdAt: body.backend.created_at,
    updatedAt: body.backend.updated_at,
  };
}

export async function CreateBackend(backend: Backend): Promise<number> {
  const resp = await fetch(`${BASE_URL}/api/v1/backend`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify({
      name: backend.name,
      protocol: backend.protocol,
      base_url: backend.baseUrl,
      enabled: backend.enabled,
      models_served: backend.modelsServed,
      weight: backend.weight,
      max_concurrent: backend.maxConcurrent,
      kv_cache_aware_routing: backend.kvCacheAwareRouting,
      metrics_url: backend.metricsUrl,
      scrape_interval: backend.scrapeInterval,
      max_idle_connections_per_host: backend.maxIdleConnectionsPerHost,
      idle_connection_timeout: backend.idleConnectionTimeout,
      dial_timeout: backend.dialTimeout,
      stream_stall_timeout: backend.streamStallTimeout,
      response_header_timeout: backend.responseHeaderTimeout,
      failure_threshold: backend.failureThreshold,
      rolling_window: backend.rollingWindow,
      open_base: backend.openBase,
      open_max: backend.openMax,
      backoff_factor: backend.backoffFactor,
      half_open_probes: backend.halfOpenProbes,
      half_open_successes: backend.halfOpenSuccesses,
      health_check_path: backend.healthCheckPath,
      health_interval: backend.healthInterval,
      verify_tls_cert: backend.verifyTlsCert,
      description: backend.description,
      labels: backend.labels,
    }),
  });

  if (!resp.ok) {
    throw new Error("Couldn't connect to server. Please try again.");
  }

  const body = await resp.json();
  return body.backend_id;
}

export async function UpdateBackend(id: number, backend: Backend): Promise<void> {
  const resp = await fetch(`${BASE_URL}/api/v1/backend/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify({
      name: backend.name,
      protocol: backend.protocol,
      base_url: backend.baseUrl,
      enabled: backend.enabled,
      models_served: backend.modelsServed,
      weight: backend.weight,
      max_concurrent: backend.maxConcurrent,
      kv_cache_aware_routing: backend.kvCacheAwareRouting,
      metrics_url: backend.metricsUrl,
      scrape_interval: backend.scrapeInterval,
      max_idle_connections_per_host: backend.maxIdleConnectionsPerHost,
      idle_connection_timeout: backend.idleConnectionTimeout,
      dial_timeout: backend.dialTimeout,
      stream_stall_timeout: backend.streamStallTimeout,
      response_header_timeout: backend.responseHeaderTimeout,
      failure_threshold: backend.failureThreshold,
      rolling_window: backend.rollingWindow,
      open_base: backend.openBase,
      open_max: backend.openMax,
      backoff_factor: backend.backoffFactor,
      half_open_probes: backend.halfOpenProbes,
      half_open_successes: backend.halfOpenSuccesses,
      health_check_path: backend.healthCheckPath,
      health_interval: backend.healthInterval,
      verify_tls_cert: backend.verifyTlsCert,
      description: backend.description,
      labels: backend.labels,
    }),
  });

  if (!resp.ok) {
    throw new Error("Couldn't connect to server. Please try again.");
  }
}

export async function DeleteBackend(id: number): Promise<void> {
  const resp = await fetch(`${BASE_URL}/api/v1/backend/${id}`, {
    method: 'DELETE',
  });

  if (!resp.ok) {
    throw new Error("Couldn't connect to server. Please try again.");
  }
}
