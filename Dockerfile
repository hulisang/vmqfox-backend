# Multi-stage Dockerfile for vmqfox-backend (ThinkPHP)
# Based on CI config: PHP 8.2 with extensions: mbstring, zip, pdo_mysql

# --- Stage 1: Composer dependencies ---
FROM composer:2 AS vendor
WORKDIR /app
# Copy composer files first to leverage docker layer cache
COPY composer.json composer.lock* ./
# Install PHP dependencies (no dev) with optimized autoloader
RUN composer install \
    --no-dev \
    --no-interaction \
    --prefer-dist \
    --no-progress \
    --optimize-autoloader

# --- Stage 2: Runtime (PHP-FPM) ---
FROM php:8.2-fpm-alpine AS runtime

ARG TZ=Asia/Shanghai
ENV TZ=${TZ}

# Install required system libs and PHP extensions
RUN set -eux; \
    apk add --no-cache \
      tzdata \
      libzip-dev \
      oniguruma-dev \
      icu-data-full icu-libs \
      freetype-dev \
      libjpeg-turbo-dev \
      libpng-dev; \
    docker-php-ext-configure gd --with-freetype --with-jpeg; \
    docker-php-ext-install \
      pdo_mysql \
      mbstring \
      zip \
      bcmath \
      gd; \
    # Configure timezone
    ln -snf /usr/share/zoneinfo/${TZ} /etc/localtime && echo ${TZ} > /etc/timezone

# (Optional) create a non-root user to run PHP-FPM
RUN addgroup -g 1000 www && adduser -D -G www -u 1000 www

WORKDIR /var/www/html

# Copy application code
COPY . .
# Copy vendor from the builder stage
COPY --from=vendor /app/vendor ./vendor

# Add entrypoint to generate .env from environment variables
COPY entrypoint.sh /entrypoint.sh
# Fix line endings and set executable permissions
RUN sed -i 's/\r$//' /entrypoint.sh && chmod +x /entrypoint.sh

# Ensure runtime/cache directories are writable (ThinkPHP uses runtime/)
RUN set -eux; \
    mkdir -p runtime && \
    chown -R www:www /var/www/html

USER www

# Expose ThinkPHP built-in server port
EXPOSE 8000

ENTRYPOINT ["/entrypoint.sh"]
CMD ["php", "think", "run"]

