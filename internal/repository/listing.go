package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/kbsch/trough/internal/domain"
)

type ListingRepository struct {
	db *sqlx.DB
}

func NewListingRepository(db *sqlx.DB) *ListingRepository {
	return &ListingRepository{db: db}
}

func (r *ListingRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Listing, error) {
	var listing domain.Listing
	err := r.db.GetContext(ctx, &listing, `
		SELECT * FROM listings WHERE id = $1 AND is_active = true
	`, id)
	if err != nil {
		return nil, err
	}
	return &listing, nil
}

func (r *ListingRepository) Search(ctx context.Context, params domain.ListingSearchParams) (*domain.ListingSearchResult, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "is_active = true")

	if params.Query != "" {
		conditions = append(conditions, fmt.Sprintf("search_vector @@ plainto_tsquery('english', $%d)", argIdx))
		args = append(args, params.Query)
		argIdx++
	}

	if params.PriceMin != nil {
		conditions = append(conditions, fmt.Sprintf("asking_price >= $%d", argIdx))
		args = append(args, *params.PriceMin)
		argIdx++
	}

	if params.PriceMax != nil {
		conditions = append(conditions, fmt.Sprintf("asking_price <= $%d", argIdx))
		args = append(args, *params.PriceMax)
		argIdx++
	}

	if params.RevenueMin != nil {
		conditions = append(conditions, fmt.Sprintf("revenue >= $%d", argIdx))
		args = append(args, *params.RevenueMin)
		argIdx++
	}

	if params.CashFlowMin != nil {
		conditions = append(conditions, fmt.Sprintf("cash_flow >= $%d", argIdx))
		args = append(args, *params.CashFlowMin)
		argIdx++
	}

	if len(params.States) > 0 {
		placeholders := make([]string, len(params.States))
		for i, s := range params.States {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, s)
			argIdx++
		}
		conditions = append(conditions, fmt.Sprintf("state IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(params.Industries) > 0 {
		placeholders := make([]string, len(params.Industries))
		for i, s := range params.Industries {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, s)
			argIdx++
		}
		conditions = append(conditions, fmt.Sprintf("industry IN (%s)", strings.Join(placeholders, ",")))
	}

	if params.Franchise != nil && *params.Franchise {
		conditions = append(conditions, "is_franchise = true")
	}

	if params.RealEstate != nil && *params.RealEstate {
		conditions = append(conditions, "real_estate_included = true")
	}

	if params.Bounds != nil {
		conditions = append(conditions, fmt.Sprintf(
			"lat BETWEEN $%d AND $%d AND lng BETWEEN $%d AND $%d",
			argIdx, argIdx+1, argIdx+2, argIdx+3,
		))
		args = append(args, params.Bounds.SouthLat, params.Bounds.NorthLat, params.Bounds.WestLng, params.Bounds.EastLng)
		argIdx += 4
	}

	whereClause := strings.Join(conditions, " AND ")

	// Order by
	orderBy := "last_seen_at DESC"
	switch params.Sort {
	case "price_asc":
		orderBy = "asking_price ASC NULLS LAST"
	case "price_desc":
		orderBy = "asking_price DESC NULLS LAST"
	case "newest":
		orderBy = "first_seen_at DESC"
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM listings WHERE %s", whereClause)
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, err
	}

	// Main query with pagination
	offset := (params.Page - 1) * params.PerPage
	query := fmt.Sprintf(`
		SELECT * FROM listings
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argIdx, argIdx+1)
	args = append(args, params.PerPage, offset)

	var listings []domain.Listing
	if err := r.db.SelectContext(ctx, &listings, query, args...); err != nil {
		return nil, err
	}

	totalPages := (total + params.PerPage - 1) / params.PerPage

	return &domain.ListingSearchResult{
		Listings:   listings,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

func (r *ListingRepository) GetFilterOptions(ctx context.Context) (*domain.FilterOptions, error) {
	var industries []domain.FilterOption
	err := r.db.SelectContext(ctx, &industries, `
		SELECT industry as value, industry as label, COUNT(*) as count
		FROM listings
		WHERE is_active = true AND industry IS NOT NULL AND industry != ''
		GROUP BY industry
		ORDER BY count DESC
		LIMIT 50
	`)
	if err != nil {
		return nil, err
	}

	var states []domain.FilterOption
	err = r.db.SelectContext(ctx, &states, `
		SELECT state as value, state as label, COUNT(*) as count
		FROM listings
		WHERE is_active = true AND state IS NOT NULL AND state != ''
		GROUP BY state
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}

	var priceRange domain.PriceRange
	err = r.db.GetContext(ctx, &priceRange, `
		SELECT COALESCE(MIN(asking_price), 0) as min, COALESCE(MAX(asking_price), 0) as max
		FROM listings
		WHERE is_active = true AND asking_price IS NOT NULL
	`)
	if err != nil {
		return nil, err
	}

	return &domain.FilterOptions{
		Industries: industries,
		States:     states,
		PriceRange: priceRange,
	}, nil
}

