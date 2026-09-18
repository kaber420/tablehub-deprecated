#include "DashboardView.h"
#include "../UIManager.h"
#include "../Themes/NeumorphicStyles.h"
#include "../Components/SmartCallModal.h"
#include "../../Core/CartManager.h"
#include <cstdio>

HeaderBar* DashboardView::headerBar = nullptr;
DashboardView::CommandCallback DashboardView::commandCb = nullptr;
lv_obj_t* DashboardView::ordersBtnLabel = nullptr;
lv_obj_t* DashboardView::ordersBtnIcon = nullptr;

void DashboardView::btn_event_cb(lv_event_t * e) {
    lv_event_code_t code = lv_event_get_code(e);
    lv_obj_t * btn = (lv_obj_t *)lv_event_get_target(e);
    
    if (code == LV_EVENT_CLICKED) {
        int id = (int)(intptr_t)lv_obj_get_user_data(btn);
        if(id == 1) {
            UIManager::getInstance().loadMyOrders();
        } else if(id == 2) {
            UIManager::getInstance().loadCallWaiter();
        } else if(id == 3) {
            if (commandCb) commandCb(3);
            UIManager::getInstance().loadRequestBill();
        } else if(id == 4) {
            UIManager::getInstance().loadMenu();
        }
    }
}

void DashboardView::refreshState() {
    if (!ordersBtnLabel) return;
    lv_label_set_text(ordersBtnLabel, "Mis Pedidos");
    lv_obj_set_style_text_color(ordersBtnLabel, NeumorphicStyles::getTextColor(), 0);
}

lv_obj_t* DashboardView::create() {
    lv_obj_t* screen = lv_obj_create(NULL);
    NeumorphicStyles::applyFlatBg(screen);
    NeumorphicStyles::disableScroll(screen);

    lv_obj_set_flex_flow(screen, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_style_pad_all(screen, 12, 0);
    lv_obj_set_style_pad_row(screen, 10, 0);

    // --- Header Neomórfico (Status Bar Flotante Reutilizable) ---
    headerBar = HeaderBar::create(screen, "", false, true);

    // --- Dashboard Grid Neomórfico ---
    lv_obj_t * grid = lv_obj_create(screen);
    lv_obj_set_width(grid, lv_pct(100));
    lv_obj_set_flex_grow(grid, 1);
    NeumorphicStyles::disableScroll(grid);
    
    lv_obj_set_style_bg_opa(grid, 0, 0);
    lv_obj_set_style_border_width(grid, 0, 0);
    lv_obj_set_style_pad_all(grid, 2, 0);
    
    static int32_t col_dsc[] = {LV_GRID_FR(1), LV_GRID_FR(1), LV_GRID_TEMPLATE_LAST};
    static int32_t row_dsc[] = {LV_GRID_FR(1), LV_GRID_FR(1), LV_GRID_TEMPLATE_LAST};
    lv_obj_set_layout(grid, LV_LAYOUT_GRID);
    lv_obj_set_grid_dsc_array(grid, col_dsc, row_dsc);
    lv_obj_set_style_pad_column(grid, 12, 0);
    lv_obj_set_style_pad_row(grid, 12, 0);

    const char* titles[] = {"Mis Pedidos", "Llamar Mesero", "Pedir Cuenta", "Menu Digital"};
    const char* icons[] = {LV_SYMBOL_LIST, LV_SYMBOL_BELL, LV_SYMBOL_FILE, LV_SYMBOL_DIRECTORY};
    lv_color_t iconColors[] = {
        NeumorphicStyles::getPrimaryAccent(),
        lv_color_hex(0xFF2E93),
        lv_color_hex(0xFFB800),
        NeumorphicStyles::getSecondaryAccent()
    };
    int ids[] = {1, 2, 3, 4};

    for(int i=0; i<4; i++) {
        uint8_t col = i % 2;
        uint8_t row = i / 2;
        
        lv_obj_t * btn = lv_button_create(grid);
        lv_obj_set_grid_cell(btn, LV_GRID_ALIGN_STRETCH, col, 1, LV_GRID_ALIGN_STRETCH, row, 1);
        NeumorphicStyles::applyButton(btn, 16);
        NeumorphicStyles::disableScroll(btn);

        lv_obj_set_user_data(btn, (void*)(intptr_t)ids[i]);
        lv_obj_add_event_cb(btn, btn_event_cb, LV_EVENT_CLICKED, NULL);

        lv_obj_set_flex_flow(btn, LV_FLEX_FLOW_COLUMN);
        lv_obj_set_flex_align(btn, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
        lv_obj_set_style_pad_all(btn, 8, 0);

        // Contenedor Neomórfico Cóncavo para el Icono
        lv_obj_t * iconContainer = lv_obj_create(btn);
        lv_obj_set_size(iconContainer, 50, 50);
        NeumorphicStyles::applySunkenCard(iconContainer, 25);
        NeumorphicStyles::disableScroll(iconContainer);
        lv_obj_set_style_pad_all(iconContainer, 0, 0);
        lv_obj_remove_flag(iconContainer, LV_OBJ_FLAG_CLICKABLE);

        lv_obj_t * icon = lv_label_create(iconContainer);
        lv_label_set_text(icon, icons[i]);
        lv_obj_set_style_text_color(icon, iconColors[i], 0);
        lv_obj_set_style_text_font(icon, &lv_font_montserrat_24, 0);
        lv_obj_center(icon);

        // Etiqueta de Texto Moderna
        lv_obj_t * label = lv_label_create(btn);
        lv_label_set_text(label, titles[i]);
        lv_obj_set_style_text_color(label, NeumorphicStyles::getTextColor(), 0);
        lv_obj_set_style_text_font(label, &lv_font_montserrat_14, 0);
        lv_obj_set_style_margin_top(label, 6, 0);

        if (ids[i] == 1) {
            ordersBtnLabel = label;
            ordersBtnIcon = icon;
        }
    }

    refreshState();
    return screen;
}





