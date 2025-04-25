package redisft

import (
	"errors"
	"fmt"
)

type GeoUnit string

const (
	Kilometers GeoUnit = "km"
	Meters     GeoUnit = "m"
	Feet       GeoUnit = "ft"
	Miles      GeoUnit = "mi"
)

var errIncomplete = errors.New("redisft: geo query requires center and radius")

// @field:[lon lat radius unit]
type GeoQuery struct {
	field     string
	lon, lat  float64
	centerSet bool
	radius    float64
	radiusSet bool
	unit      GeoUnit
}

// NewGeoQuery returns a new builder with default unit = km.
func NewGeoQuery(field string) *GeoQuery {
	return &GeoQuery{field: field, unit: Kilometers}
}

// Center sets the longitude / latitude of the search origin.
func (g *GeoQuery) Center(lon, lat float64) *GeoQuery {
	g.lon, g.lat, g.centerSet = lon, lat, true
	return g
}

// Radius sets the radius and unit (generic form).
func (g *GeoQuery) Radius(r float64, u GeoUnit) *GeoQuery {
	g.radius, g.radiusSet, g.unit = r, true, u
	return g
}

func (g *GeoQuery) Km(km float64) *GeoQuery { return g.Radius(km, Kilometers) }
func (g *GeoQuery) M(m float64) *GeoQuery   { return g.Radius(m, Meters) }
func (g *GeoQuery) Mi(mi float64) *GeoQuery { return g.Radius(mi, Miles) }
func (g *GeoQuery) Ft(ft float64) *GeoQuery { return g.Radius(ft, Feet) }

func (g *GeoQuery) Build() string {
	if (!g.centerSet || !g.radiusSet) ||
		(g.lon < -180 || g.lon > 180 || g.lat < -90 || g.lat > 90) ||
		g.radius <= 0 {
		return ""
	}
	return fmt.Sprintf("@%s:[%.6f %.6f %.4f %s]",
		g.field, g.lon, g.lat, g.radius, g.unit)
}

// MustBuild is a helper that panics if Build returns an error.
// func (g *GeoQuery) MustBuild() string {
// 	s, err := g.Build()
// 	if err != nil {
// 		panic(err)
// 	}
// 	return s
// }

func (g *GeoQuery) GetFieldName() string { return g.field }
