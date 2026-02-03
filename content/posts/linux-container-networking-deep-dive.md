---
title: "Linux Container Networking: From iptables to eBPF"
date: 2025-01-22
draft: true
description: "A deep dive into container networking: iptables, VXLAN overlays, Calico, Cilium, and eBPF. How packets actually flow in Kubernetes."
---

You deploy a pod in Kubernetes. It gets an IP address. It talks to other pods, services, and the internet.

Magic? No. Just a lot of clever engineering stacked on top of the Linux kernel.

This article explains how container networking actually works — from the ancient iptables to the modern eBPF revolution. By the end, you'll understand what Cilium, Calico, and VXLAN actually do.

## The Problem

Containers create a networking nightmare:

1. **Isolation**: Each container needs its own network namespace
2. **Connectivity**: Containers must talk to each other, across hosts
3. **Services**: Traffic must be load-balanced across pod replicas
4. **Policy**: Some pods shouldn't talk to other pods
5. **Scale**: Thousands of pods, hundreds of nodes, millions of connections

Traditional networking wasn't built for this. So we built new layers on top.

```
┌─────────────────────────────────────────────────────────────┐
│                     Your Application                        │
├─────────────────────────────────────────────────────────────┤
│                    Kubernetes Services                      │
├─────────────────────────────────────────────────────────────┤
│              CNI Plugin (Cilium / Calico)                   │
├─────────────────────────────────────────────────────────────┤
│           Overlay Network (VXLAN) or Direct Routing         │
├─────────────────────────────────────────────────────────────┤
│              iptables / nftables / eBPF                     │
├─────────────────────────────────────────────────────────────┤
│                    Linux Kernel                             │
└─────────────────────────────────────────────────────────────┘
```

Let's go bottom-up.

## Part 1: iptables and Netfilter

### The Foundation

Every packet that enters or leaves a Linux machine passes through **Netfilter** — a framework of hooks inside the kernel's networking stack.

**iptables** is the userspace tool to configure Netfilter rules. It's been the backbone of Linux firewalls since 1998.

```
Packet arrives
      │
      ▼
┌─────────────┐
│ PREROUTING  │ ← NAT, DNAT happens here
└──────┬──────┘
       │
       ▼
  Routing decision
       │
   ┌───┴───┐
   │       │
   ▼       ▼
┌──────┐ ┌─────────┐
│INPUT │ │ FORWARD │ ← For packets passing through
└──┬───┘ └────┬────┘
   │          │
   ▼          │
Local process │
   │          │
   ▼          │
┌──────┐      │
│OUTPUT│      │
└──┬───┘      │
   │          │
   ▼          ▼
┌─────────────────┐
│  POSTROUTING    │ ← SNAT, MASQUERADE
└────────┬────────┘
         │
         ▼
    Packet leaves
```

### The Five Hooks

Netfilter provides five points where you can intercept packets:

| Hook | When | Common Use |
|------|------|------------|
| PREROUTING | Right after packet arrives | DNAT (change destination) |
| INPUT | Packet destined for this host | Firewall filtering |
| FORWARD | Packet passing through | Router filtering |
| OUTPUT | Locally generated packet | Outbound filtering |
| POSTROUTING | Right before packet leaves | SNAT (change source) |

### Tables

Rules are organized into tables by purpose:

```bash
# Filter table (default) — accept/drop/reject
iptables -A INPUT -p tcp --dport 22 -j ACCEPT
iptables -A INPUT -j DROP

# NAT table — address translation
iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
iptables -t nat -A PREROUTING -p tcp --dport 80 -j DNAT --to 10.0.0.5:8080

# Mangle table — packet modification
iptables -t mangle -A PREROUTING -p tcp --dport 22 -j TOS --set-tos 0x10
```

### How Kubernetes Uses iptables

When you create a Kubernetes Service, kube-proxy creates iptables rules:

