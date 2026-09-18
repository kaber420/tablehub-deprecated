#include "RequestBillView.h"
#include "../UIManager.h"
#include "../Themes/NeumorphicStyles.h"
#include "../Components/SmartCallModal.h"
#include <Arduino.h>
#include <cstdio>
#include <cstring>
#include "../../Network/MQTTService.h"

#include "../../Core/CartManager.h"

HeaderBar* RequestBillView::headerBar = nullptr;
RequestBillView::BillCallback RequestBillView::billCb = nullptr;
float RequestBillView::currentSubtotal = 0.0f;
float RequestBillView::currentTax = 0.0f;
int RequestBillView::selectedTipPercent = 15;
std::string RequestBillView::selectedPaymentMethod = "TARJETA";

static volatile float g_pendingSubtotal = -1.0f;
static volatile float g_pendingTax = -1.0f;

lv_obj_t* RequestBillView::loadingLabel = nullptr;
lv_obj_t* RequestBillView::subtotalLabel = nullptr;
lv_obj_t* RequestBillView::taxLabel = nullptr;
lv_obj_t* RequestBillView::tipLabel = nullptr;
lv_obj_t* RequestBillView::totalLabel = nullptr;
lv_obj_t* RequestBillView::itemsContainer = nullptr;

lv_obj_t* RequestBillView::tipBtn10 = nullptr;
lv_obj_t* RequestBillView::tipBtn15 = nullptr;
lv_obj_t* RequestBillView::tipBtn20 = nullptr;
lv_obj_t* RequestBillView::tipBtn0 = nullptr;

lv_obj_t* RequestBillView::qrOverlay = nullptr;

