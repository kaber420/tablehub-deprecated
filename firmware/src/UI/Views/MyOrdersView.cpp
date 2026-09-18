#include "MyOrdersView.h"
#include "../UIManager.h"
#include "../Themes/NeumorphicStyles.h"

lv_obj_t* MyOrdersView::list = nullptr;
HeaderBar* MyOrdersView::headerBar = nullptr;
lv_obj_t* MyOrdersView::progressCard = nullptr;
lv_obj_t* MyOrdersView::progressLabel = nullptr;
lv_obj_t* MyOrdersView::progressBar = nullptr;

lv_obj_t* MyOrdersView::create() {
    lv_obj_t* screen = lv_obj_create(NULL);
    NeumorphicStyles::applyFlatBg(screen);
    lv_obj_set_flex_flow(screen, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_style_pad_all(screen, 12, 0);
    lv_obj_set_style_pad_row(screen, 12, 0);

    // Header Neomórfico Elevado (Reutilizable)
    headerBar = HeaderBar::create(screen, "Mis Pedidos", true, false);

    // Tarjeta de Progreso General del Pedido (Arriba de la lista)
    progressCard = lv_obj_create(screen);
    lv_obj_set_width(progressCard, lv_pct(100));
    lv_obj_set_height(progressCard, 68);
    NeumorphicStyles::applySunkenCard(progressCard, 14);
    lv_obj_set_style_pad_all(progressCard, 12, 0);
    lv_obj_set_flex_flow(progressCard, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_style_pad_row(progressCard, 6, 0);
    lv_obj_add_flag(progressCard, LV_OBJ_FLAG_HIDDEN); // Ocultar por defecto si no hay órdenes

    progressLabel = lv_label_create(progressCard);
    lv_label_set_text(progressLabel, "Preparación del Pedido: 0%");
    lv_obj_set_style_text_color(progressLabel, NeumorphicStyles::getTextColor(), 0);
    lv_obj_set_style_text_font(progressLabel, &lv_font_montserrat_12, 0);

    progressBar = lv_bar_create(progressCard);
    lv_obj_set_size(progressBar, lv_pct(100), 10);
    lv_obj_set_style_bg_color(progressBar, lv_color_hex(0x111318), LV_PART_MAIN);
    lv_obj_set_style_radius(progressBar, 5, 0);
    lv_obj_set_style_bg_color(progressBar, NeumorphicStyles::getPrimaryAccent(), LV_PART_INDICATOR);

    // Lista de Pedidos en Contenedores Neomórficos
    list = lv_obj_create(screen);
    lv_obj_set_width(list, lv_pct(100));
    lv_obj_set_flex_grow(list, 1);
    
    lv_obj_set_style_bg_opa(list, 0, 0);
    lv_obj_set_style_border_width(list, 0, 0);
    lv_obj_set_style_pad_all(list, 0, 0); // Quita el padding por defecto para que iguale a la tarjeta de progreso
    lv_obj_set_flex_flow(list, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_style_pad_row(list, 12, 0);

    // Evitar dangling pointers asíncronos limpiando el puntero al destruirse la lista
    lv_obj_add_event_cb(list, on_list_delete, LV_EVENT_DELETE, NULL);

    return screen;
}

void MyOrdersView::on_list_delete(lv_event_t * e) {
    list = nullptr;
    progressCard = nullptr;
    progressLabel = nullptr;
    progressBar = nullptr;
}

void MyOrdersView::updateOrders(const std::vector<OrderItem>& orders) {
    if (!list) return;

    uint32_t child_cnt = lv_obj_get_child_count(list);
    uint32_t needed_cnt = orders.size();

    // Calcular progreso general y contar estados
    int totalItems = orders.size();
    int totalProgress = 0;
    int readyItems = 0;
    int deliveredItems = 0;

    for (const auto& item : orders) {
        totalProgress += item.progress;
        if (item.status == "Servido" || item.status == "Listo" || item.status == "Ready") {
            readyItems++;
        } else if (item.status == "Entregado" || item.status == "Delivered") {
            deliveredItems++;
        }
    }
    int overallProgress = totalItems > 0 ? (totalProgress / totalItems) : 0;

    // Actualizar tarjeta de progreso general
    if (totalItems == 0) {
        if (progressCard) lv_obj_add_flag(progressCard, LV_OBJ_FLAG_HIDDEN);
    } else {
        if (progressCard) {
            lv_obj_remove_flag(progressCard, LV_OBJ_FLAG_HIDDEN);
            lv_bar_set_value(progressBar, overallProgress, LV_ANIM_OFF);

            lv_color_t overallColor = NeumorphicStyles::getPrimaryAccent();
            String statusText = "Preparación del Pedido: " + String(overallProgress) + "%";

            if (deliveredItems == totalItems) {
                overallColor = lv_color_hex(0x2ECC71); // Verde Esmeralda
                statusText = "¡Pedido Entregado! Buen provecho";
            } else if (overallProgress >= 100) {
                overallColor = lv_color_hex(0x00F5D4); // Neon Cyan
                statusText = "¡Tu Pedido está listo!";
            } else if (overallProgress == 0) {
                overallColor = lv_color_hex(0xFFB800); // Amarillo
                statusText = "Pedido Recibido: En Espera";
            }

            lv_label_set_text(progressLabel, statusText.c_str());
            lv_obj_set_style_bg_color(progressBar, overallColor, LV_PART_INDICATOR);
        }
    }

    // Reconciliar cantidad: crear nuevas cards si nos faltan
    for (uint32_t i = child_cnt; i < needed_cnt; i++) {
        lv_obj_t * card = lv_obj_create(list);
        lv_obj_set_width(card, lv_pct(100));
        lv_obj_set_height(card, 84); // Aumentado de 64 a 84 para que no parezcan "píldoras"
        NeumorphicStyles::applySunkenCard(card, 12);
        lv_obj_set_style_pad_all(card, 8, 0);
        lv_obj_set_flex_flow(card, LV_FLEX_FLOW_ROW);
        lv_obj_set_flex_align(card, LV_FLEX_ALIGN_START, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
        lv_obj_set_style_pad_column(card, 10, 0);

        // 0. Contenedor de Texto central (Flex Column)
        lv_obj_t * textContainer = lv_obj_create(card);
        lv_obj_set_width(textContainer, 170); // Ancho ajustado para no aplastar el texto derecho
        lv_obj_set_height(textContainer, LV_SIZE_CONTENT);
        lv_obj_set_style_bg_opa(textContainer, LV_OPA_TRANSP, 0);
        lv_obj_set_style_border_width(textContainer, 0, 0);
        lv_obj_set_style_pad_all(textContainer, 0, 0);
        lv_obj_set_flex_flow(textContainer, LV_FLEX_FLOW_COLUMN);
        lv_obj_set_style_pad_row(textContainer, 2, 0);
        lv_obj_remove_flag(textContainer, LV_OBJ_FLAG_CLICKABLE);

        // Nombre del Platillo
        lv_obj_t * name = lv_label_create(textContainer);
        lv_obj_set_style_text_color(name, NeumorphicStyles::getTextColor(), 0);
        lv_obj_set_style_text_font(name, &lv_font_montserrat_14, 0);
        lv_label_set_long_mode(name, LV_LABEL_LONG_WRAP);
        lv_obj_set_width(name, lv_pct(100));

        // Modificadores / Opciones (Marquesina deslizante)
        lv_obj_t * optionsLbl = lv_label_create(textContainer);
        lv_obj_set_style_text_color(optionsLbl, NeumorphicStyles::getMutedTextColor(), 0);
        lv_obj_set_style_text_font(optionsLbl, &lv_font_montserrat_12, 0);
        lv_label_set_long_mode(optionsLbl, LV_LABEL_LONG_SCROLL_CIRCULAR);
        lv_obj_set_width(optionsLbl, lv_pct(100));

        // 2. Estado del Platillo (Alineado a la derecha)
        lv_obj_t * stat = lv_label_create(card);
        lv_obj_set_flex_grow(stat, 1);
        lv_label_set_long_mode(stat, LV_LABEL_LONG_CLIP); // Evita que se vuelva vertical si falta espacio
        lv_obj_set_style_text_align(stat, LV_TEXT_ALIGN_RIGHT, 0);
        lv_obj_set_style_text_font(stat, &lv_font_montserrat_12, 0);
        lv_obj_set_style_pad_right(stat, 12, 0);
    }

    // Actualizar valores de los hijos y alternar visibilidad
    uint32_t current_children = lv_obj_get_child_count(list);
    for (uint32_t i = 0; i < current_children; i++) {
        lv_obj_t * card = lv_obj_get_child(list, i);
        if (i < needed_cnt) {
            lv_obj_remove_flag(card, LV_OBJ_FLAG_HIDDEN);
            const auto& item = orders[i];

            lv_obj_t * textContainer = lv_obj_get_child(card, 0);
            lv_obj_t * name = lv_obj_get_child(textContainer, 0);
            lv_obj_t * optionsLbl = lv_obj_get_child(textContainer, 1);
            
            lv_obj_t * stat = lv_obj_get_child(card, 1);

            // Nombre
            lv_label_set_text(name, item.name.c_str());

            // Modificadores / Opciones
            if (item.options.empty()) {
                lv_obj_add_flag(optionsLbl, LV_OBJ_FLAG_HIDDEN);
            } else {
                lv_obj_remove_flag(optionsLbl, LV_OBJ_FLAG_HIDDEN);
                String optStr = "";
                for (size_t o = 0; o < item.options.size(); o++) {
                    if (o > 0) optStr += ", ";
                    optStr += "+ " + item.options[o];
                }
                lv_label_set_text(optionsLbl, optStr.c_str());
            }

            // Estado
            lv_label_set_text(stat, item.status.c_str());

            lv_color_t statusColor = lv_color_hex(0xFFB800); // En espera (amarillo)

            if (item.status == "En Cocina" || item.status == "En cocina" || item.status == "Preparando") {
                statusColor = NeumorphicStyles::getPrimaryAccent();
            } else if (item.status == "Servido" || item.status == "Listo" || item.status == "Ready") {
                statusColor = lv_color_hex(0x00F5D4); // Cyan
            } else if (item.status == "Entregado" || item.status == "Delivered") {
                statusColor = lv_color_hex(0x2ECC71); // Verde
            }

            lv_obj_set_style_text_color(stat, statusColor, 0);
        } else {
            lv_obj_add_flag(card, LV_OBJ_FLAG_HIDDEN);
        }
    }
}


