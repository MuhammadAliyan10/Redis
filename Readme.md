```markdown
<!-- /Users/muhammadaliyan/Desktop/Redis/Readme.md -->
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
```

This will generate three executable binaries in the `bin/` directory: `redis-server`, `redis-replica`, and `redis-sentinel`.

## Usage and Cluster Setup

### 1. Start the Master Node
Run the primary database instance on the default port (6379):

```bash
make run
```
Or manually:
```bash
./bin/redis-server --port 6379
```

### 2. Start a Replica Node
Run a secondary node that syncs its data from the master:

```bash
make run-replica
```
Or manually:
```bash
./bin/redis-server --port 6380 --replicaof 127.0.0.1:6379
```

### 3. Start the Sentinel (Auto-Failover Monitor)
Run the monitoring process to ensure high availability:

```bash
make run-sentinel
```
Or manually:
```bash
./bin/redis-sentinel
```

## Supported Commands

The server parses and communicates using the REdis Serialization Protocol (RESP). The following commands are fully supported:

### Key-Value Operations
- `SET key value [EX seconds]`: Set the string value of a key, optionally with a time-to-live expiration.
- `GET key`: Get the value of a key.
- `DEL key`: Delete a key.

### Stream Operations
- `XADD stream_name event_data`: Append a new event message to a stream.
- `XREAD stream_name offset_id`: Read messages from a stream starting after the specified offset ID.

### Server Operations
- `BGREWRITEAOF`: Trigger a background process to rewrite and compress the Append-Only File.
- `PING`: Test server connectivity and latency.
- `SYNC`: Internal command used by replicas to initiate data synchronization.
- `PROMOTE`: Internal command used by Sentinel to elevate a replica to master status.

## Development and Testing

To run the test suite and ensure all components are functioning correctly:

```bash
make test
```

To format the codebase according to Go standards:

```bash
make fmt
```

## Contributing

Contributions are welcome and appreciated. Please review the `CONTRIBUTING.md` file for detailed instructions on how to submit pull requests, report bugs, or request features. Ensure that all new code adheres to the existing architecture patterns and includes appropriate test coverage.

Please note that this project is released with a Contributor Code of Conduct (`CODE_OF_CONDUCT.md`). By participating in this project you agree to abide by its terms.

## License

This project is licensed under the MIT License. See the `LICENSE` file for full details.
```
