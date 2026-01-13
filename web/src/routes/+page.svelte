<script lang="ts">
	import { goto } from '$app/navigation';

	let searchQuery = '';

	function handleSearch() {
		if (searchQuery.trim()) {
			goto(`/listings?q=${encodeURIComponent(searchQuery)}`);
		} else {
			goto('/listings');
		}
	}
</script>

<svelte:head>
	<title>Trough - Find Businesses for Sale</title>
</svelte:head>

<section class="hero">
	<div class="container">
		<h1>Find Your Next Business</h1>
		<p class="subtitle">Search thousands of businesses for sale from top brokerages nationwide</p>

		<form class="search-form" on:submit|preventDefault={handleSearch}>
			<input
				type="text"
				placeholder="Search restaurants, retail, services..."
				bind:value={searchQuery}
			/>
			<button type="submit" class="btn btn-primary">Search</button>
		</form>

		<div class="quick-links">
			<span>Popular:</span>
			<a href="/listings?industry=restaurant">Restaurants</a>
			<a href="/listings?industry=retail">Retail</a>
			<a href="/listings?industry=service">Services</a>
			<a href="/listings?franchise=true">Franchises</a>
		</div>
	</div>
</section>

<section class="features container">
	<div class="feature">
		<div class="feature-icon">&#128269;</div>
		<h3>Comprehensive Search</h3>
		<p>Search across multiple brokerage sites in one place. Filter by price, location, industry, and more.</p>
	</div>

	<div class="feature">
		<div class="feature-icon">&#127758;</div>
		<h3>Map View</h3>
		<p>Explore listings geographically. Find businesses in your target area with our interactive map.</p>
	</div>

	<div class="feature">
		<div class="feature-icon">&#128200;</div>
		<h3>Financial Details</h3>
		<p>Compare asking prices, revenue, and cash flow at a glance. Make informed decisions.</p>
	</div>
</section>

<style>
	.hero {
		background: linear-gradient(135deg, #1e3a5f 0%, #2563eb 100%);
		color: white;
		padding: 6rem 0;
		text-align: center;
	}

	h1 {
		font-size: 3rem;
		font-weight: 700;
		margin-bottom: 1rem;
	}

	.subtitle {
		font-size: 1.25rem;
		opacity: 0.9;
		margin-bottom: 2rem;
	}

	.search-form {
		display: flex;
		gap: 0.5rem;
		max-width: 600px;
		margin: 0 auto 1.5rem;
	}

	.search-form input {
		flex: 1;
		padding: 1rem 1.25rem;
		font-size: 1.125rem;
		border: none;
		border-radius: var(--radius);
	}

	.search-form .btn {
		padding: 1rem 2rem;
		font-size: 1rem;
	}

	.quick-links {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 1rem;
		font-size: 0.875rem;
	}

	.quick-links span {
		opacity: 0.7;
	}

	.quick-links a {
		color: white;
		text-decoration: none;
		padding: 0.25rem 0.75rem;
		background: rgba(255, 255, 255, 0.15);
		border-radius: 4px;
	}

	.quick-links a:hover {
		background: rgba(255, 255, 255, 0.25);
	}

	.features {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: 2rem;
		padding: 4rem 1rem;
	}

	.feature {
		text-align: center;
		padding: 2rem;
	}

	.feature-icon {
		font-size: 3rem;
		margin-bottom: 1rem;
	}

	.feature h3 {
		font-size: 1.25rem;
		margin-bottom: 0.75rem;
	}

	.feature p {
		color: var(--color-text-muted);
		line-height: 1.6;
	}
</style>
