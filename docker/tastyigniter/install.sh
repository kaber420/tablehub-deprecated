#!/bin/bash
# install.sh - Script de automatización completo para TastyIgniter

# Salir ante cualquier error
set -e

# Cambiar al directorio del script
cd "$(dirname "$0")"

echo "==============================================="
echo " Levantando contenedores de Docker... "
echo "==============================================="
docker compose up -d

echo ""
echo "==============================================="
echo " Esperando a que MySQL esté listo... "
echo "==============================================="
docker compose exec db sh -c 'until mysqladmin ping -uroot -proot --silent; do echo "Esperando a MySQL..."; sleep 2; done'

echo ""
echo "==============================================="
echo " Inicializando TastyIgniter con Composer... "
echo "==============================================="
# Si no hay composer.json, creamos el proyecto
docker compose exec app sh -c '
if [ ! -f "composer.json" ]; then
    composer create-project tastyigniter/tastyigniter . --no-interaction
    chown -R www-data:www-data /var/www/html
else
    echo "TastyIgniter ya está inicializado."
fi
'

echo ""
echo "==============================================="
echo " Copiando archivos de configuración (.env y .htaccess) "
echo "==============================================="
docker compose exec app sh -c '
if [ ! -f ".env" ]; then
    cp example.env .env
    cp example.htaccess .htaccess
    echo "Archivos .env y .htaccess copiados."
else
    echo "Los archivos .env y .htaccess ya existen."
fi
'

echo ""
echo "==============================================="
echo " Configurando variables de entorno en .env... "
echo "==============================================="
docker compose exec app sh -c '
sed -i "s/DB_HOST=/DB_HOST=db/g" .env
sed -i "s/DB_PORT=/DB_PORT=3306/g" .env
sed -i "s/DB_DATABASE=/DB_DATABASE=tastyigniter/g" .env
sed -i "s/DB_USERNAME=/DB_USERNAME=tastyuser/g" .env
sed -i "s/DB_PASSWORD=/DB_PASSWORD=tastypassword/g" .env
sed -i "s|APP_URL=|APP_URL=http://localhost:8001|g" .env
'

echo ""
echo "==============================================="
echo " Generando Clave de Encriptación (APP_KEY)... "
echo "==============================================="
docker compose exec app sh -c '
if [ -z "$(grep APP_KEY=base64: .env)" ]; then
    php artisan key:generate
fi
'

echo ""
echo "==============================================="
echo " Instalando Extensión oficial de Webhooks... "
echo "==============================================="
docker compose exec app sh -c '
composer require igniterlabs/ti-ext-webhook -W --no-interaction
'

echo ""
echo "==============================================="
echo " Ejecutando Migraciones y Siembra de BD... "
echo "==============================================="
docker compose exec app sh -c '
php artisan igniter:up
'

echo ""
echo "==============================================="
echo " Reiniciando contenedores para aplicar cambios "
echo "==============================================="
docker compose restart

# Asegurar permisos correctos del servidor web
docker compose exec app chown -R www-data:www-data /var/www/html

echo ""
echo "========================================================================="
echo " ¡PROCESO DE DOCKER COMPLETADO CON ÉXITO! "
echo "========================================================================="
echo "1. Ve a tu navegador: http://localhost:8001"
echo "2. Panel de administración: http://localhost:8001/admin"
echo "3. Configura el Webhook apuntando a:"
echo "   http://host.docker.internal:9000/v1/webhooks/tastyigniter/restaurant_1"
echo "========================================================================="
