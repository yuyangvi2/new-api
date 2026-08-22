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
/**
 * Billing expression parsing utilities.
 *
 * Mirrors the parser used by the classic frontend so that the dynamic
 * pricing breakdown UI can be rendered from the same backend expressions.
 *
 * The grammar is intentionally narrow: we only support the shapes that the
 * server emits (tiered pricing + request-rule conditional multipliers), so
 * the regular expressions are exact rather than tolerant of arbitrary
 * expression syntax.
 */

// ---------------------------------------------------------------------------
// Variable registry
// ---------------------------------------------------------------------------

export type BillingVar = {
  key: string
  field: string | null
  tierField: string | null
  label: string
  shortLabel: string
  side: 'input' | 'output' | 'condition'
  isBase?: boolean
  isConditionOnly?: boolean
  group?: string
}

export const BILLING_VARS: BillingVar[] = [
  {
    key: 'p',
    field: 'inputPrice',
    tierField: 'input_unit_cost',
    label: 'Input price',
    shortLabel: 'Input',
    side: 'input',
    isBase: true,
  },
  {
    key: 'c',
    field: 'outputPrice',
    tierField: 'output_unit_cost',
    label: 'Completion price',
    shortLabel: 'Output',
    side: 'output',
    isBase: true,
  },
  {
    key: 'len',
    field: null,
    tierField: null,
    label: 'Input length',
    shortLabel: 'Length',
    side: 'condition',
    isConditionOnly: true,
  },
  {
    key: 'cr',
    field: 'cacheReadPrice',
    tierField: 'cache_read_unit_cost',
    label: 'Cache read price',
    shortLabel: 'Cache Read',
    side: 'input',
    group: 'cache',
  },
  {
    key: 'cc',
    field: 'cacheCreatePrice',
    tierField: 'cache_create_unit_cost',
    label: 'Cache create price',
    shortLabel: 'Cache Write',
    side: 'input',
    group: 'cache',
  },
  {
    key: 'cc1h',
    field: 'cacheCreate1hPrice',
    tierField: 'cache_create_1h_unit_cost',
    label: 'Cache create (1h) price',
    shortLabel: 'Cache Write (1h)',
    side: 'input',
    group: 'cache',
  },
  {
    key: 'img',
    field: 'imagePrice',
    tierField: 'image_unit_cost',
    label: 'Image input price',
    shortLabel: 'Image In',
    side: 'input',
    group: 'media',
  },
  {
    key: 'img_o',
    field: 'imageOutputPrice',
    tierField: 'image_output_unit_cost',
    label: 'Image output price',
    shortLabel: 'Image Out',
    side: 'output',
    group: 'media',
  },
  {
    key: 'ai',
    field: 'audioInputPrice',
    tierField: 'audio_input_unit_cost',
    label: 'Audio input price',
    shortLabel: 'Audio In',
    side: 'input',
    group: 'media',
  },
  {
    key: 'ao',
    field: 'audioOutputPrice',
    tierField: 'audio_output_unit_cost',
    label: 'Audio output price',
    shortLabel: 'Audio Out',
    side: 'output',
    group: 'media',
  },
]

/** Vars that have real price fields (excludes condition-only vars like `len`) */
export const BILLING_PRICING_VARS: BillingVar[] = BILLING_VARS.filter(
  (v) => !v.isConditionOnly
)

/** Vars valid in tier conditions (`p`, `c`, `len`) */
export const BILLING_CONDITION_VARS: string[] = BILLING_VARS.filter(
  (v) => v.isBase || v.isConditionOnly
).map((v) => v.key)

const BILLING_VAR_KEY_TO_FIELD = Object.fromEntries(
  BILLING_PRICING_VARS.map((v) => [v.key, v.field as string])
) as Record<string, string>

export const BILLING_EXTRA_VARS: BillingVar[] = BILLING_VARS.filter(
  (v) => !v.isBase && !v.isConditionOnly
)

export const BILLING_CACHE_VAR_MAP = BILLING_EXTRA_VARS.map((v) => ({
  field: v.tierField as string,
  exprVar: v.key,
}))

const BILLING_VAR_REGEX = new RegExp(
  `\\b(${BILLING_PRICING_VARS.map((v) => v.key).join('|')})\\s*\\*\\s*([\\d.eE+-]+)`,
  'g'
)

// ---------------------------------------------------------------------------
// Request rule constants
// ---------------------------------------------------------------------------

export const SOURCE_PARAM = 'param'
export const SOURCE_HEADER = 'header'
export const SOURCE_TIME = 'time'

export const MATCH_EQ = 'eq'
export const MATCH_CONTAINS = 'contains'
export const MATCH_GT = 'gt'
export const MATCH_GTE = 'gte'
export const MATCH_LT = 'lt'
export const MATCH_LTE = 'lte'
export const MATCH_EXISTS = 'exists'
export const MATCH_RANGE = 'range'

