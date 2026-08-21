/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { BILLING_CACHE_VAR_MAP } from './billing-expr'

export const CACHE_MODE_TIMED = 'timed'
export const CACHE_MODE_GENERIC = 'generic'
export type CacheMode = typeof CACHE_MODE_TIMED | typeof CACHE_MODE_GENERIC

export type TokenTierConditionInput = {
  type?: 'token'
  var: 'p' | 'c' | 'len'
  op: '<' | '<=' | '>' | '>='
  value: number | string
}

export type TimePeriodTierConditionInput = {
  type: 'time_period'
  timezone: string
  start: string
  end: string
}

export type TierConditionInput =
  | TokenTierConditionInput
  | TimePeriodTierConditionInput

export type VisualTier = {
  label: string
  conditions: TierConditionInput[]
  input_unit_cost: number
  output_unit_cost: number
  cache_mode: CacheMode
  cache_read_unit_cost?: number
  cache_create_unit_cost?: number
  cache_create_1h_unit_cost?: number
  image_unit_cost?: number
  image_output_unit_cost?: number
  audio_input_unit_cost?: number
  audio_output_unit_cost?: number
  [field: string]: unknown
}

export type VisualConfig = {
  tiers: VisualTier[]
}

export function getTierCacheMode(
  tier: Partial<VisualTier> | null | undefined
): CacheMode {
  if (tier?.cache_mode === CACHE_MODE_TIMED) return CACHE_MODE_TIMED
  if (tier?.cache_mode === CACHE_MODE_GENERIC) return CACHE_MODE_GENERIC
  return Number(tier?.cache_create_1h_unit_cost) > 0
    ? CACHE_MODE_TIMED
    : CACHE_MODE_GENERIC
}

export function normalizeVisualTier(
  tier: Partial<VisualTier> = {}
): VisualTier {
  const rawConditions = Array.isArray(tier.conditions) ? tier.conditions : []
  return {
    ...tier,
    label: tier.label ?? '',
    input_unit_cost: Number(tier.input_unit_cost) || 0,
    output_unit_cost: Number(tier.output_unit_cost) || 0,
    cache_mode: getTierCacheMode(tier),
    conditions: rawConditions.map(normalizeTierCondition),
    cache_read_unit_cost: Number(tier.cache_read_unit_cost) || 0,
    cache_create_unit_cost: Number(tier.cache_create_unit_cost) || 0,
    cache_create_1h_unit_cost: Number(tier.cache_create_1h_unit_cost) || 0,
    image_unit_cost: Number(tier.image_unit_cost) || 0,
    image_output_unit_cost: Number(tier.image_output_unit_cost) || 0,
    audio_input_unit_cost: Number(tier.audio_input_unit_cost) || 0,
    audio_output_unit_cost: Number(tier.audio_output_unit_cost) || 0,
  }
}

export function createDefaultVisualConfig(): VisualConfig {
  return {
    tiers: [
      normalizeVisualTier({
        conditions: [],
        input_unit_cost: 0,
        output_unit_cost: 0,
        label: 'base',
        cache_mode: CACHE_MODE_GENERIC,
      }),
    ],
  }
}

export function normalizeVisualConfig(
  config: VisualConfig | null | undefined
): VisualConfig {
  if (!config || !Array.isArray(config.tiers) || config.tiers.length === 0) {
    return createDefaultVisualConfig()
  }
  return {
    ...config,
    tiers: config.tiers.map((tier) => normalizeVisualTier(tier)),
  }
}

export function createEmptyTokenTierCondition(): TokenTierConditionInput {
  return { type: 'token', var: 'len', op: '<', value: 200000 }
}

export function createEmptyTimePeriodTierCondition(): TimePeriodTierConditionInput {
  return {
    type: 'time_period',
    timezone: 'Asia/Shanghai',
    start: '09:00',
    end: '12:00',
  }
}

export function isTimePeriodTierCondition(
  condition: TierConditionInput
): condition is TimePeriodTierConditionInput {
  return condition.type === 'time_period'
}

