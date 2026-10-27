FROM node:alpine AS builder
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
# Vite inlines VITE_* vars at build time, so they must be build args, not runtime env.
ARG VITE_OIDC_AUTHORITY
ARG VITE_OIDC_CLIENT_ID=fai
RUN npm run build

FROM nginx:alpine
COPY --from=builder /src/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
