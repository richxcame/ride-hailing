package geo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	redisClient "github.com/richxcame/ride-hailing/pkg/redis"
	"github.com/richxcame/ride-hailing/pkg/resilience"
	"go.uber.org/zap"
)

const (
	geocodeCachePrefix = "geocode:"
	geocodeCacheTTL    = 24 * time.Hour
	autocompletePrefix = "autocomplete:"
	autocompleteTTL    = 1 * time.Hour

	googleGeocodingURL    = "https://maps.googleapis.com/maps/api/geocode/json"
	googleAutocompleteURL = "https://maps.googleapis.com/maps/api/place/autocomplete/json"
	googlePlaceDetailURL  = "https://maps.googleapis.com/maps/api/place/details/json"
)

// GeocodingResult represents a geocoded address result.
type GeocodingResult struct {
	PlaceID          string   `json:"place_id"`
	FormattedAddress string   `json:"formatted_address"`
	Latitude         float64  `json:"latitude"`
	Longitude        float64  `json:"longitude"`
	Types            []string `json:"types"`
	H3Cell           string   `json:"h3_cell,omitempty"`
}

// AutocompleteResult represents a place autocomplete suggestion.
type AutocompleteResult struct {
	PlaceID     string `json:"place_id"`
	Description string `json:"description"`
	MainText    string `json:"main_text"`
	SecondText  string `json:"secondary_text"`
}

// GeocodingService handles address geocoding and autocomplete.
type GeocodingService struct {
	apiKey     string
	httpClient *http.Client
	redis      redisClient.ClientInterface
	breaker    *resilience.CircuitBreaker
	// Optional: restrict results to a region/country
	RegionBias   string // e.g. "tm" for Turkmenistan
	LanguageBias string // e.g. "tk" for Turkmen
}

// NewGeocodingService creates a new geocoding service.
func NewGeocodingService(apiKey string, redis redisClient.ClientInterface) *GeocodingService {
	return &GeocodingService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		redis: redis,
	}
}

// SetCircuitBreaker enables circuit breaker protection for external API calls.
func (g *GeocodingService) SetCircuitBreaker(cb *resilience.CircuitBreaker) {
	g.breaker = cb
}