export const TIME_FUNCS = ['hour', 'minute', 'weekday', 'month', 'day'] as const
export type TimeFunc = (typeof TIME_FUNCS)[number]

export const COMMON_TIMEZONES: { value: string; label: string }[] = [
  { value: 'Asia/Shanghai', label: 'UTC+8 Shanghai (Asia/Shanghai)' },
  { value: 'UTC', label: 'UTC' },
  { value: 'America/New_York', label: 'UTC-5 New York (America/New_York)' },
  {
    value: 'America/Los_Angeles',
    label: 'UTC-8 Los Angeles (America/Los_Angeles)',
  },
  { value: 'America/Chicago', label: 'UTC-6 Chicago (America/Chicago)' },
  { value: 'Europe/London', label: 'UTC+0 London (Europe/London)' },
  { value: 'Europe/Berlin', label: 'UTC+1 Berlin (Europe/Berlin)' },
  { value: 'Asia/Tokyo', label: 'UTC+9 Tokyo (Asia/Tokyo)' },
  { value: 'Asia/Singapore', label: 'UTC+8 Singapore (Asia/Singapore)' },
  { value: 'Asia/Seoul', label: 'UTC+9 Seoul (Asia/Seoul)' },
  { value: 'Australia/Sydney', label: 'UTC+10 Sydney (Australia/Sydney)' },
]

const NUMERIC_LITERAL_REGEX = /^-?(?:\d+\.?\d*|\.\d+)(?:[eE][+-]?\d+)?$/

export type ParamHeaderCondition = {
  source: 'param' | 'header'
  path: string
  mode: string
  value: string
}

export type TimeCondition = {
  source: 'time'
  timeFunc: TimeFunc
  timezone: string
  mode: string
  value: string
  rangeStart: string
  rangeEnd: string
}

export type RequestCondition = TimeCondition | ParamHeaderCondition

export type RequestRuleGroup = {
  conditions: RequestCondition[]
  multiplier: string
}

export type TierCondition = {
  var: 'p' | 'c' | 'len'
  op: '<' | '<=' | '>' | '>='
  value: number
}

export type ParsedTier = {
  label: string
  conditions: TierCondition[]
  multiplierMin?: number
  multiplierMax?: number
  [field: string]: unknown
}

// ---------------------------------------------------------------------------
// Tier parser
// ---------------------------------------------------------------------------

function stripExprVersion(exprStr: string): { version: number; body: string } {
  if (!exprStr) return { version: 1, body: '' }
  const m = exprStr.match(/^v(\d+):([\s\S]*)$/)
  if (m) return { version: Number(m[1]), body: m[2] }
  return { version: 1, body: exprStr }
}

function parseTierBody(bodyStr: string): Record<string, number> {
  const coeffs: Record<string, number> = {}
  const re = new RegExp(BILLING_VAR_REGEX.source, 'g')
  let m
  while ((m = re.exec(bodyStr)) !== null) {
    if (!(m[1] in coeffs)) coeffs[m[1]] = Number(m[2])
  }
  const tier: Record<string, number> = {}
  for (const [varName, field] of Object.entries(BILLING_VAR_KEY_TO_FIELD)) {
    tier[field] = coeffs[varName] || 0
  }
  return tier
}

function findTopLevelTernary(expr: string): {
  questionIndex: number
  colonIndex: number
} | null {
  let depth = 0
  let questionIndex = -1
  for (let index = 0; index < expr.length; index += 1) {
    const char = expr[index]
    if (char === '(') depth += 1
    if (char === ')') depth -= 1
    if (depth !== 0) continue
    if (char === '?' && questionIndex < 0) {
      questionIndex = index
      continue
    }
    if (char === ':' && questionIndex >= 0) {
      return { questionIndex, colonIndex: index }
    }
  }
  return null
}

function parsePositiveNumericLiteral(expr: string): number | null {
  const body = unwrapOuterParens(expr)
  if (!NUMERIC_LITERAL_REGEX.test(body)) return null
  const value = Number(body)
  if (!Number.isFinite(value) || value <= 0) return null
  return value
}

function parseMultiplierValues(expr: string): number[] | null {
  const body = unwrapOuterParens(expr)
  const directValue = parsePositiveNumericLiteral(body)
  if (directValue !== null) return [directValue]

  const ternary = findTopLevelTernary(body)
  if (!ternary) return null

  const whenTrue = parsePositiveNumericLiteral(
    body.slice(ternary.questionIndex + 1, ternary.colonIndex)
  )
  const whenFalse = parsePositiveNumericLiteral(
    body.slice(ternary.colonIndex + 1)
  )
  if (whenTrue === null || whenFalse === null) return null
  return [whenTrue, whenFalse]
}

