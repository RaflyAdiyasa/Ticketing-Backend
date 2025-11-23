package handlers

import (
	"time"

	"github.com/Tsaniii18/Ticketing-Backend/config"
	"github.com/Tsaniii18/Ticketing-Backend/models"
	"github.com/gofiber/fiber/v2"
)

// GetTransactionHistory - Mendapatkan semua riwayat transaksi user
func GetTransactionHistory(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	// Struct untuk response dengan detail lengkap
	type TicketDetailResponse struct {
		TicketID         string    `json:"ticket_id"`
		Code             string    `json:"code"`
		Status           string    `json:"status"`
		TicketCategoryID string    `json:"ticket_category_id"`
		CategoryName     string    `json:"category_name"`
		Description      string    `json:"description"`
		Price            float64   `json:"price"`
		DateTimeStart    time.Time `json:"date_time_start"`
		DateTimeEnd      time.Time `json:"date_time_end"`
	}

	type EventInTransactionResponse struct {
		EventID        string                 `json:"event_id"`
		EventName      string                 `json:"event_name"`
		Location       string                 `json:"location"`
		City           string                 `json:"city"`
		DateStart      time.Time              `json:"date_start"`
		DateEnd        time.Time              `json:"date_end"`
		Image          string                 `json:"image"`
		TicketDetails  []TicketDetailResponse `json:"ticket_details"`
		EventSubtotal  float64                `json:"event_subtotal"`
	}

	type TransactionHistoryResponse struct {
		TransactionID     string                       `json:"transaction_id"`
		TransactionTime   time.Time                    `json:"transaction_time"`
		TransactionStatus string                       `json:"transaction_status"`
		PriceTotal        float64                      `json:"price_total"`
		Events            []EventInTransactionResponse `json:"events"`
	}

	// Get all transactions untuk user
	var transactions []models.TransactionHistory
	if err := config.DB.
		Where("owner_id = ?", user.UserID).
		Order("transaction_time DESC").
		Find(&transactions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch transaction history: " + err.Error(),
		})
	}

	// Build response dengan details
	var response []TransactionHistoryResponse

	for _, transaction := range transactions {
		// Get transaction details
		var transactionDetails []models.TransactionDetail
		if err := config.DB.
			Where("transaction_id = ?", transaction.TransactionID).
			Find(&transactionDetails).Error; err != nil {
			continue
		}

		// Group by event
		eventMap := make(map[string]*EventInTransactionResponse)
		
		for _, detail := range transactionDetails {
			// Get ticket category info
			var ticketCategory models.TicketCategory
			if err := config.DB.
				Where("ticket_category_id = ?", detail.TicketCategoryID).
				First(&ticketCategory).Error; err != nil {
				continue
			}

			// Get event info
			var event models.Event
			if err := config.DB.
				Where("event_id = ?", ticketCategory.EventID).
				First(&event).Error; err != nil {
				continue
			}

			// Get tickets untuk category ini dalam transaksi ini
			var tickets []models.Ticket
			config.DB.
				Where("ticket_category_id = ? AND owner_id = ? AND (status = ? OR status = ?)", 
					detail.TicketCategoryID, user.UserID, "active", "checked_in").
				Limit(int(detail.Quantity)).
				Find(&tickets)

			// Initialize event in map if not exists
			if _, exists := eventMap[event.EventID]; !exists {
				eventMap[event.EventID] = &EventInTransactionResponse{
					EventID:       event.EventID,
					EventName:     event.Name,
					Location:      event.Location,
					City:          event.City,
					DateStart:     event.DateStart,
					DateEnd:       event.DateEnd,
					Image:         event.Image,
					TicketDetails: []TicketDetailResponse{},
					EventSubtotal: 0,
				}
			}

			// Add ticket details
			for _, ticket := range tickets {
				eventMap[event.EventID].TicketDetails = append(
					eventMap[event.EventID].TicketDetails,
					TicketDetailResponse{
						TicketID:         ticket.TicketID,
						Code:             ticket.Code,
						Status:           ticket.Status,
						TicketCategoryID: ticketCategory.TicketCategoryID,
						CategoryName:     ticketCategory.Name,
						Description:      ticketCategory.Description,
						Price:            ticketCategory.Price,
						DateTimeStart:    ticketCategory.DateTimeStart,
						DateTimeEnd:      ticketCategory.DateTimeEnd,
					},
				)
			}

			eventMap[event.EventID].EventSubtotal += detail.Subtotal
		}

		// Convert map to slice
		var events []EventInTransactionResponse
		for _, event := range eventMap {
			events = append(events, *event)
		}

		response = append(response, TransactionHistoryResponse{
			TransactionID:     transaction.TransactionID,
			TransactionTime:   transaction.TransactionTime,
			TransactionStatus: transaction.TransactionStatus,
			PriceTotal:        transaction.PriceTotal,
			Events:            events,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":      "Transaction history retrieved successfully",
		"transactions": response,
	})
}