// ForwardGeocode converts an address string to coordinates.
func (g *GeocodingService) ForwardGeocode(ctx context.Context, address string) ([]*GeocodingResult, error) {
	if address == "" {
		return nil, common.NewBadRequestError("address is required", nil)
	}

	// Check cache
	cacheKey := fmt.Sprintf("%sforward:%s", geocodeCachePrefix, strings.ToLower(strings.TrimSpace(address)))
	if cached, err := g.getCachedResults(ctx, cacheKey); err == nil {
		return cached, nil
	}

	params := url.Values{}
	params.Set("address", address)
	params.Set("key", g.apiKey)
	if g.RegionBias != "" {
		params.Set("region", g.RegionBias)
	}
	if g.LanguageBias != "" {
		params.Set("language", g.LanguageBias)
	}

	reqURL := fmt.Sprintf("%s?%s", googleGeocodingURL, params.Encode())
	results, err := g.fetchGeocodingResults(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	// Cache results
	g.cacheResults(ctx, cacheKey, results, geocodeCacheTTL)
	return results, nil
}

// ReverseGeocode converts coordinates to an address.
func (g *GeocodingService) ReverseGeocode(ctx context.Context, latitude, longitude float64) ([]*GeocodingResult, error) {
	// Check cache
	cacheKey := fmt.Sprintf("%sreverse:%.6f,%.6f", geocodeCachePrefix, latitude, longitude)
	if cached, err := g.getCachedResults(ctx, cacheKey); err == nil {
		return cached, nil
	}

	params := url.Values{}
	params.Set("latlng", fmt.Sprintf("%f,%f", latitude, longitude))
	params.Set("key", g.apiKey)
	if g.LanguageBias != "" {
		params.Set("language", g.LanguageBias)
	}

	reqURL := fmt.Sprintf("%s?%s", googleGeocodingURL, params.Encode())
	results, err := g.fetchGeocodingResults(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	g.cacheResults(ctx, cacheKey, results, geocodeCacheTTL)
	return results, nil
}

// Autocomplete returns place suggestions for a search query.
func (g *GeocodingService) Autocomplete(ctx context.Context, input string, latitude, longitude float64) ([]*AutocompleteResult, error) {
	if input == "" {
		return nil, common.NewBadRequestError("input is required", nil)
	}

	// Check cache
	cacheKey := fmt.Sprintf("%s%.4f,%.4f:%s", autocompletePrefix, latitude, longitude, strings.ToLower(input))
	if cached, err := g.getCachedAutocomplete(ctx, cacheKey); err == nil {
		return cached, nil
	}

	params := url.Values{}
	params.Set("input", input)
	params.Set("key", g.apiKey)
	if latitude != 0 && longitude != 0 {
		params.Set("location", fmt.Sprintf("%f,%f", latitude, longitude))
		params.Set("radius", "50000") // 50km bias radius
	}
	if g.RegionBias != "" {
		params.Set("components", fmt.Sprintf("country:%s", g.RegionBias))
	}
	if g.LanguageBias != "" {
		params.Set("language", g.LanguageBias)
	}

	reqURL := fmt.Sprintf("%s?%s", googleAutocompleteURL, params.Encode())

	body, err := g.doRequest(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	var apiResp struct {
		Status      string `json:"status"`
		Predictions []struct {
			PlaceID            string `json:"place_id"`
			Description        string `json:"description"`
			StructuredFormatting struct {
				MainText      string `json:"main_text"`
				SecondaryText string `json:"secondary_text"`
			} `json:"structured_formatting"`
		} `json:"predictions"`
		ErrorMessage string `json:"error_message"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, common.NewInternalServerError("failed to parse geocoding response")
	}

	if apiResp.Status != "OK" && apiResp.Status != "ZERO_RESULTS" {
		logger.WarnContext(ctx, "geocoding API error", zap.String("status", apiResp.Status), zap.String("error", apiResp.ErrorMessage))
		return nil, common.NewInternalServerError(fmt.Sprintf("geocoding API error: %s", apiResp.Status))
	}

	results := make([]*AutocompleteResult, 0, len(apiResp.Predictions))
	for _, p := range apiResp.Predictions {
		results = append(results, &AutocompleteResult{
			PlaceID:     p.PlaceID,
			Description: p.Description,
			MainText:    p.StructuredFormatting.MainText,
			SecondText:  p.StructuredFormatting.SecondaryText,
		})
	}

	g.cacheAutocomplete(ctx, cacheKey, results, autocompleteTTL)
	return results, nil
}

// GetPlaceDetails returns geocoding details for a place ID.
func (g *GeocodingService) GetPlaceDetails(ctx context.Context, placeID string) (*GeocodingResult, error) {
	if placeID == "" {
		return nil, common.NewBadRequestError("place_id is required", nil)
	}

	cacheKey := fmt.Sprintf("%splace:%s", geocodeCachePrefix, placeID)
	if cached, err := g.getCachedResults(ctx, cacheKey); err == nil && len(cached) > 0 {
		return cached[0], nil
	}

	params := url.Values{}
	params.Set("place_id", placeID)
	params.Set("fields", "geometry,formatted_address,place_id,types")
	params.Set("key", g.apiKey)

	reqURL := fmt.Sprintf("%s?%s", googlePlaceDetailURL, params.Encode())

	body, err := g.doRequest(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	var apiResp struct {
		Status string `json:"status"`
		Result struct {
			PlaceID          string   `json:"place_id"`
			FormattedAddress string   `json:"formatted_address"`
			Types            []string `json:"types"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"result"`
		ErrorMessage string `json:"error_message"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, common.NewInternalServerError("failed to parse place details response")
	}

	if apiResp.Status != "OK" {
		return nil, common.NewNotFoundError("place not found", nil)
	}

	result := &GeocodingResult{
		PlaceID:          apiResp.Result.PlaceID,
		FormattedAddress: apiResp.Result.FormattedAddress,
		Latitude:         apiResp.Result.Geometry.Location.Lat,
		Longitude:        apiResp.Result.Geometry.Location.Lng,
		Types:            apiResp.Result.Types,
		H3Cell:           GetMatchingCell(apiResp.Result.Geometry.Location.Lat, apiResp.Result.Geometry.Location.Lng),
	}

	g.cacheResults(ctx, cacheKey, []*GeocodingResult{result}, geocodeCacheTTL)
	return result, nil
}

// fetchGeocodingResults calls the Google Geocoding API and parses results.
func (g *GeocodingService) fetchGeocodingResults(ctx context.Context, reqURL string) ([]*GeocodingResult, error) {
	body, err := g.doRequest(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	var apiResp struct {
		Status  string `json:"status"`
		Results []struct {
			PlaceID          string   `json:"place_id"`
			FormattedAddress string   `json:"formatted_address"`
			Types            []string `json:"types"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"results"`
		ErrorMessage string `json:"error_message"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, common.NewInternalServerError("failed to parse geocoding response")
	}

	if apiResp.Status != "OK" && apiResp.Status != "ZERO_RESULTS" {
		logger.WarnContext(ctx, "geocoding API error", zap.String("status", apiResp.Status), zap.String("error", apiResp.ErrorMessage))
		return nil, common.NewInternalServerError(fmt.Sprintf("geocoding API error: %s", apiResp.Status))
	}

	results := make([]*GeocodingResult, 0, len(apiResp.Results))
	for _, r := range apiResp.Results {
		results = append(results, &GeocodingResult{
			PlaceID:          r.PlaceID,
			FormattedAddress: r.FormattedAddress,
			Latitude:         r.Geometry.Location.Lat,
			Longitude:        r.Geometry.Location.Lng,
			Types:            r.Types,
			H3Cell:           GetMatchingCell(r.Geometry.Location.Lat, r.Geometry.Location.Lng),
		})
	}

	return results, nil
}

// doRequest executes an HTTP request, optionally wrapped by the circuit breaker.
func (g *GeocodingService) doRequest(ctx context.Context, reqURL string) ([]byte, error) {
	call := func(_ context.Context) (interface{}, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := g.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}

	if g.breaker != nil {
		result, err := g.breaker.Execute(ctx, call)
		if err != nil {
			return nil, common.NewInternalErrorWithError("geocoding API circuit open or request failed", err)
		}
		return result.([]byte), nil
	}

	result, err := call(ctx)
	if err != nil {
		return nil, common.NewInternalErrorWithError("geocoding API request failed", err)
	}
	return result.([]byte), nil
}

// Cache helpers

func (g *GeocodingService) getCachedResults(ctx context.Context, key string) ([]*GeocodingResult, error) {
	if g.redis == nil {
		return nil, fmt.Errorf("no cache")
	}
	data, err := g.redis.GetString(ctx, key)
	if err != nil {
		return nil, err
	}
	var results []*GeocodingResult
	if err := json.Unmarshal([]byte(data), &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (g *GeocodingService) cacheResults(ctx context.Context, key string, results []*GeocodingResult, ttl time.Duration) {
	if g.redis == nil || len(results) == 0 {
		return
	}
	data, err := json.Marshal(results)
	if err != nil {
		return
	}
	g.redis.SetWithExpiration(ctx, key, data, ttl)
}

func (g *GeocodingService) getCachedAutocomplete(ctx context.Context, key string) ([]*AutocompleteResult, error) {
	if g.redis == nil {
		return nil, fmt.Errorf("no cache")
	}
	data, err := g.redis.GetString(ctx, key)
	if err != nil {
		return nil, err
	}
	var results []*AutocompleteResult
	if err := json.Unmarshal([]byte(data), &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (g *GeocodingService) cacheAutocomplete(ctx context.Context, key string, results []*AutocompleteResult, ttl time.Duration) {
	if g.redis == nil || len(results) == 0 {
		return
	}
	data, err := json.Marshal(results)
	if err != nil {
		return
	}
	g.redis.SetWithExpiration(ctx, key, data, ttl)
}
