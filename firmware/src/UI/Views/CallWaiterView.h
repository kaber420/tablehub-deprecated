#pragma once
#include <lvgl.h>
#include "../Components/HeaderBar.h"

class CallWaiterView {
public:
    typedef void (*SmartCallCallback)(int reasonId, const char* reasonText);

    static lv_obj_t* create();
    static HeaderBar* getHeaderBar() { return headerBar; }
    static void setCallCallback(SmartCallCallback cb);

private:
    static HeaderBar* headerBar;
    static SmartCallCallback callback;

    static void option_btn_cb(lv_event_t* e);
};
