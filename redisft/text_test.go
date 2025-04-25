package redisft

import (
	"testing"
)

func TestQB_Build(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		qb   *QB
		want string
	}{
		{"empty", NewTextQuery("field"), ""},
		{"term", NewTextQuery("field").Term("foo"), "@field:(foo)"},
		{"prefix", NewTextQuery("field").Prefix("foo"), "@field:(foo*)"},
		{"suffix", NewTextQuery("field").Suffix("bar"), "@field:(*bar)"},
		{"exact", NewTextQuery("field").Exact("baz qux"), `@field:("baz qux")`},
		{"or", NewTextQuery("field").Term("a").Or().Term("b"), "@field:(a|b)"},
		{"and", NewTextQuery("field").Term("a").And().Term("b"), "@field:(a b)"},
		{"not", NewTextQuery("field").Not().Term("a"), "@field:(-a)"},
		{"must", NewTextQuery("field").Must("a"), "@field:(+a)"},
		{"any", NewTextQuery("field").Any("x", "y", "z"), "@field:(x|y|z)"},
		{"all", NewTextQuery("field").All("x", "y", "z"), "@field:(x y z)"},
		{"wild", NewTextQuery("field").Wild("foo|bar"), "@field:(foo\\|bar)"},
		{"escape", NewTextQuery("field").Term("special|chars"), "@field:(special\\|chars)"},
		{"group", NewTextQuery("field").Group(func(q *QB) {
			q.Term("a")
			q.Or()
			q.Term("b")
		}), "@field:((a|b))"},
		{"complex", NewTextQuery("field").
			Prefix("a").
			And().
			Not().
			Suffix("b").
			Or().
			Must("c"), "@field:(a* -*b|+c)"},
		{"close_without_open", NewTextQuery("field").Close().Term("x"), "@field:(x)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.qb.Build()
			if got != tc.want {
				t.Errorf("Build() = %q, want %q", got, tc.want)
			}
		})
	}
}
