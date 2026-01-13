import { writable } from 'svelte/store';
import type { MapMarker, GeoBounds } from '$lib/types/listing';

const API_URL = import.meta.env.PUBLIC_API_URL || 'http://localhost:8080';

export const mapMarkers = writable<MapMarker[]>([]);
export const mapBounds = writable<GeoBounds | null>(null);
export const isLoadingMap = writable(false);

export async function fetchMapMarkers(bounds?: GeoBounds): Promise<void> {
	isLoadingMap.set(true);

	try {
		const params = new URLSearchParams();
		if (bounds) {
			params.set('bounds', `${bounds.south_lat},${bounds.west_lng},${bounds.north_lat},${bounds.east_lng}`);
		}

		const response = await fetch(`${API_URL}/api/v1/listings/map?${params}`);

		if (!response.ok) {
			throw new Error(`Failed to fetch map data: ${response.statusText}`);
		}

		const data = await response.json();
		mapMarkers.set(data.markers || []);

		if (data.bounds) {
			mapBounds.set({
				south_lat: data.bounds.south,
				west_lng: data.bounds.west,
				north_lat: data.bounds.north,
				east_lng: data.bounds.east
			});
		}
	} catch (e) {
		console.error('Failed to fetch map markers:', e);
		mapMarkers.set([]);
	} finally {
		isLoadingMap.set(false);
	}
}
