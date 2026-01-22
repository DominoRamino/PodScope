import { describe, it, expect } from 'vitest'
import { formatBytes, formatTime } from '../utils'

describe('formatBytes', () => {
  describe('zero bytes', () => {
    it('returns "0 B" for 0', () => {
      expect(formatBytes(0)).toBe('0 B')
    })
  })

  describe('values under 1KB', () => {
    it('shows bytes for small values', () => {
      expect(formatBytes(1)).toBe('1 B')
      expect(formatBytes(512)).toBe('512 B')
      expect(formatBytes(1023)).toBe('1023 B')
    })

    it('does not add decimals for byte values', () => {
      expect(formatBytes(100)).toBe('100 B')
      expect(formatBytes(999)).toBe('999 B')
    })
  })

  describe('KB values', () => {
    it('returns "1.00 KB" for exactly 1024 bytes', () => {
      expect(formatBytes(1024)).toBe('1.00 KB')
    })

    it('formats KB values correctly', () => {
      expect(formatBytes(1536)).toBe('1.50 KB')
      expect(formatBytes(2048)).toBe('2.00 KB')
      expect(formatBytes(10240)).toBe('10.00 KB')
    })

    it('uses 2 decimal places for KB', () => {
      expect(formatBytes(1024 + 512)).toBe('1.50 KB')
      expect(formatBytes(1024 + 256)).toBe('1.25 KB')
      // 1024 + 102.4 (0.1 KB) = 1126.4 bytes = 1.10 KB
      expect(formatBytes(1126)).toBe('1.10 KB')
    })

    it('handles upper KB range', () => {
      expect(formatBytes(1024 * 1023)).toBe('1023.00 KB')
    })
  })

  describe('MB values', () => {
    it('formats MB values correctly', () => {
      expect(formatBytes(1024 * 1024)).toBe('1.00 MB')
      expect(formatBytes(1024 * 1024 * 1.5)).toBe('1.50 MB')
      expect(formatBytes(1024 * 1024 * 10)).toBe('10.00 MB')
    })

    it('uses 2 decimal places for MB', () => {
      expect(formatBytes(1024 * 1024 * 1.25)).toBe('1.25 MB')
      expect(formatBytes(1024 * 1024 * 2.75)).toBe('2.75 MB')
    })

    it('handles upper MB range', () => {
      expect(formatBytes(1024 * 1024 * 999)).toBe('999.00 MB')
    })
  })

  describe('GB values', () => {
    it('formats GB values correctly', () => {
      expect(formatBytes(1024 * 1024 * 1024)).toBe('1.00 GB')
      expect(formatBytes(1024 * 1024 * 1024 * 1.5)).toBe('1.50 GB')
      expect(formatBytes(1024 * 1024 * 1024 * 10)).toBe('10.00 GB')
    })

    it('uses 2 decimal places for GB', () => {
      expect(formatBytes(1024 * 1024 * 1024 * 1.25)).toBe('1.25 GB')
      expect(formatBytes(1024 * 1024 * 1024 * 2.75)).toBe('2.75 GB')
    })

    it('handles large GB values', () => {
      expect(formatBytes(1024 * 1024 * 1024 * 100)).toBe('100.00 GB')
    })
  })

  describe('decimal precision', () => {
    it('always shows 2 decimal places for KB and above', () => {
      // Exactly on the boundary
      expect(formatBytes(1024)).toBe('1.00 KB')
      expect(formatBytes(1024 * 1024)).toBe('1.00 MB')
      expect(formatBytes(1024 * 1024 * 1024)).toBe('1.00 GB')

      // Fractional values
      expect(formatBytes(1536)).toMatch(/^\d+\.\d{2} KB$/)
      expect(formatBytes(1024 * 1024 + 512 * 1024)).toMatch(/^\d+\.\d{2} MB$/)
      expect(formatBytes(1024 * 1024 * 1024 + 512 * 1024 * 1024)).toMatch(/^\d+\.\d{2} GB$/)
    })
  })
})

