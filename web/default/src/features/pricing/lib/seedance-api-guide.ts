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
export type SeedanceGuideLang =
  | 'curl'
  | 'python'
  | 'typescript'
  | 'javascript'

export type SeedanceGuideFlowId =
  | 'video'
  | 'asset'
  | 'real-person'
  | 'workflow'

export type SeedanceGuideSampleContext = {
  baseUrl: string
  apiKeyEnv: string
  modelName: string
  endpointPath: string
}

export type SeedanceGuideFlow = {
  id: SeedanceGuideFlowId
  titleKey: string
  descriptionKey: string
}

const SEEDANCE_GUIDE_FLOWS: SeedanceGuideFlow[] = [
  {
    id: 'video',
    titleKey: 'Video task',
    descriptionKey:
      'Create a Seedance video task, then poll the task URL or receive the result with callback_url.',
  },
  {
    id: 'asset',
    titleKey: 'Asset library',
    descriptionKey:
      'Create an asset group, upload reusable reference media, and query moderation status before generation.',
  },
  {
    id: 'real-person',
    titleKey: 'Real-person auth',
    descriptionKey:
      'Start a real-person authentication session, open the returned H5Link, then exchange BytedToken for the LivenessFace asset group.',
  },
  {
    id: 'workflow',
    titleKey: 'Complete flow',
    descriptionKey:
      'Combine asset preparation, optional real-person authentication, video task creation, and polling in one integration flow.',
  },
]

export function getSeedanceGuideFlows(): SeedanceGuideFlow[] {
  return SEEDANCE_GUIDE_FLOWS.map((flow) => ({ ...flow }))
}

export function buildSeedanceGuideSample(
  flowId: SeedanceGuideFlowId,
  lang: SeedanceGuideLang,
  ctx: SeedanceGuideSampleContext
): string {
  switch (flowId) {
    case 'asset':
      return buildAssetSample(lang, ctx)
    case 'real-person':
      return buildRealPersonSample(lang, ctx)
    case 'workflow':
      return buildWorkflowSample(lang, ctx)
    case 'video':
      return buildVideoTaskSample(lang, ctx)
  }
}

function url(ctx: SeedanceGuideSampleContext, path: string): string {
  return `${ctx.baseUrl.replace(/\/+$/, '')}${path}`
}

function body(value: unknown): string {
  return JSON.stringify(value, null, 2)
}

function curlJson(value: unknown): string {
  return body(value).replaceAll('\n', '\n     ')
}

function buildVideoTaskPayload(ctx: SeedanceGuideSampleContext) {
  return {
    model: ctx.modelName,
    content: [
      {
        type: 'text',
        text: 'Create a cinematic 5 second product video. Keep the same person and outfit as the reference image.',
      },
      {
        type: 'image_url',
        image_url: {
          url: 'https://your-cdn.example.com/reference-person.png',
        },
      },
    ],
    callback_url: 'https://your-app.example.com/seedance/task-callback',
    generate_audio: true,
    ratio: '16:9',
    resolution: '720p',
    duration: 5,
    watermark: false,
  }
}

function buildAssetGroupPayload() {
  return {
    Name: 'brand-assets',
    Description: 'Reusable character and reference media for Seedance.',
  }
}

function buildCreateAssetPayload() {
  return {
    GroupId: '<GROUP_ID>',
    URL: 'https://your-cdn.example.com/reference-person.png',
    Name: 'reference-person',
    AssetType: 'Image',
  }
}

function buildQueryAssetsPayload() {
  return {
    Filter: {
      GroupIds: ['<GROUP_ID>'],
      Statuses: ['Success'],
    },
    PageNumber: 1,
    PageSize: 20,
  }
}

function buildRealPersonSessionPayload() {
  return {
    CallbackURL: 'https://your-app.example.com/seedance/real-person-callback',
  }
}

function buildBytedTokenPayload() {
  return {
    byted_token: '<BYTED_TOKEN>',
  }
}

