package handlers

import (
	"os"
	"time"
	"log"
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
        if ticketCategory.Sold + item.Quantity > ticketCategory.Quota {
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
            "error": "Failed to create Midtrans payment: " + err.Error(),
            "transaction_id": transaction.TransactionID,
        })
    }

    return c.Status(fiber.StatusOK).JSON(fiber.Map{
        "message": "Payment initiated successfully",
        "transaction_id": transaction.TransactionID,
        "total":   total,
        "payment_url": snapResp.RedirectURL,
        "token":   snapResp.Token,
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

			var ticketCategory models.TicketCategory
			if err := config.DB.First(&ticketCategory, "ticket_category_id = ?", detail.TicketCategoryID).Error; err != nil {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "Ticket category not found",
				})
			}

			// Update sold count
			config.DB.Model(&ticketCategory).Update("sold", ticketCategory.Sold+detail.Quantity)

			for _, ticket := range tickets {
				ticket.Status = "active"
				ticket.Code = utils.GenerateTicketCode()
				if err := config.DB.Save(&ticket).Error; err != nil {
					return c.Status(500).JSON(fiber.Map{"error": "Failed to update ticket status"})
				}
			}

			// Update event total sales
			config.DB.Model(&models.Event{}).Where("event_id = ?", ticketCategory.EventID).
				Update("total_sales", gorm.Expr("total_sales + ?", detail.Subtotal))
		}

	case "deny", "cancel":
		// pembayaran gagal
		if err := config.DB.Model(&models.TransactionHistory{}).Where("transaction_id = ?", orderID).Update("transaction_status", "failed").Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to update transaction status - status cancel/deny"})
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

			for _, ticket := range tickets {
				ticket.Status = "payment_failed"
				if err := config.DB.Save(&ticket).Error; err != nil {
					return c.Status(500).JSON(fiber.Map{"error": "Failed to update ticket status"})
				}
			}
		}

	case "expire":
		// pembayaran kadaluarsa
		if err := config.DB.Model(&models.TransactionHistory{}).Where("transaction_id = ?", orderID).Update("transaction_status", "expired").Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to update transaction status - status expire"})
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

			for _, ticket := range tickets {
				ticket.Status = "payment_failed"
				if err := config.DB.Save(&ticket).Error; err != nil {
					return c.Status(500).JSON(fiber.Map{"error": "Failed to update ticket status"})
				}
			}
		}
	default:
		return c.Status(400).JSON(fiber.Map{"error": "Unknown transaction status"})
	}

	return c.JSON(fiber.Map{
		"message": "Notification processed",
		"orderID": orderID,
		"status":  status,
	})
}
