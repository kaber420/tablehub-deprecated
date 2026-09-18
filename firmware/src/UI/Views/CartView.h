#pragma once
#include <lvgl.h>
#include "../Components/HeaderBar.h"

class CartView {
public:
    static lv_obj_t* create();
    static HeaderBar* getHeaderBar() { return headerBar; }
    static void refresh();

private:
    static HeaderBar* headerBar;
    static lv_obj_t* itemsContainer;
    static lv_obj_t* totalLabel;
    static lv_obj_t* emptyLabel;
    static lv_obj_t* sendOrderBtn;

    static void renderItems();
    static void updateTotals();
    static void increase_btn_cb(lv_event_t * e);
    static void decrease_btn_cb(lv_event_t * e);
    static void remove_btn_cb(lv_event_t * e);
    static void send_order_cb(lv_event_t * e);
};
