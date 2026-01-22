import { describe, it, expect } from 'vitest'
import { formatBytes, formatTime, getProtocolColor, getStatusColor } from '../utils'

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

describe('getProtocolColor', () => {
  describe('HTTP protocol', () => {
    it('returns green color classes for HTTP', () => {
      const result = getProtocolColor('HTTP')
      expect(result).toContain('text-green-400')
      expect(result).toContain('bg-green-400/10')
    })
  })

  describe('HTTPS protocol', () => {
    it('returns yellow/amber color classes for HTTPS', () => {
      const result = getProtocolColor('HTTPS')
      expect(result).toContain('text-yellow-400')
      expect(result).toContain('bg-yellow-400/10')
    })
  })

  describe('TLS protocol', () => {
    it('returns yellow/amber color classes for TLS (same as HTTPS)', () => {
      const result = getProtocolColor('TLS')
      expect(result).toContain('text-yellow-400')
      expect(result).toContain('bg-yellow-400/10')
    })
  })

  describe('TCP protocol (default)', () => {
    it('returns blue color classes for TCP', () => {
      const result = getProtocolColor('TCP')
      expect(result).toContain('text-blue-400')
      expect(result).toContain('bg-blue-400/10')
    })
  })

  describe('consistency between protocols', () => {
    it('HTTPS and TLS return identical colors', () => {
      expect(getProtocolColor('HTTPS')).toBe(getProtocolColor('TLS'))
    })

    it('HTTP uses different colors than HTTPS', () => {
      expect(getProtocolColor('HTTP')).not.toBe(getProtocolColor('HTTPS'))
    })

    it('TCP uses different colors than HTTP', () => {
      expect(getProtocolColor('TCP')).not.toBe(getProtocolColor('HTTP'))
    })
  })
})

describe('getStatusColor', () => {
  describe('HTTP status codes', () => {
    describe('2xx success codes', () => {
      it('returns green for 200 OK', () => {
        const result = getStatusColor('CLOSED', 200)
        expect(result).toBe('text-green-400')
      })

      it('returns green for 201 Created', () => {
        const result = getStatusColor('CLOSED', 201)
        expect(result).toBe('text-green-400')
      })

      it('returns green for 204 No Content', () => {
        const result = getStatusColor('CLOSED', 204)
        expect(result).toBe('text-green-400')
      })

      it('returns green for 299 (upper boundary)', () => {
        const result = getStatusColor('CLOSED', 299)
        expect(result).toBe('text-green-400')
      })
    })

    describe('3xx redirection codes', () => {
      it('returns blue for 301 Moved Permanently', () => {
        const result = getStatusColor('CLOSED', 301)
        expect(result).toBe('text-blue-400')
      })

      it('returns blue for 302 Found', () => {
        const result = getStatusColor('CLOSED', 302)
        expect(result).toBe('text-blue-400')
      })

      it('returns blue for 304 Not Modified', () => {
        const result = getStatusColor('CLOSED', 304)
        expect(result).toBe('text-blue-400')
      })
    })

    describe('4xx client error codes', () => {
      it('returns yellow/amber for 400 Bad Request', () => {
        const result = getStatusColor('CLOSED', 400)
        expect(result).toBe('text-yellow-400')
      })

      it('returns yellow/amber for 401 Unauthorized', () => {
        const result = getStatusColor('CLOSED', 401)
        expect(result).toBe('text-yellow-400')
      })

      it('returns yellow/amber for 403 Forbidden', () => {
        const result = getStatusColor('CLOSED', 403)
        expect(result).toBe('text-yellow-400')
      })

      it('returns yellow/amber for 404 Not Found', () => {
        const result = getStatusColor('CLOSED', 404)
        expect(result).toBe('text-yellow-400')
      })

      it('returns yellow/amber for 429 Too Many Requests', () => {
        const result = getStatusColor('CLOSED', 429)
        expect(result).toBe('text-yellow-400')
      })

      it('returns yellow/amber for 499 (upper boundary)', () => {
        const result = getStatusColor('CLOSED', 499)
        expect(result).toBe('text-yellow-400')
      })
    })

    describe('5xx server error codes', () => {
      it('returns red for 500 Internal Server Error', () => {
        const result = getStatusColor('CLOSED', 500)
        expect(result).toBe('text-red-400')
      })

      it('returns red for 502 Bad Gateway', () => {
        const result = getStatusColor('CLOSED', 502)
        expect(result).toBe('text-red-400')
      })

      it('returns red for 503 Service Unavailable', () => {
        const result = getStatusColor('CLOSED', 503)
        expect(result).toBe('text-red-400')
      })

      it('returns red for 504 Gateway Timeout', () => {
        const result = getStatusColor('CLOSED', 504)
        expect(result).toBe('text-red-400')
      })

      it('returns red for high 5xx codes like 599', () => {
        const result = getStatusColor('CLOSED', 599)
        expect(result).toBe('text-red-400')
      })
    })
  })

  describe('HTTP code takes precedence over status', () => {
    it('HTTP 200 returns green even when status is RESET', () => {
      const result = getStatusColor('RESET', 200)
      expect(result).toBe('text-green-400')
    })

    it('HTTP 500 returns red even when status is CLOSED', () => {
      const result = getStatusColor('CLOSED', 500)
      expect(result).toBe('text-red-400')
    })

    it('HTTP 404 returns yellow even when status is TIMEOUT', () => {
      const result = getStatusColor('TIMEOUT', 404)
      expect(result).toBe('text-yellow-400')
    })
  })

  describe('flow status without HTTP code', () => {
    it('returns green for CLOSED status', () => {
      const result = getStatusColor('CLOSED')
      expect(result).toBe('text-green-400')
    })

    it('returns red for RESET status', () => {
      const result = getStatusColor('RESET')
      expect(result).toBe('text-red-400')
    })

    it('returns yellow for TIMEOUT status', () => {
      const result = getStatusColor('TIMEOUT')
      expect(result).toBe('text-yellow-400')
    })

    it('returns blue for OPEN status (default)', () => {
      const result = getStatusColor('OPEN')
      expect(result).toBe('text-blue-400')
    })

    it('returns blue for unknown status (default fallback)', () => {
      const result = getStatusColor('UNKNOWN')
      expect(result).toBe('text-blue-400')
    })
  })

  describe('edge cases', () => {
    it('returns status color when httpCode is undefined', () => {
      const result = getStatusColor('RESET', undefined)
      expect(result).toBe('text-red-400')
    })

    it('returns status color when httpCode is 0 (falsy)', () => {
      // 0 is falsy so status should be used
      const result = getStatusColor('RESET', 0)
      expect(result).toBe('text-red-400')
    })

    it('handles boundary between 2xx and 3xx', () => {
      expect(getStatusColor('CLOSED', 299)).toBe('text-green-400')
      expect(getStatusColor('CLOSED', 300)).toBe('text-blue-400')
    })

    it('handles boundary between 3xx and 4xx', () => {
      expect(getStatusColor('CLOSED', 399)).toBe('text-blue-400')
      expect(getStatusColor('CLOSED', 400)).toBe('text-yellow-400')
    })

    it('handles boundary between 4xx and 5xx', () => {
      expect(getStatusColor('CLOSED', 499)).toBe('text-yellow-400')
      expect(getStatusColor('CLOSED', 500)).toBe('text-red-400')
    })
  })
})
