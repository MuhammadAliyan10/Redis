# High-Performance Distributed In-Memory Key-Value Store

An industrial-grade, lock-free, distributed in-memory data structure store built entirely in Go. Designed to handle extreme throughput with zero garbage-collection latency spikes, this system translates OS-level C concepts (such as `epoll` and `fork()`) into highly concurrent, idiomatic Go.

This project was built to explore the deep internals of distributed systems, shifting from a naive multi-threaded architecture with heavy mutex contention to a highly optimized, lock-free **Actor Model** execution engine.

---

## Table of Contents
1. [Core Architecture](#core-architecture)
    - [Layer 1: The Network Boundary](#layer-1-the-network-boundary)
    - [Layer 2: The Execution Engine](#layer-2-the-execution-engine)
    - [Layer 3: The Memory Subsystem](#layer-3-the-memory-subsystem)
    - [Layer 4: Persistence & Durability](#layer-4-persistence--durability)
    - [Layer 5: Distributed Topology](#layer-5-distributed-topology)
2. [Supported Data Structures & Commands](#supported-data-structures--commands)
3. [Getting Started](#getting-started)
4. [Deployment & Clustering](#deployment--clustering)
5. [Codebase Structure](#codebase-structure)
6. [Performance Characteristics](#performance-characteristics)

---

## Core Architecture

This project is built strictly across five architectural layers, separating network I/O, execution, memory management, disk persistence, and cluster replication.

### Layer 1: The Network Boundary
- **Concurrent TCP Acceptor**: The server handles thousands of concurrent clients by spinning up a lightweight Goroutine per TCP connection.
- **Zero-Copy RESP Parser**: Traditional parsers use `bufio.Scanner` and constantly allocate new strings, triggering Go's Garbage Collector. This system uses raw byte arithmetic (`[][]byte`) and `bytes.Index` to extract commands directly from the network buffer without copying memory.
- **Pipelining Support**: The parser can handle highly fragmented TCP packets or massive pipelines of commands sent in a single burst without blocking the network thread.

### Layer 2: The Execution Engine
- **The Actor Model**: Traditional databases use `sync.RWMutex` to protect data, causing severe CPU cache-line bouncing and lock contention. This architecture funnels all parsed commands from the network Goroutines into a **single, massive buffered channel**.
- **Lock-Free State**: A dedicated single-threaded Engine Goroutine consumes commands from this channel sequentially. Because only one thread ever touches the data map, `sync.RWMutex` locks are completely eliminated. This keeps CPU L1/L2 caches blazing fast and enables millions of operations per second.

### Layer 3: The Memory Subsystem
- **Universal Object Wrapper**: All data (Strings, Hashes) is stored in a master dictionary utilizing a universal `Object` struct, preventing type confusion.
- **LRU Maxmemory Eviction**: Memory growth is strictly monitored. An O(1) doubly-linked list (`container/list`) acts as a Least Recently Used (LRU) tracker. When the configured `maxmemory` threshold is breached, the coldest keys at the back of the list are mathematically severed, protecting the host OS from Out-Of-Memory (OOM) crashes.
- **Passive & Active Expiration (TTL)**: 
  - *Passive*: Expired keys are deleted lazily if a client attempts to `GET` them.
  - *Active*: To prevent "Ghost Keys" from quietly eating RAM, a background Cron job uses Go's native randomized map iteration (a Monte Carlo probabilistic approach) to sample an isolated TTL index and sweep abandoned data continuously.

### Layer 4: Persistence & Durability
- **AOF Group Commit**: Writing directly to disk (`fsync`) pauses the CPU. Instead, mutations are pushed to a background I/O worker via a 100,000-command buffer. The worker flushes the Append-Only File (AOF) to the OS asynchronously (Appendfsync Everysec).
- **Point-in-Time Snapshotting**: Go cannot safely use the Linux `fork()` syscall to clone a multi-threaded runtime. To execute `BGREWRITEAOF` (Log Compaction), the Engine pauses for a fraction of a millisecond to perform a deep-copy clone of the memory map. The background worker serializes this clone into a compressed log, atomically replacing the old AOF without interrupting live traffic.

### Layer 5: Distributed Topology
- **Asynchronous Broadcasting**: All successful mutations in the Engine are immediately pushed to a Replication Broker channel. A dedicated background Goroutine fans out the byte-streams to all connected secondary Replicas.
- **Handshake Protocol (`SYNC`)**: When a Replica node boots and connects to the Master, it initiates a handshake. The Master instantly snapshots its state, executes a `FULLRESYNC` over the network socket, and seamlessly streams the entire database while queuing live incoming traffic for the Replica.

---

## Supported Data Structures & Commands

The Engine natively supports the exact wire protocol expected by `redis-cli`.

### Basic Operations
- `PING`: Tests connection liveness.
- `SET <key> <value> [EX <seconds>]`: Stores a string with an optional TTL expiration.
- `GET <key>`: Retrieves a string.
- `DEL <key>`: Erases a key from memory and all tracking subsystems.

### Complex Structures (Hashes)
- `HSET <key> <field> <value>`: Sets a field in a hash map.
- `HGET <key> <field>`: Retrieves a field from a hash map.

### Cluster & Maintenance
- `BGREWRITEAOF`: Triggers a Point-in-Time Snapshot and compresses the Append-Only File in the background.
- `SYNC`: Internal cluster command used by Replicas to request a full state transfer from the Master.

---

## Getting Started

### Prerequisites
- **Go**: Version 1.21 or higher.
- **Make**: Standard build automation tool.
- **redis-cli**: Standard Redis command-line interface for testing (optional but recommended).

### Compilation
Clone the repository and build the distributed binaries using the included Makefile.

```bash
make build
```
This will compile the source code and generate the executables `redis-server`, `redis-replica`, and `redis-sentinel` inside the local `bin/` directory.

---

## Deployment & Clustering

### 1. Booting the Master Node
Start the primary server on the default port. The Master node handles both Read and Write operations and manages the primary AOF file.
```bash
./bin/redis-server --port 6379
```

### 2. Booting a Replica Node
Start a secondary replica on a different port, pointing it to the Master node. The Replica will instantly execute a `SYNC`, download the Master's state, and prepare to serve Read-Only traffic.
```bash
./bin/redis-replica --port 6380 --replicaof 127.0.0.1:6379
```

### 3. Verifying the Cluster
Open a new terminal and use `redis-cli` to test the replication:

```bash
# Write data to the Master Node
redis-cli -p 6379 SET architecture "Lock-Free Actor Model"
> OK

# Read the synchronized data from the Replica Node
redis-cli -p 6380 GET architecture
> "Lock-Free Actor Model"
```

---

## Codebase Structure

The codebase is strictly modular, separated by architectural responsibilities:

- `cmd/`
  - `redis-server/main.go`: Bootstraps the Master Node, initializing the DB, AOF, Broker, and Network Acceptor.
  - `replica/main.go`: Bootstraps a Replica Node, handling the `SYNC` protocol and processing the Master's mutation stream.
- `internal/`
  - `server/tcp.go`: Layer 1 TCP Acceptor and Goroutine manager.
  - `server/engine.go`: Layer 2 Lock-Free Actor Model and command router.
  - `resp/parser.go`: The zero-copy wire protocol deserializer.
  - `storage/database.go`: Layer 3 Memory Subsystem (LRU, TTL, Map management).
  - `storage/aof.go`: Layer 4 Background Persistence and Group Commit worker.
  - `storage/rewrite.go`: Layer 4 Point-in-Time Snapshotting and AOF Compaction.
  - `replication/broker.go`: Layer 5 Asynchronous Fan-out Broadcaster for connected Replicas.

---

## Performance Characteristics

Because this system is built entirely lock-free on the execution side:
1. **Zero Context Switching**: The single Engine thread never yields to OS-level locks.
2. **Cache Locality**: Memory reads and writes stay highly localized in the CPU cache.
3. **Garbage Collection Immunity**: By recycling byte slices in the TCP layer and utilizing pointer-based memory management in the storage layer, Go's GC pressure is kept to an absolute minimum, ensuring sub-millisecond tail latencies even under maximum load.
