# LLM Gateway

This project is meant to just be a resume builder for myself. The high-level overview is to build an LLM Model serving system, and this project is deliberately structured to answer the canonical system-design interview prompt on the same challange.

## System Overview

llm-gateway is a stateless, horizontally-scalable reverse proxy that sits between OpenAI-API clients and a pool of self-hosted vLLM model servers or OpenAI-API compatible LLMs. It accepts OpenAI-compatible chat/completions traffic, decides which backend should serve each request (capability- and load-aware), enforces who may send how much (hierarchical token-bucket rate limiting), protects the fleet from overload (admission control with backpressure) and from failing backends (per-backend circuit breakers), reuses pooled HTTP connections, and relays streamed tokens to the caller with minimal added latency — all while emitting a complete OpenTelemetry signal set (traces, metrics, logs) to a Prometheus/Grafana/Loki/Tempo stack.

## Requirements

### In-scope

- Terminate client HTTP, parse request body to make policy decisions about which model and stream to use, and forward the request verbatim to the chosen LLM backend over a pooled connection.
- Capability-aware + latency-weighted routing acros N LLM backends per model.
- Per-API-key, per-model, and global token-bucket rate limiting.
- Concurrency-bounded admission with a bounded wait queue and `Retry-After`
- Circuit breaking per backend with a closed->open->half-open state machine.
- Transparent SSE / chunked streaming with client-disconnect propagation.
- Health/readiness, Prometheus metrics, and debug introspection endpoints.
- Backend discovery via static config or Kubernetes EndpointSlice watch

### Out of scope

- It does **not** run inference, batch tokens, or manage GPU/KV-cache.
- It does **not** transform prompt content, inject system prompts, do RAG, or re-rank.
- It does **not** persist request/responses, cache completions, or store chat history.
- It does **not** implement authentication beyond API-key identity resolution for backend LLMs.
- It does **not** depend on any managed cloud service.

## License

Copyright (c) 2026 Jareth Rader.

This project is licensed under the **GNU Affero General Public License v3.0 or later** (AGPL-3.0-or-later). See [LICENSE](LICENSE) for the full text.
