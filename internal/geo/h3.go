package geo

import (
	"github.com/uber/h3-go/v4"
)

// H3 resolution levels for different use cases.
// See: https://h3geo.org/docs/core-library/restable
const (
	// H3ResolutionMatching is used for driver-rider matching (~175m edge, ~0.11 km²).
	H3ResolutionMatching = 9

	// H3ResolutionSurge is used for surge pricing zones (~460m edge, ~0.74 km²).
	H3ResolutionSurge = 8

	// H3ResolutionDemand is used for demand heat maps and forecasting (~1.2 km edge, ~5.16 km²).
	H3ResolutionDemand = 7

	// H3ResolutionCity is used for city-level aggregation (~3.2 km edge, ~36.13 km²).
	H3ResolutionCity = 6

	// H3KRingMatching is the k-ring radius for nearby driver search.
	// At resolution 9, k=4 covers roughly a 1.4 km radius.
	H3KRingMatching = 4

	// H3KRingSurge is the k-ring radius for surge zone neighbours.
	H3KRingSurge = 2
)

// LatLngToCell converts latitude/longitude to an H3 cell index at the given resolution.
// Panics on invalid input (latitude/longitude out of range) which should be validated upstream.
func LatLngToCell(lat, lng float64, resolution int) h3.Cell {
	latLng := h3.NewLatLng(lat, lng)
	cell, err := h3.LatLngToCell(latLng, resolution)
	if err != nil {
		return 0
	}
	return cell
}

// CellToLatLng returns the center coordinates of an H3 cell.
func CellToLatLng(cell h3.Cell) (lat, lng float64) {
	latLng, err := cell.LatLng()
	if err != nil {
		return 0, 0
	}
	return latLng.Lat, latLng.Lng
}

// GetKRingCells returns the set of H3 cell indexes within k rings of the origin cell.
func GetKRingCells(lat, lng float64, resolution, k int) []h3.Cell {
	origin := LatLngToCell(lat, lng, resolution)
	cells, err := origin.GridDisk(k)
	if err != nil {
		return []h3.Cell{origin}
	}
	return cells
}

// GetKRingCellStrings returns k-ring cells as hex strings for Redis key usage.
func GetKRingCellStrings(lat, lng float64, resolution, k int) []string {
	cells := GetKRingCells(lat, lng, resolution, k)
	result := make([]string, len(cells))
	for i, cell := range cells {
		result[i] = cell.String()
	}
	return result
}

// CellToString converts an H3 cell to its hex string representation.
func CellToString(cell h3.Cell) string {
	return cell.String()
}

// StringToCell parses an H3 cell hex string back to a Cell.
func StringToCell(s string) h3.Cell {
	return h3.CellFromString(s)
}

// GetSurgeZone returns the H3 cell index (as string) for surge pricing at the given location.
func GetSurgeZone(lat, lng float64) string {
	return LatLngToCell(lat, lng, H3ResolutionSurge).String()
}

// GetDemandZone returns the H3 cell index (as string) for demand analytics at the given location.
func GetDemandZone(lat, lng float64) string {
	return LatLngToCell(lat, lng, H3ResolutionDemand).String()
}

// GetMatchingCell returns the H3 cell index (as string) for driver-rider matching.
func GetMatchingCell(lat, lng float64) string {
	return LatLngToCell(lat, lng, H3ResolutionMatching).String()
}

// CellDistance returns the grid distance between two H3 cells at the same resolution.
// Returns -1 if the distance cannot be computed.
func CellDistance(a, b h3.Cell) int {
	dist, err := a.GridDistance(b)
	if err != nil {
		return -1
	}
	return dist
}

// GetNeighborCells returns the immediate neighbors of a cell (k=1 ring excluding center).
func GetNeighborCells(cell h3.Cell) []h3.Cell {
	ring, err := cell.GridDisk(1)
	if err != nil {
		return nil
	}
	neighbors := make([]h3.Cell, 0, len(ring)-1)
	for _, c := range ring {
		if c != cell {
			neighbors = append(neighbors, c)
		}
	}
	return neighbors
}

// CellArea returns the approximate area of an H3 cell in square kilometers.
func CellArea(cell h3.Cell) float64 {
	area, err := h3.CellAreaKm2(cell)
	if err != nil {
		return 0
	}
	return area
}