```bash
# Service ClusterIP: 10.96.0.10 → Pods: 10.244.1.5, 10.244.2.8

# PREROUTING: Intercept traffic to service IP
-A KUBE-SERVICES -d 10.96.0.10/32 -p tcp --dport 80 \
    -j KUBE-SVC-XXXX

# Load balance across pods (random probability)
-A KUBE-SVC-XXXX -m statistic --mode random --probability 0.5 \
    -j KUBE-SEP-AAAA
-A KUBE-SVC-XXXX \
    -j KUBE-SEP-BBBB

# DNAT to actual pod IPs
-A KUBE-SEP-AAAA -p tcp -j DNAT --to-destination 10.244.1.5:80
-A KUBE-SEP-BBBB -p tcp -j DNAT --to-destination 10.244.2.8:80
```

**The problem**: With thousands of services, you get tens of thousands of iptables rules. Each packet must traverse them sequentially. O(n) lookup time.

```
Services:    100    1,000    10,000
Rules:     1,500   15,000   150,000
Latency:    ~1ms    ~5ms     ~20ms
```

This is why iptables doesn't scale.

## Part 2: Overlay Networks and VXLAN

### The Multi-Host Problem

On a single machine, containers can talk via a bridge. But what about containers on different hosts?

```
Host A (10.0.0.1)              Host B (10.0.0.2)
┌─────────────────┐            ┌─────────────────┐
│ Pod 10.244.1.5  │            │ Pod 10.244.2.8  │
│       │         │            │       │         │
│    ┌──┴──┐      │            │    ┌──┴──┐      │
│    │bridge│     │            │    │bridge│     │
│    └──┬──┘      │            │    └──┬──┘      │
│       │         │            │       │         │
│    ┌──┴──┐      │            │    ┌──┴──┐      │
│    │eth0 │      │            │    │eth0 │      │
└────┴──┬──┴──────┘            └────┴──┬──┴──────┘
        │                              │
        └──────── Network ─────────────┘
```

Pod 10.244.1.5 wants to reach Pod 10.244.2.8. But the physical network only knows about 10.0.0.1 and 10.0.0.2. It has no idea what 10.244.x.x addresses are.

Two solutions:

1. **Overlay Network**: Encapsulate pod traffic inside host traffic
2. **Direct Routing**: Teach the network about pod IPs (BGP)

### VXLAN: Virtual Extensible LAN

VXLAN wraps Layer 2 frames inside Layer 3 UDP packets:

```
Original Packet (from pod):
┌─────────────────────────────────────────┐
│ Eth │ IP Header    │ TCP │   Payload   │
│ Hdr │ Src: 10.244.1.5                   │
│     │ Dst: 10.244.2.8                   │
└─────────────────────────────────────────┘

After VXLAN Encapsulation:
┌────────────────────────────────────────────────────────────────┐
│ Outer │ Outer IP    │ UDP  │ VXLAN │ Original Packet          │
│ Eth   │ Src: 10.0.0.1      │ 4789  │ VNI   │ (unchanged)       │
│       │ Dst: 10.0.0.2      │       │       │                   │
└────────────────────────────────────────────────────────────────┘
```

**VTEP** (VXLAN Tunnel Endpoint) handles encapsulation/decapsulation:

```
Pod A                VTEP A                    VTEP B              Pod B
10.244.1.5           10.0.0.1                  10.0.0.2            10.244.2.8
   │                    │                         │                   │
   │ packet to          │                         │                   │
   │ 10.244.2.8         │                         │                   │
   │───────────────────>│                         │                   │
   │                    │ encapsulate             │                   │
   │                    │ UDP to 10.0.0.2:4789    │                   │
   │                    │────────────────────────>│                   │
   │                    │                         │ decapsulate       │
   │                    │                         │──────────────────>│
   │                    │                         │                   │
```

### VXLAN Identifier (VNI)

VXLAN uses a 24-bit VNI (VXLAN Network Identifier):

- VLANs: 12 bits → 4,094 networks
- VXLAN: 24 bits → 16,777,216 networks

This scalability is crucial for multi-tenant clouds.

### The Overhead

VXLAN adds ~50 bytes of headers:

| Component | Bytes |
|-----------|-------|
| Outer Ethernet | 14 |
| Outer IP | 20 |
| UDP | 8 |
| VXLAN | 8 |
| **Total** | **50** |

If your MTU is 1500, inner packets must be ≤1450 bytes to avoid fragmentation.

```bash
# Configure MTU for VXLAN (example with Flannel)
# Pod MTU = Physical MTU - VXLAN overhead
# 1450 = 1500 - 50
```

### Creating VXLAN in Linux

```bash
# Create VXLAN interface
ip link add vxlan0 type vxlan \
    id 42 \                    # VNI
    dstport 4789 \             # Standard VXLAN port
    local 10.0.0.1 \           # This host's IP
    group 239.1.1.1 \          # Multicast group for discovery
    dev eth0

ip link set vxlan0 up
ip addr add 10.244.1.1/24 dev vxlan0

# Or with unicast (flood-and-learn)
bridge fdb append 00:00:00:00:00:00 dev vxlan0 dst 10.0.0.2
```

## Part 3: eBPF — The Revolution

### What is eBPF?

**eBPF** (extended Berkeley Packet Filter) lets you run sandboxed programs inside the Linux kernel — without changing kernel code or loading kernel modules.

```
┌─────────────────────────────────────────────────────────────┐
│                       User Space                            │
│  ┌─────────────────────────────────────────────────────┐   │
│  │            Your eBPF Program (C)                     │   │
│  │                      │                               │   │
│  │                      ▼                               │   │
│  │               LLVM Compiler                          │   │
│  │                      │                               │   │
│  │                      ▼                               │   │
│  │              eBPF Bytecode                           │   │
│  └──────────────────────┼───────────────────────────────┘   │
│                         │ bpf() syscall                     │
└─────────────────────────┼───────────────────────────────────┘
                          │
┌─────────────────────────┼───────────────────────────────────┐
│                         ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  eBPF Verifier                        │  │
│  │  • No infinite loops                                  │  │
│  │  • No out-of-bounds memory access                     │  │
│  │  • No dangerous kernel functions                      │  │
│  └──────────────────────┬───────────────────────────────┘  │
│                         │                                   │
│                         ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  JIT Compiler                         │  │
│  │          (Bytecode → Native Machine Code)            │  │
│  └──────────────────────┬───────────────────────────────┘  │
│                         │                                   │
│                         ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Attached to Kernel Hook                  │  │
│  │       (XDP, tc, kprobe, tracepoint, etc.)            │  │
│  └──────────────────────────────────────────────────────┘  │
│                       Kernel Space                          │
└─────────────────────────────────────────────────────────────┘
```

### Why eBPF is Powerful

1. **No context switches**: Code runs in kernel, no user/kernel boundary crossing
2. **Programmable**: Write custom logic, not just configure rules
3. **Safe**: Verifier ensures programs can't crash the kernel
4. **Fast**: JIT-compiled to native machine code
5. **Dynamic**: Load/unload without rebooting

### eBPF Networking Hooks

```
Packet arrives
      │
      ▼
┌─────────────┐
│    XDP      │ ← Earliest possible hook, before memory allocation
│  (eXpress   │   Can DROP, PASS, TX (redirect), or ABORTED
│ Data Path)  │   Runs at driver level, ~10M packets/sec possible
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  tc ingress │ ← After SKB (socket buffer) allocation
│   (Traffic  │   Full packet parsing available
│  Control)   │   Can modify, redirect, or drop
└──────┬──────┘
       │
       ▼
  Netfilter/iptables
       │
       ▼
  Routing decision
       │
       ▼
┌─────────────┐
│  tc egress  │ ← Before packet leaves
└──────┬──────┘
       │
       ▼
  Packet sent
```

### XDP: Maximum Performance

**XDP** (eXpress Data Path) runs before the kernel allocates memory for the packet. It's the fastest possible hook.

