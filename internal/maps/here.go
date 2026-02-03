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
	hereRoutingBaseURL  = "https://router.hereapi.com/v8"
	hereGeocodingURL    = "https://geocode.search.hereapi.com/v1"
	hereTrafficURL      = "https://traffic.ls.hereapi.com/traffic/6.3"
)

// HEREMapsProvider implements MapsProvider for HERE Maps API
type HEREMapsProvider struct {
	apiKey        string
	routingClient *httpclient.Client
	geocodeClient *httpclient.Client
	trafficClient *httpclient.Client
}

// NewHEREMapsProvider creates a new HERE Maps provider
func NewHEREMapsProvider(config ProviderConfig) *HEREMapsProvider {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	return &HEREMapsProvider{
		apiKey:        config.APIKey,
		routingClient: httpclient.NewClient(hereRoutingBaseURL, time.Duration(timeout)*time.Second),
		geocodeClient: httpclient.NewClient(hereGeocodingURL, time.Duration(timeout)*time.Second),
		trafficClient: httpclient.NewClient(hereTrafficURL, time.Duration(timeout)*time.Second),
	}
}

// Name returns the provider name
func (h *HEREMapsProvider) Name() Provider {
	return ProviderHERE
}

// HealthCheck verifies the API key is valid
func (h *HEREMapsProvider) HealthCheck(ctx context.Context) error {
	params := url.Values{}
	params.Set("apiKey", h.apiKey)
	params.Set("q", "Berlin")
	params.Set("limit", "1")

	resp, err := h.geocodeClient.Get(ctx, "/geocode?"+params.Encode(), nil)
	if err != nil {
		return fmt.Errorf("HERE maps health check failed: %w", err)
	}

	var result struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(resp, &result); err == nil && result.Error != "" {
		return fmt.Errorf("HERE maps API error: %s", result.Error)
	}

	return nil
}

// GetRoute calculates a route between origin and destination
func (h *HEREMapsProvider) GetRoute(ctx context.Context, req *RouteRequest) (*RouteResponse, error) {
	params := url.Values{}
	params.Set("apiKey", h.apiKey)
	params.Set("transportMode", "car")
	params.Set("return", "polyline,summary,travelSummary,turnByTurnActions")

	// Origin and destination
	params.Set("origin", formatHERECoordinate(req.Origin))
	params.Set("destination", formatHERECoordinate(req.Destination))

	// Add waypoints
	for i, wp := range req.Waypoints {
		params.Add(fmt.Sprintf("via%d", i), formatHERECoordinate(wp))
	}

	// Departure time for traffic-aware routing
	if req.DepartureTime != nil {
		params.Set("departureTime", req.DepartureTime.Format(time.RFC3339))
	} else {
		params.Set("departureTime", time.Now().Format(time.RFC3339))
	}

	// Route options
	var avoid []string
	if req.AvoidTolls {
		avoid = append(avoid, "tollRoad")
	}
	if req.AvoidHighways {
		avoid = append(avoid, "controlledAccessHighway")
	}
	if req.AvoidFerries {
		avoid = append(avoid, "ferry")
	}
	if len(avoid) > 0 {
		params.Set("avoid[features]", strings.Join(avoid, ","))
	}

	if req.Alternatives {
		params.Set("alternatives", "3")
	}

	logger.Debug("HERE Maps routing request", zap.String("params", params.Encode()))

	resp, err := h.routingClient.Get(ctx, "/routes?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("HERE maps routing request failed: %w", err)
	}

	var hereResp hereRoutingResponse
	if err := json.Unmarshal(resp, &hereResp); err != nil {
		return nil, fmt.Errorf("failed to parse routing response: %w", err)
	}

	if len(hereResp.Routes) == 0 {
		return nil, fmt.Errorf("no routes found")
	}

	return h.convertRoutingResponse(&hereResp), nil
}

// GetETA calculates the estimated time of arrival
func (h *HEREMapsProvider) GetETA(ctx context.Context, req *ETARequest) (*ETAResponse, error) {
	routeReq := &RouteRequest{
		Origin:        req.Origin,
		Destination:   req.Destination,
		DepartureTime: req.DepartureTime,
	}

	routeResp, err := h.GetRoute(ctx, routeReq)
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
		Provider:            ProviderHERE,
		Confidence:          0.92, // HERE has high accuracy
	}, nil
}

