<script lang="ts">
	export let page: number;
	export let totalPages: number;
	export let onPageChange: (page: number) => void;

	$: pages = getVisiblePages(page, totalPages);

	function getVisiblePages(current: number, total: number): (number | '...')[] {
		if (total <= 7) {
			return Array.from({ length: total }, (_, i) => i + 1);
		}

		if (current <= 3) {
			return [1, 2, 3, 4, 5, '...', total];
		}

		if (current >= total - 2) {
			return [1, '...', total - 4, total - 3, total - 2, total - 1, total];
		}

		return [1, '...', current - 1, current, current + 1, '...', total];
	}
</script>

<nav class="pagination" aria-label="Pagination">
	<button
		class="page-btn"
		disabled={page === 1}
		on:click={() => onPageChange(page - 1)}
		aria-label="Previous page"
	>
		← Prev
	</button>

	<div class="page-numbers">
		{#each pages as p}
			{#if p === '...'}
				<span class="ellipsis">...</span>
			{:else}
				<button
					class="page-btn number"
					class:active={p === page}
					on:click={() => onPageChange(p)}
					aria-current={p === page ? 'page' : undefined}
				>
					{p}
				</button>
			{/if}
		{/each}
	</div>

	<button
		class="page-btn"
		disabled={page === totalPages}
		on:click={() => onPageChange(page + 1)}
		aria-label="Next page"
	>
		Next →
	</button>
</nav>

<style>
	.pagination {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 1.5rem 0;
	}

	.page-numbers {
		display: flex;
		align-items: center;
		gap: 0.25rem;
	}

	.page-btn {
		padding: 0.5rem 1rem;
		border: 1px solid var(--color-border);
		background: var(--color-bg);
		border-radius: var(--radius);
		font-size: 0.875rem;
		cursor: pointer;
		transition: all 0.2s;
	}

	.page-btn:hover:not(:disabled) {
		background: var(--color-bg-secondary);
		border-color: var(--color-primary);
	}

	.page-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.page-btn.number {
		min-width: 40px;
		padding: 0.5rem;
	}

	.page-btn.active {
		background: var(--color-primary);
		color: white;
		border-color: var(--color-primary);
	}

	.ellipsis {
		padding: 0.5rem;
		color: var(--color-text-muted);
	}

	@media (max-width: 600px) {
		.page-numbers {
			display: none;
		}
	}
</style>
