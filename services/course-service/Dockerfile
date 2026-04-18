FROM node:20-alpine AS builder

WORKDIR /build

COPY proto /build/proto
COPY services/course-service/package*.json ./

RUN --mount=type=cache,target=/root/.npm \
    npm install

COPY services/course-service/ .

RUN npm run build

# Production stage
FROM node:20-alpine

WORKDIR /app

COPY services/course-service/package*.json ./

# Production install also benefits from the npm download cache
RUN --mount=type=cache,target=/root/.npm \
    npm install --omit=dev

COPY --from=builder /build/dist ./dist
COPY --from=builder /build/proto /app/proto

ENV NODE_ENV=production

EXPOSE 3001 50052

HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
  CMD node -e "require('http').get('http://localhost:3001/health', (r) => {if(r.statusCode !== 200) throw new Error('Health check failed')})"

CMD ["node", "dist/main"]
