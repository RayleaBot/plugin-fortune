import { describe, expect, it } from 'vitest'

import {
  buildSettingsPayload,
  isSpecialDateKey,
  isSupportedTimezoneInput,
  normalizeSettings,
  validateSettings,
} from '../src/model'

describe('fortune settings model', () => {
  it('normalizes duplicate triggers while preserving explicit empty chip lists', () => {
    const settings = normalizeSettings({ trigger_commands: ['运势', '运势'], fortunes: [] })
    expect(settings.trigger_commands).toEqual(['运势'])
    expect(buildSettingsPayload(normalizeSettings({ trigger_commands: [], stats_trigger_commands: [] }))).toMatchObject({
      trigger_commands: [],
      stats_trigger_commands: [],
    })
    expect(validateSettings(settings)).toContainEqual({ scope: 'fortunes', message: '运势库至少需要一条可用运势' })
  })

  it('validates calendar dates, stars and custom timezones like the original editor', () => {
    expect(isSpecialDateKey('02-29')).toBe(true)
    expect(isSpecialDateKey('2025-02-29')).toBe(false)
    expect(isSupportedTimezoneInput('America/New_York')).toBe(true)
    expect(isSupportedTimezoneInput('UTC+08:30')).toBe(true)

    const settings = normalizeSettings({
      timezone: 'Asia/Shanghai',
      fortunes: [{ name: '大吉', stars: '☆☆☆☆☆☆☆', sign: '签文', explanation: '解签' }],
      special_dates: [],
    })
    expect(validateSettings(settings)).toContainEqual({ scope: 'fortune-0', message: '星级与运势名不匹配' })
  })
})
