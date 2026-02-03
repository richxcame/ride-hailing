package maps

import (
	"time"
)

// Provider represents the maps service provider type
type Provider string

const (
	ProviderGoogle         Provider = "google"
	ProviderHERE           Provider = "here"
	ProviderOpenRouteService Provider = "openrouteservice"
	ProviderMapbox         Provider = "mapbox"
)

// TrafficLevel indicates the current traffic conditions
type TrafficLevel string

const (
	TrafficFreeFlow   TrafficLevel = "free_flow"
	TrafficLight      TrafficLevel = "light"
	TrafficModerate   TrafficLevel = "moderate"
	TrafficHeavy      TrafficLevel = "heavy"
	TrafficSevere     TrafficLevel = "severe"
	TrafficBlocked    TrafficLevel = "blocked"
)

// Coordinate represents a geographic point
type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// RouteRequest represents a request for route calculation
type RouteRequest struct {
	Origin          Coordinate     `json:"origin"`
	Destination     Coordinate     `json:"destination"`
	Waypoints       []Coordinate   `json:"waypoints,omitempty"`
	DepartureTime   *time.Time     `json:"departure_time,omitempty"`
	ArrivalTime     *time.Time     `json:"arrival_time,omitempty"`
	Alternatives    bool           `json:"alternatives,omitempty"`
	AvoidTolls      bool           `json:"avoid_tolls,omitempty"`
	AvoidHighways   bool           `json:"avoid_highways,omitempty"`
	AvoidFerries    bool           `json:"avoid_ferries,omitempty"`
	TrafficModel    string         `json:"traffic_model,omitempty"` // best_guess, pessimistic, optimistic
	Units           string         `json:"units,omitempty"`         // metric, imperial
	VehicleType     string         `json:"vehicle_type,omitempty"`  // car, taxi, motorcycle
}

// RouteResponse represents the response from a route calculation
type RouteResponse struct {
	Routes          []Route       `json:"routes"`
	Provider        Provider      `json:"provider"`
	RequestedAt     time.Time     `json:"requested_at"`
	CacheHit        bool          `json:"cache_hit,omitempty"`
}

// Route represents a single route option
type Route struct {
	// Core metrics
	DistanceMeters    int           `json:"distance_meters"`
	DistanceKm        float64       `json:"distance_km"`
	DurationSeconds   int           `json:"duration_seconds"`
	DurationMinutes   float64       `json:"duration_minutes"`

	// Traffic-aware metrics
	DurationInTraffic    int       `json:"duration_in_traffic_seconds,omitempty"`
	DurationInTrafficMin float64   `json:"duration_in_traffic_minutes,omitempty"`
	TrafficDelaySeconds  int       `json:"traffic_delay_seconds,omitempty"`

	// Route geometry
	EncodedPolyline   string        `json:"encoded_polyline,omitempty"`
	Coordinates       []Coordinate  `json:"coordinates,omitempty"`
	BoundingBox       *BoundingBox  `json:"bounding_box,omitempty"`

	// Route details
	Legs              []RouteLeg    `json:"legs,omitempty"`
	Summary           string        `json:"summary,omitempty"`
	Warnings          []string      `json:"warnings,omitempty"`

	// Toll and fare info
	TollInfo          *TollInfo     `json:"toll_info,omitempty"`

	// Traffic conditions along route
	TrafficLevel      TrafficLevel  `json:"traffic_level,omitempty"`
	TrafficSegments   []TrafficSegment `json:"traffic_segments,omitempty"`
}

// RouteLeg represents a segment of the route between waypoints
type RouteLeg struct {
	StartLocation     Coordinate    `json:"start_location"`
	EndLocation       Coordinate    `json:"end_location"`
	StartAddress      string        `json:"start_address,omitempty"`
	EndAddress        string        `json:"end_address,omitempty"`
	DistanceMeters    int           `json:"distance_meters"`
	DurationSeconds   int           `json:"duration_seconds"`
	DurationInTraffic int           `json:"duration_in_traffic_seconds,omitempty"`
	Steps             []RouteStep   `json:"steps,omitempty"`
	TrafficLevel      TrafficLevel  `json:"traffic_level,omitempty"`
}

