#pragma once
#include <lvgl.h>

class SmartCallModal {
public:
    typedef void (*SmartCallCallback)(int reasonId, const char* reasonText);

    static void show(SmartCallCallback cb = nullptr);
    static void close();

private:
    static lv_obj_t* overlay;
    static SmartCallCallback callback;

    static void option_btn_cb(lv_event_t* e);
    static void close_btn_cb(lv_event_t* e);
};
