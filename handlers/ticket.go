package handlers

import (
	"time"

	"log"

	"gorm.io/gorm"

	"github.com/Tsaniii18/Ticketing-Backend/config"
	"github.com/Tsaniii18/Ticketing-Backend/models"
	"github.com/Tsaniii18/Ticketing-Backend/utils"
	"github.com/gofiber/fiber/v2"
)

type ticketResponse struct {
	Code           string                  `json:"code"`
	TicketID       string                  `json:"ticket_id"`
	TicketCategory *ticketCategoryResponse `json:"ticket_category"`
	Event          *eventResponse          `json:"event"`
	Tag            string                  `json:"tag"`
	Status         string                  `json:"status"`    // ADDED: Status tiket
	UsedAt         *time.Time              `json:"used_at"`   // ADDED: Waktu check-in
	CreatedAt      time.Time               `json:"created_at"` // ADDED: Waktu pembuatan
}

type ticketCategoryResponse struct {
	TicketCategoryID string    `json:"ticket_category_id"` // ADDED
	Name             string    `json:"name"`
	DateTimeStart    time.Time `json:"date_time_start"`
	DateTimeEnd      time.Time `json:"date_time_end"`
	Price            float64   `json:"price"`
	Description      string    `json:"description"`
}

type eventResponse struct {
	EventID   string    `json:"event_id"` // ADDED
	Name      string    `json:"name"`
	Location  string    `json:"location"`
	Venue     string    `json:"venue"` // Fixed: lowercase
	City      string    `json:"city"`  // ADDED (from District)
	DateStart time.Time `json:"date_start"`
	DateEnd   time.Time `json:"date_end"`
	Image     string    `json:"image"` // ADDED
}

// GetTickets - Mengambil SEMUA tiket milik user (tidak hanya active)
func GetTickets(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	// Query parameter untuk filter status (optional)
	statusFilter := c.Query("status", "") // "" = semua, "active", "used", "expired", "cancelled"

	var tickets []models.Ticket
	query := config.DB.Where("owner_id = ?", user.UserID)

	// Jika ada filter status spesifik
	if statusFilter != "" && statusFilter != "all" {
		query = query.Where("status = ?", statusFilter)
	}

	if err := query.Order("created_at DESC").Find(&tickets).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch tickets",
		})
	}

	var ticketResponses []ticketResponse

	for _, ticket := range tickets {
		var ticketCategory models.TicketCategory
		if err := config.DB.First(&ticketCategory, "ticket_category_id = ?", ticket.TicketCategoryID).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch ticket category",
			})
		}

		var event models.Event
		if err := config.DB.First(&event, "event_id = ?", ticket.EventID).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch event",
			})
		}

		// Determine computed status
		computedStatus := ticket.Status

		// Check if event has ended (expired)
		if ticket.Status == "active" && event.DateEnd.Before(time.Now()) {
			computedStatus = "expired"
		}

		ticketCategoryResponse := ticketCategoryResponse{
			TicketCategoryID: ticketCategory.TicketCategoryID,
			Name:             ticketCategory.Name,
			DateTimeStart:    ticketCategory.DateTimeStart,
			DateTimeEnd:      ticketCategory.DateTimeEnd,
			Price:            ticketCategory.Price,
			Description:      ticketCategory.Description,
		}

		eventResponse := eventResponse{
			EventID:   event.EventID,
			Name:      event.Name,
			Location:  event.Location,
			Venue:     event.Venue,
			City:      event.District, // Using District as City
			DateStart: event.DateStart,
			DateEnd:   event.DateEnd,
			Image:     event.Image,
		}

		// Determine used_at (jika status used, gunakan UpdatedAt sebagai waktu check-in)
		var usedAt *time.Time
		if ticket.Status == "used" {
			usedAt = &ticket.UpdatedAt
		}

		ticketResponse := ticketResponse{
			Code:           ticket.Code,
			TicketID:       ticket.TicketID,
			TicketCategory: &ticketCategoryResponse,
			Event:          &eventResponse,
			Tag:            ticket.Tag,
			Status:         computedStatus,
			UsedAt:         usedAt,
			CreatedAt:      ticket.CreatedAt,
		}
		ticketResponses = append(ticketResponses, ticketResponse)
	}

	return c.JSON(fiber.Map{
		"data":    ticketResponses,
		"message": "Tickets fetched successfully",
		"total":   len(ticketResponses),
	})
}

func CheckInTicket(c *fiber.Ctx) error {
	eventID := c.Params("event_id")
	codeEvent := c.Params("id")

	var ticket models.Ticket
	tx := config.DB.Begin()
	if err := tx.Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to start transaction",
		})
	}
	if err := tx.First(&ticket, "code = ? AND event_id = ?", codeEvent, eventID).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Ticket not found or Ticket code invalid",
		})
	}

	if ticket.Status == "used" {
		tx.Rollback()
		return c.Status(fiber.StatusNotAcceptable).JSON(fiber.Map{
			"error":  "Ticket already used",
			"status": "already_used",
		})
	}

	if ticket.Status == "cancelled" {
		tx.Rollback()
		return c.Status(fiber.StatusNotAcceptable).JSON(fiber.Map{
			"error":  "Ticket has been cancelled",
			"status": "cancelled",
		})
	}

	if ticket.Status != "active" {
		tx.Rollback()
		return c.Status(fiber.StatusNotAcceptable).JSON(fiber.Map{
			"error":  "Ticket not active",
			"status": "inactive",
		})
	}

	// Check if event has expired
	var event models.Event
	if err := tx.First(&event, "event_id = ?", ticket.EventID).Error; err == nil {
		if event.DateEnd.Before(time.Now()) {
			tx.Rollback()
			return c.Status(fiber.StatusNotAcceptable).JSON(fiber.Map{
				"error":  "Event has ended",
				"status": "expired",
			})
		}
	}

	ticket.Status = "used"
	if err := tx.Save(&ticket).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check in ticket",
		})
	}

	if err := tx.Model(&models.TicketCategory{}).Where("ticket_category_id = ?", ticket.TicketCategoryID).UpdateColumn("attendant", gorm.Expr("attendant + ?", 1)).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update attendant count",
		})
	}

	// Update event total attendant
	if err := tx.Model(&models.Event{}).Where("event_id = ?", ticket.EventID).UpdateColumn("total_attendant", gorm.Expr("total_attendant + ?", 1)).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update event attendant count",
		})
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to commit transaction",
		})
	}

	// Fetch ticket category and event for response
	var ticketCategory models.TicketCategory
	config.DB.First(&ticketCategory, "ticket_category_id = ?", ticket.TicketCategoryID)

	return c.JSON(fiber.Map{
		"message": "Ticket checked in successfully",
		"status":  "success",
		"ticket": fiber.Map{
			"ticket_id":       ticket.TicketID,
			"code":            ticket.Code,
			"status":          ticket.Status,
			"checked_in_at":   ticket.UpdatedAt,
			"ticket_category": ticketCategory.Name,
			"event_name":      event.Name,
		},
	})
}

