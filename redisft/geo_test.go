package redisft

import "testing"

func TestGeoQuery_Build(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		q    *GeoQuery
		want string
	}{
		{"no center and radius", NewGeoQuery("loc"), ""},
		{"only center", NewGeoQuery("loc").Center(0, 0), ""},
		{"only radius", NewGeoQuery("loc").Radius(1, Kilometers), ""},
		{"lat out of range", NewGeoQuery("loc").Center(0, 100).Km(1), ""},
		{"lon out of range", NewGeoQuery("loc").Center(200, 0).Km(1), ""},
		{"radius zero", NewGeoQuery("loc").Center(0, 0).Radius(0, Kilometers), ""},
		{"radius negative", NewGeoQuery("loc").Center(0, 0).Radius(-5, Kilometers), ""},
		{"default unit km", NewGeoQuery("loc").Center(-122.419400, 37.774900).Km(10), "@loc:[-122.419400 37.774900 10.0000 km]"},
		{"unit m", NewGeoQuery("loc").Center(-122.419400, 37.774900).M(500), "@loc:[-122.419400 37.774900 500.0000 m]"},
		{"unit mi", NewGeoQuery("loc").Center(-122.419400, 37.774900).Mi(5), "@loc:[-122.419400 37.774900 5.0000 mi]"},
		{"unit ft", NewGeoQuery("loc").Center(-122.419400, 37.774900).Ft(1000), "@loc:[-122.419400 37.774900 1000.0000 ft]"},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.q.Build()
			if got != tc.want {
				t.Errorf("Build() = %q, want %q", got, tc.want)
			}
		})
	}
}