// GetDistanceMatrix calculates distances between multiple origins and destinations
func (h *HEREMapsProvider) GetDistanceMatrix(ctx context.Context, req *DistanceMatrixRequest) (*DistanceMatrixResponse, error) {
	// HERE requires separate requests for each origin-destination pair
	// or use the Matrix Routing API (different endpoint)
	rows := make([]DistanceMatrixRow, len(req.Origins))

	for i, origin := range req.Origins {
		elements := make([]DistanceMatrixElement, len(req.Destinations))

		for j, dest := range req.Destinations {
			etaReq := &ETARequest{
				Origin:        origin,
				Destination:   dest,
				DepartureTime: req.DepartureTime,
			}

			eta, err := h.GetETA(ctx, etaReq)
			if err != nil {
				elements[j] = DistanceMatrixElement{Status: "NOT_FOUND"}
				continue
			}

			elements[j] = DistanceMatrixElement{
				Status:            "OK",
				DistanceKm:        eta.DistanceKm,
				DistanceMeters:    eta.DistanceMeters,
				DurationMinutes:   eta.DurationMinutes,
				DurationSeconds:   eta.DurationSeconds,
				DurationInTraffic: eta.DurationInTraffic,
				TrafficLevel:      eta.TrafficLevel,
			}
		}

		rows[i] = DistanceMatrixRow{Elements: elements}
	}

	return &DistanceMatrixResponse{
		Rows:        rows,
		Provider:    ProviderHERE,
		RequestedAt: time.Now(),
	}, nil
}

// GetTrafficFlow returns traffic flow information
func (h *HEREMapsProvider) GetTrafficFlow(ctx context.Context, req *TrafficFlowRequest) (*TrafficFlowResponse, error) {
	params := url.Values{}
	params.Set("apiKey", h.apiKey)
	params.Set("responseattributes", "shape")

	if req.BoundingBox != nil {
		params.Set("bbox", fmt.Sprintf("%f,%f;%f,%f",
			req.BoundingBox.Southwest.Latitude, req.BoundingBox.Southwest.Longitude,
			req.BoundingBox.Northeast.Latitude, req.BoundingBox.Northeast.Longitude))
	} else if req.Location.Latitude != 0 && req.Location.Longitude != 0 {
		radius := req.RadiusMeters
		if radius <= 0 {
			radius = 5000
		}
		params.Set("prox", fmt.Sprintf("%f,%f,%d", req.Location.Latitude, req.Location.Longitude, radius))
	}

	resp, err := h.trafficClient.Get(ctx, "/flow.json?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("HERE traffic flow request failed: %w", err)
	}

	var hereResp hereTrafficFlowResponse
	if err := json.Unmarshal(resp, &hereResp); err != nil {
		return nil, fmt.Errorf("failed to parse traffic flow response: %w", err)
	}

	return h.convertTrafficFlowResponse(&hereResp), nil
}

// GetTrafficIncidents returns traffic incidents
func (h *HEREMapsProvider) GetTrafficIncidents(ctx context.Context, req *TrafficIncidentsRequest) (*TrafficIncidentsResponse, error) {
	params := url.Values{}
	params.Set("apiKey", h.apiKey)

	if req.BoundingBox != nil {
		params.Set("bbox", fmt.Sprintf("%f,%f;%f,%f",
			req.BoundingBox.Southwest.Latitude, req.BoundingBox.Southwest.Longitude,
			req.BoundingBox.Northeast.Latitude, req.BoundingBox.Northeast.Longitude))
	} else if req.Location != nil {
		radius := req.RadiusMeters
		if radius <= 0 {
			radius = 5000
		}
		params.Set("prox", fmt.Sprintf("%f,%f,%d", req.Location.Latitude, req.Location.Longitude, radius))
	}

	resp, err := h.trafficClient.Get(ctx, "/incidents.json?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("HERE traffic incidents request failed: %w", err)
	}

	var hereResp hereTrafficIncidentsResponse
	if err := json.Unmarshal(resp, &hereResp); err != nil {
		return nil, fmt.Errorf("failed to parse traffic incidents response: %w", err)
	}

	return h.convertTrafficIncidentsResponse(&hereResp), nil
}