describe('formatTime', () => {
  describe('format output', () => {
    it('formats as HH:MM:SS.mmm', () => {
      const timestamp = '2026-01-21T14:30:45.123Z'
      const result = formatTime(timestamp)
      // Check format pattern (actual time depends on local timezone)
      expect(result).toMatch(/^\d{2}:\d{2}:\d{2}\.\d{3}$/)
    })
  })

  describe('padding', () => {
    it('pads hours with leading zeros', () => {
      // 5 AM in UTC - will be converted to local timezone
      const timestamp = '2026-01-21T05:30:45.000Z'
      const result = formatTime(timestamp)
      // Should have format with leading zeros for hours
      expect(result).toMatch(/^\d{2}:\d{2}:\d{2}\.\d{3}$/)
    })

    it('pads minutes with leading zeros', () => {
      const timestamp = '2026-01-21T14:05:45.000Z'
      const result = formatTime(timestamp)
      // Minutes should be padded
      expect(result).toMatch(/^\d{2}:\d{2}:\d{2}\.\d{3}$/)
    })

    it('pads seconds with leading zeros', () => {
      const timestamp = '2026-01-21T14:30:05.000Z'
      const result = formatTime(timestamp)
      // Seconds should be padded
      expect(result).toMatch(/^\d{2}:\d{2}:\d{2}\.\d{3}$/)
    })

    it('pads milliseconds to 3 digits', () => {
      const timestamp = '2026-01-21T14:30:45.005Z'
      const result = formatTime(timestamp)
      // Should end with .005 or similar 3-digit milliseconds
      expect(result).toMatch(/\.\d{3}$/)
    })

    it('shows 000 for zero milliseconds', () => {
      const timestamp = '2026-01-21T14:30:45.000Z'
      const result = formatTime(timestamp)
      expect(result).toMatch(/\.000$/)
    })

    it('pads single-digit milliseconds correctly', () => {
      const timestamp = '2026-01-21T14:30:45.007Z'
      const result = formatTime(timestamp)
      expect(result).toMatch(/\.007$/)
    })

    it('pads double-digit milliseconds correctly', () => {
      const timestamp = '2026-01-21T14:30:45.042Z'
      const result = formatTime(timestamp)
      expect(result).toMatch(/\.042$/)
    })
  })

  describe('midnight (00:00:00.000)', () => {
    it('handles midnight correctly in UTC', () => {
      // Midnight UTC - will be some time in local timezone
      const timestamp = '2026-01-21T00:00:00.000Z'
      const result = formatTime(timestamp)
      // Should still have valid format
      expect(result).toMatch(/^\d{2}:\d{2}:\d{2}\.000$/)
    })
  })

  describe('end of day (23:59:59.999)', () => {
    it('handles end of day correctly in UTC', () => {
      const timestamp = '2026-01-21T23:59:59.999Z'
      const result = formatTime(timestamp)
      // Should handle all maximum values
      expect(result).toMatch(/^\d{2}:\d{2}:\d{2}\.999$/)
    })
  })

  describe('local timezone conversion', () => {
    it('converts UTC timestamp to local time', () => {
      // Create a known local time to verify conversion
      const now = new Date()
      const localHours = String(now.getHours()).padStart(2, '0')
      const localMinutes = String(now.getMinutes()).padStart(2, '0')
      const localSeconds = String(now.getSeconds()).padStart(2, '0')
      const localMs = String(now.getMilliseconds()).padStart(3, '0')

      const result = formatTime(now.toISOString())
      const expected = `${localHours}:${localMinutes}:${localSeconds}.${localMs}`
      expect(result).toBe(expected)
    })
  })

  describe('various timestamps', () => {
    it('handles noon correctly', () => {
      const timestamp = '2026-01-21T12:00:00.000Z'
      const result = formatTime(timestamp)
      expect(result).toMatch(/^\d{2}:\d{2}:\d{2}\.000$/)
    })

    it('handles random timestamp with all values', () => {
      const timestamp = '2026-06-15T18:45:32.789Z'
      const result = formatTime(timestamp)
      expect(result).toMatch(/^\d{2}:\d{2}:\d{2}\.789$/)
    })
  })
})
