<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import ListingCard from '$lib/components/ListingCard.svelte';
	import FilterPanel from '$lib/components/FilterPanel.svelte';
	import Map from '$lib/components/Map.svelte';
	import Spinner from '$lib/components/Spinner.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import {
		searchResult,
		searchParams,
		isLoading,
		error,
		searchListings,
		fetchFilterOptions
	} from '$lib/stores/listings';
	import { mapMarkers, fetchMapMarkers } from '$lib/stores/map';
	import type { ListingSearchParams, GeoBounds } from '$lib/types/listing';

	let viewMode: 'list' | 'map' = 'list';

	onMount(() => {
		fetchFilterOptions();

		const params = parseUrlParams();
		searchListings(params);

		// If map view, also fetch markers
		if (viewMode === 'map') {
			fetchMapMarkers();
		}
	});

	function parseUrlParams(): ListingSearchParams {
		const urlParams = $page.url.searchParams;
		const params: ListingSearchParams = {
			page: 1,
			per_page: 24
		};

		if (urlParams.has('q')) params.q = urlParams.get('q')!;
		if (urlParams.has('price_min')) params.price_min = parseInt(urlParams.get('price_min')!);
		if (urlParams.has('price_max')) params.price_max = parseInt(urlParams.get('price_max')!);
		if (urlParams.has('state')) params.states = urlParams.get('state')!.split(',');
		if (urlParams.has('industry')) params.industries = urlParams.get('industry')!.split(',');
		if (urlParams.has('franchise')) params.franchise = urlParams.get('franchise') === 'true';
		if (urlParams.has('real_estate')) params.real_estate = urlParams.get('real_estate') === 'true';
		if (urlParams.has('sort')) params.sort = urlParams.get('sort')!;
		if (urlParams.has('page')) params.page = parseInt(urlParams.get('page')!) || 1;
		if (urlParams.has('view')) viewMode = urlParams.get('view') as 'list' | 'map';

		return params;
	}

	function handlePageChange(newPage: number) {
		const params = { ...$searchParams, page: newPage };
		searchListings(params);
		updateUrl(params);
	}

	function handleViewChange(mode: 'list' | 'map') {
		viewMode = mode;
		if (mode === 'map') {
			fetchMapMarkers();
		}
		// Update URL
		const url = new URL(window.location.href);
		url.searchParams.set('view', mode);
		goto(url.pathname + url.search, { replaceState: true, noScroll: true });
	}

	function handleMapBoundsChange(bounds: { south: number; west: number; north: number; east: number }) {
		fetchMapMarkers({
			south_lat: bounds.south,
			west_lng: bounds.west,
			north_lat: bounds.north,
			east_lng: bounds.east
		});
	}

	function updateUrl(params: ListingSearchParams) {
		const url = new URL(window.location.href);
		if (params.page && params.page > 1) {
			url.searchParams.set('page', params.page.toString());
		} else {
			url.searchParams.delete('page');
		}
		goto(url.pathname + url.search, { replaceState: true, noScroll: true });
	}
</script>

<svelte:head>
	<title>Browse Listings - Trough</title>
</svelte:head>