function normalizeTierCondition(
  condition: Partial<TierConditionInput> | null | undefined
): TierConditionInput {
  if (condition?.type === 'time_period') {
    return {
      type: 'time_period',
      timezone: condition.timezone || 'Asia/Shanghai',
      start: condition.start || '09:00',
      end: condition.end || '12:00',
    }
  }

  const tokenCondition = condition as Partial<TokenTierConditionInput>
  return {
    type: 'token',
    var:
      tokenCondition.var === 'p' ||
      tokenCondition.var === 'c' ||
      tokenCondition.var === 'len'
        ? tokenCondition.var
        : 'len',
    op:
      tokenCondition.op === '<' ||
      tokenCondition.op === '<=' ||
      tokenCondition.op === '>' ||
      tokenCondition.op === '>='
        ? tokenCondition.op
        : '<',
    value:
      tokenCondition.value == null || tokenCondition.value === ''
        ? 200000
        : tokenCondition.value,
  }
}

function parseClockMinutes(value: string): number | null {
  const match = String(value || '')
    .trim()
    .match(/^(\d{1,2}):(\d{2})$/)
  if (!match) return null
  const hour = Number(match[1])
  const minute = Number(match[2])
  if (hour < 0 || hour > 23 || minute < 0 || minute > 59) return null
  return hour * 60 + minute
}

function formatClockMinutes(minutes: number): string {
  const normalized = ((Math.trunc(minutes) % 1440) + 1440) % 1440
  const hour = Math.floor(normalized / 60)
  const minute = normalized % 60
  return `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`
}

function buildTimePeriodConditionStr(
  condition: TimePeriodTierConditionInput
): string {
  const start = parseClockMinutes(condition.start)
  const end = parseClockMinutes(condition.end)
  if (start === null || end === null || start === end) return ''

  const timezone = condition.timezone || 'Asia/Shanghai'
  const tz = JSON.stringify(timezone)
  const current = `(hour(${tz}) * 60 + minute(${tz}))`
  const op = start < end ? '&&' : '||'
  return `(${current} >= ${start} ${op} ${current} < ${end})`
}

