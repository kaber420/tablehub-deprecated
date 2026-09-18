#include "CartView.h"
#include "../../Core/CartManager.h"
#include "../Themes/NeumorphicStyles.h"
#include "../UIManager.h"
#include <cstdio>

HeaderBar* CartView::headerBar = nullptr;
lv_obj_t* CartView::itemsContainer = nullptr;
lv_obj_t* CartView::totalLabel = nullptr;
lv_obj_t* CartView::emptyLabel = nullptr;
lv_obj_t* CartView::sendOrderBtn = nullptr;

lv_obj_t* CartView::create() {
    lv_obj_t* screen = lv_obj_create(NULL);
    NeumorphicStyles::applyFlatBg(screen);
    NeumorphicStyles::disableScroll(screen);

    lv_obj_set_flex_flow(screen, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_style_pad_all(screen, 8, 0);
    lv_obj_set_style_pad_row(screen, 8, 0);

    // Header Neomórfico con botón de volver al Dashboard
    headerBar = HeaderBar::create(screen, "Mi Carrito", true, false);

    // Contenedor de Items (Vertical Scroll)
    itemsContainer = lv_obj_create(screen);
    lv_obj_set_width(itemsContainer, lv_pct(100));
    lv_obj_set_flex_grow(itemsContainer, 1);
    lv_obj_set_style_bg_opa(itemsContainer, 0, 0);
    lv_obj_set_style_border_width(itemsContainer, 0, 0);
    lv_obj_set_style_pad_all(itemsContainer, 2, 0);
    lv_obj_set_style_pad_row(itemsContainer, 10, 0);
    lv_obj_set_flex_flow(itemsContainer, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_scroll_dir(itemsContainer, LV_DIR_VER);
    // Barra Inferior de Desglose y Enviar
    lv_obj_t* bottomCard = lv_obj_create(screen);
    lv_obj_set_width(bottomCard, lv_pct(100));
    lv_obj_set_height(bottomCard, LV_SIZE_CONTENT);
    NeumorphicStyles::applyRaisedCard(bottomCard, 16);
    NeumorphicStyles::disableScroll(bottomCard);
    lv_obj_set_style_pad_all(bottomCard, 10, 0);
    lv_obj_set_flex_flow(bottomCard, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_flex_align(bottomCard, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_pad_row(bottomCard, 8, 0);

    // Etiqueta de Total
    totalLabel = lv_label_create(bottomCard);
    lv_label_set_text(totalLabel, "Total: $0.00");
    lv_obj_set_style_text_color(totalLabel, NeumorphicStyles::getPrimaryAccent(), 0);
    lv_obj_set_style_text_font(totalLabel, &lv_font_montserrat_16, 0);

    // Botón Enviar Pedido
    sendOrderBtn = lv_button_create(bottomCard);
    lv_obj_set_size(sendOrderBtn, 260, 38);
    NeumorphicStyles::applyButton(sendOrderBtn, 12);
    lv_obj_add_event_cb(sendOrderBtn, send_order_cb, LV_EVENT_CLICKED, NULL);

    lv_obj_t* sendLbl = lv_label_create(sendOrderBtn);
    lv_label_set_text(sendLbl, "Enviar Pedido a Mesero");
    lv_obj_set_style_text_color(sendLbl, NeumorphicStyles::getPrimaryAccent(), 0);
    lv_obj_set_style_text_font(sendLbl, &lv_font_montserrat_14, 0);
    lv_obj_center(sendLbl);

    renderItems();
    return screen;
}

void CartView::refresh() {
    renderItems();
}

void CartView::renderItems() {
    lv_obj_clean(itemsContainer);

    const auto& cartItems = CartManager::getInstance().getItems();

    if (cartItems.empty()) {
        emptyLabel = lv_label_create(itemsContainer);
        lv_label_set_text(emptyLabel, "Tu carrito está vacío.\nExplora la carta y agrega tus platillos favoritos.");
        lv_obj_set_style_text_color(emptyLabel, NeumorphicStyles::getMutedTextColor(), 0);
        lv_obj_set_style_text_font(emptyLabel, &lv_font_montserrat_14, 0);
        lv_obj_set_style_text_align(emptyLabel, LV_TEXT_ALIGN_CENTER, 0);
        lv_obj_align(emptyLabel, LV_ALIGN_CENTER, 0, 40);

        if (sendOrderBtn) {
            lv_obj_add_state(sendOrderBtn, LV_STATE_DISABLED);
        }
        updateTotals();
        return;
    }

    if (sendOrderBtn) {
        lv_obj_clear_state(sendOrderBtn, LV_STATE_DISABLED);
    }

    int index = 0;
    for (const auto& item : cartItems) {
        lv_obj_t* card = lv_obj_create(itemsContainer);
        lv_obj_set_width(card, lv_pct(100));
        lv_obj_set_height(card, 76);
        NeumorphicStyles::applyRaisedCard(card, 14);
        NeumorphicStyles::disableScroll(card);
        lv_obj_set_style_pad_all(card, 6, 0);
        lv_obj_remove_flag(card, LV_OBJ_FLAG_CLICKABLE);
        lv_obj_add_flag(card, LV_OBJ_FLAG_SCROLL_CHAIN);

        lv_obj_set_flex_flow(card, LV_FLEX_FLOW_ROW);
        lv_obj_set_flex_align(card, LV_FLEX_ALIGN_START, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);

        // Icon Container
        lv_obj_t* imgContainer = lv_obj_create(card);
        lv_obj_set_size(imgContainer, 46, 46);
        NeumorphicStyles::applySunkenCard(imgContainer, 10);
        NeumorphicStyles::disableScroll(imgContainer);
        lv_obj_remove_flag(imgContainer, LV_OBJ_FLAG_CLICKABLE);

        lv_obj_t* icon = lv_label_create(imgContainer);
        lv_label_set_text(icon, item.icon);
        lv_obj_set_style_text_color(icon, NeumorphicStyles::getPrimaryAccent(), 0);
        lv_obj_set_style_text_font(icon, &lv_font_montserrat_24, 0);
        lv_obj_center(icon);

        // Info
        lv_obj_t* infoCol = lv_obj_create(card);
        lv_obj_set_flex_grow(infoCol, 1);
        lv_obj_set_height(infoCol, lv_pct(100));
        lv_obj_set_style_bg_opa(infoCol, 0, 0);
        lv_obj_set_style_border_width(infoCol, 0, 0);
        lv_obj_set_style_pad_hor(infoCol, 6, 0);
        NeumorphicStyles::disableScroll(infoCol);
        lv_obj_set_flex_flow(infoCol, LV_FLEX_FLOW_COLUMN);
        lv_obj_set_flex_align(infoCol, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_START, LV_FLEX_ALIGN_START);
        lv_obj_remove_flag(infoCol, LV_OBJ_FLAG_CLICKABLE);

        lv_obj_t* nameLbl = lv_label_create(infoCol);
        lv_label_set_text(nameLbl, item.name.c_str());
        lv_obj_set_style_text_color(nameLbl, NeumorphicStyles::getTextColor(), 0);
        lv_obj_set_style_text_font(nameLbl, &lv_font_montserrat_12, 0);
        lv_label_set_long_mode(nameLbl, LV_LABEL_LONG_DOT);
        lv_obj_set_width(nameLbl, lv_pct(100));

        char priceStr[32];
        snprintf(priceStr, sizeof(priceStr), "$%.2f", item.price * item.quantity);
        lv_obj_t* priceLbl = lv_label_create(infoCol);
        lv_label_set_text(priceLbl, priceStr);
        lv_obj_set_style_text_color(priceLbl, NeumorphicStyles::getPrimaryAccent(), 0);
        lv_obj_set_style_text_font(priceLbl, &lv_font_montserrat_12, 0);

        // Controles de Cantidad [-] [Cant] [+]
        lv_obj_t* ctrlRow = lv_obj_create(card);
        lv_obj_set_size(ctrlRow, 90, 36);
        lv_obj_set_style_bg_opa(ctrlRow, 0, 0);
        lv_obj_set_style_border_width(ctrlRow, 0, 0);
        lv_obj_set_style_pad_all(ctrlRow, 0, 0);
        NeumorphicStyles::disableScroll(ctrlRow);
        lv_obj_set_flex_flow(ctrlRow, LV_FLEX_FLOW_ROW);
        lv_obj_set_flex_align(ctrlRow, LV_FLEX_ALIGN_END, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);

        // Botón Minus
        lv_obj_t* decBtn = lv_button_create(ctrlRow);
        lv_obj_set_size(decBtn, 28, 28);
        NeumorphicStyles::applyButton(decBtn, 8);
        lv_obj_add_flag(decBtn, LV_OBJ_FLAG_SCROLL_CHAIN);
        lv_obj_set_user_data(decBtn, (void*)(intptr_t)index);
        lv_obj_add_event_cb(decBtn, decrease_btn_cb, LV_EVENT_CLICKED, NULL);

        lv_obj_t* decLbl = lv_label_create(decBtn);
        lv_label_set_text(decLbl, "-");
        lv_obj_set_style_text_color(decLbl, NeumorphicStyles::getTextColor(), 0);
        lv_obj_center(decLbl);

        // Label Cantidad
        char qtyStr[16];
        snprintf(qtyStr, sizeof(qtyStr), "%d", item.quantity);
        lv_obj_t* qtyLbl = lv_label_create(ctrlRow);
        lv_label_set_text(qtyLbl, qtyStr);
        lv_obj_set_style_text_color(qtyLbl, NeumorphicStyles::getTextColor(), 0);
        lv_obj_set_style_text_font(qtyLbl, &lv_font_montserrat_12, 0);
        lv_obj_set_style_pad_hor(qtyLbl, 4, 0);

        // Botón Plus
        lv_obj_t* incBtn = lv_button_create(ctrlRow);
        lv_obj_set_size(incBtn, 28, 28);
        NeumorphicStyles::applyButton(incBtn, 8);
        lv_obj_add_flag(incBtn, LV_OBJ_FLAG_SCROLL_CHAIN);
        lv_obj_set_user_data(incBtn, (void*)(intptr_t)index);
        lv_obj_add_event_cb(incBtn, increase_btn_cb, LV_EVENT_CLICKED, NULL);

        index++;

        lv_obj_t* incLbl = lv_label_create(incBtn);
        lv_label_set_text(incLbl, "+");
        lv_obj_set_style_text_color(incLbl, NeumorphicStyles::getPrimaryAccent(), 0);
        lv_obj_center(incLbl);
    }

    updateTotals();
}

void CartView::updateTotals() {
    float total = CartManager::getInstance().getTotalPrice();
    int count = CartManager::getInstance().getItemCount();

    char str[48];
    snprintf(str, sizeof(str), "Total (%d): $%.2f", count, total);
    if (totalLabel) {
        lv_label_set_text(totalLabel, str);
    }
}

void CartView::increase_btn_cb(lv_event_t* e) {
    lv_indev_t* indev = lv_indev_active();
    if (indev && lv_indev_get_scroll_obj(indev) != NULL) return;

    lv_obj_t* btn = (lv_obj_t*)lv_event_get_target(e);
    int index = (int)(intptr_t)lv_obj_get_user_data(btn);

    const auto& items = CartManager::getInstance().getItems();
    if (index >= 0 && index < items.size()) {
        const auto& item = items[index];
        MenuItem m{item.itemId, item.name, "", item.price, 0, "", item.icon};
        CartManager::getInstance().addItem(m);
    }

    renderItems();
    UIManager::getInstance().resetInactivityTimer();
}

void CartView::decrease_btn_cb(lv_event_t* e) {
    lv_indev_t* indev = lv_indev_active();
    if (indev && lv_indev_get_scroll_obj(indev) != NULL) return;

    lv_obj_t* btn = (lv_obj_t*)lv_event_get_target(e);
    int index = (int)(intptr_t)lv_obj_get_user_data(btn);

    const auto& items = CartManager::getInstance().getItems();
    if (index >= 0 && index < items.size()) {
        CartManager::getInstance().decreaseItem(items[index].itemId);
    }
    renderItems();
    UIManager::getInstance().resetInactivityTimer();
}

void CartView::send_order_cb(lv_event_t* e) {
    if (CartManager::getInstance().getItemCount() == 0) return;

    UIManager::showToast("Pedido enviado a confirmación");
    CartManager::getInstance().clearCart();
    UIManager::getInstance().loadDashboard();
}

