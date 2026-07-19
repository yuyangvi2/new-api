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
import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import zlib from 'node:zlib'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(__dirname, '..')
const rsbuildCandidates = [
  path.join(
    root,
    '..',
    'node_modules',
    '@rsbuild',
    'core',
    'bin',
    'rsbuild.js'
  ),
  path.join(root, 'node_modules', '@rsbuild', 'core', 'bin', 'rsbuild.js'),
]
const rsbuildBin = rsbuildCandidates.find((candidate) =>
  fs.existsSync(candidate)
)

if (!rsbuildBin) {
  console.error('Unable to find @rsbuild/core/bin/rsbuild.js')
  process.exit(1)
}

const result = spawnSync(process.execPath, [rsbuildBin, 'build'], {
  cwd: root,
  env: { ...process.env, BUNDLE_STATS: '1' },
  stdio: 'inherit',
})

if (result.status !== 0) {
  process.exit(result.status ?? 1)
}

const distDir = path.join(root, 'dist')
const statsPath = path.join(distDir, 'bundle-stats.json')
const stats = JSON.parse(fs.readFileSync(statsPath, 'utf8'))

function walkFiles(dir) {
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const fullPath = path.join(dir, entry.name)
    return entry.isDirectory() ? walkFiles(fullPath) : [fullPath]
  })
}

function formatKb(bytes) {
  return `${(bytes / 1024).toFixed(1)} KB`
}

const jsAssets = walkFiles(distDir)
  .filter((file) => file.endsWith('.js'))
  .map((file) => {
    const raw = fs.statSync(file).size
    const gzip = zlib.gzipSync(fs.readFileSync(file), { level: 9 }).length
    return {
      file: path.relative(distDir, file).replaceAll(path.sep, '/'),
      raw,
      gzip,
    }
  })
  .sort((a, b) => b.raw - a.raw)

const modules = (stats.modules ?? [])
  .filter((module) => typeof module.size === 'number')
  .map((module) => ({
    name: module.name ?? module.identifier ?? '(unknown)',
    size: module.size,
  }))
  .sort((a, b) => b.size - a.size)

const reportLines = [
  '# Bundle Report',
  '',
  '## Largest JavaScript Assets',
  '',
  '| File | Raw | Gzip |',
  '| --- | ---: | ---: |',
  ...jsAssets
    .slice(0, 20)
    .map(
      (asset) =>
        `| ${asset.file} | ${formatKb(asset.raw)} | ${formatKb(asset.gzip)} |`
    ),
  '',
  '## Largest Modules',
  '',
  '| Module | Parsed Size |',
  '| --- | ---: |',
  ...modules
    .slice(0, 40)
    .map(
      (module) =>
        `| ${module.name.replaceAll('|', '\\|')} | ${formatKb(module.size)} |`
    ),
  '',
]

const reportPath = path.join(distDir, 'bundle-report.md')
fs.writeFileSync(reportPath, `${reportLines.join('\n')}\n`)

console.log(`\nBundle stats written to ${path.relative(root, statsPath)}`)
console.log(`Bundle report written to ${path.relative(root, reportPath)}`)
console.log('\nLargest JavaScript assets:')
for (const asset of jsAssets.slice(0, 10)) {
  console.log(
    `- ${asset.file}: raw ${formatKb(asset.raw)}, gzip ${formatKb(asset.gzip)}`
  )
}