```c
// Simple XDP program: drop all UDP traffic
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/udp.h>

SEC("xdp")
int xdp_drop_udp(struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    // Parse Ethernet header
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return XDP_PASS;

    // Only handle IPv4
    if (eth->h_proto != htons(ETH_P_IP))
        return XDP_PASS;

    // Parse IP header
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return XDP_PASS;

    // Drop UDP packets
    if (ip->protocol == IPPROTO_UDP)
        return XDP_DROP;

    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
```

Load it:

```bash
# Compile
clang -O2 -target bpf -c xdp_drop_udp.c -o xdp_drop_udp.o

# Load onto interface
ip link set dev eth0 xdpgeneric obj xdp_drop_udp.o sec xdp
```

### eBPF Maps: State Between Calls

eBPF programs are stateless by default. **Maps** provide shared storage:

```c
// Define a hash map: IP address → packet count
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, __u32);        // IP address
    __type(value, __u64);      // Counter
} packet_count SEC(".maps");

SEC("xdp")
int count_packets(struct xdp_md *ctx) {
    // ... parse packet to get source IP ...

    __u32 src_ip = ip->saddr;
    __u64 *count = bpf_map_lookup_elem(&packet_count, &src_ip);

    if (count) {
        (*count)++;
    } else {
        __u64 init = 1;
        bpf_map_update_elem(&packet_count, &src_ip, &init, BPF_ANY);
    }

    return XDP_PASS;
}
```

Map types include:
- `BPF_MAP_TYPE_HASH` — Hash table
- `BPF_MAP_TYPE_ARRAY` — Array
- `BPF_MAP_TYPE_LRU_HASH` — LRU cache
- `BPF_MAP_TYPE_RINGBUF` — Ring buffer for events
- `BPF_MAP_TYPE_LPM_TRIE` — Longest prefix match (for routing)

### eBPF vs iptables Performance

iptables: O(n) rule traversal — linear scan through all rules

eBPF: O(1) map lookups — hash table lookup regardless of rule count

```
                   iptables         eBPF
Rules/Policies     10,000           10,000
Lookup time        ~20ms            ~0.1ms
CPU per packet     High             Low
```

Real-world impact: One practitioner reported **33% P99 latency reduction** moving from iptables-based CNI to eBPF.

## Part 4: Calico — BGP-Based Networking

### What is Calico?

**Calico** is a CNI (Container Network Interface) plugin that provides networking and security for containers. It takes a "Layer 3" approach — treating every pod as a first-class citizen with a routable IP.

### How Calico Works

Instead of overlays, Calico uses **BGP** (Border Gateway Protocol) to distribute pod routes:

```
┌─────────────────────────────────────────────────────────────┐
│                    Physical Network / Router                │
│                                                             │
│    "10.244.1.0/24 via 10.0.0.1"                            │
│    "10.244.2.0/24 via 10.0.0.2"                            │
└──────────────────────────┬──────────────────────────────────┘
                           │
          ┌────────────────┴────────────────┐
          │                                 │
┌─────────┴─────────┐             ┌─────────┴─────────┐
│    Host A         │             │    Host B         │
│    10.0.0.1       │             │    10.0.0.2       │
│                   │             │                   │
│ ┌───────────────┐ │             │ ┌───────────────┐ │
│ │ BIRD (BGP)    │ │◄───BGP────►│ │ BIRD (BGP)    │ │
│ │               │ │             │ │               │ │
│ │ Announces:    │ │             │ │ Announces:    │ │
│ │ 10.244.1.0/24 │ │             │ │ 10.244.2.0/24 │ │
│ └───────────────┘ │             │ └───────────────┘ │
│                   │             │                   │
│   Pod             │             │   Pod             │
│   10.244.1.5      │             │   10.244.2.8      │
└───────────────────┘             └───────────────────┘
```

Each node runs **BIRD**, a BGP daemon that:
1. Announces its pod CIDR to peers
2. Learns other nodes' pod CIDRs
3. Programs routes in the Linux routing table

```bash
# Routes on Host A
$ ip route
10.244.1.0/24 dev cali0 proto bird
10.244.2.0/24 via 10.0.0.2 dev eth0 proto bird  # ← Learned from Host B
```

