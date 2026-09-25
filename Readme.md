# Go-Redis: A Distributed Database and Event Stream

## Overview

Go-Redis is a high-performance, fault-tolerant, fully concurrent distributed in-memory data structure store, used as a database, cache, and message broker. Built entirely in Go, this project implements the core architecture of modern distributed systems, including internal sharding, asynchronous replication, log compaction, and leader election.

This system is designed to provide massive read and write concurrency without CPU lock contention, making it suitable for high-throughput applications requiring low latency and reliable data persistence.

## Architecture and Core Features

### Multithreaded Storage Engine
The core storage engine utilizes FNV-1a Hashing and Map Sharding across 256 independent shards. Each shard is protected by fine-grained `sync.RWMutex` locks. This architecture ensures that operations on different keys can proceed in parallel, eliminating the traditional single-threaded bottleneck found in similar memory stores.

### Probabilistic Garbage Collection
Memory management is handled by a background worker that intelligently samples and evicts keys based on their Time-To-Live (TTL). This probabilistic approach ensures the memory footprint remains optimized without incurring the latency spikes associated with global stop-the-world sweeps.

### AOF Persistence and Log Compaction
All write commands are persisted to disk via an Append-Only File (AOF). To prevent unbounded disk growth, the system features a background AOF rewrite engine (`BGREWRITEAOF`). This mechanism atomically squashes redundant logs into a highly compressed snapshot of the current state without blocking active client requests.

### Master-Replica Replication
High availability and read scaling are achieved through asynchronous TCP broadcasting. Secondary replica nodes connect to the master node and instantly mirror its state. All write operations on the master are reliably broadcasted to all connected replicas.

### Sentinel Leader Election
The system includes a standalone Heartbeat Monitor (Sentinel) that acts as a failover judge. By continuously monitoring the health of the master node, the Sentinel can automatically promote a healthy replica and reroute cluster traffic if the master becomes unresponsive, ensuring zero-downtime tolerance for node failures.

### Distributed Message Queue
A native append-only log architecture is built directly into the engine to support event sourcing. The stream data structure allows producers to append events (`XADD`) and consumers to read them sequentially or by offset (`XREAD`), functioning as a high-throughput message queue.

## Installation

### Prerequisites
- Go 1.21 or higher
- Make (optional, but recommended for build automation)

### Building from Source

Clone the repository and build the binaries using the provided Makefile:

```bash
git clone https://github.com/MuhammadAliyan10/Redis.git
cd Redis
make build
