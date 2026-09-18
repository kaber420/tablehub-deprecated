#include "SmartCallModal.h"
#include "../Themes/NeumorphicStyles.h"
#include "../UIManager.h"
#include <cstdio>

lv_obj_t* SmartCallModal::overlay = nullptr;
SmartCallModal::SmartCallCallback SmartCallModal::callback = nullptr;

struct CallOption {
    int id;
    const char* icon;
    const char* title;
    lv_color_t iconColor;
};

static const CallOption options[] = {
    {1, LV_SYMBOL_REFRESH, "Refill / Bebida", lv_color_hex(0x00F5D4)},
    {2, LV_SYMBOL_SETTINGS, "Cubiertos / Hielo", lv_color_hex(0xFFB800)},
    {3, LV_SYMBOL_CLOSE, "Limpieza Mesa", lv_color_hex(0x9D4EDD)},
    {4, LV_SYMBOL_BELL, "Duda / Asistencia", lv_color_hex(0xFF2E93)}
};

void SmartCallModal::show(SmartCallCallback cb) {
    // Fondo / Máscara perfecto sobre la pantalla activa
    lv_obj_t* mask = lv_obj_create(lv_screen_active());
    lv_obj_add_flag(mask, LV_OBJ_FLAG_IGNORE_LAYOUT);
    lv_obj_set_size(mask, 320, 480);
    lv_obj_set_pos(mask, 0, 0);
    lv_obj_set_style_pad_all(mask, 0, 0);
    lv_obj_set_style_border_width(mask, 0, 0);
    lv_obj_set_style_radius(mask, 0, 0);
    lv_obj_set_style_bg_color(mask, lv_color_hex(0x000000), 0);
    lv_obj_set_style_bg_opa(mask, LV_OPA_70, 0);
    NeumorphicStyles::disableScroll(mask);

    // Tarjeta del Modal
    lv_obj_t* modal = lv_obj_create(mask);
    lv_obj_set_width(modal, 280);
    lv_obj_set_height(modal, LV_SIZE_CONTENT);
    NeumorphicStyles::applyRaisedCard(modal, 20);
    lv_obj_center(modal);
    lv_obj_set_style_pad_all(modal, 16, 0);
    lv_obj_set_flex_flow(modal, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_flex_align(modal, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_pad_row(modal, 10, 0);

    // Título
    lv_obj_t* title = lv_label_create(modal);
    lv_label_set_text(title, "Llamar al Mesero");
    lv_obj_set_style_text_font(title, &lv_font_montserrat_16, 0);
    lv_obj_set_style_text_color(title, NeumorphicStyles::getTextColor(), 0);

    // Subtítulo
    lv_obj_t* sub = lv_label_create(modal);
    lv_label_set_text(sub, "¿En qué te podemos ayudar?");
    lv_obj_set_style_text_font(sub, &lv_font_montserrat_12, 0);
    lv_obj_set_style_text_color(sub, NeumorphicStyles::getMutedTextColor(), 0);

    struct ModalData {
        lv_obj_t* mask;
        int optId;
        const char* optTitle;
        SmartCallCallback cb;
    };

    for (int i = 0; i < 4; i++) {
        lv_obj_t* btn = lv_button_create(modal);
        lv_obj_set_size(btn, 240, 38);
        NeumorphicStyles::applyButton(btn, 12);

        lv_obj_t* lbl = lv_label_create(btn);
        char btnText[64];
        snprintf(btnText, sizeof(btnText), "%s  %s", options[i].icon, options[i].title);
        lv_label_set_text(lbl, btnText);
        lv_obj_set_style_text_color(lbl, options[i].iconColor, 0);
        lv_obj_set_style_text_font(lbl, &lv_font_montserrat_14, 0);
        lv_obj_center(lbl);

        ModalData* data = new ModalData{mask, options[i].id, options[i].title, cb};

        lv_obj_add_event_cb(btn, [](lv_event_t* e) {
            ModalData* d = (ModalData*)lv_event_get_user_data(e);
            if (d->cb && d->optId > 0) {
                d->cb(d->optId, d->optTitle);
            }
            char msg[64];
            snprintf(msg, sizeof(msg), "Mesero notificado: %s", d->optTitle);
            UIManager::showToast(msg);
            UIManager::getInstance().resetInactivityTimer();

            if (lv_obj_is_valid(d->mask)) {
                lv_obj_delete_async(d->mask);
            }
            delete d;
        }, LV_EVENT_CLICKED, data);
    }

    // Botón Cerrar
    lv_obj_t* closeBtn = lv_button_create(modal);
    lv_obj_set_size(closeBtn, 240, 36);
    NeumorphicStyles::applyButton(closeBtn, 12);

    lv_obj_t* closeLbl = lv_label_create(closeBtn);
    lv_label_set_text(closeLbl, "Cerrar");
    lv_obj_set_style_text_color(closeLbl, NeumorphicStyles::getMutedTextColor(), 0);
    lv_obj_set_style_text_font(closeLbl, &lv_font_montserrat_14, 0);
    lv_obj_center(closeLbl);

    ModalData* closeData = new ModalData{mask, 0, "", nullptr};
    lv_obj_add_event_cb(closeBtn, [](lv_event_t* e) {
        ModalData* d = (ModalData*)lv_event_get_user_data(e);
        UIManager::getInstance().resetInactivityTimer();
        if (lv_obj_is_valid(d->mask)) {
            lv_obj_delete_async(d->mask);
        }
        delete d;
    }, LV_EVENT_CLICKED, closeData);
}

void SmartCallModal::close() {
    if (overlay && lv_obj_is_valid(overlay)) {
        lv_obj_delete_async(overlay);
        overlay = nullptr;
    }
}

void SmartCallModal::option_btn_cb(lv_event_t* e) {}
void SmartCallModal::close_btn_cb(lv_event_t* e) {}

