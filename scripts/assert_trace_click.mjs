#!/usr/bin/env node
// Playwright launcher used by scripts/assert_trace.sh --mode=playwright.
//
// Navigates to a portal URL, clicks a CTA selector, and captures the first
// X-Request-ID that the page receives back from the gateway. Emits two
// key=value lines on stdout:
//   request_id=<uuid>
//   traceparent=<w3c-string>
//
// Requires @playwright/test to be installed under tests/e2e.

import { resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { existsSync } from 'node:fs'

const args = process.argv.slice(2)
function arg(name) {
  const i = args.indexOf(name)
  return i >= 0 ? args[i + 1] : undefined
}

const url = arg('--url')
const selector = arg('--selector')
const gateway = arg('--gateway') ?? 'http://localhost:8080'
const timeoutMs = Number(arg('--timeout') ?? 10_000)

if (!url || !selector) {
  console.error('usage: assert_trace_click.mjs --url <page> --selector <css> [--gateway <url>]')
  process.exit(2)
}

const __filename = fileURLToPath(import.meta.url)
const playwrightRoot = resolve(__filename, '../../tests/e2e/node_modules/@playwright/test')
if (!existsSync(playwrightRoot)) {
  console.error(`playwright not installed at ${playwrightRoot} — run npm ci in tests/e2e first`)
  process.exit(3)
}

const { chromium } = await import(resolve(playwrightRoot, 'index.mjs')).catch(async () => {
  return await import('playwright')
})

const browser = await chromium.launch({ headless: true })
const ctx = await browser.newContext()
const page = await ctx.newPage()

let capturedRequestId = ''
let capturedTraceparent = ''

page.on('response', (resp) => {
  try {
    const u = resp.url()
    if (!u.startsWith(gateway)) return
    const rid = resp.headers()['x-request-id']
    const tp = resp.headers()['traceparent']
    if (rid && !capturedRequestId) {
      capturedRequestId = rid
      if (tp) capturedTraceparent = tp
    }
  } catch {
    /* swallow */
  }
})

try {
  await page.goto(url, { waitUntil: 'networkidle', timeout: timeoutMs })
  const loc = page.locator(selector).first()
  // If the selector is a structural element (e.g. 'main'), a page-load trace
  // is sufficient — we've already captured the XHR headers from goto().
  if (selector !== 'main' && selector !== 'body') {
    await loc.waitFor({ state: 'visible', timeout: timeoutMs }).catch(() => {})
    await loc.click({ timeout: timeoutMs }).catch(() => {})
    // Give the click-driven request a chance to finish.
    await page.waitForLoadState('networkidle', { timeout: timeoutMs }).catch(() => {})
  }
} finally {
  await browser.close()
}

if (!capturedRequestId) {
  console.error('no X-Request-ID captured from any gateway response')
  process.exit(4)
}
console.log(`request_id=${capturedRequestId}`)
console.log(`traceparent=${capturedTraceparent}`)
