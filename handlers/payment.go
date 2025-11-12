package handlers

import (
	"log"
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
	type CartWithDetails struct {
		models.Cart
		TicketCategoryName string  `json:"ticket_category_name"`
		EventID            string  `json:"event_id"`
		PricePerItem       float64 `json:"price_per_item"`
	}

	var cartItems []CartWithDetails

	if err := config.DB.
		Table("carts").
		Select("carts.*, tc.name as ticket_category_name, tc.event_id as event_id, tc.price as price_per_item").
		Joins("LEFT JOIN ticket_categories tc ON carts.ticket_category_id = tc.ticket_category_id").
		Where("carts.owner_id = ?", user.UserID).
		Find(&cartItems).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch cart: " + err.Error(),
		})
	}

	if len(cartItems) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cart is empty",
		})
	}

	// Calculate total dan validasi quota
	var total float64
	var transactionDetails []models.TransactionDetail

	for _, item := range cartItems {
		// Validasi quota tersedia
		var ticketCategory models.TicketCategory
		if err := config.DB.First(&ticketCategory, "ticket_category_id = ?", item.TicketCategoryID).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Ticket category not found: " + item.TicketCategoryID,
			})
		}

		// Cek ketersediaan quota
		if ticketCategory.Sold+item.Quantity > ticketCategory.Quota {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Not enough quota for ticket category: " + ticketCategory.Name,
			})
		}

		total += item.PriceTotal

		// Prepare transaction detail
		transactionDetail := models.TransactionDetail{
			TransactionDetailID: utils.GenerateTransactionDetailID(),
			TicketCategoryID:    item.TicketCategoryID,
			OwnerID:             user.UserID,
			Quantity:            item.Quantity,
			Subtotal:            item.PriceTotal,
		}
		transactionDetails = append(transactionDetails, transactionDetail)
	}

	// Mulai database transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to start transaction",
		})
	}

	// Create transaction
	transaction := models.TransactionHistory{
		TransactionID:     utils.GenerateTransactionID(),
		OwnerID:           user.UserID,
		TransactionTime:   time.Now(),
		PriceTotal:        total,
		CreatedAt:         time.Now(),
		TransactionStatus: "pending",
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create transaction: " + err.Error(),
		})
	}

	// Create transaction details dan pending tickets
	for _, detail := range transactionDetails {
		// Set transaction ID untuk detail
		detail.TransactionID = transaction.TransactionID

		if err := tx.Create(&detail).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create transaction detail: " + err.Error(),
			})
		}

		// Get event ID from ticket category untuk membuat tickets
		var ticketCategory models.TicketCategory
		if err := tx.First(&ticketCategory, "ticket_category_id = ?", detail.TicketCategoryID).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get ticket category: " + err.Error(),
			})
		}

		// Create pending tickets - FIX: Generate unique code untuk setiap ticket
		for i := 0; i < int(detail.Quantity); i++ {
			ticket := models.Ticket{
				TicketID:         utils.GenerateTicketID(),
				EventID:          ticketCategory.EventID,
				TicketCategoryID: detail.TicketCategoryID,
				OwnerID:          user.UserID,
				Status:           "pending",
				Code:             utils.GenerateTicketCode(), // GENERATE UNIQUE CODE
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}

			if err := tx.Create(&ticket).Error; err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to create ticket: " + err.Error(),
				})
			}
		}
	}

	// Clear cart
	if err := tx.Where("owner_id = ?", user.UserID).Delete(&models.Cart{}).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to clear cart: " + err.Error(),
		})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to commit transaction: " + err.Error(),
		})
	}

	// Create Snap client untuk Midtrans
	var s snap.Client
	s.New(os.Getenv("MIDTRANS_SERVER_KEY"), midtrans.Sandbox)

	// Prepare items untuk Midtrans
	var items []midtrans.ItemDetails
	for _, item := range cartItems {
		items = append(items, midtrans.ItemDetails{
			ID:    item.TicketCategoryID,
			Name:  item.TicketCategoryName,
			Price: int64(item.PricePerItem),
			Qty:   int32(item.Quantity),
		})
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

	// Create Midtrans transaction
	snapResp, err := s.CreateTransaction(req)
	if err != nil {
		log.Printf("Midtrans error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":          "Failed to create Midtrans payment: " + err.Error(),
			"transaction_id": transaction.TransactionID,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":        "Payment initiated successfully",
		"transaction_id": transaction.TransactionID,
		"total":          total,
		"payment_url":    snapResp.RedirectURL,
		"token":          snapResp.Token,
	})
}

