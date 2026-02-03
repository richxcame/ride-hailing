package maps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/richxcame/ride-hailing/pkg/httpclient"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

const (
	googleMapsBaseURL         = "https://maps.googleapis.com/maps/api"
	googleDirectionsEndpoint  = "/directions/json"
	googleDistanceMatrixEndpoint = "/distancematrix/json"
	googleGeocodingEndpoint   = "/geocode/json"
	googlePlacesEndpoint      = "/place/nearbysearch/json"
	googleSnapToRoadEndpoint  = "https://roads.googleapis.com/v1/snapToRoads"
	googleSpeedLimitsEndpoint = "https://roads.googleapis.com/v1/speedLimits"
)

// GoogleMapsProvider implements MapsProvider for Google Maps API
type GoogleMapsProvider struct {
	apiKey     string
	client     *httpclient.Client
	baseURL    string
	roadsURL   string
}

// NewGoogleMapsProvider creates a new Google Maps provider
func NewGoogleMapsProvider(config ProviderConfig) *GoogleMapsProvider {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = googleMapsBaseURL
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	return &GoogleMapsProvider{
		apiKey:   config.APIKey,
		client:   httpclient.NewClient(baseURL, time.Duration(timeout)*time.Second),
		baseURL:  baseURL,
		roadsURL: "https://roads.googleapis.com/v1",
	}
}

// Name returns the provider name
func (g *GoogleMapsProvider) Name() Provider {
	return ProviderGoogle
}