<div class="listings-page container">
	<aside class="sidebar">
		<FilterPanel />
	</aside>

	<section class="main-content">
		<div class="content-header">
			<div class="results-info">
				{#if $isLoading}
					<h1>Searching...</h1>
				{:else if $searchResult}
					<h1>
						{$searchResult.total.toLocaleString()} Businesses for Sale
					</h1>
				{:else}
					<h1>Browse Businesses</h1>
				{/if}
			</div>

			<div class="view-toggle">
				<button
					class="toggle-btn"
					class:active={viewMode === 'list'}
					on:click={() => handleViewChange('list')}
				>
					<svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
						<rect x="1" y="1" width="14" height="3" rx="1"/>
						<rect x="1" y="6" width="14" height="3" rx="1"/>
						<rect x="1" y="11" width="14" height="3" rx="1"/>
					</svg>
					List
				</button>
				<button
					class="toggle-btn"
					class:active={viewMode === 'map'}
					on:click={() => handleViewChange('map')}
				>
					<svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
						<path d="M8 0C5.2 0 3 2.2 3 5c0 4.2 5 10.4 5 10.4s5-6.2 5-10.4c0-2.8-2.2-5-5-5zm0 7.5c-1.4 0-2.5-1.1-2.5-2.5S6.6 2.5 8 2.5s2.5 1.1 2.5 2.5S9.4 7.5 8 7.5z"/>
					</svg>
					Map
				</button>
			</div>
		</div>

		{#if $error}
			<div class="error-message">
				<p>{$error}</p>
				<button class="btn btn-secondary" on:click={() => searchListings($searchParams)}>
					Try Again
				</button>
			</div>
		{:else if $isLoading}
			<div class="loading">
				<Spinner size="lg" />
				<p>Loading listings...</p>
			</div>
		{:else if viewMode === 'list'}
			{#if $searchResult?.listings.length}
				<div class="listings-grid">
					{#each $searchResult.listings as listing (listing.id)}
						<ListingCard {listing} />
					{/each}
				</div>

				{#if $searchResult.total_pages > 1}
					<Pagination
						page={$searchResult.page}
						totalPages={$searchResult.total_pages}
						onPageChange={handlePageChange}
					/>
				{/if}
			{:else}
				<div class="no-results">
					<h2>No listings found</h2>
					<p>Try adjusting your filters or search terms</p>
				</div>
			{/if}
		{:else}
			<div class="map-container">
				<Map
					markers={$mapMarkers}
					onBoundsChange={handleMapBoundsChange}
				/>
			</div>
			{#if $mapMarkers.length > 0}
				<p class="map-count">{$mapMarkers.length} listings shown on map</p>
			{/if}
		{/if}
	</section>
</div>

<style>
	.listings-page {
		display: grid;
		grid-template-columns: 280px 1fr;
		gap: 2rem;
		padding: 2rem 1rem;
	}

	@media (max-width: 900px) {
		.listings-page {
			grid-template-columns: 1fr;
		}

		.sidebar {
			order: 2;
		}
	}

	.content-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1.5rem;
		flex-wrap: wrap;
		gap: 1rem;
	}

	.results-info h1 {
		font-size: 1.5rem;
		font-weight: 600;
	}

	.view-toggle {
		display: flex;
		border: 1px solid var(--color-border);
		border-radius: var(--radius);
		overflow: hidden;
	}

	.toggle-btn {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 1rem;
		background: var(--color-bg);
		border: none;
		font-size: 0.875rem;
		cursor: pointer;
		transition: all 0.2s;
	}

	.toggle-btn:first-child {
		border-right: 1px solid var(--color-border);
	}

	.toggle-btn:hover {
		background: var(--color-bg-secondary);
	}

	.toggle-btn.active {
		background: var(--color-primary);
		color: white;
	}

	.listings-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
		gap: 1.5rem;
	}

	.loading {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 4rem 2rem;
		gap: 1rem;
	}

	.loading p {
		color: var(--color-text-muted);
	}

	.no-results {
		text-align: center;
		padding: 4rem 2rem;
	}

	.no-results h2 {
		margin-bottom: 0.5rem;
	}

	.no-results p {
		color: var(--color-text-muted);
	}

	.error-message {
		text-align: center;
		padding: 2rem;
		background: #fef2f2;
		color: #991b1b;
		border-radius: var(--radius);
	}

	.error-message p {
		margin-bottom: 1rem;
	}

	.map-container {
		height: 600px;
		border-radius: var(--radius);
		overflow: hidden;
	}

	.map-count {
		text-align: center;
		color: var(--color-text-muted);
		font-size: 0.875rem;
		margin-top: 1rem;
	}
</style>
