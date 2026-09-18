#include "SetupPinView.h"
#include "../../Network/ConfigManager.h"
#include "../Themes/NeumorphicStyles.h"
#include "../UIManager.h"

lv_obj_t* SetupPinView::container = nullptr;
lv_obj_t* SetupPinView::keyboard_container = nullptr;
lv_obj_t* SetupPinView::pin_label = nullptr;
std::string SetupPinView::current_pin = "";

static const char* keys[] = {
    "1", "2", "3",
    "4", "5", "6",
    "7", "8", "9",
    LV_SYMBOL_BACKSPACE, "0", LV_SYMBOL_OK
};

void SetupPinView::create(lv_obj_t* parent) {
    if (container != nullptr) return;
    current_pin = "";

    container = lv_obj_create(parent);
    lv_obj_set_size(container, LV_PCT(100), LV_PCT(100));
    NeumorphicStyles::applyFlatBg(container);
    NeumorphicStyles::disableScroll(container);

    lv_obj_t * title = lv_label_create(container);
    lv_label_set_text(title, "Configuración");
    lv_obj_align(title, LV_ALIGN_TOP_MID, 0, 20);
    lv_obj_set_style_text_color(title, NeumorphicStyles::getTextColor(), 0);
    lv_obj_set_style_text_font(title, &lv_font_montserrat_24, 0);

    lv_obj_t * sub = lv_label_create(container);
    lv_label_set_text(sub, "Ingresa el PIN de aprovisionamiento");
    lv_obj_align(sub, LV_ALIGN_TOP_MID, 0, 50);
    lv_obj_set_style_text_color(sub, NeumorphicStyles::getMutedTextColor(), 0);
    lv_obj_set_style_text_font(sub, &lv_font_montserrat_14, 0);

    // Contenedor neomórfico hundido para el PIN
    lv_obj_t* pin_display = lv_obj_create(container);
    lv_obj_set_size(pin_display, 240, 50);
    lv_obj_align(pin_display, LV_ALIGN_TOP_MID, 0, 80);
    NeumorphicStyles::applySunkenCard(pin_display, 12);
    NeumorphicStyles::disableScroll(pin_display);

    pin_label = lv_label_create(pin_display);
    lv_label_set_text(pin_label, "");
    lv_obj_center(pin_label);
    lv_obj_set_style_text_color(pin_label, NeumorphicStyles::getTextColor(), 0);
    lv_obj_set_style_text_font(pin_label, &lv_font_montserrat_24, 0);

    // Teclado tipo grid (sin btnmatrix)
    keyboard_container = lv_obj_create(container);
    lv_obj_set_size(keyboard_container, 260, 320);
    lv_obj_align(keyboard_container, LV_ALIGN_BOTTOM_MID, 0, -10);
    NeumorphicStyles::applyFlatBg(keyboard_container);
    NeumorphicStyles::disableScroll(keyboard_container);
    
    static int32_t col_dsc[] = {LV_GRID_FR(1), LV_GRID_FR(1), LV_GRID_FR(1), LV_GRID_TEMPLATE_LAST};
    static int32_t row_dsc[] = {LV_GRID_FR(1), LV_GRID_FR(1), LV_GRID_FR(1), LV_GRID_FR(1), LV_GRID_TEMPLATE_LAST};
    lv_obj_set_layout(keyboard_container, LV_LAYOUT_GRID);
    lv_obj_set_grid_dsc_array(keyboard_container, col_dsc, row_dsc);
    lv_obj_set_style_pad_all(keyboard_container, 0, 0);
    lv_obj_set_style_pad_row(keyboard_container, 8, 0);
    lv_obj_set_style_pad_column(keyboard_container, 8, 0);

    for(int i = 0; i < 12; i++) {
        uint8_t col = i % 3;
        uint8_t row = i / 3;
        
        lv_obj_t* btn = lv_button_create(keyboard_container);
        lv_obj_set_grid_cell(btn, LV_GRID_ALIGN_STRETCH, col, 1, LV_GRID_ALIGN_STRETCH, row, 1);
        
        if (i == 9) { // Borrar
            NeumorphicStyles::applyButton(btn, 12);
            lv_obj_set_style_bg_color(btn, lv_color_hex(0xFF2E93), 0); // Rosa tenue
        } else if (i == 11) { // OK
            NeumorphicStyles::applyButton(btn, 12);
            lv_obj_set_style_bg_color(btn, NeumorphicStyles::getPrimaryAccent(), 0); 
        } else {
            NeumorphicStyles::applyRaisedCard(btn, 12);
        }

        lv_obj_t* lbl = lv_label_create(btn);
        lv_label_set_text(lbl, keys[i]);
        lv_obj_set_style_text_color(lbl, NeumorphicStyles::getTextColor(), 0);
        lv_obj_set_style_text_font(lbl, &lv_font_montserrat_24, 0);
        lv_obj_center(lbl);

        // Guardamos el string en userdata para el callback
        lv_obj_add_event_cb(btn, pin_btn_event_cb, LV_EVENT_CLICKED, (void*)keys[i]);
    }
}

