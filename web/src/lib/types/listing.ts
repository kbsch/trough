export interface Listing {
	id: string;
	source_id: string;
	external_id: string;
	url: string;
	title: string;
	description?: string;
	asking_price?: number;
	revenue?: number;
	cash_flow?: number;
	ebitda?: number;
	inventory_value?: number;
	real_estate_included: boolean;
	real_estate_value?: number;
	city?: string;
	state?: string;
	zip_code?: string;
	country: string;
	lat?: number;
	lng?: number;
	industry?: string;
	industry_category?: string;
	business_type?: string;
	year_established?: number;
	employees?: number;
	reason_for_sale?: string;
	is_franchise: boolean;
	franchise_name?: string;
	first_seen_at: string;
	last_seen_at: string;
	is_active: boolean;
}

export interface ListingSearchParams {
	q?: string;
	price_min?: number;
	price_max?: number;
	revenue_min?: number;
	cash_flow_min?: number;
	states?: string[];
	industries?: string[];
	franchise?: boolean;
	real_estate?: boolean;
	bounds?: GeoBounds;
	sort?: string;
	page?: number;
	per_page?: number;
}

export interface GeoBounds {
	south_lat: number;
	west_lng: number;
	north_lat: number;
	east_lng: number;
}

export interface ListingSearchResult {
	listings: Listing[];
	total: number;
	page: number;
	per_page: number;
	total_pages: number;
}

export interface FilterOptions {
	industries: FilterOption[];
	states: FilterOption[];
	price_range: PriceRange;
}

export interface FilterOption {
	value: string;
	label: string;
	count: number;
}

export interface PriceRange {
	min: number;
	max: number;
}

export interface MapMarker {
	id: string;
	lat: number;
	lng: number;
	title: string;
	asking_price?: number;
	industry?: string;
}
