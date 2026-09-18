FROM php:8.1-apache

# Instalar dependencias del sistema
RUN apt-get update && apt-get install -y \
    libpng-dev \
    libjpeg-dev \
    libfreetype6-dev \
    libzip-dev \
    libicu-dev \
    unzip \
    git \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Configurar e instalar extensiones de PHP requeridas por TastyIgniter
RUN docker-php-ext-configure gd --with-freetype --with-jpeg \
    && docker-php-ext-install -j$(nproc) \
        bcmath \
        mysqli \
        pdo_mysql \
        gd \
        zip \
        intl \
        opcache

# Habilitar mod_rewrite de Apache para URLs amigables
RUN a2enmod rewrite

# Habilitar AllowOverride en Apache para leer el archivo .htaccess
RUN echo '<Directory /var/www/html>\n\
    Options Indexes FollowSymLinks\n\
    AllowOverride All\n\
    Require all granted\n\
</Directory>' > /etc/apache2/conf-available/tastyigniter.conf \
    && a2enconf tastyigniter

# Instalar Composer
COPY --from=composer:latest /usr/bin/composer /usr/bin/composer

WORKDIR /var/www/html

# Cambiar permisos al usuario web de Apache
RUN chown -R www-data:www-data /var/www/html
