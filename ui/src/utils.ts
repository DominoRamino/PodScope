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
