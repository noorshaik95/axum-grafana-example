# Build context: ./frontend (parent of provider/ and shared/)

# Stage 1: Builder
FROM node:20-alpine AS builder
RUN apk add --no-cache libc6-compat
WORKDIR /app

# Install shared deps before copying source so node_modules layer is preserved.
# Must run before NODE_ENV=production is set so devDependencies are included.
COPY shared/package.json shared/package-lock.json* ./shared/
RUN npm ci --prefix shared

COPY shared ./shared

WORKDIR /app/provider
COPY provider/package.json provider/package-lock.json* ./

RUN --mount=type=cache,target=/root/.npm \
    npm ci

COPY provider .

ARG NEXT_PUBLIC_API_URL=http://localhost:8080
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_TELEMETRY_DISABLED=1
ENV NODE_ENV=production

RUN npm run build

# Stage 2: Runner (standalone output only — no node_modules needed)
FROM node:20-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

COPY --from=builder --chown=nextjs:nodejs /app/provider/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/provider/.next/static ./.next/static

USER nextjs

EXPOSE 3002
ENV PORT=3002
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]
