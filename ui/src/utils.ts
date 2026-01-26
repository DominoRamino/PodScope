import type { Protocol } from './types'

/**
 * Formats a byte count into a human-readable string with appropriate units.
 * @param bytes - The number of bytes to format
 * @returns A formatted string like "1.50 KB" or "2.00 GB"
 */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

/**
 * Formats a timestamp string as HH:MM:SS.mmm (24-hour format with milliseconds).
 * @param timestamp - ISO 8601 timestamp string
 * @returns A formatted time string like "14:30:45.123"
 */
export function formatTime(timestamp: string): string {
  const date = new Date(timestamp)
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  const milliseconds = String(date.getMilliseconds()).padStart(3, '0')
  return `${hours}:${minutes}:${seconds}.${milliseconds}`
}

/**
 * Formats a duration in milliseconds into a human-readable string.
 * Uses seconds if >= 1000ms, otherwise milliseconds.
 * @param ms - Duration in milliseconds
 * @returns A formatted string like "123ms" or "1.5s"
 */
export function formatDuration(ms: number): string {
  if (!ms || ms <= 0) return '-'
  if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`
  return `${ms.toFixed(0)}ms`
}

/**
 * Returns Tailwind CSS classes for coloring protocol badges.
 * @param protocol - The protocol type ('HTTP', 'HTTPS', 'TLS', 'TCP')
 * @returns Tailwind CSS classes for text and background colors
 */
export function getProtocolColor(protocol: Protocol): string {
  switch (protocol) {
    case 'HTTP':
      return 'text-green-400 bg-green-400/10'
    case 'HTTPS':
    case 'TLS':
      return 'text-yellow-400 bg-yellow-400/10'
    default:
      return 'text-blue-400 bg-blue-400/10'
  }
}

/**
 * Returns Tailwind CSS classes for coloring status indicators.
 * @param status - The flow status ('OPEN', 'CLOSED', 'RESET', 'TIMEOUT')
 * @param httpCode - Optional HTTP status code (takes precedence if provided)
 * @returns Tailwind CSS classes for text color
 */
export function getStatusColor(status: string, httpCode?: number): string {
  if (httpCode) {
    if (httpCode >= 200 && httpCode < 300) return 'text-green-400'
    if (httpCode >= 300 && httpCode < 400) return 'text-blue-400'
    if (httpCode >= 400 && httpCode < 500) return 'text-yellow-400'
    if (httpCode >= 500) return 'text-red-400'
  }
  switch (status) {
    case 'CLOSED':
      return 'text-green-400'
    case 'RESET':
      return 'text-red-400'
    case 'TIMEOUT':
      return 'text-yellow-400'
    default:
      return 'text-blue-400'
  }
}
