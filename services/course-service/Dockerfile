# Build stage
FROM node:20-alpine AS builder

WORKDIR /build

# Copy proto files
COPY proto /build/proto

# Copy package files
COPY services/course-service/package*.json ./

# Install dependencies
RUN --mount=type=cache,target=/root/.npm \
  npm install

# Copy source code
COPY services/course-service/ .

# Build the application
RUN npm run build

# Production stage
FROM node:20-alpine

WORKDIR /app

# Install only production dependencies
COPY services/course-service/package*.json ./
RUN npm install --omit=dev

# Copy built application from builder
COPY --from=builder /build/dist ./dist
COPY --from=builder /build/proto /app/proto

# Set environment variables
ENV NODE_ENV=production

# Expose ports
EXPOSE 3001 50052

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
  CMD node -e "require('http').get('http://localhost:3001/health', (r) => {if(r.statusCode !== 200) throw new Error('Health check failed')})"

# Run the application
CMD ["node", "dist/main"]
