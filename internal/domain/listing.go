package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Listing struct {
	ID         uuid.UUID `json:"id" db:"id"`
	SourceID   uuid.UUID `json:"source_id" db:"source_id"`
	ExternalID string    `json:"external_id" db:"external_id"`
	URL        string    `json:"url" db:"url"`

	// Core fields
	Title       string `json:"title" db:"title"`
	Description string `json:"description,omitempty" db:"description"`
	AskingPrice *int64 `json:"asking_price,omitempty" db:"asking_price"` // cents
	Revenue     *int64 `json:"revenue,omitempty" db:"revenue"`           // cents, annual
	CashFlow    *int64 `json:"cash_flow,omitempty" db:"cash_flow"`       // cents, annual (SDE/EBITDA)
	EBITDA      *int64 `json:"ebitda,omitempty" db:"ebitda"`             // cents
	Inventory   *int64 `json:"inventory_value,omitempty" db:"inventory_value"`

	// Real estate
	RealEstateIncluded bool   `json:"real_estate_included" db:"real_estate_included"`
	RealEstateValue    *int64 `json:"real_estate_value,omitempty" db:"real_estate_value"`

	// Location
	City     string   `json:"city,omitempty" db:"city"`
	State    string   `json:"state,omitempty" db:"state"`
	ZipCode  string   `json:"zip_code,omitempty" db:"zip_code"`
	Country  string   `json:"country" db:"country"`
	Lat      *float64 `json:"lat,omitempty" db:"lat"`
	Lng      *float64 `json:"lng,omitempty" db:"lng"`

	// Business details
	Industry         string `json:"industry,omitempty" db:"industry"`
	IndustryCategory string `json:"industry_category,omitempty" db:"industry_category"`
	BusinessType     string `json:"business_type,omitempty" db:"business_type"`
	YearEstablished  *int   `json:"year_established,omitempty" db:"year_established"`
	Employees        *int   `json:"employees,omitempty" db:"employees"`
	ReasonForSale    string `json:"reason_for_sale,omitempty" db:"reason_for_sale"`

	// Lease
	LeaseExpiration *time.Time `json:"lease_expiration,omitempty" db:"lease_expiration"`
	MonthlyRent     *int64     `json:"monthly_rent,omitempty" db:"monthly_rent"`

	// Franchise
	IsFranchise   bool   `json:"is_franchise" db:"is_franchise"`
	FranchiseName string `json:"franchise_name,omitempty" db:"franchise_name"`

	// Raw data
	RawData json.RawMessage `json:"raw_data,omitempty" db:"raw_data"`

	// Metadata
	FirstSeenAt time.Time `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt  time.Time `json:"last_seen_at" db:"last_seen_at"`
	IsActive    bool      `json:"is_active" db:"is_active"`
}

type ListingSearchParams struct {
	Query       string   `json:"q"`
	PriceMin    *int64   `json:"price_min"`
	PriceMax    *int64   `json:"price_max"`
	RevenueMin  *int64   `json:"revenue_min"`
	CashFlowMin *int64   `json:"cash_flow_min"`
	States      []string `json:"states"`
	Industries  []string `json:"industries"`
	Franchise   *bool    `json:"franchise"`
	RealEstate  *bool    `json:"real_estate"`
	Bounds      *GeoBounds `json:"bounds"`
	Sort        string   `json:"sort"`
	Page        int      `json:"page"`
	PerPage     int      `json:"per_page"`
}

type GeoBounds struct {
	SouthLat float64 `json:"south_lat"`
	WestLng  float64 `json:"west_lng"`
	NorthLat float64 `json:"north_lat"`
	EastLng  float64 `json:"east_lng"`
}

type ListingSearchResult struct {
	Listings   []Listing `json:"listings"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PerPage    int       `json:"per_page"`
	TotalPages int       `json:"total_pages"`
}

type FilterOptions struct {
	Industries []FilterOption `json:"industries"`
	States     []FilterOption `json:"states"`
	PriceRange PriceRange     `json:"price_range"`
}

type FilterOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

type PriceRange struct {
	Min int64 `json:"min"`
	Max int64 `json:"max"`
}
