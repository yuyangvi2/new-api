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
import {
  ChevronRight,
  Gauge,
  KeyRound,
  ScrollText,
  Sigma,
  Zap,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { BundledLanguage } from 'shiki/bundle/web'

import {
  CodeBlock,
  CodeBlockCopyButton,
} from '@/components/ai-elements/code-block'
import {
  StaticDataTable,
  staticDataTableClassNames as tableStyles,
} from '@/components/data-table'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useStatus } from '@/hooks/use-status'
import { getApiBaseAddress } from '@/lib/server-address'

import { getEndpointTypeLabels } from '../constants'
import {
  buildRateLimits,
  buildSupportedParameters,
  formatRateLimit,
  type SupportedParameter,
} from '../lib/mock-stats'
import { replaceModelInPath } from '../lib/model-helpers'
import {
  buildSeedanceGuideSample,
  getSeedanceGuideFlows,
  type SeedanceGuideFlowId,
} from '../lib/seedance-api-guide'
import type { PricingModel } from '../types'

// ---------------------------------------------------------------------------
// Code-sample registry
// ---------------------------------------------------------------------------
//
// Each sample is keyed by language and endpoint type. The endpoint type comes
// from the model's `supported_endpoint_types`; we render samples only for the
// types the model actually supports. This keeps copy-pasted code accurate and
// provider-shaped (OpenAI vs Anthropic vs Gemini, etc.).

type Lang = 'curl' | 'python' | 'typescript' | 'javascript'

const LANG_LABELS: Record<Lang, string> = {
  curl: 'cURL',
  python: 'Python',
  typescript: 'TypeScript',
  javascript: 'JavaScript',
}

const LANG_HIGHLIGHT: Record<Lang, BundledLanguage> = {
  curl: 'bash',
  python: 'python',
  typescript: 'typescript',
  javascript: 'javascript',
}

type SampleContext = {
  baseUrl: string
  apiKeyEnv: string
  modelName: string
  endpointType: string
  endpointPath: string
}

function buildChatSample(lang: Lang, ctx: SampleContext): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const isResponses = ctx.endpointType === 'openai-response'
  const isReasoning = /^o[1-4]|reasoning|thinking|deepseek-r/i.test(
    ctx.modelName
  )
  const userMessage = 'Explain quantum entanglement in one paragraph.'

  const bodyJson = isResponses
    ? JSON.stringify({ model: ctx.modelName, input: userMessage }, null, 2)
    : JSON.stringify(
        {
          model: ctx.modelName,
          messages: [{ role: 'user', content: userMessage }],
          ...(isReasoning ? {} : { temperature: 0.7 }),
        },
        null,
        2
      )

  const fnCall = isResponses ? 'responses.create' : 'chat.completions.create'

  if (lang === 'curl') {
    return [
      `curl ${url} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${bodyJson.replaceAll('\n', '\n     ')}'`,
    ].join('\n')
  }

  if (lang === 'python') {
    return [
      'from openai import OpenAI',
      '',
      'client = OpenAI(',
      `    base_url="${ctx.baseUrl}/v1",`,
      `    api_key="<YOUR_API_KEY>",`,
      ')',
      '',
      isResponses
        ? `response = client.${fnCall}(\n    model="${ctx.modelName}",\n    input="${userMessage}",\n)\n\nprint(response.output_text)`
        : `completion = client.${fnCall}(\n    model="${ctx.modelName}",\n    messages=[\n        {"role": "user", "content": "${userMessage}"}\n    ],\n)\n\nprint(completion.choices[0].message.content)`,
    ].join('\n')
  }

  if (lang === 'typescript') {
    return [
      `import OpenAI from 'openai'`,
      '',
      `const client = new OpenAI({`,
      `  baseURL: '${ctx.baseUrl}/v1',`,
      `  apiKey: process.env.${ctx.apiKeyEnv},`,
      `})`,
      '',
      isResponses
        ? `const response = await client.${fnCall}({\n  model: '${ctx.modelName}',\n  input: '${userMessage}',\n})\n\nconsole.log(response.output_text)`
        : `const completion = await client.${fnCall}({\n  model: '${ctx.modelName}',\n  messages: [{ role: 'user', content: '${userMessage}' }],\n})\n\nconsole.log(completion.choices[0].message.content)`,
    ].join('\n')
  }

  return [
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `    'Content-Type': 'application/json',`,
    `  },`,
    `  body: JSON.stringify(${bodyJson}),`,
    `})`,
    '',
    `const data = await response.json()`,
    `console.log(data)`,
  ].join('\n')
}

