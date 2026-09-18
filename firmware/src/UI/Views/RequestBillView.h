#pragma once
#include <lvgl.h>
#include <vector>
#include <string>
#include "../Components/HeaderBar.h"

struct BillItem {
    std::string name;
    float price;
    int qty;
};

class RequestBillView {
public:
    typedef void (*BillCallback)(const char* paymentMethod, float subtotal, int tipPercent, float total);

    static lv_obj_t* create();
    static HeaderBar* getHeaderBar() { return headerBar; }
    static void setBillData(float subtotal, float tax, const std::vector<BillItem>& items);
    static void setBillCallback(BillCallback cb);
    static void checkPendingUpdate();

private:
    static HeaderBar* headerBar;
    static BillCallback billCb;
    static float currentSubtotal;
    static float currentTax;
    static int selectedTipPercent;
    static std::string selectedPaymentMethod;

    static lv_obj_t* loadingLabel;
    static lv_obj_t* subtotalLabel;
    static lv_obj_t* taxLabel;
    static lv_obj_t* tipLabel;
    static lv_obj_t* totalLabel;
    static lv_obj_t* itemsContainer;

    static lv_obj_t* tipBtn10;
    static lv_obj_t* tipBtn15;
    static lv_obj_t* tipBtn20;
    static lv_obj_t* tipBtn0;

    static lv_obj_t* qrOverlay;

    static void updateCalculations();
    static void tip_btn_cb(lv_event_t* e);
    static void payment_btn_cb(lv_event_t* e);
    static void showQrModal();
    static void closeQrModal(lv_event_t* e);
};
