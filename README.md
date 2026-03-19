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

## Durability Model
- A transaction record is considered durable only after WAL flush
- Records in in-memory queue may be lost on crash
- System provides at-least-once processing within batch window
- The in-memory queue by default has a size of **4KB**. Expect at least the same amount of data to be lost when a crash occurs
- This also means that **4KB** of data may be reprocessed because it was not yet written to a WAL file

