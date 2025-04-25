package redisft

import (
	"strings"
)

func escapeTag(t string) string {
	return strings.NewReplacer(
		` `, `\ `,
		`,`, `\,`,
		`{`, `\{`,
		`}`, `\}`,
		`|`, `\|`,
		`\\`, `\\\\`,
	).Replace(t)
}

// joinTags builds the `{tag1|tag2}` payload, adding a leading `+` if mandatory.
func joinTags(tags []string, mandatory bool) string {
	var b strings.Builder
	for i, t := range tags {
		if i > 0 {
			b.WriteByte('|')
		}
		if mandatory {
			b.WriteByte('+')
		}
		b.WriteString(escapeTag(t))
	}
	return b.String()
}

// TagQB builds queries for TAG fields, e.g.  @field:{tag1|tag2}
type TagQB struct {
	field string

	parts []string
	group strings.Builder

	flagOr   bool
	flagNot  bool
	flagMust bool
}

// NewTagQB creates a new builder for the given TAG field.
func NewTagQB(field string) *TagQB { return &TagQB{field: field} }

// GetFieldName returns the field name (for reflection / generic use).
func (q *TagQB) GetFieldName() string { return q.field }

// Or sets the next call to be joined with '|' inside the same block.
func (q *TagQB) Or() *TagQB { q.flagOr = true; return q }

// And flushes the current block so the next one starts a new AND section (space separated).
func (q *TagQB) And() *TagQB { q.flush(); return q }

// Not prefixes the next block with '-'.
func (q *TagQB) Not() *TagQB { q.flagNot = true; return q }

// Must prefixes the next tags with '+' (mandatory).
func (q *TagQB) Must() *TagQB { q.flagMust = true; return q }

/* ---------- TAG adders ---------- */

// Any OR‑joins the given tags in a single block: {tag1|tag2}
func (q *TagQB) Any(tags ...string) *TagQB { return q.addTags(tags, false) }

// In is an alias of Any.
func (q *TagQB) In(tags ...string) *TagQB { return q.Any(tags...) }

// Require adds mandatory tags (+tag) in OR form.
func (q *TagQB) Require(tags ...string) *TagQB { return q.addTags(tags, true) }

// NotIn adds negative tags (‑@field:{tag}).
func (q *TagQB) NotIn(tags ...string) *TagQB { q.flagNot = true; return q.Any(tags...) }

// All adds tags as separate AND blocks (+tag AND +tag …).
func (q *TagQB) All(tags ...string) *TagQB {
	for i, t := range tags {
		if i > 0 {
			q.And()
		}
		q.Require(t)
	}
	return q
}

// Group creates a parenthesised subquery.
func (q *TagQB) Group(fn func(*TagQB)) *TagQB {
	q.openParen()
	fn(q)
	q.closeParen()
	return q
}

func (q *TagQB) Build() string {
	q.flush()
	if len(q.parts) == 0 {
		return ""
	}
	return strings.Join(q.parts, " ")
}

// MustBuild panics on Build error (convenience).
// func (q *TagQB) MustBuild() string {
// 	s, err := q.Build()
// 	if err != nil {
// 		panic(err)
// 	}
// 	return s
// }

// addTags adds tags to the current block, applying the active flags.
func (q *TagQB) addTags(tags []string, mandatory bool) *TagQB {
	if len(tags) == 0 {
		return q
	}
	q.ensureBlockStarted()

	if q.flagOr && q.group.Len() > 0 {
		q.group.WriteByte('|')
	}

	if q.flagNot {
		q.group.WriteByte('-')
	}
	q.group.WriteByte('@')
	q.group.WriteString(q.field)
	q.group.WriteString(":{")
	q.group.WriteString(joinTags(tags, mandatory || q.flagMust))
	q.group.WriteByte('}')

	q.flagOr, q.flagNot, q.flagMust = false, false, false
	return q
}

// ensureBlockStarted is a no‑op for now but kept for symmetry.
func (q *TagQB) ensureBlockStarted() {}

// flush moves the current block to parts.
func (q *TagQB) flush() {
	if q.group.Len() == 0 {
		return
	}
	q.parts = append(q.parts, q.group.String())
	q.group.Reset()
}

// openParen writes '(' to the current block.
func (q *TagQB) openParen() { q.group.WriteByte('(') }

// closeParen writes ')' to the current block.
func (q *TagQB) closeParen() { q.group.WriteByte(')') }