function parseTimePeriodConditionStr(
  condition: string
): TimePeriodTierConditionInput | null {
  const match = condition.match(
    /^\(\(hour\("([^"]+)"\) \* 60 \+ minute\("\1"\)\) >= (\d+) (&&|\|\|) \(hour\("\1"\) \* 60 \+ minute\("\1"\)\) < (\d+)\)$/
  )
  if (!match) return null
  const start = Number(match[2])
  const end = Number(match[4])
  if (
    !Number.isInteger(start) ||
    !Number.isInteger(end) ||
    start < 0 ||
    start >= 1440 ||
    end < 0 ||
    end >= 1440 ||
    start === end
  ) {
    return null
  }
  const isCrossMidnight = start > end
  if ((match[3] === '||') !== isCrossMidnight) return null
  return {
    type: 'time_period',
    timezone: match[1],
    start: formatClockMinutes(start),
    end: formatClockMinutes(end),
  }
}

function buildConditionStr(conditions: TierConditionInput[]): string {
  if (!conditions || conditions.length === 0) return ''
  return conditions
    .map((condition) => {
      if (isTimePeriodTierCondition(condition)) {
        return buildTimePeriodConditionStr(condition)
      }
      if (
        !condition.var ||
        !condition.op ||
        condition.value == null ||
        condition.value === ''
      ) {
        return ''
      }
      return `${condition.var} ${condition.op} ${condition.value}`
    })
    .filter(Boolean)
    .join(' && ')
}

function buildTierBodyExpr(tier: VisualTier): string {
  const parts: string[] = []
  const ic = Number(tier.input_unit_cost) || 0
  const oc = Number(tier.output_unit_cost) || 0
  parts.push(`p * ${ic}`)
  parts.push(`c * ${oc}`)
  for (const cv of BILLING_CACHE_VAR_MAP) {
    const v = Number((tier as Record<string, unknown>)[cv.field]) || 0
    if (v !== 0) parts.push(`${cv.exprVar} * ${v}`)
  }
  return parts.join(' + ')
}

export function generateExprFromVisualConfig(
  config: VisualConfig | null | undefined
): string {
  if (!config || !config.tiers || config.tiers.length === 0) {
    return 'p * 0 + c * 0'
  }
  const tiers = config.tiers

  if (tiers.length === 1) {
    const tier = tiers[0]
    const label = tier.label || 'default'
    const body = `tier("${label}", ${buildTierBodyExpr(tier)})`
    const cond = buildConditionStr(tier.conditions)
    if (cond) {
      return `${cond} ? ${body} : p * 0 + c * 0`
    }
    return body
  }

  const parts: string[] = []
  for (let i = 0; i < tiers.length; i++) {
    const tier = tiers[i]
    const label = tier.label || `tier_${i + 1}`
    const body = `tier("${label}", ${buildTierBodyExpr(tier)})`
    const cond = buildConditionStr(tier.conditions)

    if (i < tiers.length - 1 && cond) {
      parts.push(`${cond} ? ${body}`)
    } else {
      parts.push(body)
    }
  }
  return parts.join(' : ')
}

export function tryParseVisualConfig(
  exprStr: string | null | undefined
): VisualConfig | null {
  if (!exprStr) return null
  try {
    let body = exprStr
    const versionMatch = body.match(/^v\d+:([\s\S]*)$/)
    if (versionMatch) body = versionMatch[1]
    const cacheVarNames = BILLING_CACHE_VAR_MAP.map((cv) => cv.exprVar)
    const optCacheStr = cacheVarNames
      .map((v) => `(?:\\s*\\+\\s*${v}\\s*\\*\\s*([\\d.eE+-]+))?`)
      .join('')

    const bodyPat = `p\\s*\\*\\s*([\\d.eE+-]+)\\s*\\+\\s*c\\s*\\*\\s*([\\d.eE+-]+)${optCacheStr}`

    const singleRe = new RegExp(`^tier\\("([^"]*)",\\s*${bodyPat}\\)$`)
    const simple = body.match(singleRe)
    if (simple) {
      const tier: Record<string, unknown> = {
        conditions: [],
        input_unit_cost: Number(simple[2]),
        output_unit_cost: Number(simple[3]),
        label: simple[1],
      }
      BILLING_CACHE_VAR_MAP.forEach((cv, i) => {
        const val = simple[4 + i]
        if (val != null) tier[cv.field] = Number(val)
      })
      return normalizeVisualConfig({
        tiers: [normalizeVisualTier(tier as Partial<VisualTier>)],
      })
    }

    const tierRe = new RegExp(`^tier\\("([^"]*)",\\s*${bodyPat}\\)`)
    const tiers: VisualTier[] = []
    let remaining = body.trim()
    while (remaining) {
      let condStr = ''
      if (!remaining.startsWith('tier(')) {
        const questionIndex = findTopLevelQuestionMark(remaining)
        if (questionIndex < 0) return null
        condStr = remaining.slice(0, questionIndex).trim()
        remaining = remaining.slice(questionIndex + 1).trim()
      }
      const match = remaining.match(tierRe)
      if (!match) return null
      const conditions = parseTierConditions(condStr)
      if (condStr && conditions.length === 0) return null
      const tier: Record<string, unknown> = {
        conditions,
        input_unit_cost: Number(match[2]),
        output_unit_cost: Number(match[3]),
        label: match[1],
      }
      BILLING_CACHE_VAR_MAP.forEach((cv, i) => {
        const val = match[4 + i]
        if (val != null) tier[cv.field] = Number(val)
      })
      tiers.push(normalizeVisualTier(tier as Partial<VisualTier>))
      remaining = remaining.slice(match[0].length).trim()
      if (!remaining) break
      if (!remaining.startsWith(':')) return null
      remaining = remaining.slice(1).trim()
      if (remaining === 'p * 0 + c * 0') break
    }
    if (tiers.length === 0) return null

    const cfg = normalizeVisualConfig({ tiers })
    const regenerated = generateExprFromVisualConfig(cfg)
    if (regenerated.replace(/\s+/g, '') !== body.replace(/\s+/g, '')) {
      return null
    }
    return cfg
  } catch {
    return null
  }
}

function findTopLevelQuestionMark(expr: string): number {
  let depth = 0
  for (let index = 0; index < expr.length; index += 1) {
    const char = expr[index]
    if (char === '(') depth += 1
    if (char === ')') depth -= 1
    if (char === '?' && depth === 0) return index
  }
  return -1
}

function splitTopLevelAnd(expr: string): string[] {
  const parts: string[] = []
  let start = 0
  let depth = 0
  for (let index = 0; index < expr.length; index += 1) {
    const char = expr[index]
    if (char === '(') depth += 1
    if (char === ')') depth -= 1
    if (depth === 0 && expr.slice(index, index + 4) === ' && ') {
      parts.push(expr.slice(start, index).trim())
      start = index + 4
      index += 3
    }
  }
  parts.push(expr.slice(start).trim())
  return parts.filter(Boolean)
}

function parseTierConditions(condStr: string): TierConditionInput[] {
  if (!condStr) return []
  const conditions: TierConditionInput[] = []
  for (const part of splitTopLevelAnd(condStr)) {
    const timeCondition = parseTimePeriodConditionStr(part)
    if (timeCondition) {
      conditions.push(timeCondition)
      continue
    }
    const tokenMatch = part
      .trim()
      .match(/^(p|c|len)\s*(<|<=|>|>=)\s*([\d.eE+]+)$/)
    if (!tokenMatch) return []
    conditions.push({
      type: 'token',
      var: tokenMatch[1] as TokenTierConditionInput['var'],
      op: tokenMatch[2] as TokenTierConditionInput['op'],
      value: Number(tokenMatch[3]),
    })
  }
  return conditions
}

// ---------------------------------------------------------------------------
// Local cost evaluator (for the estimator preview)
// ---------------------------------------------------------------------------

const ESTIMATOR_VARS = [
  { var: 'cr', stateKey: 'cacheReadTokens' },
  { var: 'cc', stateKey: 'cacheCreateTokens' },
  { var: 'cc1h', stateKey: 'cacheCreate1hTokens' },
  { var: 'img', stateKey: 'imageTokens' },
  { var: 'img_o', stateKey: 'imageOutputTokens' },
  { var: 'ai', stateKey: 'audioInputTokens' },
  { var: 'ao', stateKey: 'audioOutputTokens' },
] as const

export type ExtraTokenValues = Record<
  (typeof ESTIMATOR_VARS)[number]['stateKey'],
  number
>

export type EvalResult = {
  cost: number
  matchedTier: string
  error: string | null
}

export function evalExprLocally(
  exprStr: string,
  promptTokens: number,
  completionTokens: number,
  extraTokenValues: ExtraTokenValues
): EvalResult {
  try {
    if (!exprStr || !exprStr.trim()) {
      return { cost: 0, matchedTier: '', error: null }
    }
    let matchedTier = ''
    const tierFn = (name: string, value: number) => {
      matchedTier = name
      return value
    }
    const cacheReadTokens = extraTokenValues.cacheReadTokens || 0
    const cacheCreateTokens = extraTokenValues.cacheCreateTokens || 0
    const cacheCreate1hTokens = extraTokenValues.cacheCreate1hTokens || 0
    const len =
      promptTokens + cacheReadTokens + cacheCreateTokens + cacheCreate1hTokens
    const env: Record<string, unknown> = {
      p: promptTokens,
      c: completionTokens,
      len,
      tier: tierFn,
      hour: (tz: string) => getTimePart(tz, 'hour'),
      minute: (tz: string) => getTimePart(tz, 'minute'),
      weekday: (tz: string) => getTimePart(tz, 'weekday'),
      month: (tz: string) => getTimePart(tz, 'month'),
      day: (tz: string) => getTimePart(tz, 'day'),
      max: Math.max,
      min: Math.min,
      abs: Math.abs,
      ceil: Math.ceil,
      floor: Math.floor,
    }
    for (const field of ESTIMATOR_VARS) {
      env[field.var] = extraTokenValues[field.stateKey] || 0
    }
    const fn = new Function(
      ...Object.keys(env),
      `"use strict"; return (${exprStr});`
    )
    const cost = Number(fn(...Object.values(env))) || 0
    return { cost, matchedTier, error: null }
  } catch (e) {
    const message = e instanceof Error ? e.message : String(e)
    return { cost: 0, matchedTier: '', error: message }
  }
}

function getTimePart(
  timezone: string,
  part: 'hour' | 'minute' | 'weekday' | 'month' | 'day'
): number {
  const date = new Date()
  const values = new Intl.DateTimeFormat('en-US', {
    timeZone: timezone || 'UTC',
    hour: '2-digit',
    minute: '2-digit',
    weekday: 'short',
    month: 'numeric',
    day: 'numeric',
    hour12: false,
  })
    .formatToParts(date)
    .reduce<Record<string, string>>((acc, item) => {
      acc[item.type] = item.value
      return acc
    }, {})

  if (part === 'weekday') {
    const weekdays: Record<string, number> = {
      Sun: 0,
      Mon: 1,
      Tue: 2,
      Wed: 3,
      Thu: 4,
      Fri: 5,
      Sat: 6,
    }
    return weekdays[values.weekday] ?? 0
  }
  return Number(values[part]) || 0
}

export function exprUsesExtraVars(exprStr: string): boolean {
  if (!exprStr) return false
  const varNames = ESTIMATOR_VARS.map((f) => f.var).join('|')
  return new RegExp(`\\b(${varNames})\\b`).test(exprStr)
}

export const ESTIMATOR_EXTRA_FIELDS = ESTIMATOR_VARS