// RouteStep represents a single navigation instruction
type RouteStep struct {
	Instruction       string        `json:"instruction"`
	Maneuver          string        `json:"maneuver,omitempty"` // turn-left, turn-right, continue, etc.
	DistanceMeters    int           `json:"distance_meters"`
	DurationSeconds   int           `json:"duration_seconds"`
	StartLocation     Coordinate    `json:"start_location"`
	EndLocation       Coordinate    `json:"end_location"`
	RoadName          string        `json:"road_name,omitempty"`
	EncodedPolyline   string        `json:"encoded_polyline,omitempty"`
}

// BoundingBox represents the geographic bounds of a route
type BoundingBox struct {
	Northeast Coordinate `json:"northeast"`
	Southwest Coordinate `json:"southwest"`
}

// TollInfo contains toll-related information for a route
type TollInfo struct {
	HasTolls          bool          `json:"has_tolls"`
	EstimatedCost     *float64      `json:"estimated_cost,omitempty"`
	Currency          string        `json:"currency,omitempty"`
	TollRoads         []string      `json:"toll_roads,omitempty"`
}

// TrafficSegment represents traffic conditions on a route segment
type TrafficSegment struct {
	StartIndex        int           `json:"start_index"`
	EndIndex          int           `json:"end_index"`
	StartLocation     Coordinate    `json:"start_location"`
	EndLocation       Coordinate    `json:"end_location"`
	TrafficLevel      TrafficLevel  `json:"traffic_level"`
	SpeedKmh          float64       `json:"speed_kmh,omitempty"`
	FreeFlowSpeedKmh  float64       `json:"free_flow_speed_kmh,omitempty"`
	DelaySeconds      int           `json:"delay_seconds,omitempty"`
}

// ETARequest represents a request for ETA calculation
type ETARequest struct {
	Origin          Coordinate     `json:"origin"`
	Destination     Coordinate     `json:"destination"`
	DepartureTime   *time.Time     `json:"departure_time,omitempty"`
	TrafficModel    string         `json:"traffic_model,omitempty"`
}

// ETAResponse represents the ETA calculation result
type ETAResponse struct {
	DistanceKm           float64       `json:"distance_km"`
	DistanceMeters       int           `json:"distance_meters"`
	DurationMinutes      float64       `json:"duration_minutes"`
	DurationSeconds      int           `json:"duration_seconds"`
	DurationInTraffic    float64       `json:"duration_in_traffic_minutes"`
	TrafficDelayMinutes  float64       `json:"traffic_delay_minutes"`
	TrafficLevel         TrafficLevel  `json:"traffic_level"`
	EstimatedArrival     time.Time     `json:"estimated_arrival"`
	Provider             Provider      `json:"provider"`
	Confidence           float64       `json:"confidence"` // 0.0 to 1.0
	CacheHit             bool          `json:"cache_hit"`
}

// DistanceMatrixRequest represents a request for distance matrix calculation
type DistanceMatrixRequest struct {
	Origins         []Coordinate   `json:"origins"`
	Destinations    []Coordinate   `json:"destinations"`
	DepartureTime   *time.Time     `json:"departure_time,omitempty"`
	TrafficModel    string         `json:"traffic_model,omitempty"`
}

// DistanceMatrixResponse represents the distance matrix result
type DistanceMatrixResponse struct {
	Rows            []DistanceMatrixRow `json:"rows"`
	Provider        Provider            `json:"provider"`
	RequestedAt     time.Time           `json:"requested_at"`
}

// DistanceMatrixRow represents a row in the distance matrix
type DistanceMatrixRow struct {
	Elements []DistanceMatrixElement `json:"elements"`
}

