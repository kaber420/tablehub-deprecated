#!/bin/bash
# trigger_test_order.sh - Registra el webhook de TastyIgniter y dispara una orden de prueba

# Salir si ocurre un error
set -e

# Cambiar al directorio del script
cd "$(dirname "$0")"

echo "================================================================="
echo " 1. Asegurando que el Webhook esté registrado en TastyIgniter..."
echo "================================================================="

# Código PHP para registrar el Webhook si no existe
PHP_REGISTER_CODE='
try {
    $exists = \IgniterLabs\Webhook\Models\Outgoing::where("url", "like", "%v1/webhooks/tastyigniter%")->exists();
    if (!$exists) {
        $webhook = new \IgniterLabs\Webhook\Models\Outgoing();
        $webhook->name = "Tablehub Webhook Local";
        $webhook->url = "http://host.docker.internal:9000/v1/webhooks/tastyigniter/test-restaurant-001?token=secret_tasty_token_123";
        $webhook->events = ["order"];
        $webhook->is_active = true;
        $webhook->config_data = ["verify_ssl" => false, "content_type" => "application/json"];
        $webhook->save();
        echo "WEBHOOK_STATUS: Webhook registrado exitosamente.\n";
    } else {
        echo "WEBHOOK_STATUS: Webhook ya registrado previamente.\n";
    }
} catch (\Exception $e) {
    echo "WEBHOOK_ERROR: " . $e->getMessage() . "\n";
}
'

# Ejecutar el registro mediante tinker
echo "$PHP_REGISTER_CODE" | docker compose exec -T app php artisan tinker | grep -E "WEBHOOK_STATUS|WEBHOOK_ERROR" || true

echo ""
echo "================================================================="
echo " 2. Disparando un evento de Orden de prueba (Mesa 12)..."
echo "================================================================="

# Código PHP para disparar el webhook de la orden
PHP_TRIGGER_CODE='
try {
    $orderId = rand(1000, 9999);
    $payload = [
        "order" => [
            "order_id" => $orderId,
            "table_number" => "Mesa 12",
            "customer_name" => "Juan Perez (Prueba)",
            "order_type" => "dinein",
            "comment" => "Prueba automatizada de Tablehub",
            "order_totals" => [
                ["title" => "Subtotal", "value" => 25.50],
                ["title" => "Total", "value" => 25.50]
            ]
        ]
    ];
    
    \IgniterLabs\Webhook\Classes\WebhookManager::instance()->runWebhookEvent("order", "created", $payload);
    echo "TRIGGER_STATUS: Evento de webhook para la Orden #$orderId (Mesa 12) disparado correctamente.\n";
} catch (\Exception $e) {
    echo "TRIGGER_ERROR: " . $e->getMessage() . "\n";
}
'

# Ejecutar el disparo mediante tinker
echo "$PHP_TRIGGER_CODE" | docker compose exec -T app php artisan tinker | grep -E "TRIGGER_STATUS|TRIGGER_ERROR" || true

echo ""
echo "================================================================="
echo " ¡PROCESO COMPLETADO! "
echo " Revisa los logs de la Nube (puerto 9000) y de tu Hub local. "
echo "================================================================="
