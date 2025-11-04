package handlers

import (
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    "github.com/Tsaniii18/Ticketing-Backend/config"
    "github.com/Tsaniii18/Ticketing-Backend/models"
)

func Checkout(c *fiber.Ctx) error {
    user := c.Locals("user").(models.User)

    // Get user's cart items
    var cart []models.Cart
    if err := config.DB.Preload("TicketCategory").Where("owner_id = ?", user.UserID).Find(&cart).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to fetch cart",
        })
    }

    if len(cart) == 0 {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Cart is empty",
        })
    }

    // Calculate total
    var total float64
    for _, item := range cart {
        total += item.PriceTotal
    }

    // Create transaction
    transaction := models.TransactionHistory{
        OwnerID:         user.UserID,
        TransactionTime: time.Now(),
        PriceTotal:      total,
    }

    if err := config.DB.Create(&transaction).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to create transaction",
        })
    }

    // Create transaction details and tickets
    for _, cartItem := range cart {
        // Create transaction detail
        transactionDetail := models.TransactionDetail{
            TicketCategoryID: cartItem.TicketCategoryID,
            TransactionID:    transaction.TransactionID,
            OwnerID:          user.UserID,
            Quantity:         cartItem.Quantity,
            Subtotal:         cartItem.PriceTotal,
        }

        if err := config.DB.Create(&transactionDetail).Error; err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to create transaction detail",
            })
        }

        // Create tickets
        for i := 0; i < int(cartItem.Quantity); i++ {
            ticket := models.Ticket{
                EventID:          cartItem.TicketCategory.EventID,
                TicketCategoryID: cartItem.TicketCategoryID,
                OwnerID:          user.UserID,
                Status:           "active",
                Code:             generateTicketCode(),
            }

            if err := config.DB.Create(&ticket).Error; err != nil {
                return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                    "error": "Failed to create ticket",
                })
            }
        }

        // Update sold count
        config.DB.Model(&cartItem.TicketCategory).Update("sold", cartItem.TicketCategory.Sold+cartItem.Quantity)
    }

    // Clear cart
    if err := config.DB.Where("owner_id = ?", user.UserID).Delete(&models.Cart{}).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to clear cart",
        })
    }

    return c.JSON(fiber.Map{
        "message":     "Checkout successful",
        "transaction": transaction,
    })
}

func GetTransactions(c *fiber.Ctx) error {
    user := c.Locals("user").(models.User)

    var transactions []models.TransactionHistory
    if err := config.DB.Preload("TransactionDetails").Preload("TransactionDetails.TicketCategory").Where("owner_id = ?", user.UserID).Find(&transactions).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to fetch transactions",
        })
    }

    return c.JSON(transactions)
}

func GetTransaction(c *fiber.Ctx) error {
    user := c.Locals("user").(models.User)
    transactionID := c.Params("id")

    var transaction models.TransactionHistory
    if err := config.DB.Preload("TransactionDetails").Preload("TransactionDetails.TicketCategory").
        Where("transaction_id = ? AND owner_id = ?", transactionID, user.UserID).
        First(&transaction).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Transaction not found",
        })
    }

    return c.JSON(transaction)
}

func generateTicketCode() string {
    return "TICKET-" + uuid.New().String()
}
