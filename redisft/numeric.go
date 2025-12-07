package redisft

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type bound struct {
	val       float64
	exclusive bool
}

type interval struct {
	lo, hi bound
}

func (iv interval) toString(field string) string {
	lc := '['
	if iv.lo.exclusive {
		lc = '('
	}
	rc := ']'
	if iv.hi.exclusive {
		rc = ')'
	}

	loStr := fmt.Sprintf("%g", iv.lo.val)
	if math.IsInf(iv.lo.val, -1) {
		loStr = "-inf"
	}

	hiStr := fmt.Sprintf("%g", iv.hi.val)
	if math.IsInf(iv.hi.val, 1) {
		hiStr = "+inf"
	}

	return fmt.Sprintf("@%s:%c%s %s%c", field, lc, loStr, hiStr, rc)
}

type NumericQuery struct {
	field     string
	intervals []interval
}

func NewNumericQuery(field string) *NumericQuery { return &NumericQuery{field: field} }
func (n *NumericQuery) GetFieldName() string     { return n.field }

// Gt  ⇒ (v  +inf]
func (n *NumericQuery) Gt(v float64) *NumericQuery { return n.add(bound{v, true}, bound{inf, false}) }

// Ge  ⇒ [v  +inf]
func (n *NumericQuery) Ge(v float64) *NumericQuery { return n.add(bound{v, false}, bound{inf, false}) }

// Lt  ⇒ [-inf  v)
func (n *NumericQuery) Lt(v float64) *NumericQuery {
	return n.add(bound{negInf, false}, bound{v, true})
}

// Le  ⇒ [-inf  v]
func (n *NumericQuery) Le(v float64) *NumericQuery {
	return n.add(bound{negInf, false}, bound{v, false})
}

// Range – custom bounds; inclusivity defined by flags
func (n *NumericQuery) Range(lo, hi float64, inclLo, inclHi bool) *NumericQuery {
	return n.add(bound{lo, !inclLo}, bound{hi, !inclHi})
}

// Between – shortcut, both ends inclusive
func (n *NumericQuery) Between(lo, hi float64) *NumericQuery {
	return n.Range(lo, hi, true, true)
}

// OrRange – add another range joined with OR
func (n *NumericQuery) OrRange(lo, hi float64, inclLo, inclHi bool) *NumericQuery {
	return n.Range(lo, hi, inclLo, inclHi)
}

// BetweenMany – convenience: [[10,20],[30,40]]
func (n *NumericQuery) BetweenMany(ranges [][2]float64) *NumericQuery {
	for _, r := range ranges {
		n.Range(r[0], r[1], true, true)
	}
	return n
}

func (n *NumericQuery) Build() string {
	if len(n.intervals) == 0 {
		return ""
	}

	merged := merge(n.intervals)
	if len(merged) == 1 {
		return merged[0].toString(n.field)
	}

	var parts []string
	for _, iv := range merged {
		parts = append(parts, iv.toString(n.field))
	}
	return strings.Join(parts, " | ")
}

var inf = math.Inf(1)
var negInf = math.Inf(-1)

func (n *NumericQuery) add(lo, hi bound) *NumericQuery {
	if lo.val > hi.val {
		lo.val, hi.val = hi.val, lo.val
	}
	n.intervals = append(n.intervals, interval{lo: lo, hi: hi})
	return n
}

func merge(iv []interval) []interval {
	if len(iv) == 0 {
		return nil
	}
	sort.Slice(iv, func(i, j int) bool { return iv[i].lo.val < iv[j].lo.val })
	out := []interval{iv[0]}
	for _, cur := range iv[1:] {
		last := &out[len(out)-1]
		
		if cur.lo.val < last.hi.val ||
			(cur.lo.val == last.hi.val && !(cur.lo.exclusive || last.hi.exclusive)) {

			if cur.hi.val > last.hi.val ||
				(cur.hi.val == last.hi.val && !cur.hi.exclusive && last.hi.exclusive) {
				last.hi = cur.hi
			}
		} else {
			out = append(out, cur)
		}
	}
	return out
}