// DistanceMatrixElement represents a single origin-destination pair result
type DistanceMatrixElement struct {
	Status            string        `json:"status"` // OK, ZERO_RESULTS, NOT_FOUND
	DistanceKm        float64       `json:"distance_km"`
	DistanceMeters    int           `json:"distance_meters"`
	DurationMinutes   float64       `json:"duration_minutes"`
	DurationSeconds   int           `json:"duration_seconds"`
	DurationInTraffic float64       `json:"duration_in_traffic_minutes,omitempty"`
	TrafficLevel      TrafficLevel  `json:"traffic_level,omitempty"`
}

// TrafficFlowRequest represents a request for traffic flow information
type TrafficFlowRequest struct {
	Location        Coordinate     `json:"location"`
	RadiusMeters    int            `json:"radius_meters,omitempty"`
	BoundingBox     *BoundingBox   `json:"bounding_box,omitempty"`
}

// TrafficFlowResponse represents traffic flow data
type TrafficFlowResponse struct {
	Segments        []TrafficFlowSegment `json:"segments"`
	OverallLevel    TrafficLevel         `json:"overall_level"`
	UpdatedAt       time.Time            `json:"updated_at"`
	Provider        Provider             `json:"provider"`
}

// TrafficFlowSegment represents traffic on a road segment
type TrafficFlowSegment struct {
	RoadName          string        `json:"road_name,omitempty"`
	Coordinates       []Coordinate  `json:"coordinates"`
	CurrentSpeedKmh   float64       `json:"current_speed_kmh"`
	FreeFlowSpeedKmh  float64       `json:"free_flow_speed_kmh"`
	TrafficLevel      TrafficLevel  `json:"traffic_level"`
	JamFactor         float64       `json:"jam_factor"` // 0.0 (free flow) to 10.0 (blocked)
}

// GeocodingRequest represents a geocoding request
type GeocodingRequest struct {
	Address         string         `json:"address,omitempty"`
	Coordinate      *Coordinate    `json:"coordinate,omitempty"` // For reverse geocoding
	Language        string         `json:"language,omitempty"`
	Region          string         `json:"region,omitempty"` // Country code bias
	Components      map[string]string `json:"components,omitempty"` // Address component filters
}

// GeocodingResponse represents a geocoding result
type GeocodingResponse struct {
	Results         []GeocodingResult `json:"results"`
	Provider        Provider          `json:"provider"`
}

// GeocodingResult represents a single geocoding result
type GeocodingResult struct {
	FormattedAddress  string            `json:"formatted_address"`
	Coordinate        Coordinate        `json:"coordinate"`
	PlaceID           string            `json:"place_id,omitempty"`
	Types             []string          `json:"types,omitempty"` // street_address, locality, etc.
	AddressComponents []AddressComponent `json:"address_components,omitempty"`
	Confidence        float64           `json:"confidence"` // 0.0 to 1.0
}

// AddressComponent represents a component of an address
type AddressComponent struct {
	LongName  string   `json:"long_name"`
	ShortName string   `json:"short_name"`
	Types     []string `json:"types"`
}

// PlaceSearchRequest represents a place search request
type PlaceSearchRequest struct {
	Query           string         `json:"query,omitempty"`
	Location        *Coordinate    `json:"location,omitempty"`
	RadiusMeters    int            `json:"radius_meters,omitempty"`
	Types           []string       `json:"types,omitempty"` // restaurant, airport, etc.
	Language        string         `json:"language,omitempty"`
	OpenNow         bool           `json:"open_now,omitempty"`
}

// PlaceSearchResponse represents place search results
type PlaceSearchResponse struct {
	Results         []Place        `json:"results"`
	NextPageToken   string         `json:"next_page_token,omitempty"`
	Provider        Provider       `json:"provider"`
}

