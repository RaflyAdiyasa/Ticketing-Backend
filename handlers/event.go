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
	if err := config.DB.Preload("TicketCategories").Where("event_id = ?", eventID).First(&event).Error; err != nil {
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

	// Check if event can be edited (only pending or rejected)
	if event.Status != "pending" && event.Status != "rejected" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Event can only be edited when status is pending or rejected",
		})
	}

	// Parse form data
	name := c.FormValue("name")
	dateStartStr := c.FormValue("date_start")
	dateEndStr := c.FormValue("date_end")
	location := c.FormValue("location")
	city := c.FormValue("city")
	description := c.FormValue("description")
	category := c.FormValue("category")
	ticketCategoriesJSON := c.FormValue("ticket_categories")

	// Mulai transaction
	tx := config.DB.Begin()
	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to start transaction",
		})
	}

	// Update basic event info
	updateData := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if name != "" {
		updateData["name"] = name
		event.Name = name
	}
	if location != "" {
		updateData["location"] = location
		event.Location = location
	}
	if city != "" {
		updateData["city"] = city
		event.City = city
	}
	if description != "" {
		updateData["description"] = description
		event.Description = description
	}
	if category != "" {
		updateData["category"] = category
		event.Category = category
	}

	// Parse dates if provided
	if dateStartStr != "" {
		dateStart, err := time.Parse(time.RFC3339, dateStartStr)
		if err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid date_start format",
			})
		}
		updateData["date_start"] = dateStart
		event.DateStart = dateStart
	}

	if dateEndStr != "" {
		dateEnd, err := time.Parse(time.RFC3339, dateEndStr)
		if err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid date_end format",
			})
		}
		updateData["date_end"] = dateEnd
		event.DateEnd = dateEnd
	}

	// Handle image upload
	imageFile, err := c.FormFile("image")
	if err == nil {
		file, err := imageFile.Open()
		if err == nil {
			defer file.Close()
			folder := fmt.Sprintf("ticketing-app/events/%s/images", user.UserID)
			imageURL, err := config.UploadImage(context.Background(), file, folder)
			if err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to upload event image",
				})
			}
			updateData["image"] = imageURL
			event.Image = imageURL
		}
	}

	// Handle flyer upload
	flyerFile, err := c.FormFile("flyer")
	if err == nil {
		file, err := flyerFile.Open()
		if err == nil {
			defer file.Close()
			folder := fmt.Sprintf("ticketing-app/events/%s/flyers", user.UserID)
			flyerURL, err := config.UploadImage(context.Background(), file, folder)
			if err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to upload event flyer",
				})
			}
			updateData["flyer"] = flyerURL
			event.Flyer = flyerURL
		}
	}

	// Update event
	if err := tx.Model(&event).Updates(updateData).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update event: " + err.Error(),
		})
	}

	// Handle ticket categories update if provided
	if ticketCategoriesJSON != "" {
		var ticketCategories []TicketCategoryRequest
		if err := json.Unmarshal([]byte(ticketCategoriesJSON), &ticketCategories); err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid ticket categories JSON format",
			})
		}

		// Delete existing ticket categories
		if err := tx.Where("event_id = ?", event.EventID).Delete(&models.TicketCategory{}).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to delete existing ticket categories",
			})
		}

		// Create new ticket categories
		for _, tcReq := range ticketCategories {
			dateTimeStart, err := time.Parse(time.RFC3339, tcReq.DateTimeStart)
			if err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid date_time_start format",
				})
			}

			dateTimeEnd, err := time.Parse(time.RFC3339, tcReq.DateTimeEnd)
			if err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid date_time_end format",
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
					"error": "Failed to create ticket category",
				})
			}
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to commit transaction",
		})
	}

	// Reload event with relationships
	var updatedEvent models.Event
	if err := config.DB.Preload("Owner").Preload("TicketCategories").
		Where("event_id = ?", event.EventID).
		First(&updatedEvent).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load updated event data",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Event updated successfully",
		"event":   updatedEvent,
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

type EventReportResponse struct {
	Event            models.Event          `json:"event"`
	PurchaseData     []TicketCategoryStats `json:"purchase_data"`
	CheckinData      []TicketCategoryStats `json:"checkin_data"`
	AttendantData    []TicketCategoryStats `json:"attendant_data"`
	TotalIncome      float64               `json:"total_income"`
	TotalTicketsSold int                   `json:"total_tickets_sold"`
	TotalCheckins    int                   `json:"total_checkins"`
}

