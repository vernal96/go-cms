import { describe, expect, it } from 'vitest'

import { generateResourceCode } from './resource-code'

describe('generateResourceCode', () => {
  it('transliterates Russian and normalizes separators', () => {
    expect(generateResourceCode('  Новый раздел: Лето 2026! ')).toBe('novyy-razdel-leto-2026')
  })

  it('keeps Latin letters and digits in lowercase', () => {
    expect(generateResourceCode('Product API v2')).toBe('product-api-v2')
  })
})