function parseTopLevelMultiplierRange(expr: string): {
  min: number
  max: number
} | null {
  const parts = splitTopLevelMultiply(expr)
  if (parts.length < 2 || !unwrapOuterParens(parts[0]).startsWith('tier(')) {
    return null
  }

  let min = 1
  let max = 1
  for (const part of parts.slice(1)) {
    const values = parseMultiplierValues(part)
    if (!values) return null
    min *= Math.min(...values)
    max *= Math.max(...values)
  }

  return { min, max }
}

export function getRequestRuleMultiplierRange(expr: string): {
  min: number
  max: number
} | null {
  const groups = tryParseRequestRuleExpr(expr)
  if (!groups || groups.length === 0) return null

  const exclusiveMultipliers = getExclusiveTimeRangeMultipliers(groups)
  if (exclusiveMultipliers) {
    return {
      min: Math.min(1, ...exclusiveMultipliers),
      max: Math.max(1, ...exclusiveMultipliers),
    }
  }

  let min = 1
  let max = 1
  for (const group of groups) {
    const multiplier = Number(group.multiplier)
    if (!Number.isFinite(multiplier) || multiplier <= 0) return null
    min *= Math.min(multiplier, 1)
    max *= Math.max(multiplier, 1)
  }

  return { min, max }
}

function getTimeRangeDomain(timeFunc: TimeFunc): { min: number; max: number } {
  switch (timeFunc) {
    case 'hour':
      return { min: 0, max: 24 }
    case 'minute':
      return { min: 0, max: 60 }
    case 'weekday':
      return { min: 0, max: 7 }
    case 'month':
      return { min: 1, max: 13 }
    case 'day':
      return { min: 1, max: 32 }
    default:
      return { min: 0, max: 24 }
  }
}

function getTimeRangeSegments(
  condition: TimeCondition
): Array<{ start: number; end: number }> | null {
  if (condition.mode !== MATCH_RANGE) return null
  const start = Number(condition.rangeStart)
  const end = Number(condition.rangeEnd)
  if (!Number.isFinite(start) || !Number.isFinite(end) || start === end) {
    return null
  }

  const domain = getTimeRangeDomain(condition.timeFunc)
  if (
    start < domain.min ||
    start >= domain.max ||
    end < domain.min ||
    end >= domain.max
  ) {
    return null
  }

  if (start < end) return [{ start, end }]
  return [
    { start, end: domain.max },
    { start: domain.min, end },
  ]
}

function timeSegmentsOverlap(
  left: { start: number; end: number },
  right: { start: number; end: number }
): boolean {
  return left.start < right.end && right.start < left.end
}

type ExclusiveTimeRangeCandidate = {
  timeFunc: TimeFunc
  timezone: string
  guardSignature: string
  multiplier: number
  segments: Array<{ start: number; end: number }>
}

function getConditionSignature(condition: RequestCondition): string {
  if (condition.source === SOURCE_TIME) {
    return [
      condition.source,
      condition.timeFunc,
      condition.timezone,
      condition.mode,
      condition.value,
      condition.rangeStart,
      condition.rangeEnd,
    ].join(':')
  }
  return [
    condition.source,
    condition.path,
    condition.mode,
    condition.value,
  ].join(':')
}

function getGuardSignature(
  conditions: RequestCondition[],
  ignoredIndexes: Set<number>
): string {
  return conditions
    .flatMap((condition, index) =>
      ignoredIndexes.has(index) ? [] : [getConditionSignature(condition)]
    )
    .sort()
    .join('|')
}

function createRangeCandidate(
  group: RequestRuleGroup,
  condition: TimeCondition,
  ignoredIndexes: Set<number>
): ExclusiveTimeRangeCandidate | null {
  const multiplier = Number(group.multiplier)
  const segments = getTimeRangeSegments(condition)
  if (!Number.isFinite(multiplier) || multiplier <= 0 || !segments) {
    return null
  }

  return {
    timeFunc: condition.timeFunc,
    timezone: condition.timezone,
    guardSignature: getGuardSignature(group.conditions, ignoredIndexes),
    multiplier,
    segments,
  }
}

function getExclusiveTimeRangeCandidates(
  group: RequestRuleGroup
): ExclusiveTimeRangeCandidate[] {
  const candidates: ExclusiveTimeRangeCandidate[] = []

  group.conditions.forEach((condition, index) => {
    if (condition.source !== SOURCE_TIME || condition.mode !== MATCH_RANGE) {
      return
    }
    const candidate = createRangeCandidate(
      group,
      condition as TimeCondition,
      new Set([index])
    )
    if (candidate) candidates.push(candidate)
  })

  for (
    let startIndex = 0;
    startIndex < group.conditions.length;
    startIndex += 1
  ) {
    const startCondition = group.conditions[startIndex]
    if (
      startCondition?.source !== SOURCE_TIME ||
      startCondition.mode !== MATCH_GTE
    ) {
      continue
    }

    for (let endIndex = 0; endIndex < group.conditions.length; endIndex += 1) {
      const endCondition = group.conditions[endIndex]
      if (
        endCondition?.source !== SOURCE_TIME ||
        endCondition.mode !== MATCH_LT ||
        endCondition.timeFunc !== startCondition.timeFunc ||
        endCondition.timezone !== startCondition.timezone
      ) {
        continue
      }

      const candidate = createRangeCandidate(
        group,
        {
          source: SOURCE_TIME,
          timeFunc: startCondition.timeFunc,
          timezone: startCondition.timezone,
          mode: MATCH_RANGE,
          value: '',
          rangeStart: startCondition.value,
          rangeEnd: endCondition.value,
        },
        new Set([startIndex, endIndex])
      )
      if (candidate) candidates.push(candidate)
    }
  }

  return candidates
}

