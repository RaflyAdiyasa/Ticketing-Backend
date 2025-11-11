package handlers

import (
	"os"
	"time"

	"gorm.io/gorm"

	"github.com/Tsaniii18/Ticketing-Backend/config"
	"github.com/Tsaniii18/Ticketing-Backend/models"
	"github.com/Tsaniii18/Ticketing-Backend/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
)

func PaymentMidtrans(c *fiber.Ctx) error {
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
		TransactionID: utils.GenerateTransactionID(),
		OwnerID:       user.UserID,
		// TransactionTime:   time.Now(),
		PriceTotal:        total,
		CreatedAt:         time.Now(),
		TransactionStatus: "pending",
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
			TransactionDetailID: utils.GenerateTransactionDetailID(),
			TicketCategoryID:    cartItem.TicketCategoryID,
			TransactionID:       transaction.TransactionID,
			OwnerID:             user.UserID,
			Quantity:            cartItem.Quantity,
			Subtotal:            cartItem.PriceTotal,
		}

		if err := config.DB.Create(&transactionDetail).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create transaction detail",
			})
		}

		// Create tickets
		for i := 0; i < int(cartItem.Quantity); i++ {
			ticket := models.Ticket{
				TicketID:         utils.GenerateTicketID(),
				EventID:          cartItem.TicketCategory.EventID,
				TicketCategoryID: cartItem.TicketCategoryID,
				OwnerID:          user.UserID,
				Status:           "pending", // Set status to pending until payment is confirmed
				Code:             utils.GenerateTicketCode(),
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}

			if err := config.DB.Create(&ticket).Error; err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to create ticket",
				})
			}
		}

	}

	// Create Snap client
	var s snap.Client
	s.New(os.Getenv("MIDTRANS_SERVER_KEY"), midtrans.Sandbox)

	var items []midtrans.ItemDetails
	for _, cartItem := range cart {
		item := midtrans.ItemDetails{
			ID:       cartItem.TicketCategoryID,
			Name:     cartItem.TicketCategory.Name,
			Price:    int64(cartItem.PriceTotal),
			Qty:      int32(cartItem.Quantity),
			Category: "Event Ticket",
		}
		items = append(items, item)
	}

	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  transaction.TransactionID,
			GrossAmt: int64(total),
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: user.Name,
			Email: user.Email,
		},
		Items: &items,
	}

	resp, err := s.CreateTransaction(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create transaction with Midtrans",
		})
	}

	// Clear cart
	if err := config.DB.Where("owner_id = ?", user.UserID).Delete(&models.Cart{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to clear cart",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Payment Midtrans Endpoint",
		"total":   total,
		"url":     resp.RedirectURL,
		"token":   resp.Token,
	})

}

func PaymentNotificationHandler(c *fiber.Ctx) error {
	var notifPayload map[string]interface{}
	if err := c.BodyParser(&notifPayload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	orderID := notifPayload["order_id"].(string)
	status := notifPayload["transaction_status"].(string)
	transactionTimeString := notifPayload["transaction_time"].(string)

	// Update status transaksi di database kamu
	switch status {
	case "settlement":
		// pembayaran sukses

		layout := "2006-01-02 15:04:05"
		TransactionTime, _ := time.Parse(layout, transactionTimeString)
		if err := config.DB.Model(&models.TransactionHistory{}).Where("transaction_id = ?", orderID).Updates(models.TransactionHistory{
			TransactionStatus: "paid",
			TransactionTime:   TransactionTime,
		}).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to update transaction status"})
		}

		var transactionDetails []models.TransactionDetail
		if err := config.DB.Where("transaction_id = ?", orderID).Find(&transactionDetails).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch transaction details"})
		}

		for _, detail := range transactionDetails {
			var tickets []models.Ticket
			if err := config.DB.Where("ticket_category_id = ? AND owner_id = ? AND status = ?", detail.TicketCategoryID, detail.OwnerID, "pending").Limit(int(detail.Quantity)).Find(&tickets).Error; err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch tickets"})
			}

			// Update sold count
			config.DB.Model(&detail.TicketCategory).Update("sold", detail.TicketCategory.Sold+detail.Quantity)

			for _, ticket := range tickets {
				ticket.Status = "active"
				if err := config.DB.Save(&ticket).Error; err != nil {
					return c.Status(500).JSON(fiber.Map{"error": "Failed to update ticket status"})
				}
			}

			// Update event total sales
			config.DB.Model(&models.Event{}).Where("event_id = ?", detail.TicketCategory.EventID).
				Update("total_sales", gorm.Expr("total_sales + ?", detail.Subtotal))
		}

	case "pending":
		// menunggu pembayaran
	case "deny", "cancel":
		// pembayaran gagal
	}

	return c.JSON(fiber.Map{
		"message": "Notification processed",
		"orderID": orderID,
		"status":  status,
	})
}
