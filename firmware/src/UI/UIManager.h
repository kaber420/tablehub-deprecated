#pragma once
#include <lvgl.h>

class UIManager {
public:
    static UIManager& getInstance() {
        static UIManager instance;
        return instance;
    }

    void init();
    void update();
    
    void loadDashboard();
    void loadMyOrders();
    void loadRequestBill();
    void loadMenu();
    void loadCart();
    void loadCallWaiter();

    void resetInactivityTimer() {}
    static void showToast(const char* message);

private:
    UIManager() {}
    static void toast_timer_cb(lv_timer_t* timer);
    void destroyTransient();

    static lv_obj_t* toastObj;
    static lv_timer_t* toastTimer;
    
    // Pantallas permanentes (cached)
    lv_obj_t* dashboardScreen   = nullptr;

    // Pantalla transitoria activa (se destruye al navegar)
    lv_obj_t* currentTransientScreen = nullptr;
};
