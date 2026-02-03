---
title: "Go HTTP Clients: net/http vs The Fast Ones"
date: 2025-10-08
draft: false
description: "I benchmarked 4 HTTP clients. FastHTTP is 2x faster. Here's why you probably shouldn't use it."
---

Built a service that makes 10,000 API calls per second. Used `net/http`. Worked fine for 6 months.

Then someone mentioned fasthttp claims to be "10x faster". Decided to benchmark it.

Spoiler: it's 2x faster. Not 10x. And switching isn't free.

## The Clients

Four options in Go:

**[net/http](https://pkg.go.dev/net/http)** - Standard library. Everyone uses it.

**[resty](https://github.com/go-resty/resty)** - Wrapper around net/http with a nicer API. 76k stars on GitHub.

**[req](https://github.com/imroc/req)** - Another wrapper. More features. Slightly slower.

**[fasthttp](https://github.com/valyala/fasthttp)** - Not compatible with net/http. Reuses everything. Claims 10x speed.

## The Benchmarks

**Test environment:** Apple M1, Go 1.25.1, local HTTP server

### GET Requests

```
Client      ns/op    B/op    allocs   vs net/http
net/http    38,126   6,241   68       1.0x
resty       41,738   8,257   81       0.9x (slower)
req         45,950   13,501  113      0.8x (slower)
fasthttp    22,799   64      1        1.7x faster
```

FastHTTP is 1.7x faster. Not 10x.

Resty is actually slower than net/http. Convenience has a cost.

### POST Requests (JSON body)

```
Client      ns/op    B/op    allocs   vs net/http
net/http    40,100   7,043   74       1.0x
resty       45,939   10,329  106      0.9x (slower)
req         51,000   16,443  142      0.8x (slower)
fasthttp    21,188   0       0        1.9x faster
```

FastHTTP makes zero allocations on POST. That's the magic.

Net/http allocates 7KB per request. FastHTTP reuses everything.

### Connection Pooling

```
Client      ns/op    B/op    allocs   vs net/http
net/http    38,621   6,262   68       1.0x
resty       42,253   8,647   81       0.9x (slower)
fasthttp    19,417   0       0        2.0x faster
```

With connection pooling, fasthttp is exactly 2x faster.

Still not 10x. Those claims are from synthetic benchmarks where the server does nothing.

### With Headers (Auth, User-Agent, Accept)

```
Client      ns/op    B/op    allocs   vs net/http
net/http    40,171   7,159   77       1.0x
resty       43,736   8,927   86       0.9x (slower)
fasthttp    19,531   0       0        2.1x faster
```

More headers = more allocations in net/http. FastHTTP doesn't care.

## Why FastHTTP Is Faster

**Object pooling:**
```go
// net/http creates new request every time
req, _ := http.NewRequest("GET", url, nil)

// fasthttp reuses from pool
req := fasthttp.AcquireRequest()
defer fasthttp.ReleaseRequest(req)
```

**Zero-copy reads:**
Net/http copies response body to a buffer. FastHTTP gives you a slice pointing to the read buffer.

```go
// net/http
body, _ := io.ReadAll(resp.Body)

// fasthttp (points to internal buffer)
body := resp.Body()
```

**No interface{} in hot paths:**
Net/http uses `io.Reader`/`io.Writer`. FastHTTP uses concrete types. No allocations.

**Goroutine pooling:**
Net/http spawns a goroutine per connection. FastHTTP reuses worker goroutines.

## The Cost

FastHTTP isn't compatible with net/http. You can't just swap imports.

**net/http:**
```go
client := &http.Client{}
resp, err := client.Get("https://api.example.com")
if err != nil {
    return err
}
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)
```

**fasthttp:**
```go
req := fasthttp.AcquireRequest()
resp := fasthttp.AcquireResponse()
defer fasthttp.ReleaseRequest(req)
defer fasthttp.ReleaseResponse(resp)

req.SetRequestURI("https://api.example.com")
err := fasthttp.Do(req, resp)
if err != nil {
    return err
}
body := resp.Body() // Don't copy, points to buffer
```

Different API. Different semantics. Can't use with most libraries that expect `http.Client`.

No middleware. No standard library integrations. You're on your own.

## When To Use What

**Use net/http if:**
- You're making < 1000 requests/sec
- You value ecosystem compatibility
- You need standard middleware (auth, retry, logging)
- Your bottleneck is network latency, not client overhead

**Use resty if:**
- You want a nicer API
- You need features (retry, auth, debugging)
- You don't mind 10-20% slowdown
- You're still using net/http under the hood

**Use fasthttp if:**
- You're making > 10,000 requests/sec
- Client overhead shows up in profiles
- You can handle incompatibility with net/http ecosystem
- You've actually measured that net/http is your bottleneck

## Real Talk

I didn't switch. Here's why:

Our API calls spend 50ms waiting for network. Client overhead is 0.04ms (net/http) vs 0.02ms (fasthttp).

Saving 0.02ms doesn't matter when network adds 50ms.

FastHTTP is 2x faster at something that takes 0.1% of request time. That's not a 2x speedup for the system.

Plus, we use middleware that expects `http.Client`. Switching would break everything.

## The Exception

We have one service that scrapes 100,000 pages/hour from a single fast API. Network latency is <1ms. Client overhead matters there.

Switched to fasthttp. Went from 1,200 req/s to 2,000 req/s. Worth it.

For everything else? Net/http is fine.

## Code Examples

**net/http (simple):**
```go
client := &http.Client{
    Timeout: 10 * time.Second,
}
resp, err := client.Get("https://api.example.com/data")
if err != nil {
    return err
}
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)
```

**resty (convenient):**
```go
client := resty.New()
resp, err := client.R().
    SetHeader("Authorization", "Bearer token").
    SetResult(&result).
    Get("https://api.example.com/data")
```

**fasthttp (fast, manual):**
```go
req := fasthttp.AcquireRequest()
resp := fasthttp.AcquireResponse()
defer fasthttp.ReleaseRequest(req)
defer fasthttp.ReleaseResponse(resp)

req.SetRequestURI("https://api.example.com/data")
req.Header.Set("Authorization", "Bearer token")

err := fasthttp.Do(req, resp)
if err != nil {
    return err
}

body := resp.Body()
// WARNING: body becomes invalid after ReleaseResponse
// Copy it if you need it later
```

## The Numbers Don't Lie

FastHTTP is 2x faster. That's real.

But 2x faster at the client doesn't mean 2x faster requests. Network latency dominates.

Profile first. Optimize later. Use net/http until you can't.

## Run The Benchmarks

```bash
cd benchmarks/http-benchmark
./run.sh
```

Your numbers will be different. The ratios won't change much.
