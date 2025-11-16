package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Tsaniii18/Ticketing-Backend/config"
	"github.com/Tsaniii18/Ticketing-Backend/models"
	"github.com/Tsaniii18/Ticketing-Backend/utils"
	"github.com/gofiber/fiber/v2"
)

type CreateEventRequest struct {
	Name             string                  `json:"name"`
	DateStart        string                  `json:"date_start"`
	DateEnd          string                  `json:"date_end"`
	Location         string                  `json:"location"`
	City             string                  `json:"city"`
	Description      string                  `json:"description"`
	Category         string                  `json:"category"`
	TicketCategories []TicketCategoryRequest `json:"ticket_categories"`
}

type TicketCategoryRequest struct {
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Quota         uint    `json:"quota"`
	Description   string  `json:"description"`
	DateTimeStart string  `json:"date_time_start"`
	DateTimeEnd   string  `json:"date_time_end"`
}

func CreateEvent(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	name := c.FormValue("name")
	dateStartStr := c.FormValue("date_start")
	dateEndStr := c.FormValue("date_end")
	location := c.FormValue("location")
	city := c.FormValue("city")
	description := c.FormValue("description")
	category := c.FormValue("category")
	ticketCategoriesJSON := c.FormValue("ticket_categories")

	var ticketCategories []TicketCategoryRequest
	if ticketCategoriesJSON != "" {
		if err := json.Unmarshal([]byte(ticketCategoriesJSON), &ticketCategories); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid ticket categories JSON format: " + err.Error(),
			})
		}
	}

	// Validasi required fields
	if name == "" || dateStartStr == "" || dateEndStr == "" || location == "" || city == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing required fields: name, date_start, date_end, location, city",
		})
	}

	// Parse dates
	dateStart, err := time.Parse(time.RFC3339, dateStartStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid date_start format. Use RFC3339 format (e.g., 2024-07-01T18:00:00Z)",
		})
	}

	dateEnd, err := time.Parse(time.RFC3339, dateEndStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid date_end format. Use RFC3339 format (e.g., 2024-07-01T23:00:00Z)",
		})
	}

	// Handle image upload (sama seperti sebelumnya)
	var imageURL, flyerURL string

	imageFile, err := c.FormFile("image")
	if err == nil {
		file, err := imageFile.Open()
		if err == nil {
			defer file.Close()
			folder := fmt.Sprintf("ticketing-app/events/%s/images", user.UserID)
			imageURL, err = config.UploadImage(context.Background(), file, folder)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to upload event image",
				})
			}
		}
	}

	flyerFile, err := c.FormFile("flyer")
	if err == nil {
		file, err := flyerFile.Open()
		if err == nil {
			defer file.Close()
			folder := fmt.Sprintf("ticketing-app/events/%s/flyers", user.UserID)
			flyerURL, err = config.UploadImage(context.Background(), file, folder)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to upload event flyer",
				})
			}
		}
	}

	// Mulai transaction database
	tx := config.DB.Begin()
	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to start transaction",
		})
	}

	// Create event
	event := models.Event{
		EventID:     utils.GenerateEventID(),
		Name:        name,
		OwnerID:     user.UserID,
		Status:      "pending",
		DateStart:   dateStart,
		DateEnd:     dateEnd,
		Location:    location,
		City:        city,
		Description: description,
		Image:       imageURL,
		Flyer:       flyerURL,
		Category:    category,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := tx.Create(&event).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create event: " + err.Error(),
		})
	}

	// Create ticket categories jika ada
	var createdTicketCategories []models.TicketCategory
	if len(ticketCategories) > 0 {
		for _, tcReq := range ticketCategories {
			dateTimeStart, err := time.Parse(time.RFC3339, tcReq.DateTimeStart)
			if err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid date_time_start format in ticket category: " + err.Error(),
				})
			}

			dateTimeEnd, err := time.Parse(time.RFC3339, tcReq.DateTimeEnd)
			if err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid date_time_end format in ticket category: " + err.Error(),
				})
			}

			ticketCategory := models.TicketCategory{
				TicketCategoryID: utils.GenerateTicketCategoryID(),
				EventID:          event.EventID,
				Name:             tcReq.Name,
				Price:            tcReq.Price,
				Quota:            tcReq.Quota,
				Description:      tcReq.Description,
				DateTimeStart:    dateTimeStart,
				DateTimeEnd:      dateTimeEnd,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}

			if err := tx.Create(&ticketCategory).Error; err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to create ticket category: " + err.Error(),
				})
			}

			createdTicketCategories = append(createdTicketCategories, ticketCategory)
			log.Println("append result: ", createdTicketCategories)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to commit transaction: " + err.Error(),
		})
	}

	var eventWithOwner models.Event
	if err := config.DB.Preload("Owner").Preload("TicketCategories").
		Where("event_id = ?", event.EventID).
		First(&eventWithOwner).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load event data: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Event created successfully",
		"event":   eventWithOwner,
	})
}

