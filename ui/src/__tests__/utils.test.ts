import { describe, it, expect } from 'vitest'
import { formatBytes } from '../utils'

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
