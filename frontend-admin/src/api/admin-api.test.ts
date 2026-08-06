// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from 'vitest'

import { AdminAPIError, adminRequest } from './admin-api'

describe('admin API structured validation errors', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('exposes backend field errors to dynamic forms', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: {
              code: 'validation_failed',
              message: 'request data is invalid',
              details: {
                fields: [{ key: 'hero_title', rule: 'required', param: '' }],
              },
            },
          }),
          {
            status: 422,
            headers: { 'Content-Type': 'application/json' },
          },
        ),
      ),
    )

    const error = await adminRequest('/api/admin/example', 'token').catch(
      (caught) => caught,
    )
    expect(error).toBeInstanceOf(AdminAPIError)
    expect((error as AdminAPIError).fieldErrors).toEqual([
      { key: 'hero_title', rule: 'required', param: '' },
    ])
  })
})
