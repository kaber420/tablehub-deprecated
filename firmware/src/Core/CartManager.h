#pragma once
#include <vector>
#include <string>
#include "../UI/Views/MenuView.h"

struct CartItem {
    std::string itemId;
    std::string name;
    float price;
    int quantity;
    const char* icon;
};

class CartManager {
public:
    static CartManager& getInstance() {
        static CartManager instance;
        return instance;
    }

    void addItem(const MenuItem& item);
    void removeItem(const std::string& itemId);
    void decreaseItem(const std::string& itemId);
    void clearCart();

    float getTotalPrice() const;
    int getItemCount() const;
    const std::vector<CartItem>& getItems() const { return items; }

private:
    CartManager() = default;
    std::vector<CartItem> items;
};