func GetTicketCode(c *fiber.Ctx) error {
	TicketID := c.Params("id")
	var Message string = "Ticket code is valid."

	var ticket models.Ticket
	if err := config.DB.First(&ticket, "ticket_id = ?", TicketID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Ticket not found",
		})
	}

	tx := config.DB.Begin()
	if err := tx.Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to start transaction",
		})
	}

	var ticketCategory models.TicketCategory
	if err := tx.First(&ticketCategory, "ticket_category_id = ?", ticket.TicketCategoryID).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch ticket category",
		})
	}

	var event models.Event
	if err := tx.First(&event, "event_id = ?", ticket.EventID).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch event",
		})
	}

	ticketCategoryResponse := ticketCategoryResponse{
		TicketCategoryID: ticketCategory.TicketCategoryID,
		Name:             ticketCategory.Name,
		DateTimeStart:    ticketCategory.DateTimeStart,
		DateTimeEnd:      ticketCategory.DateTimeEnd,
		Price:            ticketCategory.Price,
		Description:      ticketCategory.Description,
	}

	eventResponse := eventResponse{
		EventID:   event.EventID,
		Name:      event.Name,
		Location:  event.Location,
		Venue:     event.Venue,
		City:      event.District,
		DateStart: event.DateStart,
		DateEnd:   event.DateEnd,
		Image:     event.Image,
	}

	// check if ticket is expired
	if !ticket.ExpiresAt.IsZero() && ticket.ExpiresAt.Before(time.Now()) {

		ticket.Code = utils.GenerateTicketCode()
		ticket.ExpiresAt = time.Now().Add(30 * time.Minute)

		if err := tx.Save(&ticket).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update ticket code",
			})
		} else {
			Message = "Ticket code has expired. A new code has been generated."
		}

		// re fetch ticket to ensure we have the latest data
		if err := tx.First(&ticket, "ticket_id = ?", TicketID).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to refetch ticket",
			})
		}
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to commit transaction",
		})
	}

	// Determine computed status
	computedStatus := ticket.Status
	if ticket.Status == "active" && event.DateEnd.Before(time.Now()) {
		computedStatus = "expired"
	}

	// prepare response data
	ticketResponse := ticketResponse{
		Code:           ticket.Code,
		TicketID:       ticket.TicketID,
		TicketCategory: &ticketCategoryResponse,
		Event:          &eventResponse,
		Tag:            ticket.Tag,
		Status:         computedStatus,
		CreatedAt:      ticket.CreatedAt,
	}

	return c.JSON(fiber.Map{
		"message": Message,
		"ticket":  ticketResponse,
	})
}

func UpdateTagTicket(c *fiber.Ctx) error {
	TicketID := c.Params("id")

	//parse body
	var tagMap map[string]interface{}
	if err := c.BodyParser(&tagMap); err != nil {
		log.Printf("Error parsing notification payload: %v", err)
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	newTag, ok := tagMap["tag"].(string)
	if !ok {
		log.Printf("Invalid tag in request")
		return c.Status(400).JSON(fiber.Map{"error": "Invalid tag"})
	}

	var ticket models.Ticket
	if err := config.DB.First(&ticket, "ticket_id = ?", TicketID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Ticket not found",
		})
	}

	ticket.Tag = newTag

	if err := config.DB.Save(&ticket).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update ticket tag",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Ticket tag updated successfully",
		"ticket": fiber.Map{
			"ticket_id": ticket.TicketID,
			"tag":       ticket.Tag,
		},
	})
}

// GetTicketStats - Mengambil statistik tiket user
func GetTicketStats(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	var stats struct {
		Total     int64 `json:"total"`
		Active    int64 `json:"active"`
		Used      int64 `json:"used"`
		Cancelled int64 `json:"cancelled"`
	}

	// Total tickets
	config.DB.Model(&models.Ticket{}).Where("owner_id = ?", user.UserID).Count(&stats.Total)
	// Active tickets
	config.DB.Model(&models.Ticket{}).Where("owner_id = ? AND status = ?", user.UserID, "active").Count(&stats.Active)
	// Used tickets
	config.DB.Model(&models.Ticket{}).Where("owner_id = ? AND status = ?", user.UserID, "used").Count(&stats.Used)
	// Cancelled tickets
	config.DB.Model(&models.Ticket{}).Where("owner_id = ? AND status = ?", user.UserID, "cancelled").Count(&stats.Cancelled)

	return c.JSON(fiber.Map{
		"data": stats,
	})
}