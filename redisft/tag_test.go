package redisft

import (
	"testing"
)

func TestEscapeTag(t *testing.T) {
	tests := []struct{ input, want string }{
		{"simple", "simple"},
		{"with space", `with\ space`},
		{`comma,brace{}`, `comma\,brace\{\}`},
		{`pipe|back\\`, `pipe\|back\\\\`},
	}
	for _, tc := range tests {
		tc := tc 
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := escapeTag(tc.input)
			if got != tc.want {
				t.Errorf("escapeTag(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestJoinTags(t *testing.T) {
	tests := []struct {
		tags      []string
		mandatory bool
		want      string
	}{
		{[]string{"a"}, false, "a"},
		{[]string{"a", "b"}, false, "a|b"},
		{[]string{"a", "b"}, true, "+a|+b"},
		{[]string{"a b", "c"}, true, `+a\ b|+c`},
	}
	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run("", func(t *testing.T) {
			t.Parallel()
			got := joinTags(tc.tags, tc.mandatory)
			if got != tc.want {
				t.Errorf("joinTags(%v, %v) = %q; want %q", tc.tags, tc.mandatory, got, tc.want)
			}
		})
	}
}

func TestTagQB_Build(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		q    *TagQB
		want string
	}{
		{"empty", NewTagQB("f"), ""},
		{"single any", NewTagQB("f").Any("a"), "@f:{a}"},
		{"multi any", NewTagQB("f").Any("a", "b"), "@f:{a|b}"},
		{"mandatory", NewTagQB("f").Require("a", "b"), "@f:{+a|+b}"},
		{"notin", NewTagQB("f").NotIn("a", "b"), "-@f:{a|b}"},
		{"all", NewTagQB("f").All("a", "b"), "@f:{+a} @f:{+b}"},
		{"or", NewTagQB("f").Any("a").Or().Any("b"), "@f:{a}|@f:{b}"},
		{"mixed not", NewTagQB("f").Not().Any("a"), "-@f:{a}"},
		{"group simple", NewTagQB("f").Group(func(q *TagQB) {
			q.Any("a", "b")
		}), "(@f:{a|b})"},
		{"group complex", NewTagQB("f").Group(func(q *TagQB) {
			q.Any("a").Or().Require("b")
		}), "(@f:{a}|@f:{+b})"},
	}
	for _, tc := range tests {
		tc := tc 
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.q.Build()
			if got != tc.want {
				t.Errorf("Build() = %q; want %q", got, tc.want)
			}
		})
	}
}