function getExclusiveTimeRangeMultipliers(
  groups: RequestRuleGroup[]
): number[] | null {
  const candidateGroups = groups.map(getExclusiveTimeRangeCandidates)
  if (candidateGroups.some((candidates) => candidates.length === 0)) {
    return null
  }

  for (const baseCandidate of candidateGroups[0]) {
    const timeRanges = [baseCandidate]
    for (const candidates of candidateGroups.slice(1)) {
      const match = candidates.find(
        (candidate) =>
          candidate.timeFunc === baseCandidate.timeFunc &&
          candidate.timezone === baseCandidate.timezone &&
          candidate.guardSignature === baseCandidate.guardSignature
      )
      if (!match) break
      timeRanges.push(match)
    }

    if (timeRanges.length !== groups.length) continue

    let hasOverlap = false
    for (let i = 0; i < timeRanges.length; i += 1) {
      for (let j = i + 1; j < timeRanges.length; j += 1) {
        const left = timeRanges[i]
        const right = timeRanges[j]
        if (
          left.segments.some((leftSegment) =>
            right.segments.some((rightSegment) =>
              timeSegmentsOverlap(leftSegment, rightSegment)
            )
          )
        ) {
          hasOverlap = true
          break
        }
      }
      if (hasOverlap) break
    }

    if (!hasOverlap) return timeRanges.map((range) => range.multiplier)
  }

  return null
}

export function applyMultiplierRangeToTier(
  tier: ParsedTier,
  range: { min: number; max: number } | null
): ParsedTier {
  if (!range) return tier

  const nextTier: ParsedTier = {
    ...tier,
    multiplierMin: range.min,
    multiplierMax: range.max,
  }

  for (const variable of BILLING_PRICING_VARS) {
    if (!variable.field) continue
    const value = Number(nextTier[variable.field])
    if (!Number.isFinite(value)) continue
    nextTier[variable.field] = value * range.min
    nextTier[`${variable.field}Max`] = value * range.max
  }

  return nextTier
}

export function parseTiersFromExpr(exprStr: string): ParsedTier[] {
  if (!exprStr) return []
  try {
    const { body } = stripExprVersion(exprStr)
    const multiplierRange = parseTopLevelMultiplierRange(body.trim())
    const condGroup =
      `((?:(?:p|c|len)\\s*(?:<|<=|>|>=)\\s*[\\d.eE+]+)` +
      `(?:\\s*&&\\s*(?:p|c|len)\\s*(?:<|<=|>|>=)\\s*[\\d.eE+]+)*)`
    const tierRe = new RegExp(
      `(?:${condGroup}\\s*\\?\\s*)?tier\\("([^"]*)",\\s*([^)]+)\\)`,
      'g'
    )
    const tiers: ParsedTier[] = []
    let m
    while ((m = tierRe.exec(body)) !== null) {
      const condStr = m[1] || ''
      const conditions: TierCondition[] = []
      if (condStr) {
        for (const cp of condStr.split(/\s*&&\s*/)) {
          const cm = cp.trim().match(/^(p|c|len)\s*(<|<=|>|>=)\s*([\d.eE+]+)$/)
          if (cm) {
            conditions.push({
              var: cm[1] as TierCondition['var'],
              op: cm[2] as TierCondition['op'],
              value: Number(cm[3]),
            })
          }
        }
      }
      const tier = parseTierBody(m[3]) as ParsedTier
      tier.label = m[2]
      tier.conditions = conditions
      tiers.push(applyMultiplierRangeToTier(tier, multiplierRange))
    }
    return tiers
  } catch {
    return []
  }
}

export function normalizeTierLabel(label: string | undefined): string {
  if (!label) return ''
  return label
    .replaceAll(/<[=＝]?|≤|＜[=＝]?/g, '<')
    .replaceAll(/>[=＝]?|≥|＞[=＝]?/g, '>')
    .replaceAll(/\s+/g, '')
    .toLowerCase()
}

// ---------------------------------------------------------------------------
// Request rule parser
// ---------------------------------------------------------------------------

