# syntax=docker/dockerfile:1

FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json ./
COPY source.config.ts ./
COPY next.config.mjs ./
COPY tsconfig.json ./
COPY mdx-components.tsx ./
RUN npm install

FROM node:22-alpine AS builder
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_PUBLIC_BASE_PATH=/api
ENV NEXT_PUBLIC_ASSET_PREFIX=/api
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build && npm prune --omit=dev

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production

# Copy the compiled Next.js output and runtime dependencies
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/next.config.mjs ./next.config.mjs
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/content ./content
COPY --from=builder /app/lib ./lib
COPY --from=builder /app/mdx-components.tsx ./mdx-components.tsx
COPY --from=builder /app/source.config.ts ./source.config.ts

EXPOSE 3000

CMD ["npm", "run", "start"]