function buildVideoTaskSample(
  lang: SeedanceGuideLang,
  ctx: SeedanceGuideSampleContext
): string {
  const taskUrl = url(ctx, ctx.endpointPath)
  const payload = buildVideoTaskPayload(ctx)

  if (lang === 'curl') {
    return [
      '# Create a Seedance Official task. callback_url is optional but recommended for production.',
      `curl ${taskUrl} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${curlJson(payload)}'`,
      '',
      '# Poll the task if you do not rely on callback_url.',
      `curl ${taskUrl}/<TASK_ID> \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}"`,
    ].join('\n')
  }

  if (lang === 'python') {
    return [
      'import json',
      'import time',
      'import requests',
      '',
      `task_url = "${taskUrl}"`,
      'headers = {',
      '    "Authorization": "Bearer <YOUR_API_KEY>",',
      '    "Content-Type": "application/json",',
      '}',
      `payload = json.loads('''${body(payload)}''')`,
      '',
      'create_response = requests.post(task_url, headers=headers, json=payload)',
      'create_response.raise_for_status()',
      'task_id = create_response.json()["id"]',
      '',
      'while True:',
      '    task_response = requests.get(f"{task_url}/{task_id}", headers=headers)',
      '    task_response.raise_for_status()',
      '    task = task_response.json()',
      '    if task.get("status") in {"succeeded", "failed", "expired", "cancelled", "canceled"}:',
      '        break',
      '    time.sleep(5)',
      '',
      'print(task)',
    ].join('\n')
  }

  if (lang === 'typescript') {
    return [
      'type SeedanceTask = { id: string; status?: string; content?: { video_url?: string } }',
      '',
      'const headers = {',
      `  Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
      `  'Content-Type': 'application/json',`,
      '}',
      '',
      `const createResponse = await fetch('${taskUrl}', {`,
      `  method: 'POST',`,
      '  headers,',
      `  body: JSON.stringify(${body(payload).replaceAll('\n', '\n    ')}),`,
      '})',
      'if (!createResponse.ok) throw new Error(await createResponse.text())',
      '',
      'const { id } = (await createResponse.json()) as SeedanceTask',
      'let task: SeedanceTask',
      '',
      'for (;;) {',
      `  const taskResponse = await fetch(\`${taskUrl}/\${id}\`, { headers })`,
      '  if (!taskResponse.ok) throw new Error(await taskResponse.text())',
      '  task = (await taskResponse.json()) as SeedanceTask',
      "  if (['succeeded', 'failed', 'expired', 'cancelled', 'canceled'].includes(task.status ?? '')) break",
      '  await new Promise((resolve) => setTimeout(resolve, 5000))',
      '}',
      '',
      'console.log(task)',
    ].join('\n')
  }

  return [
    'const headers = {',
    `  Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `  'Content-Type': 'application/json',`,
    '}',
    '',
    `const createResponse = await fetch('${taskUrl}', {`,
    `  method: 'POST',`,
    '  headers,',
    `  body: JSON.stringify(${body(payload).replaceAll('\n', '\n    ')}),`,
    '})',
    'if (!createResponse.ok) throw new Error(await createResponse.text())',
    '',
    'const { id } = await createResponse.json()',
    'let task',
    '',
    'for (;;) {',
    `  const taskResponse = await fetch(\`${taskUrl}/\${id}\`, { headers })`,
    '  if (!taskResponse.ok) throw new Error(await taskResponse.text())',
    '  task = await taskResponse.json()',
    "  if (['succeeded', 'failed', 'expired', 'cancelled', 'canceled'].includes(task.status ?? '')) break",
    '  await new Promise((resolve) => setTimeout(resolve, 5000))',
    '}',
    '',
    'console.log(task)',
  ].join('\n')
}

function buildAssetSample(
  lang: SeedanceGuideLang,
  ctx: SeedanceGuideSampleContext
): string {
  const groupUrl = url(ctx, '/v1/seedance-official/asset-groups')
  const assetUrl = url(ctx, '/v1/seedance-official/assets')
  const assetQueryUrl = url(ctx, '/v1/seedance-official/assets/query')

  if (lang === 'curl') {
    return [
      '# 1. Create an asset group.',
      `curl ${groupUrl} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${curlJson(buildAssetGroupPayload())}'`,
      '',
      '# 2. Create an asset in the group returned above.',
      `curl ${assetUrl} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${curlJson(buildCreateAssetPayload())}'`,
      '',
      '# 3. Query assets and wait until Status is Success before using the media URL in generation.',
      `curl ${assetQueryUrl} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${curlJson(buildQueryAssetsPayload())}'`,
    ].join('\n')
  }

  if (lang === 'python') {
    return [
      'import requests',
      '',
      `base_url = "${ctx.baseUrl.replace(/\/+$/, '')}"`,
      'headers = {',
      '    "Authorization": "Bearer <YOUR_API_KEY>",',
      '    "Content-Type": "application/json",',
      '}',
      '',
      `group_payload = ${body(buildAssetGroupPayload())}`,
      'group_response = requests.post(',
      '    f"{base_url}/v1/seedance-official/asset-groups",',
      '    headers=headers,',
      '    json=group_payload,',
      ')',
      'group_response.raise_for_status()',
      'group_id = group_response.json()["Id"]',
      '',
      `asset_payload = ${body(buildCreateAssetPayload()).replace('"<GROUP_ID>"', 'group_id')}`,
      'asset_response = requests.post(',
      '    f"{base_url}/v1/seedance-official/assets",',
      '    headers=headers,',
      '    json=asset_payload,',
      ')',
      'asset_response.raise_for_status()',
      '',
      'query_response = requests.post(',
      '    f"{base_url}/v1/seedance-official/assets/query",',
      '    headers=headers,',
      '    json={"Filter": {"GroupIds": [group_id]}, "PageNumber": 1, "PageSize": 20},',
      ')',
      'query_response.raise_for_status()',
      'print(query_response.json()["Items"])',
    ].join('\n')
  }

  if (lang === 'typescript') {
    return [
      'type AssetGroupResponse = { Id: string }',
      'type AssetListResponse = { Items?: Array<{ Id: string; URL: string; Status: string }> }',
      '',
      `const baseUrl = '${ctx.baseUrl.replace(/\/+$/, '')}'`,
      'const headers = {',
      `  Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
      `  'Content-Type': 'application/json',`,
      '}',
      '',
      'const groupResponse = await fetch(`${baseUrl}/v1/seedance-official/asset-groups`, {',
      `  method: 'POST',`,
      '  headers,',
      `  body: JSON.stringify(${body(buildAssetGroupPayload()).replaceAll('\n', '\n    ')}),`,
      '})',
      'if (!groupResponse.ok) throw new Error(await groupResponse.text())',
      'const { Id: groupId } = (await groupResponse.json()) as AssetGroupResponse',
      '',
      'const assetResponse = await fetch(`${baseUrl}/v1/seedance-official/assets`, {',
      `  method: 'POST',`,
      '  headers,',
      '  body: JSON.stringify({',
      '    GroupId: groupId,',
      "    URL: 'https://your-cdn.example.com/reference-person.png',",
      "    Name: 'reference-person',",
      "    AssetType: 'Image',",
      '  }),',
      '})',
      'if (!assetResponse.ok) throw new Error(await assetResponse.text())',
      '',
      'const queryResponse = await fetch(`${baseUrl}/v1/seedance-official/assets/query`, {',
      `  method: 'POST',`,
      '  headers,',
      '  body: JSON.stringify({',
      '    Filter: { GroupIds: [groupId], Statuses: [\'Success\'] },',
      '    PageNumber: 1,',
      '    PageSize: 20,',
      '  }),',
      '})',
      'if (!queryResponse.ok) throw new Error(await queryResponse.text())',
      'const assets = (await queryResponse.json()) as AssetListResponse',
      'console.log(assets.Items)',
    ].join('\n')
  }

  return [
    `const baseUrl = '${ctx.baseUrl.replace(/\/+$/, '')}'`,
    'const headers = {',
    `  Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `  'Content-Type': 'application/json',`,
    '}',
    '',
    'const groupResponse = await fetch(`${baseUrl}/v1/seedance-official/asset-groups`, {',
    `  method: 'POST',`,
    '  headers,',
    `  body: JSON.stringify(${body(buildAssetGroupPayload()).replaceAll('\n', '\n    ')}),`,
    '})',
    'if (!groupResponse.ok) throw new Error(await groupResponse.text())',
    'const { Id: groupId } = await groupResponse.json()',
    '',
    'const assetResponse = await fetch(`${baseUrl}/v1/seedance-official/assets`, {',
    `  method: 'POST',`,
    '  headers,',
    '  body: JSON.stringify({',
    '    GroupId: groupId,',
    "    URL: 'https://your-cdn.example.com/reference-person.png',",
    "    Name: 'reference-person',",
    "    AssetType: 'Image',",
    '  }),',
    '})',
    'if (!assetResponse.ok) throw new Error(await assetResponse.text())',
    '',
    'const queryResponse = await fetch(`${baseUrl}/v1/seedance-official/assets/query`, {',
    `  method: 'POST',`,
    '  headers,',
    '  body: JSON.stringify({',
    '    Filter: { GroupIds: [groupId], Statuses: [\'Success\'] },',
    '    PageNumber: 1,',
    '    PageSize: 20,',
    '  }),',
    '})',
    'if (!queryResponse.ok) throw new Error(await queryResponse.text())',
    'console.log(await queryResponse.json())',
  ].join('\n')
}

function buildRealPersonSample(
  lang: SeedanceGuideLang,
  ctx: SeedanceGuideSampleContext
): string {
  const sessionUrl = url(ctx, '/v1/seedance-official/real-person-auth/sessions')
  const groupUrl = url(
    ctx,
    '/v1/seedance-official/real-person-auth/asset-group/by-byted-token'
  )

  if (lang === 'curl') {
    return [
      '# 1. Create a real-person authentication session.',
      `curl ${sessionUrl} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${curlJson(buildRealPersonSessionPayload())}'`,
      '',
      '# 2. Open H5Link in your app. After the user completes verification, exchange BytedToken.',
      `curl ${groupUrl} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${curlJson(buildBytedTokenPayload())}'`,
    ].join('\n')
  }

  if (lang === 'python') {
    return [
      'import requests',
      '',
      `base_url = "${ctx.baseUrl.replace(/\/+$/, '')}"`,
      'headers = {',
      '    "Authorization": "Bearer <YOUR_API_KEY>",',
      '    "Content-Type": "application/json",',
      '}',
      '',
      'session_response = requests.post(',
      '    f"{base_url}/v1/seedance-official/real-person-auth/sessions",',
      '    headers=headers,',
      `    json=${body(buildRealPersonSessionPayload())},`,
      ')',
      'session_response.raise_for_status()',
      'session = session_response.json()',
      'print("Open this H5Link for the user:", session["H5Link"])',
      '',
      'group_response = requests.post(',
      '    f"{base_url}/v1/seedance-official/real-person-auth/asset-group/by-byted-token",',
      '    headers=headers,',
      '    json={"byted_token": session["BytedToken"]},',
      ')',
      'group_response.raise_for_status()',
      'print(group_response.json()["GroupId"])',
    ].join('\n')
  }

  if (lang === 'typescript') {
    return [
      'type RealPersonSession = { BytedToken: string; H5Link: string; CallbackURL?: string }',
      'type RealPersonGroup = { GroupId: string }',
      '',
      `const baseUrl = '${ctx.baseUrl.replace(/\/+$/, '')}'`,
      'const headers = {',
      `  Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
      `  'Content-Type': 'application/json',`,
      '}',
      '',
      'const sessionResponse = await fetch(`${baseUrl}/v1/seedance-official/real-person-auth/sessions`, {',
      `  method: 'POST',`,
      '  headers,',
      `  body: JSON.stringify(${body(buildRealPersonSessionPayload()).replaceAll('\n', '\n    ')}),`,
      '})',
      'if (!sessionResponse.ok) throw new Error(await sessionResponse.text())',
      'const session = (await sessionResponse.json()) as RealPersonSession',
      'console.log(session.H5Link)',
      '',
      'const groupResponse = await fetch(`${baseUrl}/v1/seedance-official/real-person-auth/asset-group/by-byted-token`, {',
      `  method: 'POST',`,
      '  headers,',
      '  body: JSON.stringify({ byted_token: session.BytedToken }),',
      '})',
      'if (!groupResponse.ok) throw new Error(await groupResponse.text())',
      'const group = (await groupResponse.json()) as RealPersonGroup',
      'console.log(group.GroupId)',
    ].join('\n')
  }

  return [
    `const baseUrl = '${ctx.baseUrl.replace(/\/+$/, '')}'`,
    'const headers = {',
    `  Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `  'Content-Type': 'application/json',`,
    '}',
    '',
    'const sessionResponse = await fetch(`${baseUrl}/v1/seedance-official/real-person-auth/sessions`, {',
    `  method: 'POST',`,
    '  headers,',
    `  body: JSON.stringify(${body(buildRealPersonSessionPayload()).replaceAll('\n', '\n    ')}),`,
    '})',
    'if (!sessionResponse.ok) throw new Error(await sessionResponse.text())',
    'const session = await sessionResponse.json()',
    'console.log(session.H5Link)',
    '',
    'const groupResponse = await fetch(`${baseUrl}/v1/seedance-official/real-person-auth/asset-group/by-byted-token`, {',
    `  method: 'POST',`,
    '  headers,',
    '  body: JSON.stringify({ byted_token: session.BytedToken }),',
    '})',
    'if (!groupResponse.ok) throw new Error(await groupResponse.text())',
    'console.log(await groupResponse.json())',
  ].join('\n')
}

function buildWorkflowSample(
  lang: SeedanceGuideLang,
  ctx: SeedanceGuideSampleContext
): string {
  const baseUrl = ctx.baseUrl.replace(/\/+$/, '')
  const taskUrl = url(ctx, ctx.endpointPath)
  const videoPayload = buildVideoTaskPayload(ctx)

  if (lang === 'curl') {
    return [
      '# Recommended production sequence:',
      '# 1. POST /v1/seedance-official/asset-groups',
      '# 2. POST /v1/seedance-official/assets and wait for Status=Success',
      '# 3. If the subject is a real person, create a real-person auth session and exchange BytedToken for GroupId',
      '# 4. Create the video task with content[].image_url.url or content[].video_url.url',
      '# 5. Use callback_url or poll the task until it reaches a terminal status',
      '',
      `curl ${taskUrl} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${curlJson(videoPayload)}'`,
    ].join('\n')
  }

  if (lang === 'python') {
    return [
      'import json',
      'import time',
      'import requests',
      '',
      `base_url = "${baseUrl}"`,
      `task_url = "${taskUrl}"`,
      'headers = {',
      '    "Authorization": "Bearer <YOUR_API_KEY>",',
      '    "Content-Type": "application/json",',
      '}',
      'reference_url = "https://your-cdn.example.com/reference-person.png"',
      '',
      '# Prepare the resource library.',
      'group = requests.post(',
      '    f"{base_url}/v1/seedance-official/asset-groups",',
      '    headers=headers,',
      `    json=${body(buildAssetGroupPayload())},`,
      ').json()',
      'asset = requests.post(',
      '    f"{base_url}/v1/seedance-official/assets",',
      '    headers=headers,',
      '    json={',
      '        "GroupId": group["Id"],',
      '        "URL": reference_url,',
      '        "Name": "reference-person",',
      '        "AssetType": "Image",',
      '    },',
      ').json()',
      '',
      '# Optional real-person verification. Open H5Link and continue after verification completes.',
      'session = requests.post(',
      '    f"{base_url}/v1/seedance-official/real-person-auth/sessions",',
      '    headers=headers,',
      `    json=${body(buildRealPersonSessionPayload())},`,
      ').json()',
      'print("Open H5Link:", session["H5Link"])',
      '',
      '# Use the approved media URL as the generation reference.',
      `payload = json.loads('''${body(videoPayload)}''')`,
      'payload["content"][1]["image_url"]["url"] = asset.get("URL", reference_url)',
      'create_response = requests.post(task_url, headers=headers, json=payload)',
      'create_response.raise_for_status()',
      'task_id = create_response.json()["id"]',
      '',
      'while True:',
      '    task = requests.get(f"{task_url}/{task_id}", headers=headers).json()',
      '    if task.get("status") in {"succeeded", "failed", "expired", "cancelled", "canceled"}:',
      '        break',
      '    time.sleep(5)',
      '',
      'print(task)',
    ].join('\n')
  }

  if (lang === 'typescript') {
    return [
      'type AssetGroupResponse = { Id: string }',
      'type AssetResponse = { Id: string; URL?: string }',
      'type RealPersonSession = { BytedToken: string; H5Link: string }',
      'type SeedanceTask = { id: string; status?: string }',
      '',
      `const baseUrl = '${baseUrl}'`,
      `const taskUrl = '${taskUrl}'`,
      `const referenceUrl = 'https://your-cdn.example.com/reference-person.png'`,
      'const headers = {',
      `  Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
      `  'Content-Type': 'application/json',`,
      '}',
      '',
      'const group = (await fetch(`${baseUrl}/v1/seedance-official/asset-groups`, {',
      `  method: 'POST',`,
      '  headers,',
      `  body: JSON.stringify(${body(buildAssetGroupPayload()).replaceAll('\n', '\n    ')}),`,
      '}).then((res) => res.json())) as AssetGroupResponse',
      '',
      'const asset = (await fetch(`${baseUrl}/v1/seedance-official/assets`, {',
      `  method: 'POST',`,
      '  headers,',
      '  body: JSON.stringify({',
      '    GroupId: group.Id,',
      '    URL: referenceUrl,',
      "    Name: 'reference-person',",
      "    AssetType: 'Image',",
      '  }),',
      '}).then((res) => res.json())) as AssetResponse',
      '',
      'const session = (await fetch(`${baseUrl}/v1/seedance-official/real-person-auth/sessions`, {',
      `  method: 'POST',`,
      '  headers,',
      `  body: JSON.stringify(${body(buildRealPersonSessionPayload()).replaceAll('\n', '\n    ')}),`,
      '}).then((res) => res.json())) as RealPersonSession',
      'console.log(session.H5Link)',
      '',
      `const payload = ${body(videoPayload).replaceAll('\n', '\n  ')}`,
      'payload.content[1].image_url.url = asset.URL ?? referenceUrl',
      '',
      'const created = (await fetch(taskUrl, {',
      `  method: 'POST',`,
      '  headers,',
      '  body: JSON.stringify(payload),',
      '}).then((res) => res.json())) as SeedanceTask',
      '',
      'let task: SeedanceTask',
      'for (;;) {',
      '  task = (await fetch(`${taskUrl}/${created.id}`, { headers }).then((res) => res.json())) as SeedanceTask',
      "  if (['succeeded', 'failed', 'expired', 'cancelled', 'canceled'].includes(task.status ?? '')) break",
      '  await new Promise((resolve) => setTimeout(resolve, 5000))',
      '}',
      'console.log(task)',
    ].join('\n')
  }

  return [
    `const baseUrl = '${baseUrl}'`,
    `const taskUrl = '${taskUrl}'`,
    `const referenceUrl = 'https://your-cdn.example.com/reference-person.png'`,
    'const headers = {',
    `  Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `  'Content-Type': 'application/json',`,
    '}',
    '',
    'const group = await fetch(`${baseUrl}/v1/seedance-official/asset-groups`, {',
    `  method: 'POST',`,
    '  headers,',
    `  body: JSON.stringify(${body(buildAssetGroupPayload()).replaceAll('\n', '\n    ')}),`,
    '}).then((res) => res.json())',
    '',
    'const asset = await fetch(`${baseUrl}/v1/seedance-official/assets`, {',
    `  method: 'POST',`,
    '  headers,',
    '  body: JSON.stringify({',
    '    GroupId: group.Id,',
    '    URL: referenceUrl,',
    "    Name: 'reference-person',",
    "    AssetType: 'Image',",
    '  }),',
    '}).then((res) => res.json())',
    '',
    'const session = await fetch(`${baseUrl}/v1/seedance-official/real-person-auth/sessions`, {',
    `  method: 'POST',`,
    '  headers,',
    `  body: JSON.stringify(${body(buildRealPersonSessionPayload()).replaceAll('\n', '\n    ')}),`,
    '}).then((res) => res.json())',
    'console.log(session.H5Link)',
    '',
    `const payload = ${body(videoPayload).replaceAll('\n', '\n  ')}`,
    'payload.content[1].image_url.url = asset.URL ?? referenceUrl',
    '',
    'const created = await fetch(taskUrl, {',
    `  method: 'POST',`,
    '  headers,',
    '  body: JSON.stringify(payload),',
    '}).then((res) => res.json())',
    '',
    'let task',
    'for (;;) {',
    '  task = await fetch(`${taskUrl}/${created.id}`, { headers }).then((res) => res.json())',
    "  if (['succeeded', 'failed', 'expired', 'cancelled', 'canceled'].includes(task.status ?? '')) break",
    '  await new Promise((resolve) => setTimeout(resolve, 5000))',
    '}',
    'console.log(task)',
  ].join('\n')
}
