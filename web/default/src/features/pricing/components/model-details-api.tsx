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

import {
  buildSupportedParameters,
  type SupportedParameter,
} from '../lib/mock-stats'
import { replaceModelInPath } from '../lib/model-helpers'
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

function isGrokImagineVideoModelName(modelName: string): boolean {
  return /grok-imagine-video/i.test(modelName)
}

function isGrokImagineVideo15ModelName(modelName: string): boolean {
  return /grok-imagine-video-1\.5/i.test(modelName)
}

function isGrokImagineImageModelName(modelName: string): boolean {
  return /(?:^|[-_])grok-imagine(?:$|-image)|grok-2-image/i.test(modelName)
}

function isKlingVideoModelName(modelName: string): boolean {
  return /kling/i.test(modelName)
}

function getKlingTextMode(modelName: string): string | undefined {
  const normalized = modelName.trim().toLowerCase()
  if (normalized === 'kling-v1' || normalized === 'kling-v1-5') {
    return 'pro'
  }
  if (normalized === 'kling-v1-6') {
    return 'std'
  }
  return undefined
}

function prefersKlingTextSample(modelName: string): boolean {
  return modelName.trim().toLowerCase() === 'kling-v1-5'
}

function isViduVideoModelName(modelName: string): boolean {
  return /vidu/i.test(modelName)
}

function isSeedanceVideoModelName(modelName: string): boolean {
  return /seedance|doubao-seedance/i.test(modelName)
}

function prefersViduTextSample(modelName: string): boolean {
  return /^(viduq2|viduq1|vidu1\.5|vidu2\.0)$/i.test(modelName.trim())
}

function usesOpenAIVideoGuide(modelName: string): boolean {
  return (
    isGrokImagineVideoModelName(modelName) ||
    isKlingVideoModelName(modelName) ||
    isViduVideoModelName(modelName) ||
    isSeedanceVideoModelName(modelName)
  )
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
  const isGrokImagineImage = isGrokImagineImageModelName(ctx.modelName)
  const body = isGrokImagineImage
    ? {
        model: ctx.modelName,
        prompt,
        n: 1,
        aspect_ratio: '1:1',
        resolution: '1k',
        response_format: 'url',
      }
    : { model: ctx.modelName, prompt, size: '1024x1024', n: 1 }

  if (lang === 'curl') {
    const bodyJson = JSON.stringify(body, null, 2)
    return [
      `curl ${url} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${bodyJson.replaceAll('\n', '\n     ')}'`,
    ].join('\n')
  }
  if (lang === 'python') {
    const fields = isGrokImagineImage
      ? [
          `    model="${ctx.modelName}",`,
          `    prompt="${prompt}",`,
          `    n=1,`,
          `    aspect_ratio="1:1",`,
          `    resolution="1k",`,
          `    response_format="url",`,
        ]
      : [
          `    model="${ctx.modelName}",`,
          `    prompt="${prompt}",`,
          `    size="1024x1024",`,
          `    n=1,`,
        ]

    return [
      'from openai import OpenAI',
      '',
      `client = OpenAI(base_url="${ctx.baseUrl}/v1", api_key="<YOUR_API_KEY>")`,
      '',
      'response = client.images.generate(',
      ...fields,
      ')',
      '',
      'print(response.data[0].url)',
    ].join('\n')
  }
  if (lang === 'typescript') {
    const fields = isGrokImagineImage
      ? [
          `  model: '${ctx.modelName}',`,
          `  prompt: '${prompt}',`,
          `  n: 1,`,
          `  aspect_ratio: '1:1',`,
          `  resolution: '1k',`,
          `  response_format: 'url',`,
        ]
      : [
          `  model: '${ctx.modelName}',`,
          `  prompt: '${prompt}',`,
          `  size: '1024x1024',`,
          `  n: 1,`,
        ]

    return [
      `import OpenAI from 'openai'`,
      '',
      `const client = new OpenAI({`,
      `  baseURL: '${ctx.baseUrl}/v1',`,
      `  apiKey: process.env.${ctx.apiKeyEnv},`,
      `})`,
      '',
      `const response = await client.images.generate({`,
      ...fields,
      `})`,
      '',
      `console.log(response.data[0].url)`,
    ].join('\n')
  }
  const bodyJson = JSON.stringify(body, null, 2)
  return [
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `    'Content-Type': 'application/json',`,
    `  },`,
    `  body: JSON.stringify(${bodyJson.replaceAll('\n', '\n  ')}),`,
    `})`,
    '',
    `const data = await response.json()`,
    `console.log(data.data[0].url)`,
  ].join('\n')
}