function buildAnthropicSample(lang: Lang, ctx: SampleContext): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const userMessage = 'Explain quantum entanglement in one paragraph.'

  if (lang === 'curl') {
    const body = JSON.stringify(
      {
        model: ctx.modelName,
        max_tokens: 1024,
        messages: [{ role: 'user', content: userMessage }],
      },
      null,
      2
    )
    return [
      `curl ${url} \\`,
      `  -H "x-api-key: $${ctx.apiKeyEnv}" \\`,
      `  -H "anthropic-version: 2023-06-01" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${body.replaceAll('\n', '\n     ')}'`,
    ].join('\n')
  }
  if (lang === 'python') {
    return [
      'import anthropic',
      '',
      'client = anthropic.Anthropic(',
      `    base_url="${ctx.baseUrl}",`,
      `    api_key="<YOUR_API_KEY>",`,
      ')',
      '',
      `message = client.messages.create(`,
      `    model="${ctx.modelName}",`,
      `    max_tokens=1024,`,
      `    messages=[{"role": "user", "content": "${userMessage}"}],`,
      ')',
      '',
      'print(message.content[0].text)',
    ].join('\n')
  }
  if (lang === 'typescript') {
    return [
      `import Anthropic from '@anthropic-ai/sdk'`,
      '',
      `const client = new Anthropic({`,
      `  baseURL: '${ctx.baseUrl}',`,
      `  apiKey: process.env.${ctx.apiKeyEnv},`,
      `})`,
      '',
      `const message = await client.messages.create({`,
      `  model: '${ctx.modelName}',`,
      `  max_tokens: 1024,`,
      `  messages: [{ role: 'user', content: '${userMessage}' }],`,
      `})`,
      '',
      `console.log(message.content[0].text)`,
    ].join('\n')
  }
  return [
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    'x-api-key': process.env.${ctx.apiKeyEnv},`,
    `    'anthropic-version': '2023-06-01',`,
    `    'Content-Type': 'application/json',`,
    `  },`,
    `  body: JSON.stringify({`,
    `    model: '${ctx.modelName}',`,
    `    max_tokens: 1024,`,
    `    messages: [{ role: 'user', content: '${userMessage}' }],`,
    `  }),`,
    `})`,
    '',
    `const data = await response.json()`,
    `console.log(data.content[0].text)`,
  ].join('\n')
}

function buildGeminiSample(lang: Lang, ctx: SampleContext): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}?key=$${ctx.apiKeyEnv}`
  const userMessage = 'Explain quantum entanglement in one paragraph.'

  if (lang === 'curl') {
    const body = JSON.stringify(
      { contents: [{ parts: [{ text: userMessage }] }] },
      null,
      2
    )
    return [
      `curl '${url}' \\`,
      `  -H 'Content-Type: application/json' \\`,
      `  -d '${body.replaceAll('\n', '\n     ')}'`,
    ].join('\n')
  }
  if (lang === 'python') {
    return [
      'import google.generativeai as genai',
      '',
      `genai.configure(api_key="<YOUR_API_KEY>")`,
      '',
      `model = genai.GenerativeModel("${ctx.modelName}")`,
      `response = model.generate_content("${userMessage}")`,
      '',
      `print(response.text)`,
    ].join('\n')
  }
  if (lang === 'typescript') {
    return [
      `import { GoogleGenerativeAI } from '@google/generative-ai'`,
      '',
      `const genAI = new GoogleGenerativeAI(process.env.${ctx.apiKeyEnv}!)`,
      `const model = genAI.getGenerativeModel({ model: '${ctx.modelName}' })`,
      '',
      `const result = await model.generateContent('${userMessage}')`,
      `console.log(result.response.text())`,
    ].join('\n')
  }
  return [
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: { 'Content-Type': 'application/json' },`,
    `  body: JSON.stringify({`,
    `    contents: [{ parts: [{ text: '${userMessage}' }] }],`,
    `  }),`,
    `})`,
    '',
    `const data = await response.json()`,
    `console.log(data.candidates[0].content.parts[0].text)`,
  ].join('\n')
}

function buildEmbeddingSample(lang: Lang, ctx: SampleContext): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const text = 'The food was delicious and the waiter…'

  if (lang === 'curl') {
    const body = JSON.stringify({ model: ctx.modelName, input: text }, null, 2)
    return [
      `curl ${url} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${body.replaceAll('\n', '\n     ')}'`,
    ].join('\n')
  }
  if (lang === 'python') {
    return [
      'from openai import OpenAI',
      '',
      `client = OpenAI(base_url="${ctx.baseUrl}/v1", api_key="<YOUR_API_KEY>")`,
      '',
      'response = client.embeddings.create(',
      `    model="${ctx.modelName}",`,
      `    input="${text}",`,
      ')',
      '',
      'print(response.data[0].embedding[:8])',
    ].join('\n')
  }
  if (lang === 'typescript') {
    return [
      `import OpenAI from 'openai'`,
      '',
      `const client = new OpenAI({`,
      `  baseURL: '${ctx.baseUrl}/v1',`,
      `  apiKey: process.env.${ctx.apiKeyEnv},`,
      `})`,
      '',
      `const response = await client.embeddings.create({`,
      `  model: '${ctx.modelName}',`,
      `  input: '${text}',`,
      `})`,
      '',
      `console.log(response.data[0].embedding.slice(0, 8))`,
    ].join('\n')
  }
  return [
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `    'Content-Type': 'application/json',`,
    `  },`,
    `  body: JSON.stringify({`,
    `    model: '${ctx.modelName}',`,
    `    input: '${text}',`,
    `  }),`,
    `})`,
    '',
    `const data = await response.json()`,
    `console.log(data.data[0].embedding.slice(0, 8))`,
  ].join('\n')
}

