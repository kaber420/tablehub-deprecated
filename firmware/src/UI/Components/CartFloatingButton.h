#pragma once
#include <lvgl.h>

class CartFloatingButton {
public:
    static void init();
    static void update();
    static void setVisible(bool visible);

private:
    static lv_obj_t* container;
    static lv_obj_t* badgeLbl;
    static lv_obj_t* cartBtn;
    static void btn_cb(lv_event_t* e);
};
