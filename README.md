# Go-Redis-Lite 

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)

A high-performance, persistent, and concurrent Key-Value store written in Go.
This project is a clean-room implementation of a Redis-compatible server, supporting the **RESP (Redis Serialization Protocol)**, **AOF Persistence**, and **Docker containerization**.

It handles **~123,000 requests/second** (GET) on commodity hardware.

---

##  Benchmarks

Running on Arch Linux (Ryzen 5 / RTX 3060), tested with `redis-benchmark -n 100000`:

| Operation | Requests per Second | Latency (p50) |
| :--- | :--- | :--- |
| **GET** | **123,762.38** | **0.223 ms** |
| **SET** | **63,331.22** | **1.023 ms** |

*Achieved using `sync.RWMutex` for granular locking, allowing massive read concurrency.*

---

## Features

* **TCP Listener:** Custom raw socket handling (no HTTP overhead).
* **RESP Parser:** Fully compatible with `redis-cli` and standard Redis clients.
* **Concurrency:** Handles thousands of concurrent connections using Goroutines and `sync.RWMutex`.
* **Persistence (AOF):** Append-Only File durability. Data survives server restarts.
* **Expiration (TTL):** Active "Janitor" goroutine cleans up expired keys.
* **Containerized:** Multi-stage Docker build resulting in a <15MB image.

---

##  Getting Started

### Option A: Run with Docker (Recommended)
You don't need Go installed. Just run the container.

```bash
# 1. Build the image
docker build -t go-redis .

# 2. Run the container (detached)
docker run -d -p 6379:6379 --name my-redis go-redis

Connect using redis-cli or netcat
