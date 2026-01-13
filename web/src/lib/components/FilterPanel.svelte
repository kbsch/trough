<script lang="ts">
	import { searchParams, filterOptions, searchListings } from '$lib/stores/listings';
	import type { ListingSearchParams } from '$lib/types/listing';

	let localParams: ListingSearchParams = {};
	let priceMinInput = '';
	let priceMaxInput = '';

	searchParams.subscribe(p => {
		localParams = { ...p };
		priceMinInput = p.price_min ? (p.price_min / 100).toString() : '';
		priceMaxInput = p.price_max ? (p.price_max / 100).toString() : '';
	});

	function handleSearch() {
		const params: ListingSearchParams = {
			...localParams,
			page: 1,
			per_page: 24
		};

		if (priceMinInput) {
			params.price_min = Math.round(parseFloat(priceMinInput) * 100);
		}
		if (priceMaxInput) {
			params.price_max = Math.round(parseFloat(priceMaxInput) * 100);
		}

		searchListings(params);
	}

	function handleStateChange(state: string, checked: boolean) {
		const current = localParams.states || [];
		if (checked) {
			localParams.states = [...current, state];
		} else {
			localParams.states = current.filter(s => s !== state);
		}
	}

	function handleIndustryChange(industry: string, checked: boolean) {
		const current = localParams.industries || [];
		if (checked) {
			localParams.industries = [...current, industry];
		} else {
			localParams.industries = current.filter(i => i !== industry);
		}
	}

	function clearFilters() {
		localParams = { page: 1, per_page: 24 };
		priceMinInput = '';
		priceMaxInput = '';
		searchListings(localParams);
	}
</script>

<aside class="filter-panel">
	<div class="filter-header">
		<h2>Filters</h2>
		<button class="clear-btn" on:click={clearFilters}>Clear all</button>
	</div>

	<div class="filter-section">
		<label for="search">Search</label>
		<input
			type="text"
			id="search"
			placeholder="Restaurant, retail, etc..."
			bind:value={localParams.q}
			on:keydown={(e) => e.key === 'Enter' && handleSearch()}
		/>
	</div>

	<div class="filter-section">
		<span class="section-label">Price Range</span>
		<div class="price-inputs">
			<input
				type="number"
				placeholder="Min"
				bind:value={priceMinInput}
			/>
			<span>to</span>
			<input
				type="number"
				placeholder="Max"
				bind:value={priceMaxInput}
			/>
		</div>
	</div>

	<div class="filter-section">
		<label for="sort">Sort By</label>
		<select id="sort" bind:value={localParams.sort}>
			<option value="">Most Recent</option>
			<option value="price_asc">Price: Low to High</option>
			<option value="price_desc">Price: High to Low</option>
			<option value="newest">Newest First</option>
		</select>
	</div>

	{#if $filterOptions?.states?.length}
		<div class="filter-section">
			<span class="section-label">States</span>
			<div class="checkbox-group">
				{#each $filterOptions.states.slice(0, 10) as state}
					<label class="checkbox-label">
						<input
							type="checkbox"
							checked={localParams.states?.includes(state.value)}
							on:change={(e) => handleStateChange(state.value, e.currentTarget.checked)}
						/>
						<span>{state.label}</span>
						<span class="count">({state.count})</span>
					</label>
				{/each}
			</div>
		</div>
	{/if}

	{#if $filterOptions?.industries?.length}
		<div class="filter-section">
			<span class="section-label">Industries</span>
			<div class="checkbox-group">
				{#each $filterOptions.industries.slice(0, 10) as industry}
					<label class="checkbox-label">
						<input
							type="checkbox"
							checked={localParams.industries?.includes(industry.value)}
							on:change={(e) => handleIndustryChange(industry.value, e.currentTarget.checked)}
						/>
						<span>{industry.label}</span>
						<span class="count">({industry.count})</span>
					</label>
				{/each}
			</div>
		</div>
	{/if}

	<div class="filter-section">
		<label class="checkbox-label">
			<input
				type="checkbox"
				bind:checked={localParams.franchise}
			/>
			<span>Franchise Only</span>
		</label>
		<label class="checkbox-label">
			<input
				type="checkbox"
				bind:checked={localParams.real_estate}
			/>
			<span>Includes Real Estate</span>
		</label>
	</div>

	<button class="btn btn-primary apply-btn" on:click={handleSearch}>
		Apply Filters
	</button>
</aside>

<style>
	.filter-panel {
		background: var(--color-bg);
		border: 1px solid var(--color-border);
		border-radius: var(--radius);
		padding: 1.5rem;
		height: fit-content;
		position: sticky;
		top: 1rem;
	}

	.filter-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1.5rem;
		padding-bottom: 1rem;
		border-bottom: 1px solid var(--color-border);
	}

	.filter-header h2 {
		font-size: 1.125rem;
		font-weight: 600;
	}

	.clear-btn {
		background: none;
		border: none;
		color: var(--color-primary);
		font-size: 0.875rem;
	}

	.clear-btn:hover {
		text-decoration: underline;
	}

	.filter-section {
		margin-bottom: 1.25rem;
	}

	.filter-section > label:first-child,
	.section-label {
		display: block;
		font-size: 0.875rem;
		font-weight: 500;
		margin-bottom: 0.5rem;
		color: var(--color-text);
	}

	.price-inputs {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.price-inputs input {
		flex: 1;
	}

	.price-inputs span {
		color: var(--color-text-muted);
		font-size: 0.875rem;
	}

	.checkbox-group {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		max-height: 200px;
		overflow-y: auto;
	}

	.checkbox-label {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.875rem;
		cursor: pointer;
	}

	.checkbox-label input {
		width: auto;
	}

	.count {
		color: var(--color-text-muted);
		font-size: 0.75rem;
	}

	.apply-btn {
		width: 100%;
		justify-content: center;
		margin-top: 1rem;
	}
</style>
