<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { fetchListing, formatPrice, formatNumber } from '$lib/stores/listings';
	import Spinner from '$lib/components/Spinner.svelte';
	import type { Listing } from '$lib/types/listing';

	let listing: Listing | null = null;
	let loading = true;
	let error: string | null = null;

	onMount(async () => {
		const id = $page.params.id;
		try {
			listing = await fetchListing(id);
			if (!listing) {
				error = 'Listing not found';
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load listing';
		} finally {
			loading = false;
		}
	});

	function getLocation(listing: Listing): string {
		return [listing.city, listing.state].filter(Boolean).join(', ') || 'Location not specified';
	}

	function getMultiple(price: number | undefined, cashFlow: number | undefined): string {
		if (!price || !cashFlow || cashFlow === 0) return 'N/A';
		const multiple = price / cashFlow;
		return `${multiple.toFixed(1)}x`;
	}
</script>

<svelte:head>
	<title>{listing?.title || 'Listing'} - Trough</title>
</svelte:head>

<div class="listing-detail container">
	{#if loading}
		<div class="loading">
			<Spinner size="lg" />
			<p>Loading listing...</p>
		</div>
	{:else if error}
		<div class="error">
			<h1>Error</h1>
			<p>{error}</p>
			<a href="/listings" class="btn btn-primary">Browse Listings</a>
		</div>
	{:else if listing}
		<nav class="breadcrumb">
			<a href="/">Home</a>
			<span>/</span>
			<a href="/listings">Listings</a>
			<span>/</span>
			<span class="current">{listing.industry || 'Business'}</span>
		</nav>

		<div class="detail-grid">
			<main class="main-content">
				<header class="listing-header">
					<div class="meta">
						<span class="industry">{listing.industry || 'Business'}</span>
						{#if listing.is_franchise}
							<span class="badge franchise">Franchise</span>
						{/if}
						{#if listing.real_estate_included}
							<span class="badge real-estate">Includes Real Estate</span>
						{/if}
					</div>
					<h1>{listing.title}</h1>
					<p class="location">{getLocation(listing)}</p>
				</header>

				{#if listing.description}
					<section class="section">
						<h2>Description</h2>
						<div class="description">
							{listing.description}
						</div>
					</section>
				{/if}

				{#if listing.reason_for_sale}
					<section class="section">
						<h2>Reason for Sale</h2>
						<p>{listing.reason_for_sale}</p>
					</section>
				{/if}

				<section class="section">
					<h2>Business Details</h2>
					<dl class="details-grid">
						{#if listing.year_established}
							<div class="detail-item">
								<dt>Year Established</dt>
								<dd>{listing.year_established}</dd>
							</div>
						{/if}
						{#if listing.employees}
							<div class="detail-item">
								<dt>Employees</dt>
								<dd>{formatNumber(listing.employees)}</dd>
							</div>
						{/if}
						{#if listing.business_type}
							<div class="detail-item">
								<dt>Business Type</dt>
								<dd>{listing.business_type}</dd>
							</div>
						{/if}
						{#if listing.franchise_name}
							<div class="detail-item">
								<dt>Franchise</dt>
								<dd>{listing.franchise_name}</dd>
							</div>
						{/if}
						{#if listing.industry_category}
							<div class="detail-item">
								<dt>Category</dt>
								<dd>{listing.industry_category}</dd>
							</div>
						{/if}
					</dl>
				</section>
			</main>

			<aside class="sidebar">
				<div class="price-card card">
					<div class="price-main">
						<span class="label">Asking Price</span>
						<span class="price">{formatPrice(listing.asking_price)}</span>
					</div>

					<dl class="financials">
						{#if listing.revenue}
							<div class="financial-item">
								<dt>Annual Revenue</dt>
								<dd>{formatPrice(listing.revenue)}</dd>
							</div>
						{/if}
						{#if listing.cash_flow}
							<div class="financial-item">
								<dt>Cash Flow (SDE)</dt>
								<dd>{formatPrice(listing.cash_flow)}</dd>
							</div>
						{/if}
						{#if listing.ebitda}
							<div class="financial-item">
								<dt>EBITDA</dt>
								<dd>{formatPrice(listing.ebitda)}</dd>
							</div>
						{/if}
						{#if listing.asking_price && listing.cash_flow}
							<div class="financial-item highlight">
								<dt>Multiple</dt>
								<dd>{getMultiple(listing.asking_price, listing.cash_flow)}</dd>
							</div>
						{/if}
						{#if listing.inventory_value}
							<div class="financial-item">
								<dt>Inventory</dt>
								<dd>{formatPrice(listing.inventory_value)}</dd>
							</div>
						{/if}
						{#if listing.real_estate_value}
							<div class="financial-item">
								<dt>Real Estate Value</dt>
								<dd>{formatPrice(listing.real_estate_value)}</dd>
							</div>
						{/if}
					</dl>

					<a href={listing.url} target="_blank" rel="noopener noreferrer" class="btn btn-primary view-original">
						View Original Listing
						<svg width="14" height="14" viewBox="0 0 14 14" fill="currentColor">
							<path d="M11 8V11H3V3H6V1H1V13H13V8H11Z"/>
							<path d="M8 1V3H10.59L5.29 8.29L6.71 9.71L12 4.41V7H14V1H8Z"/>
						</svg>
					</a>
				</div>

				<div class="source-info card">
					<h3>Listing Info</h3>
					<dl>
						<div class="info-item">
							<dt>First Seen</dt>
							<dd>{new Date(listing.first_seen_at).toLocaleDateString()}</dd>
						</div>
						<div class="info-item">
							<dt>Last Updated</dt>
							<dd>{new Date(listing.last_seen_at).toLocaleDateString()}</dd>
						</div>
						<div class="info-item">
							<dt>Listing ID</dt>
							<dd class="mono">{listing.external_id}</dd>
						</div>
					</dl>
				</div>
			</aside>
		</div>
	{:else}
		<div class="not-found">
			<h1>Listing Not Found</h1>
			<p>This listing may have been removed or is no longer available.</p>
			<a href="/listings" class="btn btn-primary">Browse Listings</a>
		</div>
	{/if}
</div>

<style>
	.listing-detail {
		padding: 2rem 1rem;
	}

	.loading, .error, .not-found {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		text-align: center;
		padding: 4rem 2rem;
		gap: 1rem;
	}

	.error, .not-found {
		gap: 0.5rem;
	}

	.error p, .not-found p {
		color: var(--color-text-muted);
		margin-bottom: 1rem;
	}

	.breadcrumb {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.875rem;
		margin-bottom: 2rem;
		color: var(--color-text-muted);
	}

	.breadcrumb a {
		color: var(--color-text-muted);
	}

	.breadcrumb a:hover {
		color: var(--color-primary);
	}

	.breadcrumb .current {
		color: var(--color-text);
	}

	.detail-grid {
		display: grid;
		grid-template-columns: 1fr 380px;
		gap: 2rem;
	}

	@media (max-width: 900px) {
		.detail-grid {
			grid-template-columns: 1fr;
		}

		.sidebar {
			order: -1;
		}
	}

	.listing-header {
		margin-bottom: 2rem;
	}

	.meta {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 0.75rem;
	}

	.industry {
		font-size: 0.75rem;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.badge {
		font-size: 0.7rem;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		font-weight: 500;
	}

	.franchise {
		background: #dbeafe;
		color: #1e40af;
	}

	.real-estate {
		background: #dcfce7;
		color: #166534;
	}

	h1 {
		font-size: 2rem;
		font-weight: 700;
		margin-bottom: 0.5rem;
		line-height: 1.3;
	}

	.location {
		color: var(--color-text-muted);
		font-size: 1.125rem;
	}

	.section {
		margin-bottom: 2rem;
	}

	.section h2 {
		font-size: 1.25rem;
		margin-bottom: 1rem;
		padding-bottom: 0.5rem;
		border-bottom: 1px solid var(--color-border);
	}

	.description {
		white-space: pre-wrap;
		line-height: 1.7;
	}

	.details-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 1rem;
	}

	.detail-item dt {
		font-size: 0.75rem;
		color: var(--color-text-muted);
		text-transform: uppercase;
		margin-bottom: 0.25rem;
	}

	.detail-item dd {
		font-weight: 500;
	}

	.sidebar .card {
		margin-bottom: 1rem;
	}

	.price-card {
		position: sticky;
		top: 80px;
	}

	.price-main {
		text-align: center;
		padding-bottom: 1rem;
		margin-bottom: 1rem;
		border-bottom: 1px solid var(--color-border);
	}

	.price-main .label {
		display: block;
		font-size: 0.875rem;
		color: var(--color-text-muted);
		margin-bottom: 0.25rem;
	}

	.price-main .price {
		font-size: 2rem;
		font-weight: 700;
		color: var(--color-primary);
	}

	.financials {
		margin-bottom: 1.5rem;
	}

	.financial-item {
		display: flex;
		justify-content: space-between;
		padding: 0.75rem 0;
		border-bottom: 1px solid var(--color-border);
	}

	.financial-item:last-child {
		border-bottom: none;
	}

	.financial-item dt {
		color: var(--color-text-muted);
	}

	.financial-item dd {
		font-weight: 600;
	}

	.financial-item.highlight {
		background: var(--color-bg-secondary);
		margin: 0 -1rem;
		padding: 0.75rem 1rem;
		border-radius: var(--radius);
	}

	.financial-item.highlight dd {
		color: var(--color-primary);
	}

	.view-original {
		width: 100%;
		justify-content: center;
		gap: 0.5rem;
	}

	.source-info h3 {
		font-size: 0.875rem;
		font-weight: 600;
		margin-bottom: 0.75rem;
		color: var(--color-text-muted);
	}

	.info-item {
		display: flex;
		justify-content: space-between;
		padding: 0.5rem 0;
		font-size: 0.875rem;
	}

	.info-item dt {
		color: var(--color-text-muted);
	}

	.mono {
		font-family: monospace;
		font-size: 0.8rem;
	}
</style>
