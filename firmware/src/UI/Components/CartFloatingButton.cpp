#include "CartFloatingButton.h"
#include "../Themes/NeumorphicStyles.h"
#include "../UIManager.h"
#include "../../Core/CartManager.h"
#include <cstdio>

lv_obj_t* CartFloatingButton::container = nullptr;
lv_obj_t* CartFloatingButton::badgeLbl = nullptr;
lv_obj_t* CartFloatingButton::cartBtn = nullptr;

void CartFloatingButton::init() {
    lv_obj_t* top = lv_layer_top();
    if (!top) return;
    if (container && lv_obj_is_valid(container)) {
        return;
    }

    // Floating container on lv_layer_top()
    container = lv_obj_create(top);
    lv_obj_set_size(container, 56, 56);
    lv_obj_align(container, LV_ALIGN_BOTTOM_RIGHT, -16, -16);
    lv_obj_set_style_bg_opa(container, LV_OPA_TRANSP, 0);
    lv_obj_set_style_border_width(container, 0, 0);
    lv_obj_set_style_pad_all(container, 0, 0);
    NeumorphicStyles::disableScroll(container);

    // Botón principal neumórfico circular
    cartBtn = lv_button_create(container);
    lv_obj_set_size(cartBtn, 50, 50);
    lv_obj_align(cartBtn, LV_ALIGN_BOTTOM_LEFT, 0, 0);
    NeumorphicStyles::applyRaisedCard(cartBtn, 25);

    lv_obj_t* icon = lv_label_create(cartBtn);
    lv_label_set_text(icon, LV_SYMBOL_LIST);
    lv_obj_set_style_text_color(icon, NeumorphicStyles::getPrimaryAccent(), 0);
    lv_obj_set_style_text_font(icon, &lv_font_montserrat_24, 0);
    lv_obj_remove_flag(icon, LV_OBJ_FLAG_CLICKABLE);
    lv_obj_center(icon);

    // Badge contador en la esquina superior derecha
    lv_obj_t* badge = lv_obj_create(container);
    lv_obj_set_size(badge, 22, 22);
    lv_obj_align(badge, LV_ALIGN_TOP_RIGHT, 0, 0);
    NeumorphicStyles::applySunkenCard(badge, 11);
    lv_obj_set_style_bg_color(badge, NeumorphicStyles::getPrimaryAccent(), 0);
    lv_obj_set_style_border_width(badge, 0, 0);
    lv_obj_set_style_pad_all(badge, 0, 0);
    NeumorphicStyles::disableScroll(badge);
    lv_obj_remove_flag(badge, LV_OBJ_FLAG_CLICKABLE);

    badgeLbl = lv_label_create(badge);
    lv_label_set_text(badgeLbl, "0");
    lv_obj_set_style_text_color(badgeLbl, lv_color_hex(0x0F172A), 0);
    lv_obj_set_style_text_font(badgeLbl, &lv_font_montserrat_12, 0);
    lv_obj_remove_flag(badgeLbl, LV_OBJ_FLAG_CLICKABLE);
    lv_obj_center(badgeLbl);

    lv_obj_add_event_cb(cartBtn, btn_cb, LV_EVENT_CLICKED, NULL);

    update();
}

void CartFloatingButton::update() {
    if (!container || !lv_obj_is_valid(container)) return;

    int count = CartManager::getInstance().getItemCount();
    static int lastCount = -1;
    if (count == lastCount) return;
    lastCount = count;

    char buf[16];
    snprintf(buf, sizeof(buf), "%d", count);
    if (badgeLbl && lv_obj_is_valid(badgeLbl)) {
        lv_label_set_text(badgeLbl, buf);
    }

    if (count > 0) {
        lv_obj_set_style_opa(cartBtn, LV_OPA_COVER, 0);
    } else {
        lv_obj_set_style_opa(cartBtn, LV_OPA_70, 0);
    }
}

void CartFloatingButton::setVisible(bool visible) {
    if (!container || !lv_obj_is_valid(container)) return;
    if (visible) {
        lv_obj_remove_flag(container, LV_OBJ_FLAG_HIDDEN);
        update();
    } else {
        lv_obj_add_flag(container, LV_OBJ_FLAG_HIDDEN);
    }
}

void CartFloatingButton::btn_cb(lv_event_t* e) {
    UIManager::getInstance().resetInactivityTimer();
    UIManager::getInstance().loadCart();
}
