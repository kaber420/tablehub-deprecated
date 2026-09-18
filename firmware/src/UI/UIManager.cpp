#include "UIManager.h"
#include "Views/DashboardView.h"
#include "Views/MyOrdersView.h"
#include "Views/RequestBillView.h"
#include "Views/MenuView.h"
#include "Views/CartView.h"
#include "Views/CallWaiterView.h"
#include "Components/HeaderBar.h"
#include "Themes/NeumorphicStyles.h"
#include <cstdio>

// --- Statics para el toast centralizado ---
lv_obj_t*   UIManager::toastObj   = nullptr;
lv_timer_t* UIManager::toastTimer = nullptr;

void UIManager::init() {
    Serial.println("[..] Creating DashboardView...");
    dashboardScreen = DashboardView::create();
    
    Serial.println("[..] Loading dashboard...");
    loadDashboard();
    Serial.println("[OK] UIManager init complete");
}

void UIManager::update() {
    HeaderBar::updateActiveCart();
    RequestBillView::checkPendingUpdate();
}

void UIManager::destroyTransient() {
    if (currentTransientScreen && lv_obj_is_valid(currentTransientScreen)) {
        lv_obj_delete(currentTransientScreen);
    }
    currentTransientScreen = nullptr;
}

void UIManager::loadDashboard() {
    destroyTransient();
    if (!dashboardScreen) {
        dashboardScreen = DashboardView::create();
    }
    DashboardView::refreshState();
    HeaderBar::setActiveHeader(DashboardView::getHeaderBar());
    lv_scr_load_anim(dashboardScreen, LV_SCR_LOAD_ANIM_NONE, 0, 0, false);
}

void UIManager::loadMyOrders() {
    destroyTransient();
    currentTransientScreen = MyOrdersView::create();
    HeaderBar::setActiveHeader(MyOrdersView::getHeaderBar());
    lv_scr_load_anim(currentTransientScreen, LV_SCR_LOAD_ANIM_NONE, 0, 0, false);
}

void UIManager::loadRequestBill() {
    destroyTransient();
    currentTransientScreen = RequestBillView::create();
    HeaderBar::setActiveHeader(RequestBillView::getHeaderBar());
    lv_scr_load_anim(currentTransientScreen, LV_SCR_LOAD_ANIM_NONE, 0, 0, false);
}

void UIManager::loadMenu() {
    destroyTransient();
    currentTransientScreen = MenuView::create();
    HeaderBar::setActiveHeader(MenuView::getHeaderBar());
    lv_scr_load_anim(currentTransientScreen, LV_SCR_LOAD_ANIM_NONE, 0, 0, false);
}

void UIManager::loadCart() {
    destroyTransient();
    currentTransientScreen = CartView::create();
    CartView::refresh();
    HeaderBar::setActiveHeader(CartView::getHeaderBar());
    lv_scr_load_anim(currentTransientScreen, LV_SCR_LOAD_ANIM_NONE, 0, 0, false);
}

void UIManager::loadCallWaiter() {
    destroyTransient();
    currentTransientScreen = CallWaiterView::create();
    HeaderBar::setActiveHeader(CallWaiterView::getHeaderBar());
    lv_scr_load_anim(currentTransientScreen, LV_SCR_LOAD_ANIM_NONE, 0, 0, false);
}

void UIManager::showToast(const char* message) {
    // Cancelar timer activo (si existe) antes de crear uno nuevo
    if (toastTimer) {
        lv_timer_delete(toastTimer);
        toastTimer = nullptr;
    }
    // Eliminar toast anterior de forma segura
    if (toastObj && lv_obj_is_valid(toastObj)) {
        lv_obj_delete(toastObj);
        toastObj = nullptr;
    }

    // Crear el nuevo toast sobre lv_layer_top() para que se vea
    // por encima de cualquier pantalla o modal activo
    toastObj = lv_obj_create(lv_layer_top());
    lv_obj_set_size(toastObj, 280, 44);
    NeumorphicStyles::applySunkenCard(toastObj, 22);
    lv_obj_set_style_bg_color(toastObj, lv_color_hex(0x0F172A), 0);
    lv_obj_set_style_border_color(toastObj, NeumorphicStyles::getPrimaryAccent(), 0);
    lv_obj_set_style_border_width(toastObj, 1, 0);
    lv_obj_align(toastObj, LV_ALIGN_BOTTOM_MID, 0, -20);
    lv_obj_remove_flag(toastObj, LV_OBJ_FLAG_CLICKABLE);

    lv_obj_t* label = lv_label_create(toastObj);
    lv_label_set_text(label, message);
    lv_obj_set_style_text_color(label, NeumorphicStyles::getPrimaryAccent(), 0);
    lv_obj_set_style_text_font(label, &lv_font_montserrat_12, 0);
    lv_obj_center(label);

    // Auto-destruir en 3.5s (un solo disparo)
    toastTimer = lv_timer_create(toast_timer_cb, 3500, NULL);
    lv_timer_set_repeat_count(toastTimer, 1);
}

void UIManager::toast_timer_cb(lv_timer_t* timer) {
    if (toastObj && lv_obj_is_valid(toastObj)) {
        lv_obj_delete(toastObj);
        toastObj = nullptr;
    }
    toastTimer = nullptr;
}
