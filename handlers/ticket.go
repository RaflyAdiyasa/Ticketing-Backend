package handlers

import (
	"time"
    "github.com/gofiber/fiber/v2"
    "github.com/Tsaniii18/Ticketing-Backend/config"
    "github.com/Tsaniii18/Ticketing-Backend/models"
    "github.com/Tsaniii18/Ticketing-Backend/utils"
)

func CreateTicketCategory(c *fiber.Ctx) error {
    user := c.Locals("user").(models.User)
    
    var ticketData struct {
        EventID         string  `json:"event_id"` 
        Price           float64 `json:"price"`
        Quota           uint    `json:"quota"`
        Description     string  `json:"description"`
        DateTimeStart   string  `json:"date_time_start"`
        DateTimeEnd     string  `json:"date_time_end"`
    }
    
    if err := c.BodyParser(&ticketData); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request",
        })
    }
    
    // Check if user owns the event
    var event models.Event
    if err := config.DB.First(&event, "event_id = ?", ticketData.EventID).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Event not found",
        })
    }
    
    if event.OwnerID != user.UserID && user.Role != "admin" {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "error": "Not authorized to create ticket for this event",
        })
    }
    
    ticketCategory := models.TicketCategory{
        TicketCategoryID: utils.GenerateTicketCategoryID(),
        EventID:          event.EventID,
        Price:            ticketData.Price,
        Quota:            ticketData.Quota,
        Description:      ticketData.Description,
        CreatedAt:        time.Now(),
        UpdatedAt:        time.Now(),
    }
    
    if err := config.DB.Create(&ticketCategory).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to create ticket category",
        })
    }
    
    return c.Status(fiber.StatusCreated).JSON(ticketCategory)
}

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