type TicketCategoryStats struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func GetEventReport(c *fiber.Ctx) error {
	eventID := c.Params("id")
	user := c.Locals("user").(models.User)

	var event models.Event
	if err := config.DB.Preload("Owner").Preload("TicketCategories").
		Where("event_id = ?", eventID).
		First(&event).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Event not found",
		})
	}

	// Check ownership
	if event.OwnerID != user.UserID && user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Not authorized to view this report",
		})
	}

	// Get ticket sales data per category
	var purchaseData []TicketCategoryStats
	var checkinData []TicketCategoryStats
	var attendantData []TicketCategoryStats

	// Calculate total sold and checked-in tickets for the entire event
	var totalSold int
	var totalCheckedIn int

	// Calculate purchase data and income per category
	for _, ticketCategory := range event.TicketCategories {
		var soldCount int64
		var checkedInCount int64

		// Count sold tickets for this category (status = active)
		config.DB.Model(&models.Ticket{}).
			Where("ticket_category_id = ? AND status IN (?, ?)", ticketCategory.TicketCategoryID, "active", "used").
			Count(&soldCount)

		// Count checked-in tickets for this category
		config.DB.Model(&models.Ticket{}).
			Where("ticket_category_id = ? AND status = ?", ticketCategory.TicketCategoryID, "used").
			Count(&checkedInCount)

		purchaseData = append(purchaseData, TicketCategoryStats{
			Name:  ticketCategory.Name,
			Value: int(soldCount),
		})

		checkinData = append(checkinData, TicketCategoryStats{
			Name:  ticketCategory.Name,
			Value: int(checkedInCount),
		})

		attendantData = append(attendantData, TicketCategoryStats{
			Name:  ticketCategory.Name,
			Value: int(ticketCategory.Attendant),
		})

		totalSold += int(soldCount)
		totalCheckedIn += int(checkedInCount)
	}

	// Calculate additional metrics
	soldPercentage := "0%"
	if event.TotalTicketsSold > 0 && len(event.TicketCategories) > 0 {
		// Calculate based on total quota
		totalQuota := uint(0)
		for _, tc := range event.TicketCategories {
			totalQuota += tc.Quota
		}
		if totalQuota > 0 {
			percentage := (float64(event.TotalTicketsSold) / float64(totalQuota)) * 100
			soldPercentage = fmt.Sprintf("%.1f%%", percentage)
		}
	}

	attendanceRate := "0%"
	if event.TotalTicketsSold > 0 {
		rate := (float64(event.TotalAttendant) / float64(event.TotalTicketsSold)) * 100
		attendanceRate = fmt.Sprintf("%.1f%%", rate)
	}

	// Create metrics
	metrics := fiber.Map{
		"total_attendant":    event.TotalAttendant,
		"total_tickets_sold": event.TotalTicketsSold,
		"total_sales":        event.TotalSales,
		"sold_percentage":    soldPercentage,
		"attendance_rate":    attendanceRate,
	}

	report := EventReportResponse{
		Event:            event,
		PurchaseData:     purchaseData,
		CheckinData:      checkinData,
		AttendantData:    attendantData,
		TotalIncome:      event.TotalSales,
		TotalTicketsSold: int(event.TotalTicketsSold),
		TotalCheckins:    int(event.TotalAttendant),
	}

	return c.JSON(fiber.Map{
		"report":  report,
		"metrics": metrics,
	})
}

func DownloadEventReport(c *fiber.Ctx) error {
	eventID := c.Params("id")
	user := c.Locals("user").(models.User)

	var event models.Event
	if err := config.DB.Preload("Owner").Preload("TicketCategories").
		Where("event_id = ?", eventID).
		First(&event).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Event not found",
		})
	}

	// Check ownership
	if event.OwnerID != user.UserID && user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Not authorized to download this report",
		})
	}

	// Generate CSV report
	csvData := "Kategori Tiket,Tiket Terjual,Tiket Check-in,Pendapatan Kategori\n"

	// Hitung data per kategori
	for _, ticketCategory := range event.TicketCategories {
		var soldCount int64
		var checkedInCount int64

		config.DB.Model(&models.Ticket{}).
			Where("ticket_category_id = ? AND status = ?", ticketCategory.TicketCategoryID, "paid").
			Count(&soldCount)

		config.DB.Model(&models.Ticket{}).
			Where("ticket_category_id = ? AND status = ?", ticketCategory.TicketCategoryID, "used").
			Count(&checkedInCount)

		categoryIncome := float64(soldCount) * ticketCategory.Price

		csvData += fmt.Sprintf("%s,%d,%d,%.2f\n",
			ticketCategory.Name, soldCount, checkedInCount, categoryIncome)
	}

	// Tambahkan total keseluruhan
	csvData += fmt.Sprintf("\nTotal Keseluruhan\n")
	csvData += fmt.Sprintf("Total Tiket Terjual:,%d\n", event.TotalTicketsSold)
	csvData += fmt.Sprintf("Total Check-in:,%d\n", event.TotalAttendant)
	csvData += fmt.Sprintf("Total Pendapatan:,.2f\n", event.TotalSales)

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=report-%s.csv", eventID))

	return c.SendString(csvData)
}
