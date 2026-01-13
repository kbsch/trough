<script lang="ts">
	import type { Listing } from '$lib/types/listing';
	import { formatPrice } from '$lib/stores/listings';

	export let listing: Listing;

	function getLocation(listing: Listing): string {
		const parts = [];
		if (listing.city) parts.push(listing.city);
		if (listing.state) parts.push(listing.state);
		return parts.join(', ') || 'Location not specified';
	}
</script>

<a href="/listings/{listing.id}" class="listing-card">
	<div class="card-header">
		<span class="industry">{listing.industry || 'Business'}</span>
		{#if listing.is_franchise}
			<span class="badge franchise">Franchise</span>
		{/if}
	</div>

	<h3 class="title">{listing.title}</h3>

	<p class="location">{getLocation(listing)}</p>

	{#if listing.description}
		<p class="description">{listing.description.slice(0, 150)}{listing.description.length > 150 ? '...' : ''}</p>
	{/if}

	<div class="financials">
		<div class="financial-item">
			<span class="label">Asking Price</span>
			<span class="value price">{formatPrice(listing.asking_price)}</span>
		</div>

		{#if listing.cash_flow}
			<div class="financial-item">
				<span class="label">Cash Flow</span>
				<span class="value">{formatPrice(listing.cash_flow)}</span>
			</div>
		{/if}

		{#if listing.revenue}
			<div class="financial-item">
				<span class="label">Revenue</span>
				<span class="value">{formatPrice(listing.revenue)}</span>
			</div>
		{/if}
	</div>

	{#if listing.real_estate_included}
		<span class="badge real-estate">Includes Real Estate</span>
	{/if}
</a>

<style>
	.listing-card {
		display: block;
		background: var(--color-bg);
		border: 1px solid var(--color-border);
		border-radius: var(--radius);
		padding: 1.25rem;
		transition: all 0.2s;
		text-decoration: none;
		color: inherit;
	}

	.listing-card:hover {
		box-shadow: var(--shadow-lg);
		border-color: var(--color-primary);
		transform: translateY(-2px);
	}

	.card-header {
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
		margin-top: 0.75rem;
		display: inline-block;
	}

	.title {
		font-size: 1.125rem;
		font-weight: 600;
		margin-bottom: 0.5rem;
		color: var(--color-text);
		line-height: 1.4;
	}

	.location {
		font-size: 0.875rem;
		color: var(--color-text-muted);
		margin-bottom: 0.75rem;
	}

	.description {
		font-size: 0.875rem;
		color: var(--color-secondary);
		margin-bottom: 1rem;
		line-height: 1.5;
	}

	.financials {
		display: grid;
		gap: 0.5rem;
	}

	.financial-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.label {
		font-size: 0.75rem;
		color: var(--color-text-muted);
	}

	.value {
		font-weight: 600;
		font-size: 0.875rem;
	}

	.price {
		color: var(--color-primary);
		font-size: 1rem;
	}
</style>
