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
import { cn } from '@/lib/utils'

interface HeroTerminalDemoProps {
  className?: string
}

export function HeroTerminalDemo(props: HeroTerminalDemoProps) {
  return (
    <div
      className={cn(
        'w-full max-w-xl rounded-lg border border-slate-800 bg-slate-950 text-slate-200 shadow-[0_26px_70px_-30px_rgba(15,23,42,0.75)]',
        props.className
      )}
    >
      <div className='flex items-center gap-3 border-b border-white/10 px-4 py-3'>
        <div className='flex gap-1.5'>
          <span className='size-2.5 rounded-full bg-red-400/80' />
          <span className='size-2.5 rounded-full bg-amber-300/80' />
          <span className='size-2.5 rounded-full bg-emerald-400/80' />
        </div>
        <span className='font-mono text-[10px] tracking-[0.18em] text-slate-500 uppercase'>
          API request terminal
        </span>
      </div>

      <pre className='overflow-x-auto px-4 py-5 font-mono text-[12px] leading-6 md:text-[13px]'>
        <code>
          <Line>
            <Command>curl</Command>{' '}
            <Plain>https://api.gateway.com/v1/chat/completions \</Plain>
          </Line>
          <Line indent={2}>
            <Flag>-H</Flag>{' '}
            <StringText>
              &quot;Authorization: Bearer sk-your-key&quot;
            </StringText>{' '}
            <Plain>\</Plain>
          </Line>
          <Line indent={2}>
            <Flag>-H</Flag>{' '}
            <StringText>&quot;Content-Type: application/json&quot;</StringText>{' '}
            <Plain>\</Plain>
          </Line>
          <Line indent={2}>
            <Flag>-d</Flag> <StringText>&apos;{'{'}</StringText>
          </Line>
          <Line indent={4}>
            <Key>&quot;model&quot;</Key>
            <Plain>: </Plain>
            <StringText>&quot;gpt-4o&quot;</StringText>
            <Plain>,</Plain>
          </Line>
          <Line indent={4}>
            <Key>&quot;messages&quot;</Key>
            <Plain>: [</Plain>
            <StringText>
              {'{'}&quot;role&quot;: &quot;user&quot;, &quot;content&quot;:
              &quot;Hello AI!&quot;{'}'}
            </StringText>
            <Plain>]</Plain>
          </Line>
          <Line indent={2}>
            <StringText>{'}'}&apos;</StringText>
          </Line>
          <Line />
          <Line>
            <Comment># Response Received (124ms)</Comment>
          </Line>
          <Line>
            <Plain>{'{'} </Plain>
            <Key>&quot;id&quot;</Key>
            <Plain>: </Plain>
            <StringText>&quot;chat-123&quot;</StringText>
            <Plain>, </Plain>
            <Key>&quot;object&quot;</Key>
            <Plain>: </Plain>
            <StringText>&quot;chat.completion&quot;</StringText>
            <Plain>, ... {'}'}</Plain>
          </Line>
        </code>
      </pre>
    </div>
  )
}

function Line(props: { children?: React.ReactNode; indent?: number }) {
  return (
    <span className='block whitespace-pre'>
      {props.indent ? (
        <span
          aria-hidden
          className='inline-block'
          style={{ width: `${props.indent}ch` }}
        />
      ) : null}
      {props.children}
    </span>
  )
}

function Command(props: { children: React.ReactNode }) {
  return <span className='text-sky-400'>{props.children}</span>
}

function Flag(props: { children: React.ReactNode }) {
  return <span className='text-cyan-300'>{props.children}</span>
}

function Key(props: { children: React.ReactNode }) {
  return <span className='text-amber-300'>{props.children}</span>
}

function StringText(props: { children: React.ReactNode }) {
  return <span className='text-emerald-300'>{props.children}</span>
}

function Comment(props: { children: React.ReactNode }) {
  return <span className='text-slate-500'>{props.children}</span>
}

function Plain(props: { children: React.ReactNode }) {
  return <span className='text-slate-300'>{props.children}</span>
}
