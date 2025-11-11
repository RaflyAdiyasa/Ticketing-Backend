package handlers

import (
    "context"
    "time"
    "fmt"
    "github.com/gofiber/fiber/v2"
    "github.com/Tsaniii18/Ticketing-Backend/config"
    "github.com/Tsaniii18/Ticketing-Backend/models"
    "github.com/Tsaniii18/Ticketing-Backend/utils"
)

func CreateEvent(c *fiber.Ctx) error {
    user := c.Locals("user").(models.User)
    
    // Parse form data
    name := c.FormValue("name")
    dateStartStr := c.FormValue("date_start")
    dateEndStr := c.FormValue("date_end")
    location := c.FormValue("location")
    description := c.FormValue("description")
    category := c.FormValue("category")

    // Parse dates
    dateStart, err := time.Parse(time.RFC3339, dateStartStr)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid date_start format",
        })
    }

    dateEnd, err := time.Parse(time.RFC3339, dateEndStr)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid date_end format",
        })
    }

    // Handle image upload
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

    event := models.Event{
        EventID:     utils.GenerateEventID(),
        Name:        name,
        OwnerID:     user.UserID,
        Status:      "pending",
        DateStart:   dateStart,
        DateEnd:     dateEnd,
        Location:    location,
        Description: description,
        Image:       imageURL,
        Flyer:       flyerURL,
        Category:    category,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    
    if err := config.DB.Create(&event).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to create event",
        })
    }
    
    return c.Status(fiber.StatusCreated).JSON(event)
}

func GetApprovedEvents(c *fiber.Ctx) error {
    var events []models.Event
    if err := config.DB.Preload("Owner").Where("status = ?", "approved").Find(&events).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to fetch events",
        })
    }
    
    return c.JSON(events)
}

func GetEvents(c *fiber.Ctx) error {
    var events []models.Event
    if err := config.DB.Preload("Owner").Find(&events).Error; err != nil {
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

func VerifyEvent(c *fiber.Ctx) error {
    eventID := c.Params("id")
    
    var event models.Event
    if err := config.DB.Where("event_id = ?", eventID).First(&event).Error; err != nil {
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
    
    return c.JSON(fiber.Map{
        "message": "Event verification updated",
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