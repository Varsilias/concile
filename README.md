# Concile
`Concile` is a High Thoroughput Financial-Grade Distributed Reconciliation Engine written in Go.

# Motivation
During my time at [VPay Africa](https://vpay.africa), I built an internal tool closely related to what `Concile` aims to achieve. Concile is my attempt at rebuilding that tool but in Golang but with more features that the internal version has. I am also using it as a way to strengthen my fundamental knowledge in concepts I am farmilar with but have become rusty in them. Ideas like **Concurreny Management**, **Backpressure Handling** and more. I have a list of features in mind which will be implemented gradually in the coming days

> This is a work in progress

## Features
- [ ] Multiple Transaction Sources
- [ ] Event Ingestion
- [ ] Idempotency Guarantees
- [ ] Deduplication
- [ ] Worker Pools
- [ ] Backpressure
- [ ] Persistent Storage
- [ ] Observability
- [ ] Benchmarking
- [ ] Failure Injection
- [ ] Eventually distributed coordination

## Architecture
![Concile High Level Architecture](https://github.com/Varsilias/concile/blob/main/architecture.png)
