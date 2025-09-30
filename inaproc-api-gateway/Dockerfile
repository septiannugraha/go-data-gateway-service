# syntax=docker/dockerfile:1

# Build dependencies and vendor packages
FROM composer:2.8 AS vendor
WORKDIR /app

# Copy composer manifests and install without dev dependencies
COPY composer.json composer.lock ./
RUN composer install \
    --no-dev \
    --prefer-dist \
    --no-interaction \
    --no-progress

# Copy application source and install optimized autoloader
COPY . .
RUN composer install \
    --no-dev \
    --prefer-dist \
    --no-interaction \
    --no-progress \
    --optimize-autoloader

# Runtime image
FROM php:8.2-apache

# Install required system dependencies and PHP extensions
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      git \
      unzip \
      curl \
      libicu-dev \
      libpq-dev \
      libzip-dev \
      libpng-dev \
      libxml2-dev && \
    docker-php-ext-configure intl && \
    docker-php-ext-install -j"$(nproc)" \
      intl \
      opcache \
      pdo \
      pdo_pgsql \
      pgsql \
      zip && \
    pecl install redis && \
    docker-php-ext-enable redis && \
    rm -rf /var/lib/apt/lists/*

# Configure Apache to serve the public directory and enable rewrite
ENV APACHE_DOCUMENT_ROOT=/var/www/html/public
RUN sed -ri -e 's!/var/www/html!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/sites-available/*.conf && \
    sed -ri -e 's!/var/www/html!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/apache2.conf && \
    a2enmod rewrite headers

# Copy application from builder image
COPY --from=vendor /app /var/www/html

# Ensure correct ownership for runtime writes (cache, logs, etc.)
RUN chown -R www-data:www-data /var/www/html && \
    find /var/www/html -type d -exec chmod 755 {} \; && \
    find /var/www/html -type f -exec chmod 644 {} \;

# Surface environment variables commonly configured via Kubernetes secrets/configmaps
ENV FUSIO_ENV=prod \
    PHP_MEMORY_LIMIT=512M \
    PHP_MAX_EXECUTION_TIME=60

# Basic healthcheck hitting the published root path
HEALTHCHECK --interval=30s --timeout=5s --retries=5 \
  CMD curl -fsS "http://127.0.0.1:8080/" || exit 1

EXPOSE 8080

# Switch Apache to listen on 8080 to align with container best practices
RUN sed -ri 's/80/8080/g' /etc/apache2/ports.conf /etc/apache2/sites-available/000-default.conf

CMD ["apache2-foreground"]
