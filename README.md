# redisft

**A lightweight, idiomatic Go helper layer on top of [RediSearch](https://oss.redis.com/redisearch/).**

- Minimal **connection‑pool** wrappers (single‑ or multi‑host)
- Generic **Repository[T]** with CRUD + chainable query builder
- Composable builders for **TEXT**, **NUMERIC**, **GEO**, and **TAG** fields
- Pure Go — no CGO, no reflection in hot paths

> Works with **Redis / RediSearch ≥ 2.0** and **Go 1.22+**.

---

## Installation

```bash
go get github.com/bariscan97/redis-ftsearch
```

> The module relies internally on **github.com/go-redis/redis/v8** (pulled automatically by `go get`).

---

## Quick Start

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/your‑module/redisft"
)

type Product struct {
    ID        string    `redis:"text sortable"`
    Name      string    `redis:"text"`
    Price     float64   `redis:"numeric sortable"`
    CreatedAt time.Time `redis:"numeric"`
    Location  string    `redis:"geo"`     // "lon,lat"
    Color     string    `redis:"tag"`     // product color
}

func main() {
    ctx := context.Background()

    // 🔌 connect (500 pooled conns)
    cli  := redisft.NewClient("localhost:6379", "demo", 500)
    defer cli.Close()

    // 💾 repository
    repo := redisft.NewRepo[Product](cli)

    // 🔧 index (idempotent)
    if err := repo.CreateIndex(ctx); err != nil { log.Fatal(err) }

    // ➕ insert
    _ = repo.Insert(ctx, "1", &Product{
        ID: "1", Name: "Book", Price: 19.9, CreatedAt: time.Now(),
    })

    // 🔍 query (price ASC, first 10)
    items, _ := repo.Search().SortBy("price", true).Limit(0, 10).Exec(ctx)
    log.Printf("%+v", items)
}
```

---

## Connection Pooling

### Single host

```go
pool := redisft.NewSingleHostPool("localhost:6379", 300)
```

### Multi‑host (round‑robin)

```go
pool := redisft.NewMultiHostPool([]string{
    "cache‑a:6379", "cache‑b:6379", "cache‑c:6379"}, 300)
```

`redisft.NewClient` selects the correct pool automatically from a comma‑separated list:

```go
cli := redisft.NewClient("cache‑a:6379,cache‑b:6379", "myprefix", 300)
```

---

## Builders in Action

### TEXT

```go
// Complex example using Group(), NOT and OR
nameQ := redisft.NewTextQuery("name").
            Group(func(q *redisft.QB) {         // (war* | *craft)
                q.Prefix("war").Or().Suffix("craft")
            }).
            And().Not().Exact("demo").          // -"demo"
            And().Any("guide", "tutorial")      // guide | tutorial

repo.Search(nameQ).Exec(ctx)
// → @name:((war* | *craft) -"demo" guide | tutorial)
```

### NUMERIC

```go
price := redisft.NewNumericQuery("price").
           Between(50, 120).
           OrRange(200, 300, true, false) // (200 300]

// @price:[50 120] | @price:(200 300]
```

### GEO

```go
geo := redisft.NewGeoQuery("location").
          Center(29.0, 41.0).
          Km(10)
// @location:[29.000000 41.000000 10.0000 km]
```

### TAG

```go
color := redisft.NewTagQB("color").
            Any("red", "blue").
            And().Not().In("green")
// @color:{red|blue} -@color:{green}
```

### Combining

```go
products, _ := repo.Search(price, geo, color).
                      SortBy("price", true).
                      Limit(0, 20).
                      Exec(ctx)
```

---

## Index Schema via Struct Tags

| Tag value              | RediSearch type |
|------------------------|-----------------|
| `text`, `text sortable`| TEXT            |
| `numeric`, `numeric sortable` | NUMERIC |
| `tag`                  | TAG             |
| `geo`                  | GEO             |

`generateIndexQuery` inspects the tags once (at startup) to build `FT.CREATE`.






