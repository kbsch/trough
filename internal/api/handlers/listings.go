package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kbsch/trough/internal/domain"
	"github.com/kbsch/trough/internal/repository"
)

type ListingHandler struct {
	repo *repository.ListingRepository
}

func NewListingHandler(repo *repository.ListingRepository) *ListingHandler {
	return &ListingHandler{repo: repo}
}

func (h *ListingHandler) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := parseSearchParams(r)

	result, err := h.repo.Search(ctx, params)
	if err != nil {
		InternalError(w, r, "Failed to search listings")
		return
	}

	JSON(w, http.StatusOK, result)
}

func (h *ListingHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(w, r, "Invalid listing ID format")
		return
	}

	listing, err := h.repo.GetByID(ctx, id)
	if err != nil {
		NotFound(w, r, "Listing not found")
		return
	}

	Success(w, listing)
}

func (h *ListingHandler) MapView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := parseSearchParams(r)

	// For map view, we want more results but less data per result
	params.PerPage = 1000

	result, err := h.repo.Search(ctx, params)
	if err != nil {
		InternalError(w, r, "Failed to fetch map data")
		return
	}

	// Transform to map markers (lighter weight)
	markers := make([]MapMarker, 0, len(result.Listings))
	for _, l := range result.Listings {
		if l.Lat != nil && l.Lng != nil {
			markers = append(markers, MapMarker{
				ID:          l.ID,
				Lat:         *l.Lat,
				Lng:         *l.Lng,
				Title:       l.Title,
				AskingPrice: l.AskingPrice,
				Industry:    l.Industry,
				City:        l.City,
				State:       l.State,
			})
		}
	}

	Success(w, map[string]interface{}{
		"markers": markers,
		"total":   len(markers),
		"bounds":  calculateBounds(markers),
	})
}

func (h *ListingHandler) GetFilters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filters, err := h.repo.GetFilterOptions(ctx)
	if err != nil {
		InternalError(w, r, "Failed to fetch filter options")
		return
	}

	Success(w, filters)
}

type MapMarker struct {
	ID          uuid.UUID `json:"id"`
	Lat         float64   `json:"lat"`
	Lng         float64   `json:"lng"`
	Title       string    `json:"title"`
	AskingPrice *int64    `json:"asking_price,omitempty"`
	Industry    string    `json:"industry,omitempty"`
	City        string    `json:"city,omitempty"`
	State       string    `json:"state,omitempty"`
}

type MapBounds struct {
	North float64 `json:"north"`
	South float64 `json:"south"`
	East  float64 `json:"east"`
	West  float64 `json:"west"`
}

func calculateBounds(markers []MapMarker) *MapBounds {
	if len(markers) == 0 {
		return nil
	}

	bounds := &MapBounds{
		North: markers[0].Lat,
		South: markers[0].Lat,
		East:  markers[0].Lng,
		West:  markers[0].Lng,
	}

	for _, m := range markers {
		if m.Lat > bounds.North {
			bounds.North = m.Lat
		}
		if m.Lat < bounds.South {
			bounds.South = m.Lat
		}
		if m.Lng > bounds.East {
			bounds.East = m.Lng
		}
		if m.Lng < bounds.West {
			bounds.West = m.Lng
		}
	}

	return bounds
}

func parseSearchParams(r *http.Request) domain.ListingSearchParams {
	q := r.URL.Query()

	params := domain.ListingSearchParams{
		Query:   q.Get("q"),
		Sort:    q.Get("sort"),
		Page:    1,
		PerPage: 24,
	}

	if v := q.Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			params.Page = p
		}
	}

	if v := q.Get("per_page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 && p <= 100 {
			params.PerPage = p
		}
	}

	if v := q.Get("price_min"); v != "" {
		if p, err := strconv.ParseInt(v, 10, 64); err == nil {
			params.PriceMin = &p
		}
	}

	if v := q.Get("price_max"); v != "" {
		if p, err := strconv.ParseInt(v, 10, 64); err == nil {
			params.PriceMax = &p
		}
	}

	if v := q.Get("revenue_min"); v != "" {
		if p, err := strconv.ParseInt(v, 10, 64); err == nil {
			params.RevenueMin = &p
		}
	}

	if v := q.Get("cash_flow_min"); v != "" {
		if p, err := strconv.ParseInt(v, 10, 64); err == nil {
			params.CashFlowMin = &p
		}
	}

	if v := q.Get("state"); v != "" {
		params.States = strings.Split(v, ",")
	}

	if v := q.Get("industry"); v != "" {
		params.Industries = strings.Split(v, ",")
	}

	if v := q.Get("franchise"); v != "" {
		b := v == "true"
		params.Franchise = &b
	}

	if v := q.Get("real_estate"); v != "" {
		b := v == "true"
		params.RealEstate = &b
	}

	if v := q.Get("bounds"); v != "" {
		parts := strings.Split(v, ",")
		if len(parts) == 4 {
			south, _ := strconv.ParseFloat(parts[0], 64)
			west, _ := strconv.ParseFloat(parts[1], 64)
			north, _ := strconv.ParseFloat(parts[2], 64)
			east, _ := strconv.ParseFloat(parts[3], 64)
			params.Bounds = &domain.GeoBounds{
				SouthLat: south,
				WestLng:  west,
				NorthLat: north,
				EastLng:  east,
			}
		}
	}

	return params
}
