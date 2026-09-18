#pragma once
#include <lvgl.h>
#include <vector>
#include "../../Network/MQTTService.h"
#include "../Components/HeaderBar.h"

class MyOrdersView {
public:
    static lv_obj_t* create();
    static void updateOrders(const std::vector<OrderItem>& orders);
    static HeaderBar* getHeaderBar() { return headerBar; }
private:
    static lv_obj_t* list;
    static HeaderBar* headerBar;
    static lv_obj_t* progressCard;
    static lv_obj_t* progressLabel;
    static lv_obj_t* progressBar;
    static void on_list_delete(lv_event_t * e);
};