function splitTopLevelMultiply(expr: string): string[] {
  const parts: string[] = []
  let start = 0
  let depth = 0
  for (let index = 0; index < expr.length; index += 1) {
    const char = expr[index]
    if (char === '(') depth += 1
    if (char === ')') depth -= 1
    if (depth === 0 && expr.slice(index, index + 3) === ' * ') {
      parts.push(expr.slice(start, index).trim())
      start = index + 3
      index += 2
    }
  }
  parts.push(expr.slice(start).trim())
  return parts.filter(Boolean)
}

function splitTopLevelAnd(expr: string): string[] {
  const parts: string[] = []
  let start = 0
  let depth = 0
  for (let i = 0; i < expr.length; i += 1) {
    const c = expr[i]
    if (c === '(') depth += 1
    if (c === ')') depth -= 1
    if (depth === 0 && expr.slice(i, i + 4) === ' && ') {
      parts.push(expr.slice(start, i).trim())
      start = i + 4
      i += 3
    }
  }
  parts.push(expr.slice(start).trim())
  return parts.filter(Boolean)
}

function splitTopLevelOr(expr: string): string[] {
  const parts: string[] = []
  let start = 0
  let depth = 0
  for (let i = 0; i < expr.length; i += 1) {
    const c = expr[i]
    if (c === '(') depth += 1
    if (c === ')') depth -= 1
    if (depth === 0 && expr.slice(i, i + 4) === ' || ') {
      parts.push(expr.slice(start, i).trim())
      start = i + 4
      i += 3
    }
  }
  parts.push(expr.slice(start).trim())
  return parts.filter(Boolean)
}

function parseExprLiteral(raw: string): string | null {
  const text = raw.trim()
  if (text === 'true' || text === 'false') return text
  if (NUMERIC_LITERAL_REGEX.test(text)) return text
  try {
    return JSON.parse(text) as string
  } catch {
    return null
  }
}

function tryParseMinuteOfDayRangeCondition(
  expr: string
): RequestCondition | null {
  const m = unwrapOuterParens(expr).match(
    /^\(hour\("([^"]+)"\) \* 60 \+ minute\("\1"\)\) >= (\d+) (&&|\|\|) \(hour\("\1"\) \* 60 \+ minute\("\1"\)\) < (\d+)$/
  )
  if (!m) return null

  const start = Number(m[2])
  const end = Number(m[4])
  if (
    !Number.isInteger(start) ||
    !Number.isInteger(end) ||
    start < 0 ||
    start >= 1440 ||
    end < 0 ||
    end >= 1440 ||
    start === end ||
    start % 60 !== 0 ||
    end % 60 !== 0
  ) {
    return null
  }

  const isCrossMidnight = start > end
  if ((m[3] === '||') !== isCrossMidnight) return null

  return {
    source: 'time',
    timeFunc: 'hour',
    timezone: m[1],
    mode: MATCH_RANGE,
    value: '',
    rangeStart: String(start / 60),
    rangeEnd: String(end / 60),
  }
}

function tryParseTimeCondition(expr: string): RequestCondition | null {
  const body = unwrapOuterParens(expr)
  const minuteRange = tryParseMinuteOfDayRangeCondition(body)
  if (minuteRange) return minuteRange

  let m = body.match(
    /^(hour|minute|weekday|month|day)\("([^"]+)"\) >= ([\d.eE+-]+) (&&|\|\|) \1\("\2"\) < ([\d.eE+-]+)$/
  )
  if (m) {
    const start = Number(m[3])
    const end = Number(m[5])
    if (!Number.isFinite(start) || !Number.isFinite(end) || start === end) {
      return null
    }
    const isCrossRange = start > end
    if ((m[4] === '||') !== isCrossRange) return null

    return {
      source: 'time',
      timeFunc: m[1] as TimeFunc,
      timezone: m[2],
      mode: MATCH_RANGE,
      value: '',
      rangeStart: String(start),
      rangeEnd: String(end),
    }
  }
  m = body.match(
    /^(hour|minute|weekday|month|day)\("([^"]+)"\) (==|>=|<) ([\d.eE+-]+)$/
  )
  if (m) {
    const opMap: Record<string, string> = {
      '==': MATCH_EQ,
      '>=': MATCH_GTE,
      '<': MATCH_LT,
    }
    return {
      source: 'time',
      timeFunc: m[1] as TimeFunc,
      timezone: m[2],
      mode: opMap[m[3]] || MATCH_EQ,
      value: m[4],
      rangeStart: '',
      rangeEnd: '',
    }
  }
  return null
}

