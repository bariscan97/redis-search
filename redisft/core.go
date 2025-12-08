package redisft

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisClient interface {
	Do(ctx context.Context, args ...interface{}) *redis.Cmd
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Pipeline() redis.Pipeliner
	Ping(ctx context.Context) *redis.StatusCmd
	Close() error
}

type ConnPool interface {
	Get() RedisClient
	Close() error
}

type Pool struct{ Client *redis.Client }

func NewPool(host string, maxConns int) *Pool {
	rdb := redis.NewClient(&redis.Options{
		Addr:        host,
		PoolSize:    maxConns,
		IdleTimeout: 5 * time.Minute,
		OnConnect: func(ctx context.Context, cn *redis.Conn) error {
			return cn.Ping(ctx).Err()
		},
	})
	return &Pool{Client: rdb}
}

func (p *Pool) Get() RedisClient { return p.Client }
func (p *Pool) Close() error     { return p.Client.Close() }

type Client struct {
	pool ConnPool
}

func NewClient(addr string, maxConns int) *Client {
	pool := NewPool(addr, maxConns)
	return &Client{pool: pool}
}

func (c *Client) Get() RedisClient { return c.pool.Get() }
func (c *Client) Close() error     { return c.pool.Close() }


type Repository[T any] struct {
	pool   ConnPool
	index  string
	prefix string

	qParts []string
	qSeen  map[string]struct{}
	sField string
	sAsc   bool
	sSet   bool
	off    int
	lim    int
	limSet bool
}


type Builder interface {
	GetFieldName() string
	Build() string
}

func NewRepo[T any](cli *Client) *Repository[T] {
	var z T
	t := reflect.TypeOf(z)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	name := strings.ToLower(t.Name())
	return &Repository[T]{
		pool:   cli,
		index:  "idx:" + name,
		prefix: name + ":",
		qSeen:  map[string]struct{}{},
	}
}

func (r *Repository[T]) CreateIndex(ctx context.Context) error {
	rc := r.pool.Get()
	args := generateIndexQuery(*new(T))
	_, err := rc.Do(ctx, append([]any{"FT.CREATE"}, args...)...).Result()
	if err != nil && !strings.Contains(err.Error(), "exists") {
		return err
	}
	return nil
}

func (r *Repository[T]) DropIndex(ctx context.Context, deleteDocs bool) error {
	rc := r.pool.Get()
	args := []any{"FT.DROPINDEX", r.index}
	if deleteDocs {
		args = append(args, "DD")
	}
	_, err := rc.Do(ctx, args...).Result()
	if err != nil && !strings.Contains(err.Error(), "Unknown Index name") {
		return err
	}
	return nil
}

func (r *Repository[T]) key(id string) string { return r.prefix + id }

func (r *Repository[T]) Insert(ctx context.Context, id string, doc *T) error {
	rc := r.pool.Get()
	m, err := structToMap(doc)
	if err != nil {
		return err
	}
	return rc.HSet(ctx, r.key(id), m).Err()
}

func (r *Repository[T]) InsertMany(ctx context.Context, docs map[string]*T) error {
	if len(docs) == 0 {
		return nil
	}
	rc := r.pool.Get()
	pipe := rc.Pipeline()
	for id, doc := range docs {
		m, err := structToMap(doc)
		if err != nil {
			pipe.Discard()
			return err
		}
		pipe.HSet(ctx, r.key(id), m)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (r *Repository[T]) Update(ctx context.Context, id string, patch T) error {
	rc := r.pool.Get()
	data, _ := structToMap(patch)
	return rc.HSet(ctx, r.key(id), data).Err()
}

func (r *Repository[T]) Delete(ctx context.Context, id string) error {
	rc := r.pool.Get()
	return rc.Del(ctx, r.key(id)).Err()
}

func (r *Repository[T]) Search(builders ...Builder) *Repository[T] {
	r.qParts = nil
	r.qSeen = map[string]struct{}{}
	r.sSet, r.limSet = false, false
	return r.Query(builders...)
}

func (r *Repository[T]) Query(builders ...Builder) *Repository[T] {
	for _, b := range builders {
		if _, dup := r.qSeen[b.GetFieldName()]; dup {
			continue
		}
		if p := b.Build(); p != "" {
			r.qParts = append(r.qParts, p)
			r.qSeen[b.GetFieldName()] = struct{}{}
		}
	}
	return r
}

func (r *Repository[T]) SortBy(field string, asc bool) *Repository[T] {
	r.sField, r.sAsc, r.sSet = field, asc, true
	return r
}

func (r *Repository[T]) Limit(off, cnt int) *Repository[T] {
	r.off, r.lim, r.limSet = off, cnt, true
	return r
}

func (r *Repository[T]) args() []any {
	q := "*"
	if len(r.qParts) > 0 {
		q = strings.Join(r.qParts, " ")
	}
	args := []any{r.index, q}
	if r.sSet {
		order := "ASC"
		if !r.sAsc {
			order = "DESC"
		}
		args = append(args, "SORTBY", r.sField, order)
	}
	if r.limSet {
		args = append(args, "LIMIT", r.off, r.lim)
	}
	return args
}


func (r *Repository[T]) Exec(ctx context.Context) ([]T, error) {
	rc := r.pool.Get()
	raw, err := rc.Do(ctx, append([]any{"FT.SEARCH"}, r.args()...)...).Result()
	if err != nil {
		return nil, err
	}

	rows, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}
	if len(rows) < 2 {
		return []T{}, nil
	}

	var out []T
	elemT := reflect.TypeOf(*new(T))

	for i := 1; i < len(rows); i += 2 {
		fa, _ := rows[i+1].([]interface{})
		m := map[string]any{}
		for j := 0; j < len(fa); j += 2 {
			key, _ := fa[j].(string)
			m[strings.ToLower(key)] = fa[j+1]
		}
		elem := reflect.New(elemT).Elem()
		if err := fillStruct(elem, m); err != nil {
			return nil, err
		}
		out = append(out, elem.Interface().(T))
	}
	return out, nil
}