function buildVideoSample(lang: Lang, ctx: SampleContext): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  let body: Record<string, unknown> = {
    model: ctx.modelName,
    prompt: 'Cinematic motion of a futuristic city at sunset.',
    duration: 8,
  }

  if (isSeedanceVideoModelName(ctx.modelName)) {
    const prompt =
      'Create a 10 second night city video with a slow push-in camera move and neon lights reflected on wet streets.'
    body = {
      model: ctx.modelName,
      content: [{ type: 'text', text: prompt }],
      generate_audio: true,
      ratio: '16:9',
      resolution: '720p',
      duration: 10,
      watermark: false,
    }
  } else if (isGrokImagineVideoModelName(ctx.modelName)) {
    body = isGrokImagineVideo15ModelName(ctx.modelName)
      ? {
          model: ctx.modelName,
          prompt: 'Add smooth cinematic camera motion to the product photo.',
          image: 'https://example.com/input.jpg',
          duration: 8,
          aspect_ratio: '16:9',
          resolution: '720p',
        }
      : {
          model: ctx.modelName,
          prompt: 'Cinematic motion of a futuristic city at sunset.',
          duration: 8,
          aspect_ratio: '16:9',
          resolution: '720p',
        }
  } else if (isKlingVideoModelName(ctx.modelName)) {
    if (prefersKlingTextSample(ctx.modelName)) {
      body = {
        model: ctx.modelName,
        prompt: 'A cinematic product video of an orange pencil on a clean white desk.',
        duration: 5,
        mode: 'pro',
        metadata: {
          AspectRatio: '16:9',
          NegativePrompt: 'text, watermark, distorted objects',
          CfgScale: 0.5,
        },
      }
    } else {
      body = {
        model: ctx.modelName,
        prompt: 'Add subtle cinematic motion to the product photo.',
        image: 'https://example.com/input.jpg',
        duration: 5,
        mode: 'std',
        metadata: {
          NegativePrompt: 'text, watermark, distorted objects',
          CfgScale: 0.5,
        },
      }

      const mode = getKlingTextMode(ctx.modelName)
      if (mode) {
        body.mode = mode
      }
    }
  } else if (isViduVideoModelName(ctx.modelName)) {
    body = {
      model: ctx.modelName,
      prompt: 'A product shot of an orange pencil on a clean white desk.',
      duration: 5,
      metadata: {
        CallbackUrl: 'https://example.com/callback',
        ExternalTaskId: 'demo-vidu-001',
      },
    }

    if (!prefersViduTextSample(ctx.modelName)) {
      body.images = ['https://example.com/input.jpg']
    }
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
      'import requests',
      '',
      `response = requests.post(`,
      `    "${url}",`,
      `    headers={`,
      `        "Authorization": "Bearer <YOUR_API_KEY>",`,
      `        "Content-Type": "application/json",`,
      `    },`,
      `    json=${bodyJson.replaceAll('\n', '\n    ')},`,
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
      `  body: JSON.stringify(${bodyJson.replaceAll('\n', '\n  ')}),`,
      `})`,
      '',
      `const data = await response.json()`,
      `console.log(data)`,
    ].join('\n')
  }

  return [
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `    'Content-Type': 'application/json',`,
    `  },`,
    `  body: JSON.stringify(${bodyJson.replaceAll('\n', '\n  ')}),`,
    `})`,
    '',
    `const data = await response.json()`,
    `console.log(data)`,
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
  if (endpointType === 'openai-video') {
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

  const baseUrl = useMemo(() => {
    return getApiBaseAddress(status, 'https://api.example.com')
  }, [status])

  const endpoints = useMemo(() => {
    const types = props.model.supported_endpoint_types || []
    const configuredEndpoints = types
      .map((type) => {
        const info = props.endpointMap[type] || {}
        let path = info.path || ''
        if (path && path.includes('{model}')) {
          path = replaceModelInPath(path, props.model.model_name || '')
        }
        return { type, path, method: info.method || 'POST' }
      })
      .filter((e) => Boolean(e.path))

    if (usesOpenAIVideoGuide(props.model.model_name || '')) {
      const videoEndpointIndex = configuredEndpoints.findIndex(
        (endpoint) => endpoint.type === 'openai-video'
      )
      if (videoEndpointIndex === 0) {
        return configuredEndpoints
      }
      if (videoEndpointIndex > 0) {
        const videoEndpoint = configuredEndpoints[videoEndpointIndex]
        return [
          videoEndpoint,
          ...configuredEndpoints.filter(
            (_, index) => index !== videoEndpointIndex
          ),
        ]
      }
      return [
        { type: 'openai-video', path: '/v1/videos', method: 'POST' },
        ...configuredEndpoints,
      ]
    }

    return configuredEndpoints
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
                  {ep.type}
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
            className: 'h-9 w-32',
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
      <div className='flex flex-wrap items-center gap-1'>
        <span className='text-muted-foreground text-sm'>=</span>
        <code className='bg-muted rounded px-1.5 py-0.5 font-mono text-sm'>
          {String(defaultValue)}
        </code>
        {enumValues?.map((v) => (
          <code
            key={v}
            className='bg-muted text-muted-foreground rounded px-1.5 py-0.5 font-mono text-sm'
          >
            {v}
          </code>
        ))}
        {range && (
          <span className='text-muted-foreground text-sm'>{range}</span>
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
      <AuthSection />
      <SupportedParametersSection model={props.model} />
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
