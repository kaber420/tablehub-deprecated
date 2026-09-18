#ifndef SETUP_PIN_VIEW_H
#define SETUP_PIN_VIEW_H

#include <lvgl.h>
#include <string>

class SetupPinView {
public:
    static void create(lv_obj_t* parent);
    static void close();

private:
    static void pin_btn_event_cb(lv_event_t * e);
    static void attempt_provisioning(const std::string& pin);
    
    // UI Helpers
    static void show_error_modal(const char* error_text);
    static void show_success_modal();

    static lv_obj_t* container;
    static lv_obj_t* keyboard_container;
    static lv_obj_t* pin_label;
    static std::string current_pin;
};

#endif