// Geocode converts an address to coordinates
func (h *HEREMapsProvider) Geocode(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	params := url.Values{}
	params.Set("apiKey", h.apiKey)
	params.Set("q", req.Address)

	if req.Language != "" {
		params.Set("lang", req.Language)
	}

	if req.Region != "" {
		params.Set("in", "countryCode:"+req.Region)
	}

	params.Set("limit", "5")

	resp, err := h.geocodeClient.Get(ctx, "/geocode?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("HERE geocoding request failed: %w", err)
	}

	var hereResp hereGeocodingResponse
	if err := json.Unmarshal(resp, &hereResp); err != nil {
		return nil, fmt.Errorf("failed to parse geocoding response: %w", err)
	}

	return h.convertGeocodingResponse(&hereResp), nil
}

// ReverseGeocode converts coordinates to an address
func (h *HEREMapsProvider) ReverseGeocode(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	if req.Coordinate == nil {
		return nil, fmt.Errorf("coordinate is required for reverse geocoding")
	}

	params := url.Values{}
	params.Set("apiKey", h.apiKey)
	params.Set("at", formatHERECoordinate(*req.Coordinate))

	if req.Language != "" {
		params.Set("lang", req.Language)
	}

	params.Set("limit", "5")

	resp, err := h.geocodeClient.Get(ctx, "/revgeocode?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("HERE reverse geocoding request failed: %w", err)
	}

	var hereResp hereGeocodingResponse
	if err := json.Unmarshal(resp, &hereResp); err != nil {
		return nil, fmt.Errorf("failed to parse reverse geocoding response: %w", err)
	}

	return h.convertGeocodingResponse(&hereResp), nil
}

// SearchPlaces searches for nearby places
func (h *HEREMapsProvider) SearchPlaces(ctx context.Context, req *PlaceSearchRequest) (*PlaceSearchResponse, error) {
	params := url.Values{}
	params.Set("apiKey", h.apiKey)

	if req.Query != "" {
		params.Set("q", req.Query)
	}

	if req.Location != nil {
		params.Set("at", formatHERECoordinate(*req.Location))
	}

	if req.RadiusMeters > 0 {
		params.Set("limit", "20")
	}

	if req.Language != "" {
		params.Set("lang", req.Language)
	}

	resp, err := h.geocodeClient.Get(ctx, "/discover?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("HERE places search failed: %w", err)
	}

	var hereResp hereGeocodingResponse
	if err := json.Unmarshal(resp, &hereResp); err != nil {
		return nil, fmt.Errorf("failed to parse places response: %w", err)
	}

	return h.convertPlacesResponse(&hereResp), nil
}

// SnapToRoad snaps GPS coordinates to the nearest road
func (h *HEREMapsProvider) SnapToRoad(ctx context.Context, req *SnapToRoadRequest) (*SnapToRoadResponse, error) {
	// HERE doesn't have a direct snap-to-road API in the same way
	// We can use route matching or simply return the input
	points := make([]SnappedPoint, len(req.Path))
	for i, c := range req.Path {
		points[i] = SnappedPoint{
			Location:      c,
			OriginalIndex: i,
		}
	}

	return &SnapToRoadResponse{
		SnappedPoints: points,
		Provider:      ProviderHERE,
	}, nil
}

// GetSpeedLimits returns speed limits along a path
func (h *HEREMapsProvider) GetSpeedLimits(ctx context.Context, req *SpeedLimitsRequest) (*SpeedLimitsResponse, error) {
	// HERE speed limits are available through their Fleet Telematics API
	// For now, return empty as it requires different credentials
	return &SpeedLimitsResponse{
		SpeedLimits: []SpeedLimit{},
		Provider:    ProviderHERE,
	}, nil
}

// Helper functions

func formatHERECoordinate(c Coordinate) string {
	return fmt.Sprintf("%f,%f", c.Latitude, c.Longitude)
}

