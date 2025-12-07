package redisft

import (
	"fmt"
	"strings"
)

func NewTextQuery(field string) *QB { return &QB{field: field} }

type QB struct {
	field string
	sb    strings.Builder
	stack []int // parenthesis depth

	nextOR  bool // join with "|"
	nextNOT bool // prefix with "-"
	nextMND bool // prefix with "+"
}

// And – no explicit token; RediSearch treats a space as AND.
// We simply clear the OR flag.
func (q *QB) And() *QB { q.nextOR = false; return q }

// Or – the next token will be joined with '|'.
func (q *QB) Or() *QB { q.nextOR = true; return q }

// Not – the next token will be prefixed with '-'.
func (q *QB) Not() *QB { q.nextNOT = true; return q }

// Must – the next token will be prefixed with '+' (mandatory).
func (q *QB) Must(term string) *QB { q.nextMND = true; return q.Term(term) }

// Term adds a raw term (already escaped if needed).
func (q *QB) Term(term string) *QB { return q.raw(escape(term)) }

// Prefix adds a “foo*” style token.
func (q *QB) Prefix(p string) *QB { return q.raw(escape(p) + "*") }

// Suffix adds a “*foo” style token.
func (q *QB) Suffix(s string) *QB { return q.raw("*" + escape(s)) }

// Wild adds a fully‑wild token (“*foo*” etc.).
func (q *QB) Wild(w string) *QB { return q.raw(escape(w)) }

// Exact adds a quoted exact‑match phrase.
func (q *QB) Exact(phrase string) *QB { return q.raw("\"" + escape(phrase) + "\"") }

// Any – add all terms joined with OR.
func (q *QB) Any(ts ...string) *QB {
	for i, t := range ts {
		if i > 0 {
			q.Or()
		}
		q.Term(t)
	}
	return q
}

// All – add all terms with implicit AND (space).
func (q *QB) All(ts ...string) *QB {
	for _, t := range ts {
		q.Term(t)
	}
	return q
}

// Group creates “( … )” by executing the callback inside.
func (q *QB) Group(fn func(*QB)) *QB {
	q.Open()
	fn(q)
	return q.Close()
}

// Open writes '('.
func (q *QB) Open() *QB {
	q.flushOp()
	q.sb.WriteByte('(')
	q.stack = append(q.stack, 1)
	return q
}

// Close writes ')'.
func (q *QB) Close() *QB {
	if len(q.stack) == 0 {
		return q
	}
	q.sb.WriteByte(')')
	q.stack = q.stack[:len(q.stack)-1]
	return q
}

func (q *QB) Build() string {
	for len(q.stack) > 0 {
		q.sb.WriteByte(')')
		q.stack = q.stack[:len(q.stack)-1]
	}
	expr := strings.TrimSpace(q.sb.String())
	if expr == "" {
		return ""
	}
	return fmt.Sprintf("@%s:(%s)", q.field, expr)
}

func (q *QB) raw(token string) *QB {
	q.flushOp()
	if q.nextNOT {
		q.sb.WriteByte('-')
	}
	if q.nextMND {
		q.sb.WriteByte('+')
	}
	q.sb.WriteString(token)
	q.nextNOT, q.nextMND = false, false
	return q
}

func (q *QB) flushOp() {
	if q.sb.Len() == 0 {
		return
	}
	s := q.sb.String()
	if last := s[len(s)-1]; last == '(' {
		// no op
	} else if q.nextOR {
		q.sb.WriteByte('|')
	} else {
		q.sb.WriteByte(' ')
	}
	q.nextOR = false
}

func (q *QB) GetFieldName() string { return q.field }

// Fuzzy adds a fuzzy search term.
// distance 1 => %term%
// distance 2 => %%term%%
// otherwise => wraps in %..% (defaulting to 1 if invalid, or just clamping)
func (q *QB) Fuzzy(term string, distance int) *QB {
	t := escape(term)
	if distance > 2 {
		distance = 2
	}
	if distance < 1 {
		distance = 1
	}
	if distance == 1 {
		return q.raw("%" + t + "%")
	}
	return q.raw("%%" + t + "%%")
}

func escape(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if isSpecial(r) {
			sb.WriteByte('\\')
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

func isSpecial(r rune) bool {
	switch r {
	case ',', '.', '<', '>', '{', '}', '[', ']', '"', '\'', ':', ';', '!', '@', '#', '$', '%', '^', '&', '*', '(', ')', '-', '+', '=', '~', '\\', '|', '/':
		return true
	}
	return false
}