lv_obj_t* RequestBillView::create() {
    Serial.println("[..] RequestBillView::create starting"); Serial.flush();
    lv_obj_t* screen = lv_obj_create(NULL);
    NeumorphicStyles::applyFlatBg(screen);
    lv_obj_set_flex_flow(screen, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_style_pad_all(screen, 10, 0);
    lv_obj_set_style_pad_row(screen, 8, 0);

    // 1. HeaderBar con botón Atrás
    Serial.println("[..] RequestBillView headerBar creating"); Serial.flush();
    headerBar = HeaderBar::create(screen, "Pedir Cuenta", true, false);

    // Contenedor scrolleable para el contenido principal
    Serial.println("[..] RequestBillView scrollArea creating"); Serial.flush();
    lv_obj_t* scrollArea = lv_obj_create(screen);
    lv_obj_set_width(scrollArea, lv_pct(100));
    lv_obj_set_flex_grow(scrollArea, 1);
    lv_obj_set_style_bg_opa(scrollArea, 0, 0);
    lv_obj_set_style_border_width(scrollArea, 0, 0);
    lv_obj_set_style_pad_all(scrollArea, 0, 0);
    lv_obj_set_flex_flow(scrollArea, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_style_pad_row(scrollArea, 10, 0);

    // 2. Tarjeta Neomórfica de Resumen de Cuenta
    lv_obj_t* summaryCard = lv_obj_create(scrollArea);
    lv_obj_set_width(summaryCard, lv_pct(100));
    NeumorphicStyles::applySunkenCard(summaryCard, 16);
    lv_obj_set_flex_flow(summaryCard, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_style_pad_all(summaryCard, 10, 0);
    lv_obj_set_style_pad_row(summaryCard, 4, 0);

    lv_obj_t* cardTitle = lv_label_create(summaryCard);
    lv_label_set_text(cardTitle, "Resumen de Consumo");
    lv_obj_set_style_text_font(cardTitle, &lv_font_montserrat_14, 0);
    lv_obj_set_style_text_color(cardTitle, NeumorphicStyles::getPrimaryAccent(), 0);

    // Subtotal
    subtotalLabel = lv_label_create(summaryCard);
    lv_obj_set_style_text_font(subtotalLabel, &lv_font_montserrat_12, 0);
    lv_obj_set_style_text_color(subtotalLabel, NeumorphicStyles::getTextColor(), 0);

    // Impuestos (IVA)
    taxLabel = lv_label_create(summaryCard);
    lv_obj_set_style_text_font(taxLabel, &lv_font_montserrat_12, 0);
    lv_obj_set_style_text_color(taxLabel, NeumorphicStyles::getMutedTextColor(), 0);

    // Propina
    tipLabel = lv_label_create(summaryCard);
    lv_obj_set_style_text_font(tipLabel, &lv_font_montserrat_12, 0);
    lv_obj_set_style_text_color(tipLabel, NeumorphicStyles::getTextColor(), 0);

    // Línea separadora
    lv_obj_t* line = lv_obj_create(summaryCard);
    lv_obj_set_size(line, lv_pct(100), 2);
    lv_obj_set_style_bg_color(line, lv_color_hex(0x334155), 0);
    lv_obj_set_style_bg_opa(line, LV_OPA_COVER, 0);
    lv_obj_set_style_border_width(line, 0, 0);
    lv_obj_set_style_pad_all(line, 0, 0);
    NeumorphicStyles::disableScroll(line);

    // Total General
    totalLabel = lv_label_create(summaryCard);
    lv_obj_set_style_text_font(totalLabel, &lv_font_montserrat_16, 0);
    lv_obj_set_style_text_color(totalLabel, NeumorphicStyles::getPrimaryAccent(), 0);

    // Loading Label
    loadingLabel = lv_label_create(summaryCard);
    lv_label_set_text(loadingLabel, "Calculando cuenta...");
    lv_obj_set_style_text_color(loadingLabel, NeumorphicStyles::getMutedTextColor(), 0);
    lv_obj_set_style_text_font(loadingLabel, &lv_font_montserrat_12, 0);
    lv_obj_center(loadingLabel);

    lv_obj_add_flag(subtotalLabel, LV_OBJ_FLAG_HIDDEN);
    lv_obj_add_flag(taxLabel, LV_OBJ_FLAG_HIDDEN);
    lv_obj_add_flag(tipLabel, LV_OBJ_FLAG_HIDDEN);
    lv_obj_add_flag(totalLabel, LV_OBJ_FLAG_HIDDEN);

    // Calcular inmediatamente usando los productos locales del carrito
    float cartTotal = CartManager::getInstance().getTotalPrice();
    if (cartTotal > 0.0f) {
        float subtotal = cartTotal / 1.16f;
        float tax = cartTotal - subtotal;
        std::vector<BillItem> emptyItems;
        setBillData(subtotal, tax, emptyItems);
    }
    
    // Configurar callback de red seguro
    MQTTService::getInstance().setBillStateCallback([](float subtotal, float tax, float total) {
        g_pendingSubtotal = subtotal;
        g_pendingTax = tax;
    });

    // 3. Selector de Propina (10%, 15%, 20%, 0%)
    lv_obj_t* tipSectionTitle = lv_label_create(scrollArea);
    lv_label_set_text(tipSectionTitle, "Propina Sugerida");
    lv_obj_set_style_text_font(tipSectionTitle, &lv_font_montserrat_12, 0);
    lv_obj_set_style_text_color(tipSectionTitle, NeumorphicStyles::getMutedTextColor(), 0);

    lv_obj_t* tipGrid = lv_obj_create(scrollArea);
    lv_obj_set_width(tipGrid, lv_pct(100));
    lv_obj_set_height(tipGrid, 44);
    lv_obj_set_style_bg_opa(tipGrid, 0, 0);
    lv_obj_set_style_border_width(tipGrid, 0, 0);
    lv_obj_set_style_pad_all(tipGrid, 0, 0);
    NeumorphicStyles::disableScroll(tipGrid);

    static int32_t tip_col_dsc[] = {LV_GRID_FR(1), LV_GRID_FR(1), LV_GRID_FR(1), LV_GRID_FR(1), LV_GRID_TEMPLATE_LAST};
    static int32_t tip_row_dsc[] = {LV_GRID_FR(1), LV_GRID_TEMPLATE_LAST};
    lv_obj_set_layout(tipGrid, LV_LAYOUT_GRID);
    lv_obj_set_grid_dsc_array(tipGrid, tip_col_dsc, tip_row_dsc);
    lv_obj_set_style_pad_column(tipGrid, 6, 0);

    const int tipPercents[] = {10, 15, 20, 0};
    const char* tipTitles[] = {"10%", "15%", "20%", "0%"};
    lv_obj_t** tipBtns[] = {&tipBtn10, &tipBtn15, &tipBtn20, &tipBtn0};

    for (int i = 0; i < 4; i++) {
        lv_obj_t* btn = lv_button_create(tipGrid);
        lv_obj_set_grid_cell(btn, LV_GRID_ALIGN_STRETCH, i, 1, LV_GRID_ALIGN_STRETCH, 0, 1);
        NeumorphicStyles::applyButton(btn, 12);
        lv_obj_set_user_data(btn, (void*)(intptr_t)tipPercents[i]);
        lv_obj_add_event_cb(btn, tip_btn_cb, LV_EVENT_CLICKED, NULL);

        lv_obj_t* lbl = lv_label_create(btn);
        lv_label_set_text(lbl, tipTitles[i]);
        lv_obj_set_style_text_font(lbl, &lv_font_montserrat_12, 0);
        lv_obj_remove_flag(lbl, LV_OBJ_FLAG_CLICKABLE);
        lv_obj_center(lbl);

        *tipBtns[i] = btn;
    }

    // 4. Métodos de Pago
    lv_obj_t* paySectionTitle = lv_label_create(scrollArea);
    lv_label_set_text(paySectionTitle, "Método de Pago");
    lv_obj_set_style_text_font(paySectionTitle, &lv_font_montserrat_12, 0);
    lv_obj_set_style_text_color(paySectionTitle, NeumorphicStyles::getMutedTextColor(), 0);

    lv_obj_t* payGrid = lv_obj_create(scrollArea);
    lv_obj_set_width(payGrid, lv_pct(100));
    lv_obj_set_height(payGrid, 60);
    lv_obj_set_style_bg_opa(payGrid, 0, 0);
    lv_obj_set_style_border_width(payGrid, 0, 0);
    lv_obj_set_style_pad_all(payGrid, 0, 0);
    NeumorphicStyles::disableScroll(payGrid);

    static int32_t pay_col_dsc[] = {LV_GRID_FR(1), LV_GRID_FR(1), LV_GRID_FR(1), LV_GRID_TEMPLATE_LAST};
    static int32_t pay_row_dsc[] = {LV_GRID_FR(1), LV_GRID_TEMPLATE_LAST};
    lv_obj_set_layout(payGrid, LV_LAYOUT_GRID);
    lv_obj_set_grid_dsc_array(payGrid, pay_col_dsc, pay_row_dsc);
    lv_obj_set_style_pad_column(payGrid, 8, 0);

    const char* payIcons[] = {LV_SYMBOL_IMAGE, LV_SYMBOL_DIRECTORY, LV_SYMBOL_FILE};
    const char* payTitles[] = {"Efectivo", "Tarjeta", "QR Digital"};
    const int payIds[] = {1, 2, 3};

    for (int i = 0; i < 3; i++) {
        lv_obj_t* btn = lv_button_create(payGrid);
        lv_obj_set_grid_cell(btn, LV_GRID_ALIGN_STRETCH, i, 1, LV_GRID_ALIGN_STRETCH, 0, 1);
        NeumorphicStyles::applyButton(btn, 12);
        lv_obj_set_user_data(btn, (void*)(intptr_t)payIds[i]);
        lv_obj_add_event_cb(btn, payment_btn_cb, LV_EVENT_CLICKED, NULL);

        lv_obj_set_flex_flow(btn, LV_FLEX_FLOW_COLUMN);
        lv_obj_set_flex_align(btn, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);

        lv_obj_t* icon = lv_label_create(btn);
        lv_label_set_text(icon, payIcons[i]);
        lv_obj_set_style_text_color(icon, NeumorphicStyles::getPrimaryAccent(), 0);
        lv_obj_set_style_text_font(icon, &lv_font_montserrat_16, 0);
        lv_obj_remove_flag(icon, LV_OBJ_FLAG_CLICKABLE);

        lv_obj_t* lbl = lv_label_create(btn);
        lv_label_set_text(lbl, payTitles[i]);
        lv_obj_set_style_text_font(lbl, &lv_font_montserrat_12, 0);
        lv_obj_set_style_text_color(lbl, NeumorphicStyles::getTextColor(), 0);
        lv_obj_remove_flag(lbl, LV_OBJ_FLAG_CLICKABLE);
    }

    updateCalculations();
    
    // Publish MQTT Request
    MQTTService::getInstance().publishBillRequest();

    return screen;
}

void RequestBillView::updateCalculations() {
    float tipAmount = (currentSubtotal * selectedTipPercent) / 100.0f;
    float grandTotal = currentSubtotal + currentTax + tipAmount;

    char buf[64];

    snprintf(buf, sizeof(buf), "Subtotal: $%.2f", currentSubtotal);
    if (subtotalLabel) lv_label_set_text(subtotalLabel, buf);

    snprintf(buf, sizeof(buf), "Impuestos (IVA): $%.2f", currentTax);
    if (taxLabel) lv_label_set_text(taxLabel, buf);

    snprintf(buf, sizeof(buf), "Propina (%d%%): $%.2f", selectedTipPercent, tipAmount);
    if (tipLabel) lv_label_set_text(tipLabel, buf);

    snprintf(buf, sizeof(buf), "TOTAL: $%.2f", grandTotal);
    if (totalLabel) lv_label_set_text(totalLabel, buf);
}

void RequestBillView::tip_btn_cb(lv_event_t* e) {
    lv_obj_t* btn = (lv_obj_t*)lv_event_get_target(e);
    selectedTipPercent = (int)(intptr_t)lv_obj_get_user_data(btn);
    updateCalculations();
    UIManager::getInstance().resetInactivityTimer();
}

void RequestBillView::payment_btn_cb(lv_event_t* e) {
    lv_obj_t* btn = (lv_obj_t*)lv_event_get_target(e);
    int payId = (int)(intptr_t)lv_obj_get_user_data(btn);

    float tipAmount = (currentSubtotal * selectedTipPercent) / 100.0f;
    float grandTotal = currentSubtotal + currentTax + tipAmount;

    if (payId == 1) selectedPaymentMethod = "EFECTIVO";
    else if (payId == 2) selectedPaymentMethod = "TARJETA";
    else if (payId == 3) selectedPaymentMethod = "QR_DIGITAL";

    if (billCb) {
        billCb(selectedPaymentMethod.c_str(), currentSubtotal, selectedTipPercent, grandTotal);
    }

    if (payId == 3) {
        showQrModal();
    } else {
        char msg[80];
        snprintf(msg, sizeof(msg), "Cuenta solicitada: Pago en %s ($%.2f)", 
                 selectedPaymentMethod.c_str(), grandTotal);
        UIManager::showToast(msg);
        UIManager::getInstance().loadDashboard();
    }
    UIManager::getInstance().resetInactivityTimer();
}

void RequestBillView::showQrModal() {
    if (qrOverlay && lv_obj_is_valid(qrOverlay)) {
        lv_obj_delete(qrOverlay);
        qrOverlay = nullptr;
    }

    qrOverlay = lv_obj_create(lv_layer_top());
    lv_obj_set_size(qrOverlay, lv_pct(100), lv_pct(100));
    lv_obj_set_style_bg_color(qrOverlay, lv_color_black(), 0);
    lv_obj_set_style_bg_opa(qrOverlay, LV_OPA_80, 0);
    lv_obj_set_style_border_width(qrOverlay, 0, 0);
    NeumorphicStyles::disableScroll(qrOverlay);

    lv_obj_t* card = lv_obj_create(qrOverlay);
    lv_obj_set_width(card, 280);
    lv_obj_set_height(card, LV_SIZE_CONTENT);
    NeumorphicStyles::applyRaisedCard(card, 20);
    lv_obj_center(card);
    lv_obj_set_flex_flow(card, LV_FLEX_FLOW_COLUMN);
    lv_obj_set_flex_align(card, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER, LV_FLEX_ALIGN_CENTER);
    lv_obj_set_style_pad_all(card, 16, 0);
    lv_obj_set_style_pad_row(card, 10, 0);

    lv_obj_t* title = lv_label_create(card);
    lv_label_set_text(title, "Pago Móvil / Factura");
    lv_obj_set_style_text_font(title, &lv_font_montserrat_16, 0);
    lv_obj_set_style_text_color(title, NeumorphicStyles::getPrimaryAccent(), 0);

    lv_obj_t* sub = lv_label_create(card);
    lv_label_set_text(sub, "Escanea con tu cámara para pagar");
    lv_obj_set_style_text_font(sub, &lv_font_montserrat_12, 0);
    lv_obj_set_style_text_color(sub, NeumorphicStyles::getMutedTextColor(), 0);

    // Contenedor del Código QR en LVGL 9
    float tipAmount = (currentSubtotal * selectedTipPercent) / 100.0f;
    float grandTotal = currentSubtotal + currentTax + tipAmount;

    char qrPayload[128];
    snprintf(qrPayload, sizeof(qrPayload), "https://pay.tablehub.app/bill/MESA04?total=%.2f", grandTotal);

    lv_obj_t* qr = lv_qrcode_create(card);
    lv_qrcode_set_size(qr, 140);
    lv_qrcode_set_dark_color(qr, lv_color_black());
    lv_qrcode_set_light_color(qr, lv_color_white());
    lv_qrcode_update(qr, qrPayload, strlen(qrPayload));

    // Botón Cerrar
    lv_obj_t* closeBtn = lv_button_create(card);
    lv_obj_set_size(closeBtn, 120, 36);
    NeumorphicStyles::applyButton(closeBtn, 18);
    lv_obj_add_event_cb(closeBtn, closeQrModal, LV_EVENT_CLICKED, NULL);

    lv_obj_t* closeLabel = lv_label_create(closeBtn);
    lv_label_set_text(closeLabel, "Entendido");
    lv_obj_set_style_text_font(closeLabel, &lv_font_montserrat_12, 0);
    lv_obj_set_style_text_color(closeLabel, NeumorphicStyles::getTextColor(), 0);
    lv_obj_center(closeLabel);
}

void RequestBillView::closeQrModal(lv_event_t* e) {
    if (qrOverlay && lv_obj_is_valid(qrOverlay)) {
        lv_obj_delete(qrOverlay);
        qrOverlay = nullptr;
    }
    UIManager::getInstance().loadDashboard();
}

void RequestBillView::checkPendingUpdate() {
    if (g_pendingSubtotal >= 0.0f) {
        float sub = g_pendingSubtotal;
        float tax = g_pendingTax;
        g_pendingSubtotal = -1.0f;
        std::vector<BillItem> emptyItems;
        setBillData(sub, tax, emptyItems);
    }
}

void RequestBillView::setBillData(float subtotal, float tax, const std::vector<BillItem>& items) {
    currentSubtotal = subtotal;
    currentTax = tax;
    
    if (loadingLabel && lv_obj_is_valid(loadingLabel)) {
        lv_obj_add_flag(loadingLabel, LV_OBJ_FLAG_HIDDEN);
    }
    if (subtotalLabel) lv_obj_clear_flag(subtotalLabel, LV_OBJ_FLAG_HIDDEN);
    if (taxLabel) lv_obj_clear_flag(taxLabel, LV_OBJ_FLAG_HIDDEN);
    if (tipLabel) lv_obj_clear_flag(tipLabel, LV_OBJ_FLAG_HIDDEN);
    if (totalLabel) lv_obj_clear_flag(totalLabel, LV_OBJ_FLAG_HIDDEN);
    
    updateCalculations();
}

void RequestBillView::setBillCallback(BillCallback cb) {
    billCb = cb;
}