### Calico Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Kubernetes API                         │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────┴──────────────────────────────────┐
│                      calico/kube-controllers                │
│          Syncs network policies from K8s to Calico          │
└──────────────────────────┬──────────────────────────────────┘
                           │
          ┌────────────────┼────────────────┐
          │                │                │
┌─────────┴────────┐ ┌─────┴──────┐ ┌───────┴──────┐
│   Felix          │ │   Felix    │ │    Felix     │
│   (per node)     │ │            │ │              │
│                  │ │            │ │              │
│ • Programs routes│ │            │ │              │
│ • Programs       │ │            │ │              │
│   iptables/eBPF  │ │            │ │              │
│ • Enforces       │ │            │ │              │
│   policies       │ │            │ │              │
└─────────┬────────┘ └─────┬──────┘ └───────┬──────┘
          │                │                │
┌─────────┴────────┐ ┌─────┴──────┐ ┌───────┴──────┐
│   BIRD (BGP)     │ │   BIRD     │ │    BIRD      │
│   Daemon         │ │            │ │              │
└──────────────────┘ └────────────┘ └──────────────┘
```

**Felix**: The agent on each node that:
- Watches for NetworkPolicy changes
- Programs iptables/eBPF rules
- Manages routes

**BIRD**: BGP daemon for route distribution

### Calico Network Policies

Calico extends Kubernetes NetworkPolicy:

```yaml
# Calico GlobalNetworkPolicy
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: deny-external-egress
spec:
  selector: app == 'backend'
  types:
  - Egress
  egress:
  # Allow traffic to other pods
  - action: Allow
    destination:
      selector: all()
  # Allow DNS
  - action: Allow
    protocol: UDP
    destination:
      ports:
      - 53
  # Deny everything else (implicit with Calico)
```

### Calico Data Planes

Calico supports multiple data planes:

| Data Plane | Pros | Cons |
|------------|------|------|
| **iptables** (default) | Mature, widely understood | O(n) scaling |
| **eBPF** | O(1) lookups, lower latency | Linux 5.3+, less mature |
| **Windows HNS** | Windows support | Windows only |
| **VPP** | Very high performance | Complex |

Enable eBPF mode:

```bash
# Switch to eBPF data plane
calicoctl patch felixconfiguration default \
    --patch='{"spec": {"bpfEnabled": true}}'
```

## Part 5: Cilium — eBPF-Native Networking

### What is Cilium?

**Cilium** is a CNI plugin built from the ground up on eBPF. It's the default CNI in Google GKE and Azure AKS.

Where Calico added eBPF as an option, Cilium was designed for eBPF from day one.

### Cilium Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes API                           │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────┴──────────────────────────────────┐
│                    Cilium Operator                          │
│        • Manages CiliumNode CRDs                            │
│        • IPAM allocation                                    │
│        • Garbage collection                                 │
└──────────────────────────┬──────────────────────────────────┘
                           │
          ┌────────────────┼────────────────┐
          │                │                │
┌─────────┴────────┐ ┌─────┴──────┐ ┌───────┴──────┐
│   Cilium Agent   │ │   Agent    │ │    Agent     │
│   (per node)     │ │            │ │              │
│                  │ │            │ │              │
│ • Loads eBPF     │ │            │ │              │
│   programs       │ │            │ │              │
│ • Manages eBPF   │ │            │ │              │
│   maps           │ │            │ │              │
│ • Identity       │ │            │ │              │
│   management     │ │            │ │              │
└─────────┬────────┘ └─────┬──────┘ └───────┬──────┘
          │                │                │
          ▼                ▼                ▼
┌─────────────────────────────────────────────────────────────┐
│                    eBPF Programs in Kernel                  │
│                                                             │
│   XDP ──► tc ingress ──► Routing ──► tc egress             │
│                                                             │
│   eBPF Maps:                                                │
│   • Endpoint map (pod identity → IP)                        │
│   • Policy map (identity → allowed identities)              │
│   • NAT map (service VIP → backend pods)                    │
│   • Connection tracking map                                 │
└─────────────────────────────────────────────────────────────┘
```

