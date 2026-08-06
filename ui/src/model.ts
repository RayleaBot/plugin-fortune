export const FORTUNE_NAMES = ['大吉', '吉', '中吉', '小吉', '末吉', '凶', '大凶', '吉凶未定'] as const

export const EXPECTED_STARS: Record<string, readonly string[]> = {
  大吉: ['★★★★★★★'],
  吉: ['★★★★★★☆'],
  中吉: ['★★★★★☆☆'],
  小吉: ['★★★★☆☆☆'],
  末吉: ['★★★☆☆☆☆'],
  凶: ['★★☆☆☆☆☆', '★☆☆☆☆☆☆'],
  大凶: ['☆☆☆☆☆☆☆'],
  吉凶未定: ['???????'],
}

export const TIMEZONE_OPTIONS = [
  { value: 'Asia/Shanghai', offset: 'UTC+08:00', label: '中国标准时间' },
  { value: 'UTC', offset: 'UTC+00:00', label: '协调世界时' },
  { value: 'Etc/UTC', offset: 'UTC+00:00', label: 'UTC 标准名称' },
  { value: 'PRC', offset: 'UTC+08:00', label: '中国时区别名' },
  { value: 'Asia/Tokyo', offset: 'UTC+09:00', label: '日本标准时间' },
  { value: 'Asia/Seoul', offset: 'UTC+09:00', label: '韩国标准时间' },
  { value: 'Asia/Singapore', offset: 'UTC+08:00', label: '新加坡时间' },
  { value: 'Europe/London', offset: 'UTC+00:00/UTC+01:00', label: '伦敦时间' },
  { value: 'Europe/Paris', offset: 'UTC+01:00/UTC+02:00', label: '巴黎时间' },
  { value: 'America/New_York', offset: 'UTC-05:00/UTC-04:00', label: '纽约时间' },
  { value: 'America/Los_Angeles', offset: 'UTC-08:00/UTC-07:00', label: '洛杉矶时间' },
  { value: 'UTC+08:00', offset: 'UTC+08:00', label: '固定东八区' },
  { value: '+08:00', offset: 'UTC+08:00', label: '固定东八区简写' },
] as const

export interface FortuneItem {
  name: string
  stars: string
  sign: string
  explanation: string
}

export interface SpecialDate {
  date: string
  fortune_name: string
}

export interface FortuneSettings {
  trigger_commands: string[]
  stats_trigger_commands: string[]
  timezone: string
  fortunes: FortuneItem[]
  special_dates: SpecialDate[]
  good_actions: string[]
  bad_actions: string[]
}

export interface ValidationIssue {
  scope: string
  message: string
}

const defaultTriggerCommands = ['我的运势']
const defaultStatsTriggerCommands = ['运势统计']
const defaultGoodActions = ['整理计划']
const defaultBadActions = ['熬夜']

export function normalizeStringList(value: unknown, fallback: readonly string[] = []): string[] {
  const source = Array.isArray(value) ? value : fallback
  return [...new Set(source.map((item) => String(item ?? '').trim()).filter(Boolean))]
}

export function firstStarsForName(name: string): string {
  return EXPECTED_STARS[name]?.[0] ?? EXPECTED_STARS['大吉']![0]!
}

export function normalizeSettings(
  value: Record<string, unknown>,
  fallback: Partial<FortuneSettings> = {},
): FortuneSettings {
  const fortunesSource = Array.isArray(value.fortunes) ? value.fortunes : fallback.fortunes ?? []
  const specialDatesSource = Array.isArray(value.special_dates) ? value.special_dates : fallback.special_dates ?? []

  return {
    trigger_commands: normalizeStringList(
      value.trigger_commands,
      fallback.trigger_commands ?? defaultTriggerCommands,
    ),
    stats_trigger_commands: normalizeStringList(
      value.stats_trigger_commands,
      fallback.stats_trigger_commands ?? defaultStatsTriggerCommands,
    ),
    timezone: String(value.timezone || fallback.timezone || 'Asia/Shanghai').trim() || 'Asia/Shanghai',
    fortunes: fortunesSource.map((item) => {
      const source = isRecord(item) ? item : {}
      const name = String(source.name || '大吉').trim()
      return {
        name,
        stars: String(source.stars || firstStarsForName(name)).trim(),
        sign: String(source.sign || '').trim(),
        explanation: String(source.explanation || '').trim(),
      }
    }),
    special_dates: specialDatesSource.map((item) => {
      const source = isRecord(item) ? item : {}
      const legacyFortune = isRecord(source.fortune) ? source.fortune.name : source.fortune
      return {
        date: String(source.date || '').trim(),
        fortune_name: String(source.fortune_name || legacyFortune || '').trim(),
      }
    }),
    good_actions: normalizeStringList(value.good_actions, fallback.good_actions ?? defaultGoodActions),
    bad_actions: normalizeStringList(value.bad_actions, fallback.bad_actions ?? defaultBadActions),
  }
}

