package handlers

import (
	"time"

	"github.com/Tsaniii18/Ticketing-Backend/config"
	"github.com/Tsaniii18/Ticketing-Backend/models"
	"github.com/Tsaniii18/Ticketing-Backend/utils"
	"github.com/gofiber/fiber/v2"
)

func GetTickets(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	var tickets []models.Ticket
	if err := config.DB.Preload("Event").Preload("TicketCategory").Where("owner_id = ?", user.UserID).Find(&tickets).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch tickets",
		})
	}

	return c.JSON(tickets)
}

func CheckInTicket(c *fiber.Ctx) error {
	ticketID := c.Params("id")

	var ticket models.Ticket
	if err := config.DB.Preload("Event").First(&ticket, ticketID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Ticket not found",
		})
	}

	if ticket.Status == "used" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Ticket already used",
		})
	}

	ticket.Status = "used"
	if err := config.DB.Save(&ticket).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check in ticket",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Ticket checked in successfully",
		"ticket":  ticket,
	})
}

type ticketResponse struct {
	Code           string                  `json:"code"`
	TicketID       string                  `json:"ticket_id"`
	TicketCategory *ticketCategoryResponse `json:"ticket_category"`
	Event          *eventResponse          `json:"event"`
}

type ticketCategoryResponse struct {
	DateTimeStart time.Time `json:"date_time_start"`
	DateTimeEnd   time.Time `json:"date_time_end"`
	Price         float64   `json:"price"`
	Description   string    `json:"description"`
}

type eventResponse struct {
	Name     string `json:"name"`
	Location string `json:"location"`
	City     string `json:"city"`
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
		DateTimeStart: ticketCategory.DateTimeStart,
		DateTimeEnd:   ticketCategory.DateTimeEnd,
		Price:         ticketCategory.Price,
		Description:   ticketCategory.Description,
	}

	eventResponse := eventResponse{
		Name:     event.Name,
		Location: event.Location,
		City:     event.City,
	}

	// prepare response data
	ticketResponse := ticketResponse{
		Code:           ticket.Code,
		TicketID:       ticket.TicketID,
		TicketCategory: &ticketCategoryResponse,
		Event:          &eventResponse,
	}

	// check if ticket is expired
	if !ticket.ExpiresAt.IsZero() && ticket.ExpiresAt.Before(time.Now()) {

		ticket.Code = utils.GenerateTicketCode()
		ticket.ExpiresAt = time.Now().Add(3 * time.Minute)

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

	return c.JSON(fiber.Map{
		"message": Message,
		"ticket":  ticketResponse,
	})
}
