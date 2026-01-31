package geo

import (
	"math"
	"testing"
)

func TestHaversineDistance(t *testing.T) {
	// NYC to LAX ~3940 km
	dist := haversineDistance(40.7128, -74.0060, 33.9425, -118.4081)
	if dist < 3900 || dist > 4000 {
		t.Fatalf("NYC-LAX expected ~3940km, got %f", dist)
	}

	// Same point
	dist = haversineDistance(37.7749, -122.4194, 37.7749, -122.4194)
	if dist != 0 {
		t.Fatalf("same point expected 0km, got %f", dist)
	}

	// Short distance (~1km)
	dist = haversineDistance(37.7749, -122.4194, 37.7839, -122.4094)
	if dist < 0.8 || dist > 1.5 {
		t.Fatalf("short distance expected ~1km, got %f", dist)
	}
}

func TestEstimateETAMinutes_WithSpeed(t *testing.T) {
	// 10km at 60km/h = 10 minutes
	eta := estimateETAMinutes(10.0, 60.0)
	if eta != 10 {
		t.Fatalf("expected 10 minutes, got %d", eta)
	}

	// 5km at 30km/h = 10 minutes
	eta = estimateETAMinutes(5.0, 30.0)
	if eta != 10 {
		t.Fatalf("expected 10 minutes, got %d", eta)
	}
}

func TestEstimateETAMinutes_FallbackSpeed(t *testing.T) {
	// Speed < 5 km/h should use fallback (30 km/h)
	// 3km at fallback 30km/h = 6 minutes
	eta := estimateETAMinutes(3.0, 2.0)
	if eta != 6 {
		t.Fatalf("expected 6 minutes (fallback speed), got %d", eta)
	}

	// Zero speed
	eta = estimateETAMinutes(3.0, 0.0)
	if eta != 6 {
		t.Fatalf("expected 6 minutes (zero speed fallback), got %d", eta)
	}
}

func TestEstimateETAMinutes_CeilRounding(t *testing.T) {
	// 1km at 60km/h = 1 minute exactly
	eta := estimateETAMinutes(1.0, 60.0)
	if eta != 1 {
		t.Fatalf("expected 1 minute, got %d", eta)
	}

	// 1.5km at 60km/h = 1.5 minutes -> ceil to 2
	eta = estimateETAMinutes(1.5, 60.0)
	if eta != 2 {
		t.Fatalf("expected 2 minutes (ceil), got %d", eta)
	}
}

func TestEstimateETAMinutes_VeryShortDistance(t *testing.T) {
	// 0.1km at 30km/h = 0.2 min -> ceil to 1
	eta := estimateETAMinutes(0.1, 30.0)
	if eta != 1 {
		t.Fatalf("expected 1 minute minimum, got %d", eta)
	}
}

func TestHaversineDistance_Symmetry(t *testing.T) {
	d1 := haversineDistance(37.7749, -122.4194, 34.0522, -118.2437)
	d2 := haversineDistance(34.0522, -118.2437, 37.7749, -122.4194)

	if math.Abs(d1-d2) > 0.001 {
		t.Fatalf("haversine should be symmetric: %f != %f", d1, d2)
	}
}