func (h *HEREMapsProvider) convertRoutingResponse(resp *hereRoutingResponse) *RouteResponse {
	routes := make([]Route, len(resp.Routes))

	for i, r := range resp.Routes {
		route := Route{}

		for _, section := range r.Sections {
			route.DistanceMeters += section.Summary.Length
			route.DurationSeconds += section.Summary.Duration
			if section.Summary.BaseDuration > 0 {
				route.DurationInTraffic += section.Summary.Duration
			}

			// Decode polyline
			if section.Polyline != "" {
				route.EncodedPolyline = section.Polyline
			}

			// Convert actions to steps
			for _, action := range section.Actions {
				step := RouteStep{
					Instruction:     action.Instruction,
					Maneuver:        action.Action,
					DistanceMeters:  action.Length,
					DurationSeconds: action.Duration,
				}
				route.Legs = append(route.Legs, RouteLeg{
					DistanceMeters:  action.Length,
					DurationSeconds: action.Duration,
					Steps:           []RouteStep{step},
				})
			}
		}

		route.DistanceKm = float64(route.DistanceMeters) / 1000
		route.DurationMinutes = float64(route.DurationSeconds) / 60
		if route.DurationInTraffic > 0 {
			route.DurationInTrafficMin = float64(route.DurationInTraffic) / 60
			route.TrafficDelaySeconds = route.DurationInTraffic - route.DurationSeconds
		}

		route.TrafficLevel = calculateTrafficLevel(route.DurationSeconds, route.DurationInTraffic)

		routes[i] = route
	}

	return &RouteResponse{
		Routes:      routes,
		Provider:    ProviderHERE,
		RequestedAt: time.Now(),
	}
}

func (h *HEREMapsProvider) convertGeocodingResponse(resp *hereGeocodingResponse) *GeocodingResponse {
	results := make([]GeocodingResult, len(resp.Items))

	for i, item := range resp.Items {
		result := GeocodingResult{
			FormattedAddress: item.Address.Label,
			Coordinate: Coordinate{
				Latitude:  item.Position.Lat,
				Longitude: item.Position.Lng,
			},
			PlaceID:    item.ID,
			Confidence: float64(item.Scoring.QueryScore),
		}

		// Add address components
		if item.Address.Country != "" {
			result.AddressComponents = append(result.AddressComponents, AddressComponent{
				LongName:  item.Address.Country,
				ShortName: item.Address.CountryCode,
				Types:     []string{"country"},
			})
		}
		if item.Address.State != "" {
			result.AddressComponents = append(result.AddressComponents, AddressComponent{
				LongName: item.Address.State,
				Types:    []string{"administrative_area_level_1"},
			})
		}
		if item.Address.City != "" {
			result.AddressComponents = append(result.AddressComponents, AddressComponent{
				LongName: item.Address.City,
				Types:    []string{"locality"},
			})
		}
		if item.Address.Street != "" {
			result.AddressComponents = append(result.AddressComponents, AddressComponent{
				LongName: item.Address.Street,
				Types:    []string{"route"},
			})
		}

		results[i] = result
	}

	return &GeocodingResponse{
		Results:  results,
		Provider: ProviderHERE,
	}
}

func (h *HEREMapsProvider) convertPlacesResponse(resp *hereGeocodingResponse) *PlaceSearchResponse {
	results := make([]Place, len(resp.Items))

	for i, item := range resp.Items {
		results[i] = Place{
			PlaceID:          item.ID,
			Name:             item.Title,
			FormattedAddress: item.Address.Label,
			Coordinate: Coordinate{
				Latitude:  item.Position.Lat,
				Longitude: item.Position.Lng,
			},
			Types: item.Categories,
		}
	}

	return &PlaceSearchResponse{
		Results:  results,
		Provider: ProviderHERE,
	}
}

