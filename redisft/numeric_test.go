package redisft

import (
	"testing"
)

func TestNumericQuery_Build(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		q    *NumericQuery
		want string
	}{
		{"empty", NewNumericQuery("f"), ""},
		{"gt", NewNumericQuery("f").Gt(5), "@f:(5 +inf]"},
		{"ge", NewNumericQuery("f").Ge(5), "@f:[5 +inf]"},
		{"lt", NewNumericQuery("f").Lt(5), "@f:[-inf 5)"},
		{"le", NewNumericQuery("f").Le(5), "@f:[-inf 5]"},
		{"range_exclusive", NewNumericQuery("f").Range(1, 2, false, false), "@f:(1 2)"},
		{"range_inclusive", NewNumericQuery("f").Range(1, 2, true, true), "@f:[1 2]"},
		{"range_mixed", NewNumericQuery("f").Range(1, 2, true, false), "@f:[1 2)"},
		{"between", NewNumericQuery("f").Between(3, 4), "@f:[3 4]"},
		{"between_many", NewNumericQuery("f").BetweenMany([][2]float64{{1, 2}, {3, 4}}), "@f:[1 2] | @f:[3 4]"},
		{"merge_overlap", NewNumericQuery("f").Between(1, 3).Between(2, 4), "@f:[1 4]"},
		{"merge_adjacent", NewNumericQuery("f").Between(1, 2).Between(2, 3), "@f:[1 3]"},
		{"reverse", NewNumericQuery("f").Range(5, 1, true, true), "@f:[1 5]"},
		{"reverse_excl", NewNumericQuery("f").Range(5, 1, false, true), "@f:(1 5]"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.q.Build()
			if got != tc.want {
				t.Errorf("Build() = %q, want %q", got, tc.want)
			}
		})
	}
}
