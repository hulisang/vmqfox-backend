# Multi-stage Dockerfile for vmqfox-backend (ThinkPHP)
# Optimized per bt.md: PHP 8.2 with required extensions and PHP settings

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

# Install required system libs and PHP extensions (align with bt.md)
RUN set -eux; \
    # base packages + build deps
    apk add --no-cache \
      tzdata \
      libzip-dev zlib-dev \
      oniguruma-dev \
      libxml2-dev \
      curl-dev \
      freetype-dev \
      libjpeg-turbo-dev \
      libpng-dev \
      bash; \
    apk add --no-cache --virtual .build-deps $PHPIZE_DEPS build-base; \
    # gd
    docker-php-ext-configure gd --with-freetype --with-jpeg; \
    # core extensions
    docker-php-ext-install -j"$(nproc)" \
      pdo_mysql \
      mysqli \
      mbstring \
      zip \
      bcmath \
      gd \
      curl \
      xml \
      opcache; \
    # redis via pecl
    pecl install redis && docker-php-ext-enable redis; \
    # cleanup build deps
    apk del .build-deps; \
    rm -rf /tmp/pear; \
    # Configure timezone
    ln -snf /usr/share/zoneinfo/${TZ} /etc/localtime && echo ${TZ} > /etc/timezone

# (Optional) create a non-root user to run PHP-FPM
RUN addgroup -g 1000 www && adduser -D -G www -u 1000 www

WORKDIR /var/www/html

# Copy application code
# Copy vendor from the builder stage first for better layer caching
COPY --from=vendor /app/vendor ./vendor
# Copy application code
COPY . .

# PHP configuration aligned with bt.md recommendations
# Also setting up directory permissions in the same layer
RUN set -eux; \
    { \
        echo "memory_limit=256M"; \
        echo "max_execution_time=300"; \
        echo "post_max_size=50M"; \
        echo "upload_max_filesize=50M"; \
        echo "date.timezone=${TZ}"; \
    } > /usr/local/etc/php/conf.d/vmqfox.ini; \
    { \
        echo "opcache.enable=1"; \
        echo "opcache.validate_timestamps=1"; \
        echo "opcache.memory_consumption=128"; \
        echo "opcache.interned_strings_buffer=16"; \
        echo "opcache.max_accelerated_files=10000"; \
        echo "opcache.revalidate_freq=2"; \
    } > /usr/local/etc/php/conf.d/opcache.ini; \
    # Ensure runtime/cache directories exist and are writable
    mkdir -p runtime public/qr-code; \
    chown -R www:www /var/www/html; \
    chmod -R 777 runtime public/qr-code

# Add entrypoint to generate .env from environment variables
COPY entrypoint.sh /entrypoint.sh
# Fix line endings and set executable permissions
RUN sed -i 's/\r$//' /entrypoint.sh && chmod +x /entrypoint.sh

USER www

# Expose PHP-FPM port
EXPOSE 9000

ENTRYPOINT ["/entrypoint.sh"]
# The main command to run PHP-FPM service. This will be executed by the entrypoint script.
CMD ["php-fpm"]