function buildImageSample(lang: Lang, ctx: SampleContext): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const prompt = 'A serene koi pond at sunset, ukiyo-e style.'

  if (lang === 'curl') {
    const body = JSON.stringify(
      { model: ctx.modelName, prompt, size: '1024x1024', n: 1 },
      null,
      2
    )
    return [
      `curl ${url} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${body.replaceAll('\n', '\n     ')}'`,
    ].join('\n')
  }
  if (lang === 'python') {
    return [
      'from openai import OpenAI',
      '',
      `client = OpenAI(base_url="${ctx.baseUrl}/v1", api_key="<YOUR_API_KEY>")`,
      '',
      'response = client.images.generate(',
      `    model="${ctx.modelName}",`,
      `    prompt="${prompt}",`,
      `    size="1024x1024",`,
      `    n=1,`,
      ')',
      '',
      'print(response.data[0].url)',
    ].join('\n')
  }
  if (lang === 'typescript') {
    return [
      `import OpenAI from 'openai'`,
      '',
      `const client = new OpenAI({`,
      `  baseURL: '${ctx.baseUrl}/v1',`,
      `  apiKey: process.env.${ctx.apiKeyEnv},`,
      `})`,
      '',
      `const response = await client.images.generate({`,
      `  model: '${ctx.modelName}',`,
      `  prompt: '${prompt}',`,
      `  size: '1024x1024',`,
      `  n: 1,`,
      `})`,
      '',
      `console.log(response.data[0].url)`,
    ].join('\n')
  }
  return [
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `    'Content-Type': 'application/json',`,
    `  },`,
    `  body: JSON.stringify({`,
    `    model: '${ctx.modelName}',`,
    `    prompt: '${prompt}',`,
    `    size: '1024x1024',`,
    `    n: 1,`,
    `  }),`,
    `})`,
    '',
    `const data = await response.json()`,
    `console.log(data.data[0].url)`,
  ].join('\n')
}

function isSeedanceVideoModel(modelName: string): boolean {
  return /seedance|doubao-seedance/i.test(modelName)
}