void SetupPinView::close() {
    if (container != nullptr && lv_obj_is_valid(container)) {
        lv_obj_delete_async(container);
        container = nullptr;
        keyboard_container = nullptr;
        pin_label = nullptr;
    }
}

void SetupPinView::pin_btn_event_cb(lv_event_t * e) {
    const char* txt = (const char*)lv_event_get_user_data(e);
    if(txt == NULL) return;

    if(strcmp(txt, LV_SYMBOL_BACKSPACE) == 0) {
        if(!current_pin.empty()) {
            current_pin.pop_back();
        }
    } else if(strcmp(txt, LV_SYMBOL_OK) == 0) {
        if(!current_pin.empty()) {
            attempt_provisioning(current_pin);
        }
    } else {
        if(current_pin.length() < 8) { // Max 8 digits
            current_pin += txt;
        }
    }

    std::string display_text = "";
    for(size_t i = 0; i < current_pin.length(); i++) {
        display_text += "* ";
    }
    
    if(pin_label) {
        lv_label_set_text(pin_label, display_text.c_str());
    }
}

struct AsyncProvData {
    std::string pin;
};

void SetupPinView::attempt_provisioning(const std::string& pin) {
    if(keyboard_container) {
        lv_obj_add_state(keyboard_container, LV_STATE_DISABLED);
    }
    
#ifdef ARDUINO
    AsyncProvData* data = new AsyncProvData{pin};
    xTaskCreatePinnedToCore([](void* param) {
        AsyncProvData* d = (AsyncProvData*)param;
        String err;
        bool success = ConfigManager::getInstance().attemptProvisioning(d->pin, err);
        
        struct ResultData { bool ok; String e; };
        ResultData* res = new ResultData{success, err};
        
        // Regresar a hilo LVGL
        lv_async_call([](void* rData) {
            ResultData* r = (ResultData*)rData;
            if (r->ok) {
                show_success_modal();
            } else {
                current_pin = "";
                if(pin_label) lv_label_set_text(pin_label, "");
                if(keyboard_container) lv_obj_clear_state(keyboard_container, LV_STATE_DISABLED);
                show_error_modal(r->e.c_str());
            }
            delete r;
        }, res);
        
        delete d;
        vTaskDelete(NULL);
    }, "ProvTask", 8192, data, 1, NULL, 1);
#else
    // Fallback para emulador / PC
    std::string err;
    if(ConfigManager::getInstance().attemptProvisioning(pin, err)) {
        show_success_modal();
    } else {
        current_pin = "";
        if(pin_label) lv_label_set_text(pin_label, "");
        if(keyboard_container) lv_obj_clear_state(keyboard_container, LV_STATE_DISABLED);
        show_error_modal(err.c_str());
    }
#endif
}

