import { NextResponse, type NextRequest } from 'next/server'

/**
 * Admin portal middleware — matches provider's auth-guard pattern.
 *
 * Responsibilities:
 *   1. Legacy redirect: `/universities/*` → `/schools/*` (plan R7 bookmark preservation).
 *   2. Auth guard: unauthed traffic to any non-public route → `/login`.
 *                  Authed traffic hitting `/login` → `/ops` (admin landing).
 *
 * Matcher excludes API routes + Next static + favicon so tracing + asset
 * serving stay zero-cost.
 */
export function middleware(request: NextRequest) {
  const { pathname, search } = request.nextUrl

  // Legacy redirect: /universities/* → /schools/*
  if (pathname === '/universities' || pathname.startsWith('/universities/')) {
    const url = request.nextUrl.clone()
    url.pathname = pathname.replace(/^\/universities/, '/schools')
    url.search = search
    return NextResponse.redirect(url)
  }

  // Auth guard
  const token = request.cookies.get('slate_token')?.value
  const isAuthPage = pathname.startsWith('/login')

  if (!token && !isAuthPage) {
    return NextResponse.redirect(new URL('/login', request.url))
  }

  if (token && isAuthPage) {
    return NextResponse.redirect(new URL('/ops', request.url))
  }

  return NextResponse.next()
}

export const config = {
  matcher: ['/((?!api|_next/static|_next/image|favicon.ico).*)'],
}