// Place represents a place/point of interest
type Place struct {
	PlaceID           string        `json:"place_id"`
	Name              string        `json:"name"`
	FormattedAddress  string        `json:"formatted_address"`
	Coordinate        Coordinate    `json:"coordinate"`
	Types             []string      `json:"types,omitempty"`
	Rating            *float64      `json:"rating,omitempty"`
	UserRatingsTotal  *int          `json:"user_ratings_total,omitempty"`
	PriceLevel        *int          `json:"price_level,omitempty"`
	OpeningHours      *OpeningHours `json:"opening_hours,omitempty"`
	Icon              string        `json:"icon,omitempty"`
	Photos            []PhotoRef    `json:"photos,omitempty"`
}

// OpeningHours represents place opening hours
type OpeningHours struct {
	OpenNow     bool     `json:"open_now"`
	WeekdayText []string `json:"weekday_text,omitempty"`
}

// PhotoRef represents a reference to a place photo
type PhotoRef struct {
	Reference     string `json:"reference"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	Attributions  []string `json:"attributions,omitempty"`
}

// TrafficIncident represents a traffic incident
type TrafficIncident struct {
	ID                string        `json:"id"`
	Type              string        `json:"type"` // accident, construction, road_closure, etc.
	Severity          string        `json:"severity"` // minor, moderate, major, critical
	Location          Coordinate    `json:"location"`
	Description       string        `json:"description"`
	StartTime         *time.Time    `json:"start_time,omitempty"`
	EndTime           *time.Time    `json:"end_time,omitempty"`
	RoadClosed        bool          `json:"road_closed"`
	AffectedRoads     []string      `json:"affected_roads,omitempty"`
	DelayMinutes      int           `json:"delay_minutes,omitempty"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// TrafficIncidentsRequest represents a request for traffic incidents
type TrafficIncidentsRequest struct {
	BoundingBox     *BoundingBox   `json:"bounding_box,omitempty"`
	Location        *Coordinate    `json:"location,omitempty"`
	RadiusMeters    int            `json:"radius_meters,omitempty"`
	Types           []string       `json:"types,omitempty"` // Filter by incident type
}

// TrafficIncidentsResponse represents traffic incidents response
type TrafficIncidentsResponse struct {
	Incidents       []TrafficIncident `json:"incidents"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Provider        Provider          `json:"provider"`
}

// SnapToRoadRequest represents a request to snap coordinates to roads
type SnapToRoadRequest struct {
	Path            []Coordinate   `json:"path"`
	Interpolate     bool           `json:"interpolate,omitempty"` // Include interpolated points
}

// SnapToRoadResponse represents snapped coordinates
type SnapToRoadResponse struct {
	SnappedPoints   []SnappedPoint `json:"snapped_points"`
	Provider        Provider       `json:"provider"`
}

// SnappedPoint represents a point snapped to a road
type SnappedPoint struct {
	Location        Coordinate     `json:"location"`
	OriginalIndex   int            `json:"original_index,omitempty"` // -1 if interpolated
	PlaceID         string         `json:"place_id,omitempty"`
	RoadName        string         `json:"road_name,omitempty"`
}

// SpeedLimit represents a speed limit on a road
type SpeedLimit struct {
	PlaceID         string         `json:"place_id"`
	SpeedLimitKmh   int            `json:"speed_limit_kmh"`
	SpeedLimitMph   int            `json:"speed_limit_mph"`
}

// SpeedLimitsRequest represents a request for speed limits
type SpeedLimitsRequest struct {
	Path            []Coordinate   `json:"path,omitempty"`
	PlaceIDs        []string       `json:"place_ids,omitempty"`
}

// SpeedLimitsResponse represents speed limits along a path
type SpeedLimitsResponse struct {
	SpeedLimits     []SpeedLimit   `json:"speed_limits"`
	SnappedPoints   []SnappedPoint `json:"snapped_points,omitempty"`
	Provider        Provider       `json:"provider"`
}