void SetupPinView::show_error_modal(const char* error_text) {
    lv_obj_t* mask = lv_obj_create(lv_screen_active());
    lv_obj_add_flag(mask, LV_OBJ_FLAG_IGNORE_LAYOUT);
    lv_obj_set_size(mask, 320, 480);
    lv_obj_set_style_bg_color(mask, lv_color_hex(0x000000), 0);
    lv_obj_set_style_bg_opa(mask, LV_OPA_80, 0);
    lv_obj_set_style_border_width(mask, 0, 0);

    lv_obj_t* modal = lv_obj_create(mask);
    lv_obj_set_width(modal, 260);
    lv_obj_set_height(modal, LV_SIZE_CONTENT);
    NeumorphicStyles::applyRaisedCard(modal, 20);
    lv_obj_center(modal);
    lv_obj_set_flex_flow(modal, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_flex_align(modal, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_pad_all(modal, 20, 0);

    lv_obj_t* title = lv_label_create(modal);
    lv_label_set_text(title, "Error de Sistema");
    lv_obj_set_style_text_color(title, lv_color_hex(0xFF2E93), 0);
    lv_obj_set_style_text_font(title, &lv_font_montserrat_16, 0);

    lv_obj_t* msg = lv_label_create(modal);
    lv_label_set_text(msg, error_text);
    lv_label_set_long_mode(msg, LV_LABEL_LONG_WRAP);
    lv_obj_set_width(msg, 220);
    lv_obj_set_style_text_align(msg, LV_TEXT_ALIGN_CENTER, 0);
    lv_obj_set_style_text_color(msg, NeumorphicStyles::getMutedTextColor(), 0);
    lv_obj_set_style_margin_top(msg, 10, 0);

    lv_obj_t* closeBtn = lv_button_create(modal);
    lv_obj_set_size(closeBtn, 200, 40);
    NeumorphicStyles::applyButton(closeBtn, 12);
    lv_obj_set_style_margin_top(closeBtn, 20, 0);

    lv_obj_t* lbl = lv_label_create(closeBtn);
    lv_label_set_text(lbl, "Reintentar");
    lv_obj_center(lbl);

    lv_obj_add_event_cb(closeBtn, [](lv_event_t* e) {
        lv_obj_t* m = (lv_obj_t*)lv_event_get_user_data(e);
        if (lv_obj_is_valid(m)) lv_obj_delete_async(m);
    }, LV_EVENT_CLICKED, mask);
}

void SetupPinView::show_success_modal() {
    lv_obj_t* mask = lv_obj_create(lv_screen_active());
    lv_obj_add_flag(mask, LV_OBJ_FLAG_IGNORE_LAYOUT);
    lv_obj_set_size(mask, 320, 480);
    lv_obj_set_style_bg_color(mask, lv_color_hex(0x000000), 0);
    lv_obj_set_style_bg_opa(mask, LV_OPA_80, 0);
    lv_obj_set_style_border_width(mask, 0, 0);

    lv_obj_t* modal = lv_obj_create(mask);
    lv_obj_set_width(modal, 260);
    lv_obj_set_height(modal, LV_SIZE_CONTENT);
    NeumorphicStyles::applyRaisedCard(modal, 20);
    lv_obj_center(modal);
    lv_obj_set_flex_flow(modal, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_flex_align(modal, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_pad_all(modal, 20, 0);

    lv_obj_t* title = lv_label_create(modal);
    lv_label_set_text(title, "¡Éxito!");
    lv_obj_set_style_text_color(title, NeumorphicStyles::getPrimaryAccent(), 0);
    lv_obj_set_style_text_font(title, &lv_font_montserrat_24, 0);

    lv_obj_t* msg = lv_label_create(modal);
    lv_label_set_text(msg, "Mesa aprovisionada.\nReiniciando...");
    lv_obj_set_style_text_align(msg, LV_TEXT_ALIGN_CENTER, 0);
    lv_obj_set_style_margin_top(msg, 10, 0);

#ifdef ARDUINO
    // Usamos el hilo secundario para reiniciar sin bloquear la UI inmediatamente
    xTaskCreatePinnedToCore([](void* p) {
        delay(2000);
        ESP.restart();
    }, "Reboot", 2048, NULL, 1, NULL, 1);
#else
    exit(0);
#endif
}