func PaymentNotificationHandler(c *fiber.Ctx) error {
	var notifPayload map[string]interface{}
	if err := c.BodyParser(&notifPayload); err != nil {
		log.Printf("Error parsing notification payload: %v", err)
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	// Log received payload untuk debugging
	log.Printf("Received Midtrans notification: %+v", notifPayload)

	orderID, ok := notifPayload["order_id"].(string)
	if !ok {
		log.Printf("Invalid order_id in notification")
		return c.Status(400).JSON(fiber.Map{"error": "Invalid order_id"})
	}

	transactionStatus, ok := notifPayload["transaction_status"].(string)
	if !ok {
		log.Printf("Invalid transaction_status in notification")
		return c.Status(400).JSON(fiber.Map{"error": "Invalid transaction_status"})
	}

	log.Printf("Processing notification for OrderID: %s, Status: %s", orderID, transactionStatus)

	// Handle different transaction status
	switch transactionStatus {
	case "settlement":
		return handleSettlement(c, orderID, notifPayload)
	case "deny", "cancel", "expire":
		return handleFailure(c, orderID, transactionStatus)
	case "pending":
		return handlePending(c, orderID)
	default:
		log.Printf("Unhandled transaction status: %s", transactionStatus)
		return c.Status(400).JSON(fiber.Map{"error": "Unknown transaction status"})
	}
}

func handleSettlement(c *fiber.Ctx, orderID string, notifPayload map[string]interface{}) error {
	// Start database transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		log.Printf("Failed to start transaction: %v", tx.Error)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to start transaction"})
	}

	// Update transaction status
	result := tx.Model(&models.TransactionHistory{}).
		Where("transaction_id = ?", orderID).
		Updates(map[string]interface{}{
			"transaction_status": "paid",
			"transaction_time":   time.Now(), // Gunakan waktu server sebagai fallback
		})

	if result.Error != nil {
		tx.Rollback()
		log.Printf("Failed to update transaction status: %v", result.Error)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update transaction status"})
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		log.Printf("No transaction found with ID: %s", orderID)
		return c.Status(404).JSON(fiber.Map{"error": "Transaction not found"})
	}

	// Get transaction details
	var transactionDetails []models.TransactionDetail
	if err := tx.Where("transaction_id = ?", orderID).Find(&transactionDetails).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to fetch transaction details: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch transaction details"})
	}

	// Process each transaction detail
	for _, detail := range transactionDetails {
		// Update ticket category sold count
		if err := tx.Model(&models.TicketCategory{}).
			Where("ticket_category_id = ?", detail.TicketCategoryID).
			Update("sold", gorm.Expr("sold + ?", detail.Quantity)).Error; err != nil {
			tx.Rollback()
			log.Printf("Failed to update ticket category sold count: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to update ticket category"})
		}

		// Update tickets status from pending to active
		if err := tx.Model(&models.Ticket{}).
			Where("ticket_category_id = ? AND owner_id = ? AND status = ?",
				detail.TicketCategoryID, detail.OwnerID, "pending").
			Updates(map[string]interface{}{
				"status": "active",
				// "code":   utils.GenerateTicketCode(),
			}).Error; err != nil {
			tx.Rollback()
			log.Printf("Failed to update tickets status: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to update tickets"})
		}

		// Get event ID for updating total sales
		var ticketCategory models.TicketCategory
		if err := tx.First(&ticketCategory, "ticket_category_id = ?", detail.TicketCategoryID).Error; err != nil {
			tx.Rollback()
			log.Printf("Failed to get ticket category: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to get ticket category"})
		}

		// Update event total sales
		if err := tx.Model(&models.Event{}).
			Where("event_id = ?", ticketCategory.EventID).
			Update("total_sales", gorm.Expr("total_sales + ?", detail.Subtotal)).Error; err != nil {
			tx.Rollback()
			log.Printf("Failed to update event total sales: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to update event sales"})
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to commit transaction"})
	}

	log.Printf("Successfully processed settlement for OrderID: %s", orderID)
	return c.JSON(fiber.Map{
		"message": "Payment successful and processed",
		"orderID": orderID,
		"status":  "paid",
	})
}

func handleFailure(c *fiber.Ctx, orderID string, status string) error {
	// Map Midtrans status to your status
	var newStatus string
	switch status {
	case "deny", "cancel":
		newStatus = "failed"
	case "expire":
		newStatus = "expired"
	default:
		newStatus = "failed"
	}

	// Start transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to start transaction"})
	}

	// Update transaction status
	result := tx.Model(&models.TransactionHistory{}).
		Where("transaction_id = ?", orderID).
		Update("transaction_status", newStatus)

	if result.Error != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update transaction status"})
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		return c.Status(404).JSON(fiber.Map{"error": "Transaction not found"})
	}

	// Update tickets status to payment_failed
	var transactionDetails []models.TransactionDetail
	if err := tx.Where("transaction_id = ?", orderID).Find(&transactionDetails).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch transaction details"})
	}

	for _, detail := range transactionDetails {
		if err := tx.Model(&models.Ticket{}).
			Where("ticket_category_id = ? AND owner_id = ? AND status = ?",
				detail.TicketCategoryID, detail.OwnerID, "pending").
			Update("status", "payment_failed").Error; err != nil {
			tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"error": "Failed to update ticket status"})
		}
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to commit transaction"})
	}

	log.Printf("Transaction %s marked as %s", orderID, newStatus)
	return c.JSON(fiber.Map{
		"message": "Transaction status updated",
		"orderID": orderID,
		"status":  newStatus,
	})
}

func handlePending(c *fiber.Ctx, orderID string) error {
	// Untuk status pending, tidak perlu melakukan perubahan besar
	log.Printf("Transaction %s is pending payment", orderID)
	return c.JSON(fiber.Map{
		"message": "Payment pending",
		"orderID": orderID,
		"status":  "pending",
	})
}
