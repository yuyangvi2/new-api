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
import type { WebSearchStrategy } from '../../types'

export type WebSearchCompatibility = {
  supported: boolean
  strategy?: WebSearchStrategy
  labelKey: string
}

type WebSearchRule = {
  groups: string[]
  labelKey: string
  modelPrefixes: string[]
  strategy: WebSearchStrategy
}

const WEB_SEARCH_RULES: WebSearchRule[] = [
  {
    groups: ['gemini'],
    labelKey: 'Uses Gemini Google Search',
    modelPrefixes: ['gemini-'],
    strategy: 'gemini_google_search',
  },
  {
    groups: ['qwen'],
    labelKey: 'Uses Qwen native search',
    modelPrefixes: ['qwen'],
    strategy: 'qwen_enable_search',
  },
  {
    groups: ['claude'],
    labelKey: 'Uses OpenAI-compatible web search',
    modelPrefixes: ['claude-'],
    strategy: 'web_search_options',
  },
  {
    groups: ['openai'],
    labelKey: 'Uses OpenAI-compatible web search',
    modelPrefixes: ['gpt-', 'o3', 'o4', 'chatgpt-'],
    strategy: 'web_search_options',
  },
]

function matchesWebSearchRule(
  rule: WebSearchRule,
  group: string,
  model: string,
): boolean {
  const normalizedGroup = group.trim().toLowerCase()
  const normalizedModel = model.trim().toLowerCase()

  if (rule.groups.includes(normalizedGroup)) {
    return true
  }

  return rule.modelPrefixes.some((prefix) => normalizedModel.startsWith(prefix))
}

export function getWebSearchCompatibility(
  group: string,
  model: string,
): WebSearchCompatibility {
  if (!model.trim()) {
    return {
      supported: false,
      labelKey: 'Select a model to enable web search',
    }
  }

  for (const rule of WEB_SEARCH_RULES) {
    if (matchesWebSearchRule(rule, group, model)) {
      return {
        supported: true,
        strategy: rule.strategy,
        labelKey: rule.labelKey,
      }
    }
  }

  return {
    supported: false,
    labelKey: 'Web search is not available for this model',
  }
}
