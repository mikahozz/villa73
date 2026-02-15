# Build stage
FROM node:22.22.0-bookworm-slim AS build
WORKDIR /app

ARG PNPM_VERSION=10.29.3
RUN npm install -g "pnpm@${PNPM_VERSION}"

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# Copy only files required for the Vite build.
# Keep nginx.conf out of this stage so nginx-only changes do not invalidate pnpm build cache.
COPY index.html ./
COPY tsconfig.json tsconfig.app.json tsconfig.node.json ./
COPY vite.config.ts postcss.config.js tailwind.config.js ./
COPY public ./public
COPY src ./src
RUN pnpm run build

# Production stage
FROM nginx:alpine AS production

COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 3000
CMD ["nginx", "-g", "daemon off;"]
