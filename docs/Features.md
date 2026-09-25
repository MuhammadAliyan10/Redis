# Architectural Features & Subsystems

This document provides a highly detailed breakdown of the distributed architecture. Every feature implemented in this project exists to solve a specific distributed systems or high-performance computing problem.

---

### 1. Zero-Copy RESP Protocol Parser
- **File Implementation:** `internal/resp/parser.go`
- **The Problem:** Traditional parsers read network streams into standard strings. This heavily fragments heap memory and forces the Garbage Collector (GC) to constantly pause the runtime to clean up, destroying throughput.
- **The Implementation:** The parser relies entirely on raw byte slices (`[][]byte`) and `bytes.Index`. It scans the persistent TCP buffer and returns index pointers referencing the original memory allocation.
- **Why it is the Best Solution:** It completely bypasses Go's string allocation overhead. By passing memory pointers instead of copying data, heap memory remains flat and GC latency spikes are effectively eliminated.

### 2. Lock-Free Actor Model Engine
- **File Implementation:** `internal/server/engine.go`
- **The Problem:** Multi-threaded databases experience severe CPU cache-line bouncing and lock contention (`sync.RWMutex`) when thousands of threads attempt to read and write data simultaneously.
- **The Implementation:** All validated network commands are pushed into a master buffered channel (`commandCh`). A single, isolated Engine Goroutine executes commands sequentially from this channel.
- **Why it is the Best Solution:** By eliminating concurrent writes at the application level, the Engine can process millions of commands per second entirely out of the CPU's L1/L2 cache, without ever yielding to the OS scheduler for a lock.

### 3. Universal Object Wrapper
- **File Implementation:** `internal/storage/database.go`
- **The Problem:** A strongly-typed language like Go cannot natively store Strings, Hashes, and Sets in the same primitive `map` without complex, slow reflection overhead.
- **The Implementation:** An `Object` struct is defined containing an `ObjectType` enum and a generic `Ptr` (`any`) pointing to the underlying data structure, alongside metadata like `ExpiresAt` and `Size`.
- **Why it is the Best Solution:** It provides absolute O(1) type safety via the enum while allowing a single unified dictionary map to natively house, track, and execute highly complex, heterogeneous data structures.

### 4. LRU Maxmemory Eviction
- **File Implementation:** `internal/storage/database.go`
- **The Problem:** In-memory databases will eventually consume all available RAM, triggering the Linux OOM (Out Of Memory) Killer to violently crash the process.
- **The Implementation:** A Doubly Linked List (`container/list`) acts as a strict timeline. Accessing a key instantly moves its node to the front. When memory exceeds the `maxmemory` limit, `enforceMaxMemory()` mathematically deletes the nodes at the extreme tail.
- **Why it is the Best Solution:** Doubly linked lists provide absolute O(1) time complexity for both updating a key's usage status and evicting the oldest key, guaranteeing zero latency overhead regardless of the database's scale.

### 5. Probabilistic Active Expiration (Cron)
- **File Implementation:** `internal/storage/database.go`
- **The Problem:** Passive expiration (lazy TTL) only deletes keys if the client explicitly asks for them. "Ghost keys" (keys that expire but are never accessed again) will silently leak memory indefinitely.
- **The Implementation:** `ActiveExpireCron()` runs continuously, sampling keys from a dedicated `ttlMap`. It leverages Go's native randomized map iteration to instantly grab 20 random expiring keys.
- **Why it is the Best Solution:** Iterating over millions of keys would freeze the lock-free Engine. A probabilistic Monte Carlo approach instantly clears backlogs by recursively executing if >25% of the random sample is expired, ensuring memory is reclaimed aggressively without locking the CPU.

### 6. AOF Group Commit Persistence
- **File Implementation:** `internal/storage/aof.go`
- **The Problem:** Executing an `fsync` directly to a spinning disk or SSD takes milliseconds. Doing this inside the main Engine thread would instantly cripple the database's throughput.
- **The Implementation:** The Engine pushes the wire-protocol bytes into an enormous non-blocking buffered channel (`writeCh`). A separate background I/O Goroutine batches these commands and flushes them to the OS every second.
- **Why it is the Best Solution:** It strictly decouples hardware disk latency from memory throughput. This provides robust durability (only losing up to 1 second of data in a critical power failure) while maintaining maximum in-memory speed.

### 7. Point-in-Time Snapshotting (BGREWRITEAOF)
- **File Implementation:** `internal/storage/rewrite.go`
- **The Problem:** The AOF file grows indefinitely and must be compacted. However, Go's multi-threaded runtime cannot safely use the `fork()` syscall to clone the process memory in the background (which is how C Redis achieves this).
- **The Implementation:** The Engine pauses for a fraction of a millisecond to execute `Snapshot()`, performing a rapid deep-copy of the active `dict`. This clone is passed to a background Goroutine that serializes it into a dense, compressed log.
- **Why it is the Best Solution:** It guarantees absolute point-in-time consistency and prevents race conditions by deep-copying volatile pointers, allowing the heavy disk serialization to happen fully asynchronously without corrupting live data.

### 8. Asynchronous Replication Fan-Out
- **File Implementation:** `internal/replication/broker.go`
- **The Problem:** A Master node must propagate writes to Replicas. If a Replica experiences a network lag, blocking the Master to wait for the network packet to arrive would destroy global cluster throughput.
- **The Implementation:** The Engine pushes successful mutations to the `streamCh` inside the Broker. The Broker uses a dedicated background Goroutine to iterate over all connected Replicas and stream the bytes entirely independently of the Master Engine.
- **Why it is the Best Solution:** It perfectly isolates the Lock-Free Engine from the unpredictability of Wide-Area Networks (WAN). The Master never waits for a Replica to acknowledge a write.

### 9. Handshake Protocol (`SYNC`)
- **File Implementation:** `cmd/replica/main.go` & `internal/server/engine.go`
- **The Problem:** Replicas boot up completely empty. They require a robust mechanism to safely acquire the Master's exact state before joining the live replication stream.
- **The Implementation:** The Replica issues a `SYNC` command. The Master instantly takes a `Snapshot()`, registers the Replica with the `Broker` to queue future commands, and uses a background Goroutine to serialize and stream the Snapshot (`FULLRESYNC`) directly to the Replica.
- **Why it is the Best Solution:** It achieves seamless, zero-downtime clustering. By queueing future commands in the Broker *before* the massive snapshot transfer begins, we guarantee absolutely zero data loss during the synchronization window.

### 10. TCP Pipelining & Multiplexing
- **File Implementation:** `internal/server/tcp.go`
- **The Problem:** Creating a new TCP connection for every single command introduces massive networking overhead and latency.
- **The Implementation:** The TCP handler reads the incoming socket continuously into a 4096-byte buffer. The inner loop mathematically processes `processIndex` against `readIndex`, allowing it to unpack and route hundreds of pipelined commands from a single network packet simultaneously.
- **Why it is the Best Solution:** It drastically reduces the number of expensive `read()` OS syscalls and maximizes network bandwidth utilization, which is critical for high-throughput distributed microservice architectures.
