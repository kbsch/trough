<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Loader } from '@googlemaps/js-api-loader';
	import type { MapMarker } from '$lib/types/listing';
	import { formatPrice } from '$lib/stores/listings';

	export let markers: MapMarker[] = [];
	export let onBoundsChange: ((bounds: { south: number; west: number; north: number; east: number }) => void) | null = null;

	let mapContainer: HTMLDivElement;
	let map: google.maps.Map | null = null;
	let markerCluster: any = null;
	let googleMarkers: google.maps.Marker[] = [];
	let infoWindow: google.maps.InfoWindow | null = null;
	let boundsChangeTimeout: ReturnType<typeof setTimeout>;

	const apiKey = import.meta.env.PUBLIC_GOOGLE_MAPS_API_KEY || '';

	onMount(async () => {
		if (!apiKey) {
			console.warn('Google Maps API key not configured');
			return;
		}

		const loader = new Loader({
			apiKey,
			version: 'weekly',
			libraries: ['marker']
		});

		try {
			const google = await loader.load();

			// Default center: US center
			const center = { lat: 39.8283, lng: -98.5795 };

			map = new google.maps.Map(mapContainer, {
				center,
				zoom: 4,
				mapTypeControl: false,
				streetViewControl: false,
				fullscreenControl: true,
				zoomControl: true,
				styles: [
					{
						featureType: 'poi',
						elementType: 'labels',
						stylers: [{ visibility: 'off' }]
					}
				]
			});

			infoWindow = new google.maps.InfoWindow();

			// Listen for bounds changes
			if (onBoundsChange) {
				map.addListener('idle', () => {
					clearTimeout(boundsChangeTimeout);
					boundsChangeTimeout = setTimeout(() => {
						if (map) {
							const bounds = map.getBounds();
							if (bounds) {
								onBoundsChange({
									south: bounds.getSouthWest().lat(),
									west: bounds.getSouthWest().lng(),
									north: bounds.getNorthEast().lat(),
									east: bounds.getNorthEast().lng()
								});
							}
						}
					}, 500);
				});
			}

			updateMarkers();
		} catch (error) {
			console.error('Failed to load Google Maps:', error);
		}
	});

	onDestroy(() => {
		clearTimeout(boundsChangeTimeout);
		clearMarkers();
	});

	function clearMarkers() {
		googleMarkers.forEach(marker => marker.setMap(null));
		googleMarkers = [];
	}

	function updateMarkers() {
		if (!map) return;

		clearMarkers();

		markers.forEach(marker => {
			const gMarker = new google.maps.Marker({
				position: { lat: marker.lat, lng: marker.lng },
				map,
				title: marker.title,
				icon: {
					path: google.maps.SymbolPath.CIRCLE,
					scale: 8,
					fillColor: '#2563eb',
					fillOpacity: 0.9,
					strokeColor: '#ffffff',
					strokeWeight: 2
				}
			});

			gMarker.addListener('click', () => {
				if (infoWindow) {
					const content = `
						<div style="max-width: 250px; padding: 8px;">
							<h3 style="margin: 0 0 8px; font-size: 14px; font-weight: 600;">
								${marker.title}
							</h3>
							${marker.city || marker.state ? `
								<p style="margin: 0 0 8px; font-size: 12px; color: #666;">
									${[marker.city, marker.state].filter(Boolean).join(', ')}
								</p>
							` : ''}
							${marker.asking_price ? `
								<p style="margin: 0 0 8px; font-size: 14px; font-weight: 600; color: #2563eb;">
									${formatPrice(marker.asking_price)}
								</p>
							` : ''}
							${marker.industry ? `
								<p style="margin: 0 0 8px; font-size: 11px; color: #888; text-transform: uppercase;">
									${marker.industry}
								</p>
							` : ''}
							<a href="/listings/${marker.id}"
							   style="color: #2563eb; font-size: 12px; text-decoration: none;">
								View Details →
							</a>
						</div>
					`;
					infoWindow.setContent(content);
					infoWindow.open(map, gMarker);
				}
			});

			googleMarkers.push(gMarker);
		});

		// Fit bounds if we have markers
		if (markers.length > 0) {
			const bounds = new google.maps.LatLngBounds();
			markers.forEach(m => bounds.extend({ lat: m.lat, lng: m.lng }));
			map.fitBounds(bounds);
		}
	}

	// React to marker changes
	$: if (map && markers) {
		updateMarkers();
	}
</script>

<div class="map-wrapper">
	{#if !apiKey}
		<div class="map-placeholder">
			<p>Google Maps API key not configured.</p>
			<p class="hint">Set PUBLIC_GOOGLE_MAPS_API_KEY in your environment.</p>
		</div>
	{:else}
		<div bind:this={mapContainer} class="map-container"></div>
	{/if}
</div>

<style>
	.map-wrapper {
		width: 100%;
		height: 100%;
		min-height: 500px;
		border-radius: var(--radius);
		overflow: hidden;
	}

	.map-container {
		width: 100%;
		height: 100%;
		min-height: 500px;
	}

	.map-placeholder {
		width: 100%;
		height: 100%;
		min-height: 500px;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		background: var(--color-bg-secondary);
		color: var(--color-text-muted);
		text-align: center;
		padding: 2rem;
	}

	.hint {
		font-size: 0.875rem;
		margin-top: 0.5rem;
		opacity: 0.7;
	}
</style>