func (h *HEREMapsProvider) convertTrafficFlowResponse(resp *hereTrafficFlowResponse) *TrafficFlowResponse {
	segments := make([]TrafficFlowSegment, 0)

	if resp.RWS != nil {
		for _, rw := range resp.RWS {
			for _, road := range rw.RW {
				for _, fi := range road.FIS {
					for _, flow := range fi.FI {
						segment := TrafficFlowSegment{
							RoadName:         road.DE,
							CurrentSpeedKmh:  float64(flow.CF[0].SU),
							FreeFlowSpeedKmh: float64(flow.CF[0].FF),
							JamFactor:        flow.CF[0].JF,
						}

						// Calculate traffic level from jam factor
						switch {
						case flow.CF[0].JF <= 2:
							segment.TrafficLevel = TrafficFreeFlow
						case flow.CF[0].JF <= 4:
							segment.TrafficLevel = TrafficLight
						case flow.CF[0].JF <= 6:
							segment.TrafficLevel = TrafficModerate
						case flow.CF[0].JF <= 8:
							segment.TrafficLevel = TrafficHeavy
						case flow.CF[0].JF <= 10:
							segment.TrafficLevel = TrafficSevere
						default:
							segment.TrafficLevel = TrafficBlocked
						}

						segments = append(segments, segment)
					}
				}
			}
		}
	}

	// Determine overall traffic level
	overallLevel := TrafficFreeFlow
	if len(segments) > 0 {
		var totalJam float64
		for _, s := range segments {
			totalJam += s.JamFactor
		}
		avgJam := totalJam / float64(len(segments))
		switch {
		case avgJam <= 2:
			overallLevel = TrafficFreeFlow
		case avgJam <= 4:
			overallLevel = TrafficLight
		case avgJam <= 6:
			overallLevel = TrafficModerate
		case avgJam <= 8:
			overallLevel = TrafficHeavy
		default:
			overallLevel = TrafficSevere
		}
	}

	return &TrafficFlowResponse{
		Segments:     segments,
		OverallLevel: overallLevel,
		UpdatedAt:    time.Now(),
		Provider:     ProviderHERE,
	}
}

func (h *HEREMapsProvider) convertTrafficIncidentsResponse(resp *hereTrafficIncidentsResponse) *TrafficIncidentsResponse {
	incidents := make([]TrafficIncident, 0)

	if resp.TRAFFICITEMS != nil {
		for _, ti := range resp.TRAFFICITEMS.TRAFFICITEM {
			incident := TrafficIncident{
				ID:          strconv.Itoa(ti.TRAFFICITEMID),
				Type:        ti.TRAFFICITEMTYPEDESC,
				Description: ti.TRAFFICITEMDESCRIPTION.Value,
				UpdatedAt:   time.Now(),
			}

			// Set severity
			switch ti.CRITICALITY.ID {
			case "0", "1":
				incident.Severity = "minor"
			case "2":
				incident.Severity = "moderate"
			case "3":
				incident.Severity = "major"
			default:
				incident.Severity = "critical"
			}

			// Set location from origin
			if ti.LOCATION.GEOLOC.ORIGIN != nil {
				incident.Location = Coordinate{
					Latitude:  ti.LOCATION.GEOLOC.ORIGIN.LATITUDE,
					Longitude: ti.LOCATION.GEOLOC.ORIGIN.LONGITUDE,
				}
			}

			// Parse times if available
			if ti.STARTTIME != "" {
				if t, err := time.Parse(time.RFC3339, ti.STARTTIME); err == nil {
					incident.StartTime = &t
				}
			}
			if ti.ENDTIME != "" {
				if t, err := time.Parse(time.RFC3339, ti.ENDTIME); err == nil {
					incident.EndTime = &t
				}
			}

			incident.RoadClosed = ti.TRAFFICITEMTYPEDESC == "ROAD_CLOSURE"

			incidents = append(incidents, incident)
		}
	}

	return &TrafficIncidentsResponse{
		Incidents: incidents,
		UpdatedAt: time.Now(),
		Provider:  ProviderHERE,
	}
}

// HERE Maps API response structures

type hereRoutingResponse struct {
	Routes []hereRoute `json:"routes"`
}

type hereRoute struct {
	ID       string        `json:"id"`
	Sections []hereSection `json:"sections"`
}

type hereSection struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Departure herePlace   `json:"departure"`
	Arrival   herePlace   `json:"arrival"`
	Summary   hereSummary `json:"summary"`
	Polyline  string      `json:"polyline"`
	Actions   []hereAction `json:"actions"`
}

