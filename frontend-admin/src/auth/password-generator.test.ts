import { describe, expect, it } from 'vitest'

import { generatePassword } from './password-generator'

describe('generatePassword', () => {
  it('creates a 20 character password with every required character class', () => {
    const password = generatePassword()
    expect(password).toHaveLength(20)
    expect(password).toMatch(/[A-Z]/)
    expect(password).toMatch(/[a-z]/)
    expect(password).toMatch(/[0-9]/)
    expect(password).toMatch(/[!@#$%&*+\-_=]/)
  })
})
