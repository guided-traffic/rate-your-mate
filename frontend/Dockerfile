# Build stage
FROM node:22-alpine AS builder

# Accept build arguments
ARG BUILD_NUMBER
ARG GIT_COMMIT
ARG BUILD_TIME

WORKDIR /app
COPY package*.json ./
RUN npm ci

COPY . .

RUN npm run build -- --configuration=production

# Production stage
FROM nginx:alpine

# Remove default entrypoint scripts that try to modify config files
RUN rm -rf /docker-entrypoint.d/*

# Create temp directories for read-only filesystem and set permissions
RUN mkdir -p /tmp/nginx-client-body /tmp/nginx-proxy /tmp/nginx-fastcgi /tmp/nginx-uwsgi /tmp/nginx-scgi && \
    mkdir -p /usr/share/nginx/html && \
    chown -R nginx:nginx /tmp/nginx-* /var/cache/nginx /usr/share/nginx/html && \
    chmod -R 755 /tmp/nginx-* /var/cache/nginx

# Copy custom nginx config
COPY nginx.conf /etc/nginx/nginx.conf

# Copy built app
COPY --from=builder /app/dist/frontend/browser /usr/share/nginx/html

# Set ownership and permissions
RUN chown -R nginx:nginx /usr/share/nginx/html

# Expose port
EXPOSE 8080

# Run as nginx user
USER nginx

CMD ["nginx", "-g", "daemon off;"]
