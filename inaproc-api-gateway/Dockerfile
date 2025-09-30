# syntax=docker/dockerfile:1

# Simple PHP-FPM + Nginx approach
FROM php:8.2-fpm

# Install required system dependencies and PHP extensions
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      nginx \
      supervisor \
      git \
      unzip \
      curl \
      libicu-dev \
      libpq-dev \
      libzip-dev \
      libpng-dev \
      libxml2-dev \
      libmariadb-dev && \
    docker-php-ext-configure intl && \
    docker-php-ext-install -j"$(nproc)" \
      intl \
      opcache \
      pdo \
      pdo_mysql \
      pdo_pgsql \
      pgsql \
      zip \
      soap \
      sockets && \
    pecl install redis && \
    docker-php-ext-enable redis && \
    rm -rf /var/lib/apt/lists/*

# Copy pre-built application (including vendor/)
COPY . /var/www/html

# Configure nginx
COPY <<EOF /etc/nginx/sites-available/default
server {
    listen 8080;
    server_name _;
    root /var/www/html/public;
    index index.php;

    location / {
        try_files \$uri /index.php\$is_args\$args;
    }

    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass 127.0.0.1:9000;
        fastcgi_param SCRIPT_FILENAME \$document_root\$fastcgi_script_name;
        include fastcgi_params;
    }
}
EOF

# Configure supervisor to run both nginx and php-fpm
COPY <<EOF /etc/supervisor/conf.d/supervisord.conf
[supervisord]
nodaemon=true
user=root

[program:php-fpm]
command=php-fpm
autostart=true
autorestart=true
stdout_logfile=/var/log/php-fpm.log
stderr_logfile=/var/log/php-fpm.log

[program:nginx]
command=nginx -g "daemon off;"
autostart=true
autorestart=true
stdout_logfile=/var/log/nginx.log
stderr_logfile=/var/log/nginx.log
EOF

# Configure PHP-FPM to listen on 127.0.0.1:9000
RUN sed -i 's/listen = \/run\/php\/php8.2-fpm.sock/listen = 127.0.0.1:9000/' /usr/local/etc/php-fpm.d/www.conf

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

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