function buildSeedanceVideoSample(lang: Lang, ctx: SampleContext): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const prompt =
    'Create a cinematic 5 second night city video with a slow push-in camera move and neon lights reflected on wet streets.'
  const body = {
    model: ctx.modelName,
    content: [{ type: 'text', text: prompt }],
    generate_audio: true,
    ratio: '16:9',
    resolution: '720p',
    duration: 5,
    watermark: false,
  }
  const bodyJson = JSON.stringify(body, null, 2)

  if (lang === 'curl') {
    return [
      '# Create a video generation task',
      `curl ${url} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${bodyJson.replaceAll('\n', '\n     ')}'`,
      '',
      '# Query the task with the id returned above',
      `curl ${url}/<TASK_ID> \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}"`,
    ].join('\n')
  }
  if (lang === 'python') {
    return [
      'import json',
      'import time',
      'import requests',
      '',
      `url = "${url}"`,
      `headers = {`,
      `    "Authorization": "Bearer <YOUR_API_KEY>",`,
      `    "Content-Type": "application/json",`,
      `}`,
      `payload = json.loads('''${bodyJson}''')`,
      '',
      `create_response = requests.post(url, headers=headers, json=payload)`,
      `create_response.raise_for_status()`,
      `task_id = create_response.json()["id"]`,
      '',
      `while True:`,
      `    task_response = requests.get(f"{url}/{task_id}", headers=headers)`,
      `    task_response.raise_for_status()`,
      `    task = task_response.json()`,
      `    if task.get("status") in {"succeeded", "failed", "expired", "cancelled", "canceled"}:`,
      `        break`,
      `    time.sleep(5)`,
      '',
      `print(task)`,
    ].join('\n')
  }
  if (lang === 'typescript') {
    return [
      `const headers = {`,
      `  Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
      `  'Content-Type': 'application/json',`,
      `}`,
      '',
      `const createResponse = await fetch('${url}', {`,
      `  method: 'POST',`,
      `  headers,`,
      `  body: JSON.stringify(${bodyJson.replaceAll('\n', '\n    ')}),`,
      `})`,
      `if (!createResponse.ok) throw new Error(await createResponse.text())`,
      '',
      `const { id } = (await createResponse.json()) as { id: string }`,
      `let task: { status?: string; content?: { video_url?: string } } | undefined`,
      '',
      `for (;;) {`,
      `  const taskResponse = await fetch(\`${url}/\${id}\`, { headers })`,
      `  if (!taskResponse.ok) throw new Error(await taskResponse.text())`,
      `  task = await taskResponse.json()`,
      `  if (['succeeded', 'failed', 'expired', 'cancelled', 'canceled'].includes(task.status ?? '')) break`,
      `  await new Promise((resolve) => setTimeout(resolve, 5000))`,
      `}`,
      '',
      `console.log(task)`,
    ].join('\n')
  }
  return [
    `const headers = {`,
    `  Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `  'Content-Type': 'application/json',`,
    `}`,
    '',
    `const createResponse = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers,`,
    `  body: JSON.stringify(${bodyJson.replaceAll('\n', '\n    ')}),`,
    `})`,
    `if (!createResponse.ok) throw new Error(await createResponse.text())`,
    '',
    `const { id } = await createResponse.json()`,
    `let task`,
    '',
    `for (;;) {`,
    `  const taskResponse = await fetch(\`${url}/\${id}\`, { headers })`,
    `  if (!taskResponse.ok) throw new Error(await taskResponse.text())`,
    `  task = await taskResponse.json()`,
    `  if (['succeeded', 'failed', 'expired', 'cancelled', 'canceled'].includes(task.status ?? '')) break`,
    `  await new Promise((resolve) => setTimeout(resolve, 5000))`,
    `}`,
    '',
    `console.log(task)`,
  ].join('\n')
}

