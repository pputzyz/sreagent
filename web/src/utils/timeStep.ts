/**
 * Computes an appropriate time step/interval for a given time range.
 * Used by query engine and variable auto-refresh to pick a reasonable resolution.
 *
 * @param durationSec - time range duration in seconds
 * @returns step string like '15s', '1m', '5m', etc.
 */
export function computeTimeStep(durationSec: number): string {
  if (durationSec <= 300) return '15s'       // 5min
  if (durationSec <= 3600) return '30s'      // 1h
  if (durationSec <= 21600) return '1m'      // 6h
  if (durationSec <= 86400) return '5m'      // 24h
  if (durationSec <= 604800) return '15m'    // 7d
  return '1h'
}
