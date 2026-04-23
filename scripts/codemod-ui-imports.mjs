#!/usr/bin/env node
/**
 * One-shot codemod: rewrite portal imports from `@/components/ui/<name>` and
 * relative `./components/ui/<name>` / `../components/ui/<name>` forms to
 * `<relative-path>/shared/components/ui/<name>`.
 *
 * Usage: node scripts/codemod-ui-imports.mjs
 *
 * Run once — then delete the portal shim files.
 */

import fs from 'node:fs'
import path from 'node:path'

const PORTALS = ['admin', 'provider', 'student']
const FRONTEND = path.resolve(path.dirname(new URL(import.meta.url).pathname), '..', 'frontend')
const EXTS = new Set(['.ts', '.tsx', '.js', '.jsx'])

// Files in these dirs are the *sources* we rewrite. Portal component/app code.
const INCLUDED_DIRS = ['app', 'components', 'lib', 'hooks']
// Don't rewrite portal ui/ files themselves — those get deleted wholesale.
const EXCLUDED_SEGMENTS = [`components${path.sep}ui${path.sep}`]

function walk(dir, out) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === 'node_modules' || entry.name.startsWith('.')) continue
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) walk(full, out)
    else if (EXTS.has(path.extname(entry.name))) out.push(full)
  }
}

function relativeFromFileToShared(filePath, portalRoot) {
  const fromDir = path.dirname(filePath)
  const sharedUi = path.resolve(FRONTEND, 'shared', 'components', 'ui')
  let rel = path.relative(fromDir, sharedUi).split(path.sep).join('/')
  if (!rel.startsWith('.')) rel = './' + rel
  return rel
}

let stats = { files: 0, edits: 0, perPortal: {} }

for (const portal of PORTALS) {
  const portalRoot = path.join(FRONTEND, portal)
  const files = []
  for (const sub of INCLUDED_DIRS) {
    const subDir = path.join(portalRoot, sub)
    if (fs.existsSync(subDir)) walk(subDir, files)
  }
  stats.perPortal[portal] = { files: 0, edits: 0 }

  for (const file of files) {
    if (EXCLUDED_SEGMENTS.some((s) => file.includes(s))) continue
    const src = fs.readFileSync(file, 'utf8')
    let next = src
    let localEdits = 0

    // Pattern matches three forms:
    //   from '@/components/ui/<name>'
    //   from '../components/ui/<name>' (any ../ depth, ./ too)
    //   from '@/components/ui' (index form, less common)
    // Rewrites to `<rel-to-shared>/components/ui/<name>` or `<rel>/components/ui`.
    next = next.replace(
      /from\s+(['"])((?:@\/|\.{1,2}\/(?:\.{1,2}\/)*)components\/ui(?:\/[a-zA-Z0-9_-]+)?)\1/g,
      (match, quote, spec) => {
        // Extract component name (or empty if barrel).
        const m = spec.match(/components\/ui(?:\/([a-zA-Z0-9_-]+))?$/)
        const name = m && m[1] ? `/${m[1]}` : ''
        const rel = relativeFromFileToShared(file, portalRoot)
        localEdits += 1
        return `from ${quote}${rel}${name}${quote}`
      }
    )

    if (localEdits > 0) {
      fs.writeFileSync(file, next, 'utf8')
      stats.files += 1
      stats.edits += localEdits
      stats.perPortal[portal].files += 1
      stats.perPortal[portal].edits += localEdits
    }
  }
}

console.log(JSON.stringify(stats, null, 2))
