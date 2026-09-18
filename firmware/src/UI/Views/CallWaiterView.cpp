#include "CallWaiterView.h"
#include "../Themes/NeumorphicStyles.h"
#include "../UIManager.h"
#include <Arduino.h>
#include <cstdio>

HeaderBar* CallWaiterView::headerBar = nullptr;
CallWaiterView::SmartCallCallback CallWaiterView::callback = nullptr;

struct CallWaitOption {
    int id;
    const char* icon;
    const char* title;
    const char* subtitle;
    uint32_t iconColorHex;
};

static const CallWaitOption options[] = {
    {1, LV_SYMBOL_REFRESH, "Refill / Bebida", "Pedir otra ronda o bebida", 0x00F5D4},
    {2, LV_SYMBOL_SETTINGS, "Cubiertos / Hielo", "Cubiertos extra, servilletas o hielo", 0xFFB800},
    {3, LV_SYMBOL_CLOSE, "Limpieza de Mesa", "Retirar platos o limpiar derrame", 0x9D4EDD},
    {4, LV_SYMBOL_BELL, "Asistencia General", "Atención del mesero a la mesa", 0xFF2E93}
};

void CallWaiterView::setCallCallback(SmartCallCallback cb) {
    callback = cb;
}

lv_obj_t* CallWaiterView::create() {
    Serial.println("[..] CallWaiterView::create starting");
    lv_obj_t* screen = lv_obj_create(NULL);
    NeumorphicStyles::applyFlatBg(screen);
    NeumorphicStyles::disableScroll(screen);

    lv_obj_set_flex_flow(screen, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_style_pad_all(screen, 8, 0);
    lv_obj_set_style_pad_row(screen, 8, 0);

    Serial.println("[..] CallWaiterView HeaderBar creating");
    headerBar = HeaderBar::create(screen, "Llamar Mesero", true, false);

    Serial.println("[..] CallWaiterView content container creating");
    lv_obj_t* content = lv_obj_create(screen);
    lv_obj_set_width(content, lv_pct(100));
    lv_obj_set_flex_grow(content, 1);
    lv_obj_set_style_bg_opa(content, LV_OPA_TRANSP, 0);
    lv_obj_set_style_border_width(content, 0, 0);
    lv_obj_set_style_pad_all(content, 16, 0);
    lv_obj_set_flex_flow(content, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_flex_align(content, LV_FLEX_ALIGN_START, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_pad_row(content, 14, 0);

    Serial.println("[..] CallWaiterView subtitle creating");
    lv_obj_t* sub = lv_label_create(content);
    lv_label_set_text(sub, "¿En qué te podemos ayudar?");
    lv_obj_set_style_text_font(sub, &lv_font_montserrat_14, 0);
    lv_obj_set_style_text_color(sub, NeumorphicStyles::getMutedTextColor(), 0);

    for (int i = 0; i < 4; i++) {
        Serial.printf("[..] CallWaiterView card %d starting\n", i); Serial.flush();
        lv_obj_t* card = lv_button_create(content);
        lv_obj_set_size(card, 280, 70);
        NeumorphicStyles::applyRaisedCard(card, 16);
        lv_obj_set_flex_flow(card, LV_FLEX_FLOW_ROW);
        lv_obj_set_flex_align(card, LV_FLEX_ALIGN_START, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
        lv_obj_set_style_pad_all(card, 12, 0);
        lv_obj_set_style_pad_column(card, 14, 0);

        Serial.printf("[..] CallWaiterView card %d iconCircle\n", i); Serial.flush();
        lv_obj_t* iconCircle = lv_obj_create(card);
        lv_obj_set_size(iconCircle, 44, 44);
        NeumorphicStyles::applySunkenCard(iconCircle, 22);
        NeumorphicStyles::disableScroll(iconCircle);
        lv_obj_remove_flag(iconCircle, LV_OBJ_FLAG_CLICKABLE);

        lv_obj_t* iconLbl = lv_label_create(iconCircle);
        lv_label_set_text(iconLbl, options[i].icon);
        lv_obj_set_style_text_color(iconLbl, lv_color_hex(options[i].iconColorHex), 0);
        lv_obj_set_style_text_font(iconLbl, &lv_font_montserrat_16, 0);
        lv_obj_center(iconLbl);

        Serial.printf("[..] CallWaiterView card %d textContainer\n", i); Serial.flush();
        lv_obj_t* textContainer = lv_obj_create(card);
        lv_obj_set_size(textContainer, 180, LV_SIZE_CONTENT);
        lv_obj_set_style_bg_opa(textContainer, LV_OPA_TRANSP, 0);
        lv_obj_set_style_border_width(textContainer, 0, 0);
        lv_obj_set_style_pad_all(textContainer, 0, 0);
        lv_obj_set_flex_flow(textContainer, LV_FLEX_FLOW_COLUMN);
        lv_obj_remove_flag(textContainer, LV_OBJ_FLAG_CLICKABLE);

        Serial.printf("[..] CallWaiterView card %d titleLbl\n", i); Serial.flush();
        lv_obj_t* titleLbl = lv_label_create(textContainer);
        lv_label_set_text(titleLbl, options[i].title);
        lv_obj_set_style_text_color(titleLbl, NeumorphicStyles::getTextColor(), 0);
        lv_obj_set_style_text_font(titleLbl, &lv_font_montserrat_14, 0);
        lv_label_set_long_mode(titleLbl, LV_LABEL_LONG_WRAP);
        lv_obj_set_width(titleLbl, lv_pct(100));

        Serial.printf("[..] CallWaiterView card %d descLbl\n", i); Serial.flush();
        lv_obj_t* descLbl = lv_label_create(textContainer);
        lv_label_set_text(descLbl, options[i].subtitle);
        lv_obj_set_style_text_color(descLbl, NeumorphicStyles::getMutedTextColor(), 0);
        lv_obj_set_style_text_font(descLbl, &lv_font_montserrat_12, 0);
        lv_label_set_long_mode(descLbl, LV_LABEL_LONG_WRAP);
        lv_obj_set_width(descLbl, lv_pct(100));

        intptr_t optIndex = i;
        lv_obj_add_event_cb(card, option_btn_cb, LV_EVENT_CLICKED, (void*)optIndex);
        Serial.printf("[..] CallWaiterView card %d done\n", i); Serial.flush();
    }

    Serial.println("[OK] CallWaiterView::create finished"); Serial.flush();
    return screen;
}

void CallWaiterView::option_btn_cb(lv_event_t* e) {
    UIManager::getInstance().resetInactivityTimer();
    intptr_t index = (intptr_t)lv_event_get_user_data(e);
    if (index >= 0 && index < 4) {
        if (callback) {
            callback(options[index].id, options[index].title);
        }
        char msg[64];
        snprintf(msg, sizeof(msg), "Solicitud enviada: %s", options[index].title);
        UIManager::showToast(msg);

        // Regresar al dashboard después de seleccionar
        UIManager::getInstance().loadDashboard();
    }
}