type herePlace struct {
	Time  string       `json:"time"`
	Place hereLocation `json:"place"`
}

type hereLocation struct {
	Type          string       `json:"type"`
	Location      herePosition `json:"location"`
	OriginalPlace herePosition `json:"originalLocation"`
}

type herePosition struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type hereSummary struct {
	Duration     int `json:"duration"`
	Length       int `json:"length"`
	BaseDuration int `json:"baseDuration"`
}

type hereAction struct {
	Action      string `json:"action"`
	Duration    int    `json:"duration"`
	Length      int    `json:"length"`
	Instruction string `json:"instruction"`
	Direction   string `json:"direction"`
}

type hereGeocodingResponse struct {
	Items []hereGeocodingItem `json:"items"`
}

type hereGeocodingItem struct {
	Title      string        `json:"title"`
	ID         string        `json:"id"`
	ResultType string        `json:"resultType"`
	Address    hereAddress   `json:"address"`
	Position   herePosition  `json:"position"`
	Scoring    hereScoring   `json:"scoring"`
	Categories []string      `json:"categories"`
}

type hereAddress struct {
	Label       string `json:"label"`
	CountryCode string `json:"countryCode"`
	Country     string `json:"countryName"`
	State       string `json:"state"`
	City        string `json:"city"`
	District    string `json:"district"`
	Street      string `json:"street"`
	PostalCode  string `json:"postalCode"`
}

type hereScoring struct {
	QueryScore float64 `json:"queryScore"`
	FieldScore hereFieldScore `json:"fieldScore"`
}

type hereFieldScore struct {
	Country    float64 `json:"country"`
	City       float64 `json:"city"`
	Streets    []float64 `json:"streets"`
	HouseNumber float64 `json:"houseNumber"`
}

type hereTrafficFlowResponse struct {
	RWS []hereRWS `json:"RWS"`
}

type hereRWS struct {
	RW []hereRW `json:"RW"`
}

type hereRW struct {
	DE  string    `json:"DE"` // Description
	FIS []hereFIS `json:"FIS"`
}

type hereFIS struct {
	FI []hereFI `json:"FI"`
}

type hereFI struct {
	CF []hereCF `json:"CF"` // Current flow
}

type hereCF struct {
	SP float64 `json:"SP"` // Speed (capped)
	SU float64 `json:"SU"` // Speed (uncapped)
	FF float64 `json:"FF"` // Free flow speed
	JF float64 `json:"JF"` // Jam factor (0-10)
	CN float64 `json:"CN"` // Confidence
}

type hereTrafficIncidentsResponse struct {
	TRAFFICITEMS *hereTrafficItems `json:"TRAFFIC_ITEMS"`
}

type hereTrafficItems struct {
	TRAFFICITEM []hereTrafficItem `json:"TRAFFIC_ITEM"`
}

type hereTrafficItem struct {
	TRAFFICITEMID           int                        `json:"TRAFFIC_ITEM_ID"`
	TRAFFICITEMTYPEDESC     string                     `json:"TRAFFIC_ITEM_TYPE_DESC"`
	TRAFFICITEMDESCRIPTION  hereTrafficDescription     `json:"TRAFFIC_ITEM_DESCRIPTION"`
	LOCATION                hereTrafficLocation        `json:"LOCATION"`
	CRITICALITY             hereTrafficCriticality     `json:"CRITICALITY"`
	STARTTIME               string                     `json:"START_TIME"`
	ENDTIME                 string                     `json:"END_TIME"`
}

type hereTrafficDescription struct {
	Value string `json:"value"`
}

type hereTrafficLocation struct {
	GEOLOC hereGeoLoc `json:"GEOLOC"`
}

type hereGeoLoc struct {
	ORIGIN *hereGeoPoint `json:"ORIGIN"`
}

type hereGeoPoint struct {
	LATITUDE  float64 `json:"LATITUDE"`
	LONGITUDE float64 `json:"LONGITUDE"`
}

type hereTrafficCriticality struct {
	ID          string `json:"ID"`
	DESCRIPTION string `json:"DESCRIPTION"`
}