func (r *ListingRepository) Upsert(ctx context.Context, listing *domain.Listing) error {
	query := `
		INSERT INTO listings (
			id, source_id, external_id, url, title, description,
			asking_price, revenue, cash_flow, ebitda, inventory_value,
			real_estate_included, real_estate_value,
			city, state, zip_code, country, lat, lng,
			industry, industry_category, business_type, year_established, employees, reason_for_sale,
			lease_expiration, monthly_rent,
			is_franchise, franchise_name,
			raw_data, first_seen_at, last_seen_at, is_active,
			search_vector
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13,
			$14, $15, $16, $17, $18, $19,
			$20, $21, $22, $23, $24, $25,
			$26, $27,
			$28, $29,
			$30, $31, $32, $33,
			to_tsvector('english', COALESCE($5, '') || ' ' || COALESCE($6, '') || ' ' || COALESCE($20, ''))
		)
		ON CONFLICT (source_id, external_id) DO UPDATE SET
			url = EXCLUDED.url,
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			asking_price = EXCLUDED.asking_price,
			revenue = EXCLUDED.revenue,
			cash_flow = EXCLUDED.cash_flow,
			ebitda = EXCLUDED.ebitda,
			inventory_value = EXCLUDED.inventory_value,
			real_estate_included = EXCLUDED.real_estate_included,
			real_estate_value = EXCLUDED.real_estate_value,
			city = EXCLUDED.city,
			state = EXCLUDED.state,
			zip_code = EXCLUDED.zip_code,
			lat = EXCLUDED.lat,
			lng = EXCLUDED.lng,
			industry = EXCLUDED.industry,
			industry_category = EXCLUDED.industry_category,
			business_type = EXCLUDED.business_type,
			year_established = EXCLUDED.year_established,
			employees = EXCLUDED.employees,
			reason_for_sale = EXCLUDED.reason_for_sale,
			lease_expiration = EXCLUDED.lease_expiration,
			monthly_rent = EXCLUDED.monthly_rent,
			is_franchise = EXCLUDED.is_franchise,
			franchise_name = EXCLUDED.franchise_name,
			raw_data = EXCLUDED.raw_data,
			last_seen_at = EXCLUDED.last_seen_at,
			is_active = true,
			search_vector = to_tsvector('english', COALESCE(EXCLUDED.title, '') || ' ' || COALESCE(EXCLUDED.description, '') || ' ' || COALESCE(EXCLUDED.industry, ''))
	`

	_, err := r.db.ExecContext(ctx, query,
		listing.ID, listing.SourceID, listing.ExternalID, listing.URL, listing.Title, listing.Description,
		listing.AskingPrice, listing.Revenue, listing.CashFlow, listing.EBITDA, listing.Inventory,
		listing.RealEstateIncluded, listing.RealEstateValue,
		listing.City, listing.State, listing.ZipCode, listing.Country, listing.Lat, listing.Lng,
		listing.Industry, listing.IndustryCategory, listing.BusinessType, listing.YearEstablished, listing.Employees, listing.ReasonForSale,
		listing.LeaseExpiration, listing.MonthlyRent,
		listing.IsFranchise, listing.FranchiseName,
		listing.RawData, listing.FirstSeenAt, listing.LastSeenAt, listing.IsActive,
	)
	return err
}

func (r *ListingRepository) MarkStale(ctx context.Context, sourceID uuid.UUID, beforeTime string) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE listings SET is_active = false
		WHERE source_id = $1 AND last_seen_at < $2 AND is_active = true
	`, sourceID, beforeTime)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