### Identity-Based Security

Cilium doesn't use IP addresses for security. Instead, it assigns **identities** to pods based on labels:

```
Pod Labels                    Cilium Identity
─────────────────────────────────────────────
app=frontend, env=prod   →    Identity 12345
app=backend, env=prod    →    Identity 12346
app=frontend, env=dev    →    Identity 12347
```

When a packet arrives, Cilium:
1. Looks up source identity in eBPF map
2. Looks up destination identity
3. Checks policy map: is (src_identity → dst_identity) allowed?

```
┌──────────────┐         ┌──────────────┐
│   frontend   │         │   backend    │
│  ID: 12345   │────────►│  ID: 12346   │
└──────────────┘         └──────────────┘
                   │
                   ▼
    Policy Map: {12345 → 12346: ALLOW}
```

Why is this better?
- IP addresses change constantly (pod restarts, scaling)
- Labels are stable and semantic
- Policy doesn't break when pods move

### L7 (Application Layer) Policies

Cilium can inspect HTTP, gRPC, Kafka, and more:

```yaml
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: api-access
spec:
  endpointSelector:
    matchLabels:
      app: backend
  ingress:
  - fromEndpoints:
    - matchLabels:
        app: frontend
    toPorts:
    - ports:
      - port: "80"
        protocol: TCP
      rules:
        http:
        - method: "GET"
          path: "/api/v1/users"
        - method: "POST"
          path: "/api/v1/orders"
```

This allows:
- `GET /api/v1/users` from frontend → backend ✓
- `POST /api/v1/orders` from frontend → backend ✓
- `DELETE /api/v1/users` from frontend → backend ✗

### Hubble: Observability

**Hubble** is Cilium's observability layer. It uses eBPF to provide real-time visibility:

```bash
# Watch all flows
hubble observe

# Filter by pod
hubble observe --pod frontend-xxx

# Filter by verdict
hubble observe --verdict DROPPED

# Export to Prometheus/Grafana
hubble metrics
```

```
TIMESTAMP             SOURCE                    DESTINATION              VERDICT
12:34:56.789          frontend/pod-abc          backend/pod-xyz          FORWARDED
12:34:56.790          frontend/pod-abc          database/pod-123         DROPPED (policy)
12:34:56.791          backend/pod-xyz           external/1.2.3.4         FORWARDED
```

### Cilium Service Mesh

Cilium can replace sidecar proxies (like Envoy in Istio):

```
Traditional Service Mesh:
┌─────────────────────────────────┐
│           Pod                   │
│  ┌───────────┐  ┌───────────┐  │
│  │   App     │──│  Envoy    │──│──► Network
│  │           │  │  Sidecar  │  │
│  └───────────┘  └───────────┘  │
└─────────────────────────────────┘
  Extra container, extra latency, extra memory

Cilium Service Mesh:
┌─────────────────────────────────┐
│           Pod                   │
│  ┌───────────┐                  │
│  │   App     │─────────────────│──► Network
│  └───────────┘                  │     │
└─────────────────────────────────┘     │
                                        │
                        eBPF handles L7 │
                        in kernel       │
                                        ▼
```

No sidecar needed. L7 features (mTLS, retries, timeouts) implemented in eBPF.

## Part 6: Comparison and When to Use What

### Feature Comparison

| Feature | iptables | Calico (iptables) | Calico (eBPF) | Cilium |
|---------|----------|-------------------|---------------|--------|
| Scalability | Poor | Moderate | Good | Excellent |
| L3/L4 Policy | Yes | Yes | Yes | Yes |
| L7 Policy | No | Limited | Limited | Yes |
| Observability | tcpdump | Basic | Better | Hubble |
| Windows | Yes | Yes | No | No |
| Kernel version | Any | Any | 5.3+ | 4.19+ |
| Learning curve | High | Moderate | Moderate | Higher |

