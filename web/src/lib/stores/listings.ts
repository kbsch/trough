import { writable, derived } from 'svelte/store';
import type { Listing, ListingSearchParams, ListingSearchResult, FilterOptions } from '$lib/types/listing';

const API_URL = import.meta.env.PUBLIC_API_URL || 'http://localhost:8080';

export const searchParams = writable<ListingSearchParams>({
	page: 1,
	per_page: 24
});

export const searchResult = writable<ListingSearchResult | null>(null);
export const filterOptions = writable<FilterOptions | null>(null);
export const isLoading = writable(false);
export const error = writable<string | null>(null);

export async function searchListings(params: ListingSearchParams): Promise<void> {
	isLoading.set(true);
	error.set(null);

	try {
		const queryParams = new URLSearchParams();

		if (params.q) queryParams.set('q', params.q);
		if (params.price_min) queryParams.set('price_min', params.price_min.toString());
		if (params.price_max) queryParams.set('price_max', params.price_max.toString());
		if (params.revenue_min) queryParams.set('revenue_min', params.revenue_min.toString());
		if (params.cash_flow_min) queryParams.set('cash_flow_min', params.cash_flow_min.toString());
		if (params.states?.length) queryParams.set('state', params.states.join(','));
		if (params.industries?.length) queryParams.set('industry', params.industries.join(','));
		if (params.franchise !== undefined) queryParams.set('franchise', params.franchise.toString());
		if (params.real_estate !== undefined) queryParams.set('real_estate', params.real_estate.toString());
		if (params.sort) queryParams.set('sort', params.sort);
		if (params.page) queryParams.set('page', params.page.toString());
		if (params.per_page) queryParams.set('per_page', params.per_page.toString());
		if (params.bounds) {
			queryParams.set('bounds', `${params.bounds.south_lat},${params.bounds.west_lng},${params.bounds.north_lat},${params.bounds.east_lng}`);
		}

		const response = await fetch(`${API_URL}/api/v1/listings?${queryParams}`);

		if (!response.ok) {
			throw new Error(`Search failed: ${response.statusText}`);
		}

		const result: ListingSearchResult = await response.json();
		searchResult.set(result);
		searchParams.set(params);
	} catch (e) {
		error.set(e instanceof Error ? e.message : 'An error occurred');
	} finally {
		isLoading.set(false);
	}
}

export async function fetchListing(id: string): Promise<Listing | null> {
	try {
		const response = await fetch(`${API_URL}/api/v1/listings/${id}`);
		if (!response.ok) {
			throw new Error(`Failed to fetch listing: ${response.statusText}`);
		}
		return await response.json();
	} catch (e) {
		error.set(e instanceof Error ? e.message : 'An error occurred');
		return null;
	}
}

export async function fetchFilterOptions(): Promise<void> {
	try {
		const response = await fetch(`${API_URL}/api/v1/filters`);
		if (!response.ok) {
			throw new Error(`Failed to fetch filters: ${response.statusText}`);
		}
		const options: FilterOptions = await response.json();
		filterOptions.set(options);
	} catch (e) {
		console.error('Failed to fetch filter options:', e);
	}
}

export function formatPrice(cents: number | undefined): string {
	if (!cents) return 'Price not disclosed';
	return new Intl.NumberFormat('en-US', {
		style: 'currency',
		currency: 'USD',
		minimumFractionDigits: 0,
		maximumFractionDigits: 0
	}).format(cents / 100);
}

export function formatNumber(value: number | undefined): string {
	if (!value) return 'N/A';
	return new Intl.NumberFormat('en-US').format(value);
}
