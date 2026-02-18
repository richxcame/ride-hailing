package geo

import "math"

const (
	earthRadiusKm  = 6371.0
	averageSpeedKmh = 40.0 // city traffic average
)

// Haversine calculates the great-circle distance in kilometres between two
// coordinates. The result is rounded to two decimal places.
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return math.Round(earthRadiusKm*c*100) / 100
}

// EstimateDuration returns the estimated travel time in minutes for a given
// distance in kilometres, assuming an average city speed of 40 km/h.
func EstimateDuration(distanceKm float64) int {
	return int(math.Round((distanceKm / averageSpeedKmh) * 60))
}