function tryParseRequestCondition(expr: string): RequestCondition | null {
  const tc = tryParseTimeCondition(expr)
  if (tc) return tc

  let m = expr.match(/^header\("([^"]+)"\) != ""$/)
  if (m) return { source: 'header', path: m[1], mode: MATCH_EXISTS, value: '' }

  m = expr.match(/^param\("([^"]+)"\) != nil$/)
  if (m) return { source: 'param', path: m[1], mode: MATCH_EXISTS, value: '' }

  m = expr.match(/^has\(header\("([^"]+)"\), ((?:"(?:[^"\\]|\\.)*"))\)$/)
  if (m) {
    return {
      source: 'header',
      path: m[1],
      mode: MATCH_CONTAINS,
      value: JSON.parse(m[2]) as string,
    }
  }

  m = expr.match(
    /^param\("([^"]+)"\) != nil && has\(param\("([^"]+)"\), ((?:"(?:[^"\\]|\\.)*"))\)$/
  )
  if (m && m[1] === m[2]) {
    return {
      source: 'param',
      path: m[1],
      mode: MATCH_CONTAINS,
      value: JSON.parse(m[3]) as string,
    }
  }

  m = expr.match(
    /^param\("([^"]+)"\) != nil && param\("([^"]+)"\) (>|>=|<|<=) ([\d.eE+-]+)$/
  )
  if (m && m[1] === m[2]) {
    const opMap: Record<string, string> = {
      '>': MATCH_GT,
      '>=': MATCH_GTE,
      '<': MATCH_LT,
      '<=': MATCH_LTE,
    }
    return { source: 'param', path: m[1], mode: opMap[m[3]], value: m[4] }
  }

  m = expr.match(/^(param|header)\("([^"]+)"\) == (.+)$/)
  if (m) {
    const parsedValue = parseExprLiteral(m[3])
    if (parsedValue === null) return null
    return {
      source: m[1] as 'param' | 'header',
      path: m[2],
      mode: MATCH_EQ,
      value: String(parsedValue),
    }
  }

  return null
}

function tryParseConjunctionConditions(
  expr: string
): RequestCondition[] | null {
  const directCondition = tryParseRequestCondition(expr)
  if (directCondition) return [directCondition]

  const andParts = splitTopLevelAnd(unwrapOuterParens(expr))
  if (andParts.length <= 1) return null

  const conditions: RequestCondition[] = []
  for (const ap of andParts) {
    const cond = tryParseRequestCondition(ap.trim())
    if (!cond) return null
    conditions.push(cond)
  }
  if (conditions.length === 0) return null
  return conditions
}

function tryParseRuleConditionAlternatives(
  expr: string
): RequestCondition[][] | null {
  const body = unwrapOuterParens(expr)
  const directCondition = tryParseRequestCondition(body)
  if (directCondition) return [[directCondition]]

  const topLevelOrParts = splitTopLevelOr(body)
  if (topLevelOrParts.length > 1) {
    const alternatives: RequestCondition[][] = []
    for (const part of topLevelOrParts) {
      const conditions = tryParseConjunctionConditions(part)
      if (!conditions) return null
      alternatives.push(conditions)
    }
    return alternatives
  }

  const andParts = splitTopLevelAnd(body)
  if (andParts.length <= 1) {
    const conditions = tryParseConjunctionConditions(body)
    return conditions ? [conditions] : null
  }

  const sharedConditions: RequestCondition[] = []
  let nestedAlternatives: RequestCondition[][] | null = null
  for (const part of andParts) {
    const directPart = tryParseRequestCondition(part.trim())
    if (directPart) {
      sharedConditions.push(directPart)
      continue
    }

    const orParts = splitTopLevelOr(unwrapOuterParens(part))
    if (orParts.length <= 1 || nestedAlternatives) return null

    nestedAlternatives = []
    for (const orPart of orParts) {
      const conditions = tryParseConjunctionConditions(orPart)
      if (!conditions) return null
      nestedAlternatives.push(conditions)
    }
  }

  if (!nestedAlternatives) {
    return sharedConditions.length > 0 ? [sharedConditions] : null
  }

  return nestedAlternatives.map((conditions) => [
    ...sharedConditions,
    ...conditions,
  ])
}

function tryParseRuleGroupFactors(part: string): RequestRuleGroup[] | null {
  const body = unwrapOuterParens(part)
  const ternary = findTopLevelTernary(body)
  if (!ternary) return null

  const conditionStr = body.slice(0, ternary.questionIndex)
  const multiplier = parsePositiveNumericLiteral(
    body.slice(ternary.questionIndex + 1, ternary.colonIndex)
  )
  const fallback = parsePositiveNumericLiteral(
    body.slice(ternary.colonIndex + 1)
  )
  if (multiplier === null || fallback !== 1) return null

  const alternatives = tryParseRuleConditionAlternatives(conditionStr)
  if (!alternatives) return null

  const groups = alternatives.map((conditions) => ({
    conditions,
    multiplier: String(multiplier),
  }))
  if (groups.length <= 1) return groups
  return getExclusiveTimeRangeMultipliers(groups) ? groups : null
}

export function tryParseRequestRuleExpr(
  expr: string
): RequestRuleGroup[] | null {
  const trimmed = (expr || '').trim()
  if (!trimmed) return []

  const parts = splitTopLevelMultiply(trimmed)
  const groups: RequestRuleGroup[] = []
  for (const part of parts) {
    const nextGroups = tryParseRuleGroupFactors(part)
    if (!nextGroups) return null
    groups.push(...nextGroups)
  }
  return groups
}