### Performance Comparison

Benchmarks from Cilium's CNI performance tests:

```
Throughput (TCP, single stream):
─────────────────────────────────
Cilium eBPF:    43.5 Gbps
Calico eBPF:    40.2 Gbps
Calico iptables: 38.1 Gbps

Latency (P99, 1000 services):
─────────────────────────────────
Cilium:         0.8ms
Calico eBPF:    0.9ms
Calico iptables: 1.2ms
```

At small scale, the differences are negligible. At 10,000+ services, eBPF wins decisively.

### When to Choose Each

**Choose iptables (kube-proxy) when:**
- Simple cluster, <100 services
- Need maximum compatibility
- Team knows iptables well
- Don't need advanced policies

**Choose Calico when:**
- Need BGP integration with physical network
- Running on-premises with existing network infrastructure
- Need Windows node support
- Want flexibility (can switch data planes)
- Need GlobalNetworkPolicy across namespaces

**Choose Cilium when:**
- Running at scale (1000+ pods)
- Need L7 visibility and policy
- Want service mesh without sidecars
- Running on GKE, AKS (native support)
- Need advanced observability (Hubble)
- Building zero-trust network

### Migration Path

Most organizations follow this path:

```
iptables/kube-proxy
        │
        │ Scale issues, need policies
        ▼
    Calico (iptables)
        │
        │ Need better performance, observability
        ▼
    Calico (eBPF) or Cilium
        │
        │ Need L7 policies, service mesh
        ▼
      Cilium
```

## Quick Reference

### Commands Cheat Sheet

```bash
# iptables
iptables -L -n -v                    # List all rules
iptables -t nat -L -n -v             # List NAT rules
iptables-save > backup.rules         # Backup
iptables-restore < backup.rules      # Restore

# VXLAN
ip link show type vxlan              # Show VXLAN interfaces
bridge fdb show dev vxlan0           # Show forwarding database

# eBPF
bpftool prog list                    # List loaded programs
bpftool map list                     # List maps
bpftool net list                     # List network attachments

# Calico
calicoctl get nodes                  # List nodes
calicoctl get networkpolicies -A     # List policies
calicoctl node status                # Check BGP status

# Cilium
cilium status                        # Cluster status
cilium endpoint list                 # List endpoints
cilium policy get                    # Get policies
hubble observe                       # Watch traffic flows
```

### MTU Quick Reference

| Encapsulation | Overhead | Inner MTU (if outer=1500) |
|---------------|----------|---------------------------|
| None (native) | 0 | 1500 |
| VXLAN | 50 | 1450 |
| Geneve | 50 | 1450 |
| WireGuard | 60 | 1440 |
| IPsec (AES) | 73 | 1427 |

## Summary

Container networking is a stack of technologies:

1. **Netfilter/iptables**: The original Linux firewall. Still works, doesn't scale.

2. **VXLAN**: Overlay networks that tunnel L2 over L3. Solves multi-host networking.

3. **eBPF**: Programmable kernel. O(1) lookups. The foundation of modern CNIs.

4. **Calico**: BGP-based networking. Flexible data planes. Great for hybrid environments.

5. **Cilium**: eBPF-native. Identity-based security. L7 policies. Best observability.

Choose based on your scale, requirements, and team expertise. Start simple (Calico), evolve when needed (Cilium).

## References

- [eBPF Official Site](https://ebpf.io/)
- [Cilium Documentation](https://docs.cilium.io/)
- [Calico Documentation](https://docs.tigera.io/)
- [VXLAN RFC 7348](https://datatracker.ietf.org/doc/html/rfc7348)
- [Netfilter/iptables Deep Dive - DigitalOcean](https://www.digitalocean.com/community/tutorials/a-deep-dive-into-iptables-and-netfilter-architecture)
- [CNI Benchmark - Cilium](https://cilium.io/blog/2021/05/11/cni-benchmark/)
- [Calico vs Cilium - Tigera](https://www.tigera.io/learn/guides/cilium-vs-calico/)