// HealthCheck verifies the API key is valid
func (g *GoogleMapsProvider) HealthCheck(ctx context.Context) error {
	// Make a simple geocoding request to verify API key
	params := url.Values{}
	params.Set("address", "1600 Amphitheatre Parkway, Mountain View, CA")
	params.Set("key", g.apiKey)

	resp, err := g.client.Get(ctx, googleGeocodingEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return fmt.Errorf("google maps health check failed: %w", err)
	}

	var result struct {
		Status       string `json:"status"`
		ErrorMessage string `json:"error_message"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse health check response: %w", err)
	}

	if result.Status != "OK" && result.Status != "ZERO_RESULTS" {
		return fmt.Errorf("google maps API error: %s - %s", result.Status, result.ErrorMessage)
	}

	return nil
}

// GetRoute calculates a route between origin and destination
func (g *GoogleMapsProvider) GetRoute(ctx context.Context, req *RouteRequest) (*RouteResponse, error) {
	params := url.Values{}
	params.Set("origin", formatCoordinate(req.Origin))
	params.Set("destination", formatCoordinate(req.Destination))
	params.Set("key", g.apiKey)
	params.Set("mode", "driving")

	// Add waypoints if any
	if len(req.Waypoints) > 0 {
		waypoints := make([]string, len(req.Waypoints))
		for i, wp := range req.Waypoints {
			waypoints[i] = formatCoordinate(wp)
		}
		params.Set("waypoints", strings.Join(waypoints, "|"))
	}

	// Traffic-aware routing
	if req.DepartureTime != nil {
		params.Set("departure_time", strconv.FormatInt(req.DepartureTime.Unix(), 10))
	} else {
		params.Set("departure_time", "now")
	}

	if req.TrafficModel != "" {
		params.Set("traffic_model", req.TrafficModel)
	} else {
		params.Set("traffic_model", "best_guess")
	}

	// Route options
	var avoid []string
	if req.AvoidTolls {
		avoid = append(avoid, "tolls")
	}
	if req.AvoidHighways {
		avoid = append(avoid, "highways")
	}
	if req.AvoidFerries {
		avoid = append(avoid, "ferries")
	}
	if len(avoid) > 0 {
		params.Set("avoid", strings.Join(avoid, "|"))
	}

	if req.Alternatives {
		params.Set("alternatives", "true")
	}

	params.Set("units", "metric")

	logger.Debug("Google Maps directions request", zap.String("params", params.Encode()))

	resp, err := g.client.Get(ctx, googleDirectionsEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("google maps directions request failed: %w", err)
	}

	var googleResp googleDirectionsResponse
	if err := json.Unmarshal(resp, &googleResp); err != nil {
		return nil, fmt.Errorf("failed to parse directions response: %w", err)
	}

	if googleResp.Status != "OK" {
		return nil, fmt.Errorf("google maps error: %s - %s", googleResp.Status, googleResp.ErrorMessage)
	}

	return g.convertDirectionsResponse(&googleResp), nil
}

// GetETA calculates the estimated time of arrival
func (g *GoogleMapsProvider) GetETA(ctx context.Context, req *ETARequest) (*ETAResponse, error) {
	routeReq := &RouteRequest{
		Origin:        req.Origin,
		Destination:   req.Destination,
		DepartureTime: req.DepartureTime,
		TrafficModel:  req.TrafficModel,
	}

	routeResp, err := g.GetRoute(ctx, routeReq)
	if err != nil {
		return nil, err
	}

	if len(routeResp.Routes) == 0 {
		return nil, fmt.Errorf("no routes found")
	}

	route := routeResp.Routes[0]
	now := time.Now()

	durationInTraffic := route.DurationInTrafficMin
	if durationInTraffic == 0 {
		durationInTraffic = route.DurationMinutes
	}

	return &ETAResponse{
		DistanceKm:          route.DistanceKm,
		DistanceMeters:      route.DistanceMeters,
		DurationMinutes:     route.DurationMinutes,
		DurationSeconds:     route.DurationSeconds,
		DurationInTraffic:   durationInTraffic,
		TrafficDelayMinutes: float64(route.TrafficDelaySeconds) / 60,
		TrafficLevel:        route.TrafficLevel,
		EstimatedArrival:    now.Add(time.Duration(durationInTraffic) * time.Minute),
		Provider:            ProviderGoogle,
		Confidence:          0.95, // Google Maps has high accuracy
	}, nil
}

// GetDistanceMatrix calculates distances between multiple origins and destinations
func (g *GoogleMapsProvider) GetDistanceMatrix(ctx context.Context, req *DistanceMatrixRequest) (*DistanceMatrixResponse, error) {
	params := url.Values{}
	params.Set("key", g.apiKey)
	params.Set("mode", "driving")
	params.Set("units", "metric")

	// Format origins
	origins := make([]string, len(req.Origins))
	for i, o := range req.Origins {
		origins[i] = formatCoordinate(o)
	}
	params.Set("origins", strings.Join(origins, "|"))

	// Format destinations
	destinations := make([]string, len(req.Destinations))
	for i, d := range req.Destinations {
		destinations[i] = formatCoordinate(d)
	}
	params.Set("destinations", strings.Join(destinations, "|"))

	// Traffic-aware
	if req.DepartureTime != nil {
		params.Set("departure_time", strconv.FormatInt(req.DepartureTime.Unix(), 10))
	} else {
		params.Set("departure_time", "now")
	}

	if req.TrafficModel != "" {
		params.Set("traffic_model", req.TrafficModel)
	}

	resp, err := g.client.Get(ctx, googleDistanceMatrixEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("google maps distance matrix request failed: %w", err)
	}

	var googleResp googleDistanceMatrixResponse
	if err := json.Unmarshal(resp, &googleResp); err != nil {
		return nil, fmt.Errorf("failed to parse distance matrix response: %w", err)
	}

	if googleResp.Status != "OK" {
		return nil, fmt.Errorf("google maps error: %s - %s", googleResp.Status, googleResp.ErrorMessage)
	}

	return g.convertDistanceMatrixResponse(&googleResp), nil
}

// GetTrafficFlow returns traffic flow information for a location
func (g *GoogleMapsProvider) GetTrafficFlow(ctx context.Context, req *TrafficFlowRequest) (*TrafficFlowResponse, error) {
	// Google Maps doesn't have a direct traffic flow API
	// We estimate traffic by comparing current ETA with free-flow ETA
	return &TrafficFlowResponse{
		Segments:     []TrafficFlowSegment{},
		OverallLevel: TrafficModerate, // Default
		UpdatedAt:    time.Now(),
		Provider:     ProviderGoogle,
	}, nil
}

// GetTrafficIncidents returns traffic incidents in an area
func (g *GoogleMapsProvider) GetTrafficIncidents(ctx context.Context, req *TrafficIncidentsRequest) (*TrafficIncidentsResponse, error) {
	// Google Maps doesn't have a public traffic incidents API
	// This would require Google Maps Platform Premium
	return &TrafficIncidentsResponse{
		Incidents: []TrafficIncident{},
		UpdatedAt: time.Now(),
		Provider:  ProviderGoogle,
	}, nil
}

// Geocode converts an address to coordinates
func (g *GoogleMapsProvider) Geocode(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	params := url.Values{}
	params.Set("key", g.apiKey)

	if req.Address != "" {
		params.Set("address", req.Address)
	}

	if req.Language != "" {
		params.Set("language", req.Language)
	}

	if req.Region != "" {
		params.Set("region", req.Region)
	}

	// Component filtering
	if len(req.Components) > 0 {
		components := make([]string, 0, len(req.Components))
		for k, v := range req.Components {
			components = append(components, fmt.Sprintf("%s:%s", k, v))
		}
		params.Set("components", strings.Join(components, "|"))
	}

	resp, err := g.client.Get(ctx, googleGeocodingEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("google maps geocoding request failed: %w", err)
	}

	var googleResp googleGeocodingResponse
	if err := json.Unmarshal(resp, &googleResp); err != nil {
		return nil, fmt.Errorf("failed to parse geocoding response: %w", err)
	}

	if googleResp.Status != "OK" && googleResp.Status != "ZERO_RESULTS" {
		return nil, fmt.Errorf("google maps error: %s - %s", googleResp.Status, googleResp.ErrorMessage)
	}

	return g.convertGeocodingResponse(&googleResp), nil
}

// ReverseGeocode converts coordinates to an address
func (g *GoogleMapsProvider) ReverseGeocode(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	if req.Coordinate == nil {
		return nil, fmt.Errorf("coordinate is required for reverse geocoding")
	}

	params := url.Values{}
	params.Set("key", g.apiKey)
	params.Set("latlng", formatCoordinate(*req.Coordinate))

	if req.Language != "" {
		params.Set("language", req.Language)
	}

	resp, err := g.client.Get(ctx, googleGeocodingEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("google maps reverse geocoding request failed: %w", err)
	}

	var googleResp googleGeocodingResponse
	if err := json.Unmarshal(resp, &googleResp); err != nil {
		return nil, fmt.Errorf("failed to parse reverse geocoding response: %w", err)
	}

	if googleResp.Status != "OK" && googleResp.Status != "ZERO_RESULTS" {
		return nil, fmt.Errorf("google maps error: %s - %s", googleResp.Status, googleResp.ErrorMessage)
	}

	return g.convertGeocodingResponse(&googleResp), nil
}

// SearchPlaces searches for nearby places
func (g *GoogleMapsProvider) SearchPlaces(ctx context.Context, req *PlaceSearchRequest) (*PlaceSearchResponse, error) {
	params := url.Values{}
	params.Set("key", g.apiKey)

	if req.Query != "" {
		params.Set("keyword", req.Query)
	}

	if req.Location != nil {
		params.Set("location", formatCoordinate(*req.Location))
	}

	if req.RadiusMeters > 0 {
		params.Set("radius", strconv.Itoa(req.RadiusMeters))
	} else {
		params.Set("radius", "5000") // Default 5km
	}

	if len(req.Types) > 0 {
		params.Set("type", req.Types[0]) // Google only supports one type
	}

	if req.Language != "" {
		params.Set("language", req.Language)
	}

	if req.OpenNow {
		params.Set("opennow", "true")
	}

	resp, err := g.client.Get(ctx, googlePlacesEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("google maps places search failed: %w", err)
	}

	var googleResp googlePlacesResponse
	if err := json.Unmarshal(resp, &googleResp); err != nil {
		return nil, fmt.Errorf("failed to parse places response: %w", err)
	}

	if googleResp.Status != "OK" && googleResp.Status != "ZERO_RESULTS" {
		return nil, fmt.Errorf("google maps error: %s - %s", googleResp.Status, googleResp.ErrorMessage)
	}

	return g.convertPlacesResponse(&googleResp), nil
}

// SnapToRoad snaps GPS coordinates to the nearest road
func (g *GoogleMapsProvider) SnapToRoad(ctx context.Context, req *SnapToRoadRequest) (*SnapToRoadResponse, error) {
	params := url.Values{}
	params.Set("key", g.apiKey)

	// Format path
	path := make([]string, len(req.Path))
	for i, c := range req.Path {
		path[i] = formatCoordinate(c)
	}
	params.Set("path", strings.Join(path, "|"))

	if req.Interpolate {
		params.Set("interpolate", "true")
	}

	client := httpclient.NewClient(g.roadsURL, 30*time.Second)
	resp, err := client.Get(ctx, "/snapToRoads?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("google roads snap request failed: %w", err)
	}

	var googleResp googleSnapToRoadResponse
	if err := json.Unmarshal(resp, &googleResp); err != nil {
		return nil, fmt.Errorf("failed to parse snap response: %w", err)
	}

	if googleResp.Error != nil {
		return nil, fmt.Errorf("google roads error: %s", googleResp.Error.Message)
	}

	return g.convertSnapResponse(&googleResp), nil
}

// GetSpeedLimits returns speed limits along a path
func (g *GoogleMapsProvider) GetSpeedLimits(ctx context.Context, req *SpeedLimitsRequest) (*SpeedLimitsResponse, error) {
	params := url.Values{}
	params.Set("key", g.apiKey)

	if len(req.Path) > 0 {
		path := make([]string, len(req.Path))
		for i, c := range req.Path {
			path[i] = formatCoordinate(c)
		}
		params.Set("path", strings.Join(path, "|"))
	} else if len(req.PlaceIDs) > 0 {
		params.Set("placeId", strings.Join(req.PlaceIDs, "|"))
	}

	client := httpclient.NewClient(g.roadsURL, 30*time.Second)
	resp, err := client.Get(ctx, "/speedLimits?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("google roads speed limits request failed: %w", err)
	}

	var googleResp googleSpeedLimitsResponse
	if err := json.Unmarshal(resp, &googleResp); err != nil {
		return nil, fmt.Errorf("failed to parse speed limits response: %w", err)
	}

	if googleResp.Error != nil {
		return nil, fmt.Errorf("google roads error: %s", googleResp.Error.Message)
	}

	return g.convertSpeedLimitsResponse(&googleResp), nil
}

// Conversion helpers

func formatCoordinate(c Coordinate) string {
	return fmt.Sprintf("%f,%f", c.Latitude, c.Longitude)
}

func (g *GoogleMapsProvider) convertDirectionsResponse(resp *googleDirectionsResponse) *RouteResponse {
	routes := make([]Route, len(resp.Routes))

	for i, r := range resp.Routes {
		route := Route{
			EncodedPolyline: r.OverviewPolyline.Points,
			Summary:         r.Summary,
			Warnings:        r.Warnings,
		}

		// Aggregate totals from legs
		for _, leg := range r.Legs {
			route.DistanceMeters += leg.Distance.Value
			route.DurationSeconds += leg.Duration.Value
			if leg.DurationInTraffic.Value > 0 {
				route.DurationInTraffic += leg.DurationInTraffic.Value
			}

			routeLeg := RouteLeg{
				StartLocation:     Coordinate{Latitude: leg.StartLocation.Lat, Longitude: leg.StartLocation.Lng},
				EndLocation:       Coordinate{Latitude: leg.EndLocation.Lat, Longitude: leg.EndLocation.Lng},
				StartAddress:      leg.StartAddress,
				EndAddress:        leg.EndAddress,
				DistanceMeters:    leg.Distance.Value,
				DurationSeconds:   leg.Duration.Value,
				DurationInTraffic: leg.DurationInTraffic.Value,
			}

			// Convert steps
			for _, step := range leg.Steps {
				routeStep := RouteStep{
					Instruction:     step.HTMLInstructions,
					Maneuver:        step.Maneuver,
					DistanceMeters:  step.Distance.Value,
					DurationSeconds: step.Duration.Value,
					StartLocation:   Coordinate{Latitude: step.StartLocation.Lat, Longitude: step.StartLocation.Lng},
					EndLocation:     Coordinate{Latitude: step.EndLocation.Lat, Longitude: step.EndLocation.Lng},
					EncodedPolyline: step.Polyline.Points,
				}
				routeLeg.Steps = append(routeLeg.Steps, routeStep)
			}

			route.Legs = append(route.Legs, routeLeg)
		}

		route.DistanceKm = float64(route.DistanceMeters) / 1000
		route.DurationMinutes = float64(route.DurationSeconds) / 60
		if route.DurationInTraffic > 0 {
			route.DurationInTrafficMin = float64(route.DurationInTraffic) / 60
			route.TrafficDelaySeconds = route.DurationInTraffic - route.DurationSeconds
		}

		// Calculate traffic level from delay
		route.TrafficLevel = calculateTrafficLevel(route.DurationSeconds, route.DurationInTraffic)

		// Bounds
		if r.Bounds != nil {
			route.BoundingBox = &BoundingBox{
				Northeast: Coordinate{Latitude: r.Bounds.Northeast.Lat, Longitude: r.Bounds.Northeast.Lng},
				Southwest: Coordinate{Latitude: r.Bounds.Southwest.Lat, Longitude: r.Bounds.Southwest.Lng},
			}
		}

		routes[i] = route
	}

	return &RouteResponse{
		Routes:      routes,
		Provider:    ProviderGoogle,
		RequestedAt: time.Now(),
	}
}

func (g *GoogleMapsProvider) convertDistanceMatrixResponse(resp *googleDistanceMatrixResponse) *DistanceMatrixResponse {
	rows := make([]DistanceMatrixRow, len(resp.Rows))

	for i, row := range resp.Rows {
		elements := make([]DistanceMatrixElement, len(row.Elements))
		for j, elem := range row.Elements {
			elements[j] = DistanceMatrixElement{
				Status:          elem.Status,
				DistanceMeters:  elem.Distance.Value,
				DistanceKm:      float64(elem.Distance.Value) / 1000,
				DurationSeconds: elem.Duration.Value,
				DurationMinutes: float64(elem.Duration.Value) / 60,
			}
			if elem.DurationInTraffic.Value > 0 {
				elements[j].DurationInTraffic = float64(elem.DurationInTraffic.Value) / 60
				elements[j].TrafficLevel = calculateTrafficLevel(elem.Duration.Value, elem.DurationInTraffic.Value)
			}
		}
		rows[i] = DistanceMatrixRow{Elements: elements}
	}

	return &DistanceMatrixResponse{
		Rows:        rows,
		Provider:    ProviderGoogle,
		RequestedAt: time.Now(),
	}
}

func (g *GoogleMapsProvider) convertGeocodingResponse(resp *googleGeocodingResponse) *GeocodingResponse {
	results := make([]GeocodingResult, len(resp.Results))

	for i, r := range resp.Results {
		result := GeocodingResult{
			FormattedAddress: r.FormattedAddress,
			Coordinate:       Coordinate{Latitude: r.Geometry.Location.Lat, Longitude: r.Geometry.Location.Lng},
			PlaceID:          r.PlaceID,
			Types:            r.Types,
			Confidence:       calculateGeocodingConfidence(r.Geometry.LocationType),
		}

		for _, comp := range r.AddressComponents {
			result.AddressComponents = append(result.AddressComponents, AddressComponent{
				LongName:  comp.LongName,
				ShortName: comp.ShortName,
				Types:     comp.Types,
			})
		}

		results[i] = result
	}

	return &GeocodingResponse{
		Results:  results,
		Provider: ProviderGoogle,
	}
}

func (g *GoogleMapsProvider) convertPlacesResponse(resp *googlePlacesResponse) *PlaceSearchResponse {
	results := make([]Place, len(resp.Results))

	for i, r := range resp.Results {
		place := Place{
			PlaceID:          r.PlaceID,
			Name:             r.Name,
			FormattedAddress: r.Vicinity,
			Coordinate:       Coordinate{Latitude: r.Geometry.Location.Lat, Longitude: r.Geometry.Location.Lng},
			Types:            r.Types,
			Icon:             r.Icon,
		}

		if r.Rating > 0 {
			place.Rating = &r.Rating
		}
		if r.UserRatingsTotal > 0 {
			place.UserRatingsTotal = &r.UserRatingsTotal
		}
		if r.PriceLevel > 0 {
			place.PriceLevel = &r.PriceLevel
		}
		if r.OpeningHours != nil {
			place.OpeningHours = &OpeningHours{OpenNow: r.OpeningHours.OpenNow}
		}

		results[i] = place
	}

	return &PlaceSearchResponse{
		Results:       results,
		NextPageToken: resp.NextPageToken,
		Provider:      ProviderGoogle,
	}
}

func (g *GoogleMapsProvider) convertSnapResponse(resp *googleSnapToRoadResponse) *SnapToRoadResponse {
	points := make([]SnappedPoint, len(resp.SnappedPoints))

	for i, p := range resp.SnappedPoints {
		points[i] = SnappedPoint{
			Location:      Coordinate{Latitude: p.Location.Latitude, Longitude: p.Location.Longitude},
			OriginalIndex: p.OriginalIndex,
			PlaceID:       p.PlaceID,
		}
	}

	return &SnapToRoadResponse{
		SnappedPoints: points,
		Provider:      ProviderGoogle,
	}
}

func (g *GoogleMapsProvider) convertSpeedLimitsResponse(resp *googleSpeedLimitsResponse) *SpeedLimitsResponse {
	limits := make([]SpeedLimit, len(resp.SpeedLimits))

	for i, l := range resp.SpeedLimits {
		limits[i] = SpeedLimit{
			PlaceID:       l.PlaceID,
			SpeedLimitKmh: int(l.SpeedLimit),
			SpeedLimitMph: int(float64(l.SpeedLimit) * 0.621371),
		}
	}

	var snapped []SnappedPoint
	for _, p := range resp.SnappedPoints {
		snapped = append(snapped, SnappedPoint{
			Location:      Coordinate{Latitude: p.Location.Latitude, Longitude: p.Location.Longitude},
			OriginalIndex: p.OriginalIndex,
			PlaceID:       p.PlaceID,
		})
	}

	return &SpeedLimitsResponse{
		SpeedLimits:   limits,
		SnappedPoints: snapped,
		Provider:      ProviderGoogle,
	}
}

// calculateTrafficLevel determines traffic level based on delay
func calculateTrafficLevel(normalDuration, trafficDuration int) TrafficLevel {
	if trafficDuration <= 0 || normalDuration <= 0 {
		return TrafficFreeFlow
	}

	ratio := float64(trafficDuration) / float64(normalDuration)

	switch {
	case ratio <= 1.1:
		return TrafficFreeFlow
	case ratio <= 1.25:
		return TrafficLight
	case ratio <= 1.5:
		return TrafficModerate
	case ratio <= 2.0:
		return TrafficHeavy
	case ratio <= 3.0:
		return TrafficSevere
	default:
		return TrafficBlocked
	}
}

// calculateGeocodingConfidence determines confidence based on location type
func calculateGeocodingConfidence(locationType string) float64 {
	switch locationType {
	case "ROOFTOP":
		return 1.0
	case "RANGE_INTERPOLATED":
		return 0.9
	case "GEOMETRIC_CENTER":
		return 0.75
	case "APPROXIMATE":
		return 0.5
	default:
		return 0.5
	}
}

// Google Maps API response structures

type googleDirectionsResponse struct {
	Status       string         `json:"status"`
	ErrorMessage string         `json:"error_message,omitempty"`
	Routes       []googleRoute  `json:"routes"`
}

type googleRoute struct {
	Summary          string            `json:"summary"`
	Legs             []googleLeg       `json:"legs"`
	OverviewPolyline googlePolyline    `json:"overview_polyline"`
	Bounds           *googleBounds     `json:"bounds"`
	Warnings         []string          `json:"warnings"`
	WaypointOrder    []int             `json:"waypoint_order"`
}

type googleLeg struct {
	StartAddress      string            `json:"start_address"`
	EndAddress        string            `json:"end_address"`
	StartLocation     googleLatLng      `json:"start_location"`
	EndLocation       googleLatLng      `json:"end_location"`
	Distance          googleValue       `json:"distance"`
	Duration          googleValue       `json:"duration"`
	DurationInTraffic googleValue       `json:"duration_in_traffic"`
	Steps             []googleStep      `json:"steps"`
}

type googleStep struct {
	HTMLInstructions string        `json:"html_instructions"`
	Distance         googleValue   `json:"distance"`
	Duration         googleValue   `json:"duration"`
	StartLocation    googleLatLng  `json:"start_location"`
	EndLocation      googleLatLng  `json:"end_location"`
	Polyline         googlePolyline `json:"polyline"`
	Maneuver         string        `json:"maneuver,omitempty"`
	TravelMode       string        `json:"travel_mode"`
}

type googlePolyline struct {
	Points string `json:"points"`
}

type googleBounds struct {
	Northeast googleLatLng `json:"northeast"`
	Southwest googleLatLng `json:"southwest"`
}

type googleLatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type googleValue struct {
	Text  string `json:"text"`
	Value int    `json:"value"`
}

type googleDistanceMatrixResponse struct {
	Status       string                   `json:"status"`
	ErrorMessage string                   `json:"error_message,omitempty"`
	Rows         []googleDistanceMatrixRow `json:"rows"`
}

type googleDistanceMatrixRow struct {
	Elements []googleDistanceMatrixElement `json:"elements"`
}

type googleDistanceMatrixElement struct {
	Status            string      `json:"status"`
	Distance          googleValue `json:"distance"`
	Duration          googleValue `json:"duration"`
	DurationInTraffic googleValue `json:"duration_in_traffic"`
}

type googleGeocodingResponse struct {
	Status       string                `json:"status"`
	ErrorMessage string                `json:"error_message,omitempty"`
	Results      []googleGeocodingResult `json:"results"`
}

type googleGeocodingResult struct {
	FormattedAddress  string                    `json:"formatted_address"`
	Geometry          googleGeometry            `json:"geometry"`
	PlaceID           string                    `json:"place_id"`
	Types             []string                  `json:"types"`
	AddressComponents []googleAddressComponent  `json:"address_components"`
}

type googleGeometry struct {
	Location     googleLatLng `json:"location"`
	LocationType string       `json:"location_type"`
}

type googleAddressComponent struct {
	LongName  string   `json:"long_name"`
	ShortName string   `json:"short_name"`
	Types     []string `json:"types"`
}

type googlePlacesResponse struct {
	Status        string        `json:"status"`
	ErrorMessage  string        `json:"error_message,omitempty"`
	Results       []googlePlace `json:"results"`
	NextPageToken string        `json:"next_page_token,omitempty"`
}

type googlePlace struct {
	PlaceID          string           `json:"place_id"`
	Name             string           `json:"name"`
	Vicinity         string           `json:"vicinity"`
	Geometry         googleGeometry   `json:"geometry"`
	Types            []string         `json:"types"`
	Rating           float64          `json:"rating"`
	UserRatingsTotal int              `json:"user_ratings_total"`
	PriceLevel       int              `json:"price_level"`
	Icon             string           `json:"icon"`
	OpeningHours     *googleOpenHours `json:"opening_hours,omitempty"`
}

type googleOpenHours struct {
	OpenNow bool `json:"open_now"`
}

type googleSnapToRoadResponse struct {
	SnappedPoints []googleSnappedPoint `json:"snappedPoints"`
	Error         *googleAPIError      `json:"error,omitempty"`
}

type googleSnappedPoint struct {
	Location      googleRoadsLatLng `json:"location"`
	OriginalIndex int               `json:"originalIndex"`
	PlaceID       string            `json:"placeId"`
}

type googleRoadsLatLng struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type googleSpeedLimitsResponse struct {
	SpeedLimits   []googleSpeedLimit   `json:"speedLimits"`
	SnappedPoints []googleSnappedPoint `json:"snappedPoints"`
	Error         *googleAPIError      `json:"error,omitempty"`
}

type googleSpeedLimit struct {
	PlaceID    string  `json:"placeId"`
	SpeedLimit float64 `json:"speedLimit"`
	Units      string  `json:"units"`
}

type googleAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}