// ---------------------------------------------------------------------------
// Combine / split billing expr and request rules
// ---------------------------------------------------------------------------

function hasFullOuterParens(expr: string): boolean {
  if (!expr.startsWith('(') || !expr.endsWith(')')) return false
  let depth = 0
  for (let i = 0; i < expr.length; i += 1) {
    if (expr[i] === '(') depth += 1
    if (expr[i] === ')') depth -= 1
    if (depth === 0 && i < expr.length - 1) return false
  }
  return depth === 0
}

function unwrapOuterParens(expr: string): string {
  let current = (expr || '').trim()
  while (hasFullOuterParens(current)) {
    current = current.slice(1, -1).trim()
  }
  return current
}

export function splitBillingExprAndRequestRules(expr: string): {
  billingExpr: string
  requestRuleExpr: string
} {
  const trimmed = (expr || '').trim()
  if (!trimmed) return { billingExpr: '', requestRuleExpr: '' }

  const parts = splitTopLevelMultiply(trimmed)
  if (parts.length <= 1) return { billingExpr: trimmed, requestRuleExpr: '' }

  const ruleParts: string[] = []
  const baseParts: string[] = []

  parts.forEach((part) => {
    const parsed = tryParseRequestRuleExpr(part)
    if (parsed && parsed.length > 0) {
      ruleParts.push(part)
    } else {
      baseParts.push(part)
    }
  })

  if (ruleParts.length === 0 || baseParts.length !== 1) {
    return { billingExpr: trimmed, requestRuleExpr: '' }
  }

  return {
    billingExpr: unwrapOuterParens(baseParts[0]),
    requestRuleExpr: ruleParts.join(' * '),
  }
}

export function combineBillingExpr(
  baseExpr: string,
  requestRuleExpr: string
): string {
  const base = (baseExpr || '').trim()
  const rules = (requestRuleExpr || '').trim()
  if (!base) return ''
  if (!rules) return base
  return `(${base}) * ${rules}`
}

// ---------------------------------------------------------------------------
// Editor: empty constructors
// ---------------------------------------------------------------------------

export function createEmptyCondition(): ParamHeaderCondition {
  return { source: 'param', path: '', mode: MATCH_EQ, value: '' }
}

export function createEmptyTimeCondition(): TimeCondition {
  return {
    source: 'time',
    timeFunc: 'hour',
    timezone: 'Asia/Shanghai',
    mode: MATCH_GTE,
    value: '',
    rangeStart: '',
    rangeEnd: '',
  }
}

export function createEmptyRuleGroup(): RequestRuleGroup {
  return { conditions: [createEmptyCondition()], multiplier: '' }
}

export function createEmptyTimeRuleGroup(): RequestRuleGroup {
  return { conditions: [createEmptyTimeCondition()], multiplier: '' }
}

// ---------------------------------------------------------------------------
// Editor: match option helpers
// ---------------------------------------------------------------------------

export type MatchOption = { value: string; labelKey: string }

export function getRequestRuleMatchOptions(source: string): MatchOption[] {
  if (source === SOURCE_TIME) {
    return [
      { value: MATCH_EQ, labelKey: 'Equals' },
      { value: MATCH_GTE, labelKey: 'Greater than or equal' },
      { value: MATCH_LT, labelKey: 'Less than' },
      { value: MATCH_RANGE, labelKey: 'Time range' },
    ]
  }
  const base: MatchOption[] = [
    { value: MATCH_EQ, labelKey: 'Equals' },
    { value: MATCH_CONTAINS, labelKey: 'Contains' },
    { value: MATCH_EXISTS, labelKey: 'Exists' },
  ]
  if (source === SOURCE_HEADER) return base
  return [
    ...base,
    { value: MATCH_GT, labelKey: 'Greater than' },
    { value: MATCH_GTE, labelKey: 'Greater than or equal' },
    { value: MATCH_LT, labelKey: 'Less than' },
    { value: MATCH_LTE, labelKey: 'Less than or equal' },
  ]
}

// ---------------------------------------------------------------------------
// Editor: normalize a single condition
// ---------------------------------------------------------------------------

function isTimeFunc(value: unknown): value is TimeFunc {
  return typeof value === 'string' && TIME_FUNCS.includes(value as TimeFunc)
}