func GetApprovedEvents(c *fiber.Ctx) error {
	var events []models.Event
	if err := config.DB.Preload("Owner").Preload("TicketCategories").Where("status = ?", "approved").Find(&events).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch events",
		})
	}

	return c.JSON(events)
}

func GetMyEvents(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	var events []models.Event
	if err := config.DB.Preload("Owner").Preload("TicketCategories").
		Where("owner_id = ?", user.UserID).
		Order("created_at DESC").
		Find(&events).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch your events",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Events retrieved successfully",
		"events":  events,
	})
}

func GetEvents(c *fiber.Ctx) error {
	var events []models.Event
	if err := config.DB.Preload("Owner").Preload("TicketCategories").Find(&events).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch events",
		})
	}

	return c.JSON(events)
}

func GetEvent(c *fiber.Ctx) error {
	eventID := c.Params("id")

	var event models.Event
	if err := config.DB.Preload("Owner").Preload("TicketCategories").
		Where("event_id = ?", eventID).
		First(&event).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Event not found",
		})
	}

	return c.JSON(event)
}

func VerifyEvent(c *fiber.Ctx) error {
	eventID := c.Params("id")

	var event models.Event
	if err := config.DB.Preload("Owner").Where("event_id = ?", eventID).First(&event).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Event not found",
		})
	}

	var req struct {
		Status          string `json:"status"`
		ApprovalComment string `json:"approval_comment,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	event.Status = req.Status
	event.ApprovalComment = req.ApprovalComment

	if err := config.DB.Save(&event).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to verify event",
		})
	}

	var eventWithOwner models.Event
	if err := config.DB.Preload("Owner").Preload("TicketCategories").
		Where("event_id = ?", event.EventID).
		First(&eventWithOwner).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load event data: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Event verification updated",
		"event":   event,
	})
}

func UpdateEvent(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	eventID := c.Params("id")

	var event models.Event
	if err := config.DB.First(&event, eventID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Event not found",
		})
	}

	// Check ownership
	if event.OwnerID != user.UserID && user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Not authorized to update this event",
		})
	}

	var updateData map[string]interface{}
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	if err := config.DB.Model(&event).Updates(updateData).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update event",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Event updated successfully",
		"event":   event,
	})
}

func DeleteEvent(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	eventID := c.Params("id")

	var event models.Event
	if err := config.DB.First(&event, eventID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Event not found",
		})
	}

	// Check ownership or admin
	if event.OwnerID != user.UserID && user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Not authorized to delete this event",
		})
	}

	if err := config.DB.Delete(&event).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete event",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Event deleted successfully",
	})
}

func GetEventsPopular(c *fiber.Ctx) error {

	var events []models.Event
	if err := config.DB.Preload("Owner").Preload("TicketCategories").
		Where("status = ?", "approved").
		Order("total_tickets_sold DESC").
		Limit(4).
		Find(&events).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch events",
		})
	}

	return c.JSON(fiber.Map{
		"events": events,
	})
}