function buildVideoSample(lang: Lang, ctx: SampleContext): string {
  if (isSeedanceVideoModel(ctx.modelName)) {
    return buildSeedanceVideoSample(lang, ctx)
  }

  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const prompt =
    'Create a 10 second night city video with a slow push-in camera move and neon lights reflected on wet streets.'
  const body = {
    model: ctx.modelName,
    prompt,
    seconds: '10',
    size: '1280x720',
  }
  const bodyJson = JSON.stringify(body, null, 2)

  if (lang === 'curl') {
    return [
      `curl ${url} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${bodyJson.replaceAll('\n', '\n     ')}'`,
    ].join('\n')
  }
  if (lang === 'python') {
    return [
      'import json',
      'import requests',
      '',
      `payload = json.loads('''${bodyJson}''')`,
      '',
      `response = requests.post(`,
      `    "${url}",`,
      `    headers={`,
      `        "Authorization": "Bearer <YOUR_API_KEY>",`,
      `        "Content-Type": "application/json",`,
      `    },`,
      `    json=payload,`,
      `)`,
      `response.raise_for_status()`,
      `print(response.json())`,
    ].join('\n')
  }
  if (lang === 'typescript') {
    return [
      `const response = await fetch('${url}', {`,
      `  method: 'POST',`,
      `  headers: {`,
      `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
      `    'Content-Type': 'application/json',`,
      `  },`,
      `  body: JSON.stringify(${bodyJson.replaceAll('\n', '\n    ')}),`,
      `})`,
      '',
      `if (!response.ok) throw new Error(await response.text())`,
      `console.log(await response.json())`,
    ].join('\n')
  }
  return [
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `    'Content-Type': 'application/json',`,
    `  },`,
    `  body: JSON.stringify(${bodyJson.replaceAll('\n', '\n    ')}),`,
    `})`,
    '',
    `if (!response.ok) throw new Error(await response.text())`,
    `console.log(await response.json())`,
  ].join('\n')
}

function buildSample(
  lang: Lang,
  endpointType: string,
  ctx: SampleContext
): string {
  if (endpointType === 'anthropic') {
    return buildAnthropicSample(lang, ctx)
  }
  if (endpointType === 'gemini') {
    return buildGeminiSample(lang, ctx)
  }
  if (endpointType === 'embeddings' || endpointType === 'jina-rerank') {
    return buildEmbeddingSample(lang, ctx)
  }
  if (endpointType === 'image-generation') {
    return buildImageSample(lang, ctx)
  }
  if (endpointType === 'openai-video' || endpointType === 'seedance') {
    return buildVideoSample(lang, ctx)
  }
  return buildChatSample(lang, ctx)
}

// ---------------------------------------------------------------------------
// Code samples section
// ---------------------------------------------------------------------------

function CodeSamplesSection(props: {
  model: PricingModel
  endpointMap: Record<string, { path?: string; method?: string }>
}) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const endpointTypeLabels = getEndpointTypeLabels(t) as Record<string, string>

  const baseUrl = useMemo(() => {
    return getApiBaseAddress(status, 'https://api.example.com')
  }, [status])

  const endpoints = useMemo(() => {
    const types = props.model.supported_endpoint_types || []
    return types
      .map((type) => {
        const info = props.endpointMap[type] || {}
        let path = info.path || ''
        if (path && path.includes('{model}')) {
          path = replaceModelInPath(path, props.model.model_name || '')
        }
        return { type, path, method: info.method || 'POST' }
      })
      .filter((e) => Boolean(e.path))
  }, [props.model, props.endpointMap])

  const [endpointType, setEndpointType] = useState<string>(
    endpoints[0]?.type ?? ''
  )
  const [lang, setLang] = useState<Lang>('curl')

  const activeEndpoint = useMemo(() => {
    return endpoints.find((e) => e.type === endpointType) ?? endpoints[0]
  }, [endpointType, endpoints])

  if (endpoints.length === 0 || !activeEndpoint) {
    return null
  }

  const code = buildSample(lang, activeEndpoint.type, {
    baseUrl,
    apiKeyEnv: 'NEW_API_KEY',
    modelName: props.model.model_name || '',
    endpointType: activeEndpoint.type,
    endpointPath: activeEndpoint.path,
  })

  return (
    <section>
      <SectionTitle icon={ScrollText}>{t('Code samples')}</SectionTitle>

      <div className='flex flex-wrap items-center gap-2'>
        {endpoints.length > 1 && (
          <Tabs value={endpointType} onValueChange={setEndpointType}>
            <TabsList className='bg-muted/40 h-8 p-0.5'>
              {endpoints.map((ep) => (
                <TabsTrigger
                  key={ep.type}
                  value={ep.type}
                  className='h-7 px-2.5 text-xs'
                >
                  {endpointTypeLabels[ep.type] ?? ep.type}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        )}

        <Tabs
          value={lang}
          onValueChange={(v) => setLang(v as Lang)}
          className='ml-auto'
        >
          <TabsList className='bg-muted/40 h-8 p-0.5'>
            {(Object.keys(LANG_LABELS) as Lang[]).map((l) => (
              <TabsTrigger key={l} value={l} className='h-7 px-2.5 text-xs'>
                {LANG_LABELS[l]}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>

      <div className='mt-3'>
        <CodeBlock code={code} language={LANG_HIGHLIGHT[lang]}>
          <CodeBlockCopyButton />
        </CodeBlock>
      </div>

      <p className='text-muted-foreground mt-2 text-xs'>
        {t('Replace')}{' '}
        <code className='bg-muted rounded px-1 py-0.5 font-mono text-[11px]'>
          {'<YOUR_API_KEY>'}
        </code>{' '}
        {t('with the API key from your token settings.')}
      </p>
    </section>
  )
}

// ---------------------------------------------------------------------------
// Seedance Official workflow guide
// ---------------------------------------------------------------------------

function resolveSeedanceVideoEndpoint(
  model: PricingModel,
  endpointMap: Record<string, { path?: string; method?: string }>
): string {
  const endpointTypes = model.supported_endpoint_types || []
  const preferredType =
    endpointTypes.find((type) => type === 'seedance') ??
    endpointTypes.find((type) => type === 'openai-video') ??
    endpointTypes[0]

  let path = endpointMap[preferredType || '']?.path || '/v1/videos/generations'
  if (path.includes('{model}')) {
    path = replaceModelInPath(path, model.model_name || '')
  }
  return path
}

function isFastOrMiniSeedanceModel(modelName: string): boolean {
  const normalized = modelName.toLowerCase()
  return normalized.includes('fast') || normalized.includes('mini')
}

function SeedanceOfficialGuideSection(props: {
  model: PricingModel
  endpointMap: Record<string, { path?: string; method?: string }>
}) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const modelName = props.model.model_name || ''
  const [flowId, setFlowId] = useState<SeedanceGuideFlowId>('video')
  const [lang, setLang] = useState<Lang>('curl')

  const baseUrl = useMemo(() => {
    return getApiBaseAddress(status, 'https://api.example.com')
  }, [status])

  const flows = useMemo(() => getSeedanceGuideFlows(), [])
  const activeFlow = flows.find((flow) => flow.id === flowId) ?? flows[0]
  const endpointPath = resolveSeedanceVideoEndpoint(
    props.model,
    props.endpointMap
  )

  if (!isSeedanceVideoModel(modelName) || !activeFlow) {
    return null
  }

  const code = buildSeedanceGuideSample(activeFlow.id, lang, {
    baseUrl,
    apiKeyEnv: 'NEW_API_KEY',
    modelName,
    endpointPath,
  })

  return (
    <section>
      <SectionTitle icon={ScrollText}>
        {t('Seedance Official workflow')}
      </SectionTitle>

      <div className='border-border/60 bg-muted/15 rounded-lg border p-3'>
        <div className='flex flex-wrap items-center gap-2'>
          <Tabs
            value={flowId}
            onValueChange={(value) => setFlowId(value as SeedanceGuideFlowId)}
          >
            <TabsList className='bg-muted/40 h-8 p-0.5'>
              {flows.map((flow) => (
                <TabsTrigger
                  key={flow.id}
                  value={flow.id}
                  className='h-7 px-2.5 text-xs'
                >
                  {t(flow.titleKey)}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>

          <Tabs
            value={lang}
            onValueChange={(value) => setLang(value as Lang)}
            className='ml-auto'
          >
            <TabsList className='bg-muted/40 h-8 p-0.5'>
              {(Object.keys(LANG_LABELS) as Lang[]).map((l) => (
                <TabsTrigger key={l} value={l} className='h-7 px-2.5 text-xs'>
                  {LANG_LABELS[l]}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        </div>

        <div className='mt-3 space-y-2 text-xs leading-relaxed'>
          <p className='text-muted-foreground'>
            {t(activeFlow.descriptionKey)}
          </p>
          <p className='text-muted-foreground'>
            {t(
              'Asset and real-person routes use the same Bearer token as video generation.'
            )}{' '}
            {t(
              'Generation requests reference media through content[].image_url.url or content[].video_url.url; manage reusable assets separately and use approved media URLs in tasks.'
            )}
          </p>
          <p className='text-muted-foreground'>
            {isFastOrMiniSeedanceModel(modelName)
              ? t(
                  'Fast and Mini variants should use the resolutions shown in the parameter table below, typically 480p or 720p.'
                )
              : t(
                  'Use the resolutions enabled in the parameter table below; standard Seedance models may include higher resolutions when configured.'
                )}
          </p>
        </div>

        <div className='mt-3'>
          <CodeBlock code={code} language={LANG_HIGHLIGHT[lang]}>
            <CodeBlockCopyButton />
          </CodeBlock>
        </div>
      </div>
    </section>
  )
}

// ---------------------------------------------------------------------------
// Supported parameters table
// ---------------------------------------------------------------------------

function SupportedParametersSection(props: { model: PricingModel }) {
  const { t } = useTranslation()
  const params = useMemo(
    () => buildSupportedParameters(props.model),
    [props.model]
  )

  if (params.length === 0) return null

  return (
    <section>
      <SectionTitle icon={Sigma}>{t('Supported parameters')}</SectionTitle>
      <StaticDataTable
        className={tableStyles.sectionContainer}
        headerRowClassName={tableStyles.mutedHeaderRow}
        data={params}
        getRowKey={(param) => param.name}
        getRowClassName={() => 'hover:bg-muted/20'}
        columns={[
          {
            id: 'parameter',
            header: t('Parameter'),
            className: 'h-9 w-44',
            cellClassName: tableStyles.topCell,
            cell: (p) => (
              <div className='flex items-center gap-1.5'>
                <code className='font-mono text-sm font-medium'>{p.name}</code>
                {p.required && (
                  <Badge
                    variant='outline'
                    className='h-6 border-rose-500/40 px-2 text-sm text-rose-600 dark:text-rose-400'
                  >
                    {t('required')}
                  </Badge>
                )}
              </div>
            ),
          },
          {
            id: 'type',
            header: t('Type'),
            className: 'h-9 w-24',
            cellClassName: tableStyles.topCell,
            cell: (p) => (
              <Badge
                variant='secondary'
                className='h-7 rounded-full px-2.5 font-mono text-sm font-normal'
              >
                {p.type}
              </Badge>
            ),
          },
          {
            id: 'range',
            header: t('Default / range'),
            className: 'h-9 w-48',
            cellClassName: tableStyles.topCell,
            cell: (p) => <ParamRangeCell param={p} />,
          },
          {
            id: 'description',
            header: t('Description'),
            className: 'h-9',
            cellClassName: tableStyles.topMutedCell,
            cell: (p) => t(p.descriptionKey),
          },
        ]}
      />
    </section>
  )
}

function ParamRangeCell(props: { param: SupportedParameter }) {
  const { defaultValue, range, enumValues } = props.param
  if (defaultValue !== undefined) {
    return (
      <div className='flex flex-col items-start gap-1'>
        <div className='flex items-center gap-1'>
          <span className='text-muted-foreground text-sm'>=</span>
          <code className='bg-muted rounded px-1.5 py-0.5 font-mono text-sm'>
            {String(defaultValue)}
          </code>
        </div>
        {range && (
          <span className='text-muted-foreground font-mono text-sm leading-5'>
            {range}
          </span>
        )}
        {enumValues && enumValues.length > 0 && (
          <div className='flex flex-wrap gap-0.5'>
            {enumValues.map((v) => (
              <code
                key={v}
                className='bg-muted text-muted-foreground rounded px-1.5 py-0.5 font-mono text-sm'
              >
                {v}
              </code>
            ))}
          </div>
        )}
      </div>
    )
  }
  if (range) {
    return (
      <span className='text-muted-foreground font-mono text-sm'>{range}</span>
    )
  }
  if (enumValues && enumValues.length > 0) {
    return (
      <div className='flex flex-wrap gap-0.5'>
        {enumValues.map((v) => (
          <code
            key={v}
            className='bg-muted text-muted-foreground rounded px-1.5 py-0.5 font-mono text-sm'
          >
            {v}
          </code>
        ))}
      </div>
    )
  }
  return <span className='text-muted-foreground/60 text-sm'>—</span>
}

// ---------------------------------------------------------------------------
// Rate-limits table
// ---------------------------------------------------------------------------

function RateLimitsSection(props: { model: PricingModel }) {
  const { t } = useTranslation()
  const limits = useMemo(() => buildRateLimits(props.model), [props.model])

  if (limits.length === 0) return null

  return (
    <section>
      <SectionTitle icon={Gauge}>{t('Rate limits')}</SectionTitle>
      <StaticDataTable
        className={tableStyles.sectionContainer}
        headerRowClassName={tableStyles.mutedHeaderRow}
        data={limits}
        getRowKey={(limit) => limit.group}
        getRowClassName={() => 'hover:bg-muted/20'}
        columns={[
          {
            id: 'group',
            header: t('Group'),
            className: 'h-9',
            cellClassName: 'py-2 font-mono',
            cell: (limit) => limit.group,
          },
          {
            id: 'rpm',
            header: 'RPM',
            className: 'h-9 text-right',
            cellClassName: tableStyles.topNumericCell,
            cell: (limit) => formatRateLimit(limit.rpm),
          },
          {
            id: 'tpm',
            header: 'TPM',
            className: 'h-9 text-right',
            cellClassName: tableStyles.topNumericCell,
            cell: (limit) => formatRateLimit(limit.tpm),
          },
          {
            id: 'rpd',
            header: 'RPD',
            className: 'h-9 text-right',
            cellClassName: tableStyles.topNumericCell,
            cell: (limit) => formatRateLimit(limit.rpd),
          },
        ]}
      />
      <p className='text-muted-foreground mt-2 text-[11px] leading-relaxed'>
        {t(
          'RPM = requests per minute, TPM = tokens per minute, RPD = requests per day. Limits apply per token group.'
        )}
      </p>
    </section>
  )
}

// ---------------------------------------------------------------------------
// Authentication preview
// ---------------------------------------------------------------------------

function AuthSection() {
  const { t } = useTranslation()
  return (
    <section>
      <SectionTitle icon={KeyRound}>{t('Authentication')}</SectionTitle>
      <div className='border-border/60 bg-muted/20 flex items-start gap-2 rounded-lg border p-3'>
        <ChevronRight className='text-muted-foreground mt-0.5 size-3.5 shrink-0' />
        <div className='space-y-1.5 text-xs leading-relaxed'>
          <p>
            {t('All requests must include')}{' '}
            <code className='bg-muted rounded px-1 py-0.5 font-mono text-[11px]'>
              Authorization: Bearer &lt;TOKEN&gt;
            </code>{' '}
            {t('header. Anthropic-formatted endpoints accept the')}{' '}
            <code className='bg-muted rounded px-1 py-0.5 font-mono text-[11px]'>
              x-api-key
            </code>{' '}
            {t('header instead.')}
          </p>
          <p className='text-muted-foreground'>
            {t(
              'Generate tokens from the Tokens page; you can scope them to specific models, groups, IPs, and rate-limits.'
            )}
          </p>
        </div>
      </div>
    </section>
  )
}

// ---------------------------------------------------------------------------
// Composite API tab
// ---------------------------------------------------------------------------

export function ModelDetailsApi(props: {
  model: PricingModel
  endpointMap: Record<string, { path?: string; method?: string }>
}) {
  return (
    <div className='space-y-6'>
      <CodeSamplesSection model={props.model} endpointMap={props.endpointMap} />
      <SeedanceOfficialGuideSection
        model={props.model}
        endpointMap={props.endpointMap}
      />
      <AuthSection />
      <SupportedParametersSection model={props.model} />
      <RateLimitsSection model={props.model} />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Local UI helpers
// ---------------------------------------------------------------------------

function SectionTitle(props: {
  children: React.ReactNode
  icon: React.ComponentType<{ className?: string }>
}) {
  const Icon = props.icon
  return (
    <h3 className='text-foreground mb-3 flex items-center gap-1.5 text-sm font-semibold'>
      <Icon className='text-muted-foreground/70 size-3.5' />
      {props.children}
    </h3>
  )
}

// Re-export so the parent can keep its own SectionTitle if it wants:
export { Zap as ApiTabIcon }