export function buildSettingsPayload(value: FortuneSettings): FortuneSettings {
  return {
    trigger_commands: normalizeStringList(value.trigger_commands),
    stats_trigger_commands: normalizeStringList(value.stats_trigger_commands),
    timezone: value.timezone.trim() || 'Asia/Shanghai',
    fortunes: value.fortunes.map((item) => ({
      name: item.name.trim(),
      stars: item.stars.trim(),
      sign: item.sign.trim(),
      explanation: item.explanation.trim(),
    })),
    special_dates: value.special_dates.map((item) => ({
      date: item.date.trim(),
      fortune_name: item.fortune_name.trim(),
    })),
    good_actions: normalizeStringList(value.good_actions),
    bad_actions: normalizeStringList(value.bad_actions),
  }
}

export function validateSettings(value: FortuneSettings): ValidationIssue[] {
  const errors: ValidationIssue[] = []
  const fortuneNames = new Set(value.fortunes.map((fortune) => fortune.name.trim()).filter(Boolean))

  if (value.fortunes.length === 0) {
    errors.push({ scope: 'fortunes', message: '运势库至少需要一条可用运势' })
  }

  value.fortunes.forEach((item, index) => {
    const scope = `fortune-${index}`
    if (!item.name.trim()) {
      errors.push({ scope, message: '运势名不能为空' })
    } else if (!EXPECTED_STARS[item.name]) {
      errors.push({ scope, message: '运势名不在支持范围内' })
    }
    if (!item.stars.trim()) {
      errors.push({ scope, message: '星级不能为空' })
    } else if (!(EXPECTED_STARS[item.name] ?? []).includes(item.stars)) {
      errors.push({ scope, message: '星级与运势名不匹配' })
    }
    if (!item.sign.trim()) errors.push({ scope, message: '签文不能为空' })
    if (!item.explanation.trim()) errors.push({ scope, message: '解签不能为空' })
  })

  value.special_dates.forEach((item, index) => {
    const scope = `special-${index}`
    if (!isSpecialDateKey(item.date)) {
      errors.push({ scope, message: '日期格式应为 YYYY-MM-DD 或 MM-DD，且须为真实有效日期' })
    }
    if (!item.fortune_name.trim()) {
      errors.push({ scope, message: '特殊日期需要指定运势' })
    } else if (!fortuneNames.has(item.fortune_name.trim())) {
      errors.push({ scope, message: '指定运势不在当前运势库中' })
    }
  })

  if (!isSupportedTimezoneInput(value.timezone)) {
    errors.push({ scope: 'timezone', message: '时区格式不正确' })
  }
  return errors
}

export function isSpecialDateKey(value: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value) && !/^\d{2}-\d{2}$/.test(value)) return false
  const parts = value.split('-').map(Number)
  if (parts.length === 3) {
    const [year, month, day] = parts as [number, number, number]
    const date = new Date(year, month - 1, day)
    return date.getFullYear() === year && date.getMonth() === month - 1 && date.getDate() === day
  }
  const [month, day] = parts as [number, number]
  const date = new Date(2024, month - 1, day)
  return date.getFullYear() === 2024 && date.getMonth() === month - 1 && date.getDate() === day
}

export function isSupportedTimezoneInput(value: string): boolean {
  const text = value.trim()
  if (!text) return false
  if (TIMEZONE_OPTIONS.some((item) => item.value === text)) return true
  if (/^(?:UTC)?[+-](?:\d|0\d|1[0-4])(?::?[0-5]\d)?$/i.test(text)) {
    const match = text.match(/([+-])(\d{1,2})(?::?(\d{2}))?$/)
    if (!match) return false
    const hours = Number(match[2])
    const minutes = Number(match[3] || '0')
    return hours < 14 || (hours === 14 && minutes === 0)
  }
  return /^[A-Za-z_]+(?:\/[A-Za-z0-9_+-]+)+$/.test(text)
}

export function stableSettingsJSON(value: FortuneSettings): string {
  return JSON.stringify(buildSettingsPayload(value))
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}