export function normalizeCondition(
  cond: Partial<RequestCondition> | null | undefined
): RequestCondition {
  let source: RequestCondition['source'] = 'param'
  if (cond?.source === 'time') {
    source = 'time'
  } else if (cond?.source === 'header') {
    source = 'header'
  }

  if (source === 'time') {
    const timeCond = cond as Partial<TimeCondition> | null | undefined
    const timeFunc: TimeFunc = isTimeFunc(timeCond?.timeFunc)
      ? timeCond.timeFunc
      : 'hour'
    const options = getRequestRuleMatchOptions(SOURCE_TIME)
    const mode = options.some((item) => item.value === timeCond?.mode)
      ? (timeCond?.mode as string)
      : MATCH_GTE
    return {
      source: 'time',
      timeFunc,
      timezone: timeCond?.timezone || 'Asia/Shanghai',
      mode,
      value: timeCond?.value == null ? '' : String(timeCond.value),
      rangeStart:
        timeCond?.rangeStart == null ? '' : String(timeCond.rangeStart),
      rangeEnd: timeCond?.rangeEnd == null ? '' : String(timeCond.rangeEnd),
    }
  }

  const phCond = cond as Partial<ParamHeaderCondition> | null | undefined
  const options = getRequestRuleMatchOptions(source)
  const mode = options.some((item) => item.value === phCond?.mode)
    ? (phCond?.mode as string)
    : MATCH_EQ
  return {
    source,
    path: phCond?.path || '',
    mode,
    value: phCond?.value == null ? '' : String(phCond.value),
  }
}

// ---------------------------------------------------------------------------
// Editor: build expression strings
// ---------------------------------------------------------------------------

function buildExprLiteral(mode: string, value: string): string {
  const text = String(value || '').trim()
  if (mode === MATCH_CONTAINS) return JSON.stringify(text)
  if (text === 'true' || text === 'false') return text
  if (NUMERIC_LITERAL_REGEX.test(text)) return text
  return JSON.stringify(text)
}

function buildTimeConditionExpr(cond: TimeCondition): string {
  const normalized = normalizeCondition(cond) as TimeCondition
  const { timeFunc, timezone, mode } = normalized
  const tz = JSON.stringify(timezone)
  const fn = `${timeFunc}(${tz})`

  if (mode === MATCH_RANGE) {
    const s = normalized.rangeStart.trim()
    const e = normalized.rangeEnd.trim()
    if (!NUMERIC_LITERAL_REGEX.test(s) || !NUMERIC_LITERAL_REGEX.test(e)) {
      return ''
    }
    return Number(s) < Number(e)
      ? `${fn} >= ${s} && ${fn} < ${e}`
      : `${fn} >= ${s} || ${fn} < ${e}`
  }
  const v = normalized.value.trim()
  if (!NUMERIC_LITERAL_REGEX.test(v)) return ''
  const opMap: Record<string, string> = {
    [MATCH_EQ]: '==',
    [MATCH_GTE]: '>=',
    [MATCH_LT]: '<',
  }
  return `${fn} ${opMap[mode] || '=='} ${v}`
}

function buildRequestConditionExpr(cond: RequestCondition): string {
  if (cond.source === 'time') return buildTimeConditionExpr(cond)
  const normalized = normalizeCondition(cond) as ParamHeaderCondition
  const path = normalized.path.trim()
  if (!path) return ''

  const sourceExpr =
    normalized.source === 'header'
      ? `header(${JSON.stringify(path)})`
      : `param(${JSON.stringify(path)})`

  switch (normalized.mode) {
    case MATCH_EXISTS:
      return normalized.source === 'header'
        ? `${sourceExpr} != ""`
        : `${sourceExpr} != nil`
    case MATCH_CONTAINS:
      return normalized.source === 'header'
        ? `has(${sourceExpr}, ${buildExprLiteral(normalized.mode, normalized.value)})`
        : `${sourceExpr} != nil && has(${sourceExpr}, ${buildExprLiteral(normalized.mode, normalized.value)})`
    case MATCH_GT:
    case MATCH_GTE:
    case MATCH_LT:
    case MATCH_LTE: {
      const opMap: Record<string, string> = {
        [MATCH_GT]: '>',
        [MATCH_GTE]: '>=',
        [MATCH_LT]: '<',
        [MATCH_LTE]: '<=',
      }
      const numText = String(normalized.value).trim()
      if (!NUMERIC_LITERAL_REGEX.test(numText)) return ''
      return `${sourceExpr} != nil && ${sourceExpr} ${opMap[normalized.mode]} ${numText}`
    }
    case MATCH_EQ:
    default:
      return `${sourceExpr} == ${buildExprLiteral(normalized.mode, normalized.value)}`
  }
}

function buildRuleGroupFactor(group: RequestRuleGroup): string {
  const multiplier = (group.multiplier || '').trim()
  if (!NUMERIC_LITERAL_REGEX.test(multiplier)) return ''
  const condExprs = (group.conditions || [])
    .map(buildRequestConditionExpr)
    .filter(Boolean)
  if (condExprs.length === 0) return ''

  const combined =
    condExprs.length === 1
      ? condExprs[0]
      : condExprs.map((e) => (e.includes(' || ') ? `(${e})` : e)).join(' && ')
  return `(${combined} ? ${multiplier} : 1)`
}

export function buildRequestRuleExpr(groups: RequestRuleGroup[]): string {
  return (groups || []).map(buildRuleGroupFactor).filter(Boolean).join(' * ')
}