// GetTransactionDetail - Mendapatkan detail satu transaksi
func GetTransactionDetail(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	transactionID := c.Params("id")

	// Struct untuk response detail
	type TicketDetailResponse struct {
		TicketID         string    `json:"ticket_id"`
		Code             string    `json:"code"`
		Status           string    `json:"status"`
		TicketCategoryID string    `json:"ticket_category_id"`
		CategoryName     string    `json:"category_name"`
		Description      string    `json:"description"`
		Price            float64   `json:"price"`
		DateTimeStart    time.Time `json:"date_time_start"`
		DateTimeEnd      time.Time `json:"date_time_end"`
	}

	type EventDetailResponse struct {
		EventID       string                 `json:"event_id"`
		EventName     string                 `json:"event_name"`
		Location      string                 `json:"location"`
		City          string                 `json:"city"`
		DateStart     time.Time              `json:"date_start"`
		DateEnd       time.Time              `json:"date_end"`
		Image         string                 `json:"image"`
		Flyer         string                 `json:"flyer"`
		TicketDetails []TicketDetailResponse `json:"ticket_details"`
		EventSubtotal float64                `json:"event_subtotal"`
	}

	// Get transaction
	var transaction models.TransactionHistory
	if err := config.DB.
		Where("transaction_id = ? AND owner_id = ?", transactionID, user.UserID).
		First(&transaction).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Transaction not found",
		})
	}

	// Get transaction details
	var transactionDetails []models.TransactionDetail
	if err := config.DB.
		Where("transaction_id = ?", transaction.TransactionID).
		Find(&transactionDetails).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch transaction details: " + err.Error(),
		})
	}

	// Build response
	eventMap := make(map[string]*EventDetailResponse)

	for _, detail := range transactionDetails {
		// Get ticket category
		var ticketCategory models.TicketCategory
		if err := config.DB.
			Where("ticket_category_id = ?", detail.TicketCategoryID).
			First(&ticketCategory).Error; err != nil {
			continue
		}

		// Get event
		var event models.Event
		if err := config.DB.
			Where("event_id = ?", ticketCategory.EventID).
			First(&event).Error; err != nil {
			continue
		}

		// Get tickets
		var tickets []models.Ticket
		config.DB.
			Where("ticket_category_id = ? AND owner_id = ? AND (status = ? OR status = ?)", 
				detail.TicketCategoryID, user.UserID, "active", "checked_in").
			Limit(int(detail.Quantity)).
			Find(&tickets)

		// Initialize event
		if _, exists := eventMap[event.EventID]; !exists {
			eventMap[event.EventID] = &EventDetailResponse{
				EventID:       event.EventID,
				EventName:     event.Name,
				Location:      event.Location,
				City:          event.City,
				DateStart:     event.DateStart,
				DateEnd:       event.DateEnd,
				Image:         event.Image,
				Flyer:         event.Flyer,
				TicketDetails: []TicketDetailResponse{},
				EventSubtotal: 0,
			}
		}

		// Add tickets
		for _, ticket := range tickets {
			eventMap[event.EventID].TicketDetails = append(
				eventMap[event.EventID].TicketDetails,
				TicketDetailResponse{
					TicketID:         ticket.TicketID,
					Code:             ticket.Code,
					Status:           ticket.Status,
					TicketCategoryID: ticketCategory.TicketCategoryID,
					CategoryName:     ticketCategory.Name,
					Description:      ticketCategory.Description,
					Price:            ticketCategory.Price,
					DateTimeStart:    ticketCategory.DateTimeStart,
					DateTimeEnd:      ticketCategory.DateTimeEnd,
				},
			)
		}

		eventMap[event.EventID].EventSubtotal += detail.Subtotal
	}

	// Convert to slice
	var events []EventDetailResponse
	for _, event := range eventMap {
		events = append(events, *event)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Transaction detail retrieved successfully",
		"transaction": fiber.Map{
			"transaction_id":     transaction.TransactionID,
			"transaction_time":   transaction.TransactionTime,
			"transaction_status": transaction.TransactionStatus,
			"price_total":        transaction.PriceTotal,
			"events":             events,
		},
	})
}