<script lang="ts">
	import '../app.css';
</script>

<header class="header">
	<div class="container header-content">
		<a href="/" class="logo">
			<span class="logo-icon">T</span>
			<span class="logo-text">Trough</span>
		</a>
		<nav class="nav">
			<a href="/listings">Browse Listings</a>
			<a href="/listings?view=map">Map View</a>
		</nav>
	</div>
</header>

<main>
	<slot />
</main>

<footer class="footer">
	<div class="container">
		<p>&copy; {new Date().getFullYear()} Trough. Business listings aggregated from multiple sources.</p>
	</div>
</footer>

<style>
	.header {
		background: var(--color-bg);
		border-bottom: 1px solid var(--color-border);
		position: sticky;
		top: 0;
		z-index: 100;
	}

	.header-content {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: 64px;
	}

	.logo {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		text-decoration: none;
		color: var(--color-text);
	}

	.logo-icon {
		width: 32px;
		height: 32px;
		background: var(--color-primary);
		color: white;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-weight: 700;
		font-size: 1.25rem;
	}

	.logo-text {
		font-size: 1.25rem;
		font-weight: 700;
	}

	.nav {
		display: flex;
		gap: 1.5rem;
	}

	.nav a {
		color: var(--color-text);
		text-decoration: none;
		font-weight: 500;
	}

	.nav a:hover {
		color: var(--color-primary);
	}

	main {
		min-height: calc(100vh - 64px - 80px);
	}

	.footer {
		background: var(--color-bg-secondary);
		border-top: 1px solid var(--color-border);
		padding: 2rem 0;
		text-align: center;
		color: var(--color-text-muted);
		font-size: 0.875rem;
	}
</style>
