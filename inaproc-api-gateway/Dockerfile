# Apache-based Dockerfile for Fusio
# Based on official fusio-docker implementation
FROM php:8.3-apache

LABEL maintainer="Generated for LKPP Deployment"
LABEL version="1.0.0"
LABEL description="Fusio API Gateway - Apache Version"

# Environment variables
ENV FUSIO_PROJECT_KEY="cee262a5a1bdd205cbfe0dfc2ad4eddf"
ENV FUSIO_ENV="prod"
ENV FUSIO_DEBUG="false"
ENV FUSIO_CONNECTION="pdo-pgsql://airbyte:airbyte@host.docker.internal:5434/fusio"
ENV FUSIO_BACKEND_USER="admin"
ENV FUSIO_BACKEND_EMAIL="admin@fusio-project.org"
ENV FUSIO_BACKEND_PW="fusio123"
ENV FUSIO_MAILER="native://default"
ENV FUSIO_MESSENGER="doctrine://default"
ENV FUSIO_URL="http://localhost"
ENV FUSIO_APPS_URL="http://localhost/apps"

# Install system packages and PHP extensions
RUN apt-get update && apt-get -y install \
    wget \
    git \
    unzip \
    cron \
    sudo \
    supervisor \
    default-mysql-client \
    libpq-dev \
    libxml2-dev \
    libcurl3-dev \
    libzip-dev \
    libonig-dev \
    libpng-dev \
    libmemcached-dev \
    openssl \
    libssl-dev \
    libicu-dev \
    libcurl4-openssl-dev

# Install PHP extensions
RUN docker-php-ext-install \
    pgsql \
    mysqli \
    pdo \
    pdo_mysql \
    pdo_pgsql \
    simplexml \
    dom \
    bcmath \
    curl \
    zip \
    mbstring \
    opcache \
    intl \
    xml \
    gd \
    soap \
    sockets

# Install PECL extensions
RUN pecl install memcached && docker-php-ext-enable memcached
RUN pecl install apcu && docker-php-ext-enable apcu

# Install Composer
ENV COMPOSER_VERSION="2.8.2"
RUN wget -O /usr/bin/composer "https://getcomposer.org/download/${COMPOSER_VERSION}/composer.phar"
RUN chmod +x /usr/bin/composer

# Copy and install Fusio application
WORKDIR /var/www/html/fusio
COPY . .
RUN /usr/bin/composer install --no-dev --optimize-autoloader --classmap-authoritative
RUN /usr/bin/composer dump-autoload --no-dev --optimize --classmap-authoritative
RUN chmod +x /var/www/html/fusio/bin/fusio

# Apache configuration
RUN rm /etc/apache2/sites-available/*.conf
RUN rm /etc/apache2/sites-enabled/*.conf

# Create Apache configuration
RUN echo '<VirtualHost *:80>' > /etc/apache2/sites-available/fusio.conf && \
    echo '    ServerName localhost' >> /etc/apache2/sites-available/fusio.conf && \
    echo '    DocumentRoot /var/www/html/fusio/public' >> /etc/apache2/sites-available/fusio.conf && \
    echo '    <Directory /var/www/html/fusio/public>' >> /etc/apache2/sites-available/fusio.conf && \
    echo '        Options FollowSymLinks' >> /etc/apache2/sites-available/fusio.conf && \
    echo '        AllowOverride All' >> /etc/apache2/sites-available/fusio.conf && \
    echo '        Require all granted' >> /etc/apache2/sites-available/fusio.conf && \
    echo '        RewriteEngine On' >> /etc/apache2/sites-available/fusio.conf && \
    echo '        RewriteBase /' >> /etc/apache2/sites-available/fusio.conf && \
    echo '        RewriteCond %{REQUEST_FILENAME} !-f' >> /etc/apache2/sites-available/fusio.conf && \
    echo '        RewriteCond %{REQUEST_FILENAME} !-d' >> /etc/apache2/sites-available/fusio.conf && \
    echo '        RewriteRule (.*) /index.php/$1 [L]' >> /etc/apache2/sites-available/fusio.conf && \
    echo '        RewriteCond %{HTTP:Authorization} ^(.*)' >> /etc/apache2/sites-available/fusio.conf && \
    echo '        RewriteRule .* - [e=HTTP_AUTHORIZATION:%1]' >> /etc/apache2/sites-available/fusio.conf && \
    echo '    </Directory>' >> /etc/apache2/sites-available/fusio.conf && \
    echo '    ErrorLog ${APACHE_LOG_DIR}/fusio.error.log' >> /etc/apache2/sites-available/fusio.conf && \
    echo '    CustomLog ${APACHE_LOG_DIR}/fusio.access.log combined' >> /etc/apache2/sites-available/fusio.conf && \
    echo '</VirtualHost>' >> /etc/apache2/sites-available/fusio.conf

RUN a2enmod rewrite
RUN a2ensite fusio

# PHP configuration
RUN mv "${PHP_INI_DIR}/php.ini-production" "${PHP_INI_DIR}/php.ini"

# Create PHP configuration for Fusio
RUN echo 'memory_limit = 512M' > "${PHP_INI_DIR}/conf.d/fusio.ini" && \
    echo 'post_max_size = 96M' >> "${PHP_INI_DIR}/conf.d/fusio.ini" && \
    echo 'upload_max_filesize = 64M' >> "${PHP_INI_DIR}/conf.d/fusio.ini" && \
    echo 'date.timezone = UTC' >> "${PHP_INI_DIR}/conf.d/fusio.ini" && \
    echo 'pdo_mysql.default_socket = /var/run/mysqld/mysqld.sock' >> "${PHP_INI_DIR}/conf.d/fusio.ini"

# Clean up installation file for security
RUN rm -f /var/www/html/fusio/public/install.php

# Set proper permissions
RUN chown -R www-data:www-data /var/www/html/fusio
RUN chmod -R 755 /var/www/html/fusio

# Create cron jobs
RUN echo "" > /etc/cron.d/fusio
RUN echo "* * * * * www-data cd /var/www/html/fusio && php bin/fusio cronjob" >> /etc/cron.d/fusio
RUN echo "0 0 1 * * www-data cd /var/www/html/fusio && php bin/fusio system:log_rotate" >> /etc/cron.d/fusio
RUN echo "0 0 1 * * www-data cd /var/www/html/fusio && php bin/fusio system:clean" >> /etc/cron.d/fusio
RUN chmod 0644 /etc/cron.d/fusio

# Copy entrypoint script
COPY apache-docker-entrypoint.sh /apache-docker-entrypoint.sh
RUN chmod +x /apache-docker-entrypoint.sh

# Expose port 80
EXPOSE 80

# Set entrypoint
ENTRYPOINT ["/apache-docker-entrypoint.sh"]