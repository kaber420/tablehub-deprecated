#include "CartManager.h"

void CartManager::addItem(const MenuItem& item) {
    for (auto& cartItem : items) {
        if (cartItem.itemId == item.id) {
            cartItem.quantity++;
            return;
        }
    }

    CartItem newItem;
    newItem.itemId = item.id;
    newItem.name = item.name;
    newItem.price = item.price;
    newItem.quantity = 1;
    newItem.icon = item.icon;
    items.push_back(newItem);
}

void CartManager::decreaseItem(const std::string& itemId) {
    for (auto it = items.begin(); it != items.end(); ++it) {
        if (it->itemId == itemId) {
            it->quantity--;
            if (it->quantity <= 0) {
                items.erase(it);
            }
            return;
        }
    }
}

void CartManager::removeItem(const std::string& itemId) {
    for (auto it = items.begin(); it != items.end(); ++it) {
        if (it->itemId == itemId) {
            items.erase(it);
            return;
        }
    }
}

void CartManager::clearCart() {
    items.clear();
}

float CartManager::getTotalPrice() const {
    float total = 0.0f;
    for (const auto& item : items) {
        total += (item.price * item.quantity);
    }
    return total;
}

int CartManager::getItemCount() const {
    int count = 0;
    for (const auto& item : items) {
        count += item.quantity;
    }
    return count;
}
