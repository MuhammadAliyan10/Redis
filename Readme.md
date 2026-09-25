# Go-Redis: A Distributed Database & Event Stream

A high-performance, fault-tolerant, fully concurrent Redis clone built entirely from scratch in Go.
This project implements the core architecture of modern distributed systems, including internal sharding, asynchronous replication, log compaction, and leader election.

## 🚀 Core Features

*   **Multithreaded Storage Engine:** Implements FNV-1a Hashing and Map Sharding (256 shards) with fine-grained `sync.RWMutex` locks, allowing massive read/write concurrency without CPU lock contention.
*   **Probabilistic Garbage Collection:** A background worker intelligently samples and evicts keys based on their Time-To-Live (`EX`), ensuring the memory footprint remains optimized.
*   **AOF Persistence & Log Compaction:** Commands are persisted to disk via an Append-Only File (AOF). Includes a `BGREWRITEAOF` engine that atomically squashes redundant logs into perfect snapshots.
*   **Master-Replica Replication:** Asynchronous TCP broadcasting allows secondary nodes to instantly mirror the Master's state.
*   **Sentinel Leader Election:** A standalone Heartbeat Monitor that acts as a Chaos-Monkey-proof failover judge. If the Master dies, Sentinel automatically promotes a Replica and reroutes cluster traffic.
*   **Distributed Message Queue:** Includes "Redis Streams" (`XADD`, `XREAD`)—an Append-Only Log architecture natively built into the engine for event sourcing.

## 🛠️ Supported Commands

*   `SET key value [EX seconds]`
*   `GET key`
*   `DEL key`
*   `XADD stream_name event_data`
*   `XREAD stream_name offset_id`
*   `BGREWRITEAOF` (Triggers Log Compaction)
*   `PING`

## 💻 How to Run the Distributed Cluster

**1. Start the Master Node:**
\`\`\`bash
go run cmd/redis-server/main.go --port 6379
\`\`\`

**2. Start a Replica Node:**
\`\`\`bash
go run cmd/redis-server/main.go --port 6380 --replicaof 127.0.0.1:6379
\`\`\`

**3. Start the Sentinel (Auto-Failover Monitor):**
\`\`\`bash
go run cmd/sentinel/main.go
\`\`\`
