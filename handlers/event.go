package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/Tsaniii18/Ticketing-Backend/config"
	"github.com/Tsaniii18/Ticketing-Backend/models"
	"github.com/Tsaniii18/Ticketing-Backend/utils"
	"github.com/gofiber/fiber/v2"
)

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
	venue := c.FormValue("venue")
	district := c.FormValue("district")
	description := c.FormValue("description")
	rules := c.FormValue("rules") // Tambahkan rules
	category := c.FormValue("category")
	childCategory := c.FormValue("child_category")
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
	if name == "" || dateStartStr == "" || dateEndStr == "" || location == "" || venue == "" || district == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing required fields: name, date_start, date_end, location, venue, district",
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

	// Mulai transaction database
	tx := config.DB.Begin()
	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to start transaction",
		})
	}

	// Create event
	event := models.Event{
		EventID:       utils.GenerateEventID(),
		Name:          name,
		OwnerID:       user.UserID,
		Status:        "pending",
		DateStart:     dateStart,
		DateEnd:       dateEnd,
		Location:      location,
		Venue:         venue,
		District:      district,
		Description:   description,
		Rules:         rules,
		TotalLikes:    0,
		Image:         imageURL,
		Flyer:         flyerURL,
		Category:      category,
		ChildCategory: childCategory,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
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
			var ticketName models.TicketCategory
			if err := tx.Model(&ticketName).Where("name = ? && event_id = ?", tcReq.Name, event.EventID).First(&ticketName).Error; err == nil {
				tx.Rollback()
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Duplicate ticket category name : " + tcReq.Name,
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

	// ScheduleEventEnd(config.DB, event)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Event created successfully",
		"event":   eventWithOwner,
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
	venue := c.FormValue("venue")
	district := c.FormValue("district")
	description := c.FormValue("description")
	rules := c.FormValue("rules") // Tambahkan rules
	category := c.FormValue("category")
	childCategory := c.FormValue("child_category")
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
	if venue != "" {
		updateData["venue"] = venue
		event.Venue = venue
	}
	if district != "" {
		updateData["district"] = district
		event.District = district
	}
	if description != "" {
		updateData["description"] = description
		event.Description = description
	}
	if rules != "" {
		updateData["rules"] = rules // Tambahkan rules
		event.Rules = rules
	}
	if category != "" {
		updateData["category"] = category
		event.Category = category
	}
	if childCategory != "" {
		updateData["child_category"] = childCategory
		event.ChildCategory = childCategory
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

	updateData["status"] = "pending"

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

func GetApprovedEvents(c *fiber.Ctx) error {
	var events []models.Event
	if err := config.DB.Preload("Owner").Preload("TicketCategories").Where("status IN (?)", []string{"active", "approved", "ended"}).Find(&events).Error; err != nil {
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
	if err := config.DB.Preload("Owner").Preload("TicketCategories").Preload("LikedBy").Find(&events).Error; err != nil {
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

	if event.Status == "approved" {
		ScheduleEventEnd(config.DB, event)
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
	if err := config.DB.First(&event, "event_id", eventID).Error; err != nil {
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
		Order("total_likes DESC").
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
	TotalLikes       uint                  `json:"total_likes"`
	TotalQuota       int                   `json:"total_quota"`
}

type TicketCategoryStats struct {
	Name       string  `json:"name"`
	Value      int     `json:"value"`
	Quota      int     `json:"quota"`
	Price      float64 `json:"price"`
	Percentage float64 `json:"percentage"`
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

	// Initialize data arrays - gunakan make untuk memastikan tidak nil
	purchaseData := make([]TicketCategoryStats, 0)
	checkinData := make([]TicketCategoryStats, 0)
	attendantData := make([]TicketCategoryStats, 0)

	// Calculate totals
	var totalSold int64 = 0
	var totalCheckedIn int64 = 0
	var totalQuota int = 0
	var totalIncome float64 = 0

	// Log untuk debug
	log.Printf("Event ID: %s, TicketCategories count: %d", eventID, len(event.TicketCategories))

	// Calculate data per category - langsung dari TicketCategory
	for _, ticketCategory := range event.TicketCategories {
		// Langsung ambil dari field Sold dan Attendant di TicketCategory
		soldCount := int64(ticketCategory.Sold)
		checkedInCount := int64(ticketCategory.Attendant)

		// Log untuk debug setiap kategori
		log.Printf("Category: %s, Sold: %d, Attendant: %d, Quota: %d, Price: %.2f",
			ticketCategory.Name, soldCount, checkedInCount, ticketCategory.Quota, ticketCategory.Price)

		// Calculate percentage of sold tickets for this category
		soldPercentage := float64(0)
		if ticketCategory.Quota > 0 {
			soldPercentage = (float64(soldCount) / float64(ticketCategory.Quota)) * 100
		}

		// Calculate percentage of check-ins for this category
		checkinPercentage := float64(0)
		if soldCount > 0 {
			checkinPercentage = (float64(checkedInCount) / float64(soldCount)) * 100
		}

		// Calculate income for this category
		categoryIncome := float64(soldCount) * ticketCategory.Price

		purchaseData = append(purchaseData, TicketCategoryStats{
			Name:       ticketCategory.Name,
			Value:      int(soldCount),
			Quota:      int(ticketCategory.Quota),
			Price:      ticketCategory.Price,
			Percentage: soldPercentage,
		})

		checkinData = append(checkinData, TicketCategoryStats{
			Name:       ticketCategory.Name,
			Value:      int(checkedInCount),
			Quota:      int(soldCount),
			Price:      ticketCategory.Price,
			Percentage: checkinPercentage,
		})

		attendantData = append(attendantData, TicketCategoryStats{
			Name:       ticketCategory.Name,
			Value:      int(checkedInCount),
			Quota:      int(ticketCategory.Quota),
			Price:      ticketCategory.Price,
			Percentage: checkinPercentage,
		})

		totalSold += soldCount
		totalCheckedIn += checkedInCount
		totalQuota += int(ticketCategory.Quota)
		totalIncome += categoryIncome
	}

	// Log totals untuk debug
	log.Printf("Totals - Sold: %d, CheckedIn: %d, Quota: %d, Income: %.2f",
		totalSold, totalCheckedIn, totalQuota, totalIncome)

	// Calculate overall metrics
	soldPercentage := "0%"
	if totalQuota > 0 {
		percentage := (float64(totalSold) / float64(totalQuota)) * 100
		soldPercentage = fmt.Sprintf("%.1f%%", percentage)
	}

	attendanceRate := "0%"
	if totalSold > 0 {
		rate := (float64(totalCheckedIn) / float64(totalSold)) * 100
		attendanceRate = fmt.Sprintf("%.1f%%", rate)
	}

	// Create metrics map
	metrics := fiber.Map{
		"total_attendant":    totalCheckedIn,
		"total_tickets_sold": totalSold,
		"total_sales":        totalIncome,
		"total_quota":        totalQuota,
		"sold_percentage":    soldPercentage,
		"attendance_rate":    attendanceRate,
	}

	report := EventReportResponse{
		Event:            event,
		PurchaseData:     purchaseData,
		CheckinData:      checkinData,
		AttendantData:    attendantData,
		TotalIncome:      totalIncome,
		TotalLikes:       event.TotalLikes,
		TotalTicketsSold: int(totalSold),
		TotalCheckins:    int(totalCheckedIn),
		TotalQuota:       totalQuota,
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
	csvData := "Kategori_Tiket,Harga_Tiket,Kuota_Tiket,Tiket_Terjual,Persentase_Terjual,Tiket_Check_in,Persentase_Check_in,Pendapatan_Kategori\n"

	var grandTotalSold int64 = 0
	var grandTotalCheckedIn int64 = 0
	var grandTotalQuota int64 = 0
	var grandTotalIncome float64 = 0

	// Calculate data per category - langsung dari TicketCategory
	for _, ticketCategory := range event.TicketCategories {
		// Langsung ambil dari field Sold dan Attendant di TicketCategory
		soldCount := int64(ticketCategory.Sold)
		checkedInCount := int64(ticketCategory.Attendant)

		// Log untuk debug
		log.Printf("Download Report - Category: %s, Sold: %d, Attendant: %d",
			ticketCategory.Name, soldCount, checkedInCount)

		// Handle division by zero
		soldPercentage := float64(0)
		if ticketCategory.Quota > 0 {
			soldPercentage = (float64(soldCount) / float64(ticketCategory.Quota)) * 100
		}

		checkInPercentage := float64(0)
		if soldCount > 0 {
			checkInPercentage = (float64(checkedInCount) / float64(soldCount)) * 100
		}

		categoryIncome := float64(soldCount) * ticketCategory.Price

		csvData += fmt.Sprintf("%s,%.0f,%d,%d,%.2f%%,%d,%.2f%%,%.0f\n",
			ticketCategory.Name,
			ticketCategory.Price,
			ticketCategory.Quota,
			soldCount,
			soldPercentage,
			checkedInCount,
			checkInPercentage,
			categoryIncome)

		grandTotalSold += soldCount
		grandTotalCheckedIn += checkedInCount
		grandTotalQuota += int64(ticketCategory.Quota)
		grandTotalIncome += categoryIncome
	}

	// Calculate overall percentages
	overallSoldPercentage := float64(0)
	if grandTotalQuota > 0 {
		overallSoldPercentage = (float64(grandTotalSold) / float64(grandTotalQuota)) * 100
	}

	overallCheckInPercentage := float64(0)
	if grandTotalSold > 0 {
		overallCheckInPercentage = (float64(grandTotalCheckedIn) / float64(grandTotalSold)) * 100
	}

	// Add summary section
	csvData += "\n"
	csvData += fmt.Sprintf("RINGKASAN LAPORAN EVENT: %s\n", event.Name)
	csvData += fmt.Sprintf("Tanggal Event:,%s - %s\n", event.DateStart.Format("02-01-2006"), event.DateEnd.Format("02-01-2006"))
	csvData += fmt.Sprintf("Lokasi:,%s - %s\n", event.Venue, event.Location)
	csvData += "\n"
	csvData += fmt.Sprintf("Total Kuota Tiket:,%d\n", grandTotalQuota)
	csvData += fmt.Sprintf("Total Tiket Terjual:,%d (%.2f%%)\n", grandTotalSold, overallSoldPercentage)
	csvData += fmt.Sprintf("Total Check-in:,%d (%.2f%%)\n", grandTotalCheckedIn, overallCheckInPercentage)
	csvData += fmt.Sprintf("Total Pendapatan:,Rp %.0f\n", grandTotalIncome)
	csvData += fmt.Sprintf("Total Like:,%d\n", event.TotalLikes)

	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=laporan_%s_%s.csv", event.Name, time.Now().Format("2006-01-02")))

	return c.SendString(csvData)
}

func AddLike(c *fiber.Ctx) error {
	eventID := c.Params("id")
	user := c.Locals("user").(models.User)

	if user.Role == "admin" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User only",
		})
	}

	if user.Role == "organizer" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User only",
		})
	}

	var event models.Event
	if err := config.DB.Where("event_id = ?", eventID).First(&event).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Event not found",
		})
	}

	var existingLike models.EventLike
	if err := config.DB.Where("user_id = ? AND event_id = ?", user.UserID, eventID).First(&existingLike).Error; err == nil {

		if err := config.DB.Delete(&existingLike).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to unlike event: " + err.Error(),
			})
		}

		if event.TotalLikes > 0 {
			event.TotalLikes--
		}

		if err := config.DB.Save(&event).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update like count: " + err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message":          "Unliked",
			"action":           "unlike",
			"event_total_like": event.TotalLikes,
		})
	}

	eventLike := models.EventLike{
		UserID:  user.UserID,
		EventID: eventID,
	}

	if err := config.DB.Create(&eventLike).Error; err != nil {
		log.Printf("Error creating like: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to like event: " + err.Error(),
		})
	}

	event.TotalLikes++

	if err := config.DB.Save(&event).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update like count: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":          "Liked",
		"action":           "like",
		"event_total_like": event.TotalLikes,
	})
}

func MyLikedEvent(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	var userFigure models.User
	if err := config.DB.Model(&models.User{}).Preload("LikedEvents").Where("user_id = ?", user.UserID).Find(&userFigure).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Users event not found",
		})
	}

	var number_of_likes = len(userFigure.LikedEvents)
	return c.JSON(fiber.Map{
		"message":         "Successfully",
		"number_of_likes": number_of_likes,
		"liked_event":     userFigure.LikedEvents,
	})

}

func ScheduleEventEnd(db *gorm.DB, event models.Event) {
	durationEnd := time.Until(event.DateEnd)
	durationStart := time.Until(event.DateStart)

	go func() {
		if durationStart > 0 {
			log.Println("Event " + event.Name + " ---START in " + fmt.Sprintf("%.2f", durationStart.Hours()) + " hours")
			time.Sleep(durationStart)
		}

		if err := db.Model(&event).
			Where("event_id = ?", event.EventID).
			Update("status", "active").Error; err != nil {

			log.Println("Failed to update event start:", err)
		}
	}()

	go func() {
		if durationEnd > 0 {
			log.Println("Event " + event.Name + " ---END in " + fmt.Sprintf("%.2f", durationEnd.Hours()) + " hours")
			time.Sleep(durationEnd)
		}

		if err := db.Model(&event).
			Where("event_id = ?", event.EventID).
			Update("status", "ended").Error; err != nil {

			log.Println("Failed to update event end:", err)
		}
	}()
}

func InitialScheduleEventEnd(db *gorm.DB) error {
	var events []models.Event
	if err := db.Model(&models.Event{}).Find(&events).Error; err != nil {
		return err
	}

	var count uint = 0
	for _, event := range events {
		if event.Status == "approved" || event.Status == "active" {
			ScheduleEventEnd(db, event)
			count++
		}

	}
	log.Println(" --  Start Goroutine for " + strconv.Itoa(int(count)) + " events")
	return nil
}

func GetEventCategories(c *fiber.Ctx) error {

	var allCategory []models.EventCategory
	if err := config.DB.Model(&allCategory).Preload("ChildEventCategory").Find(&allCategory).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Event not found",
		})
	}

	return c.JSON(fiber.Map{
		"event_category": allCategory,
	})
}

type ChildEventCategoryRequest struct {
	ParentCategoryID       string `json:"event_category_id"`
	ChildEventCategoryName string `json:"child_event_category_name"`
}

func AddEventCategory(c *fiber.Ctx) error {
	var nameReq map[string]interface{}

	if err := c.BodyParser(&nameReq); err != nil {
		log.Printf("Error parsing : %v", err)
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	categoryName, _ := nameReq["category_event"].(string)

	if categoryName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Input is Empty : ",
		})
	}

	eventCategory := models.EventCategory{
		EventCategoryID:   utils.GenerateEventCategoryID(),
		EventCategoryName: categoryName,
	}

	if err := config.DB.Model(&models.EventCategory{}).Create(&eventCategory).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add event category",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Success create event categories: s" + categoryName,
	})

}

func AddSubEventCategory(c *fiber.Ctx) error {
	var nameReq map[string]interface{}

	if err := c.BodyParser(&nameReq); err != nil {
		log.Printf("Error parsing : %v", err)
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	categoryName, _ := nameReq["child_category_event"].(string)
	parentCategoryName, _ := nameReq["category_event"].(string)

	if categoryName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Input is Empty  ",
		})
	}

	if parentCategoryName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Input is Empty  ",
		})
	}
	var checkParent models.EventCategory
	if err := config.DB.Model(&models.EventCategory{}).Where("event_category_name = ?", parentCategoryName).First(&checkParent).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "main category not found",
		})

	}

	eventSubCategory := models.ChildEventCategory{
		ChildEventCategoryID:   utils.GenerateChildEventCategoryID(),
		ParentCategoryID:       checkParent.EventCategoryID,
		ChildEventCategoryName: categoryName,
		ParentCategoryName:     parentCategoryName,
	}

	if err := config.DB.Model(&models.ChildEventCategory{}).Create(&eventSubCategory).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add event category",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Success create event categories: " + categoryName,
	})

}

func AddEventCategoryAll(c *fiber.Ctx) error {
	childCategoryName := c.FormValue("child_categories")
	EventCategoryName := c.FormValue("event_category")

	var childCategories []ChildEventCategoryRequest
	if childCategoryName != "" {
		if err := json.Unmarshal([]byte(childCategoryName), &childCategories); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid ticket categories JSON format: " + err.Error(),
			})
		}
	}

	tx := config.DB.Begin()
	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to start transaction",
		})
	}

	var EventCategory models.EventCategory
	if err := config.DB.Model(&models.EventCategory{}).Where(&EventCategory, "event_category_name = ?", EventCategoryName).Error; err != nil {

		eventCategory := models.EventCategory{
			EventCategoryID:   utils.GenerateEventCategoryID(),
			EventCategoryName: EventCategoryName,
		}

		if err := tx.Model(&eventCategory).Create(&eventCategory).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create event category: " + err.Error(),
			})
		}
	}

	if len(childCategories) > 0 {
		for _, tcReq := range childCategories {

			childCategory := models.ChildEventCategory{
				ChildEventCategoryID:   utils.GenerateChildEventCategoryID(),
				ParentCategoryID:       EventCategory.EventCategoryID,
				ChildEventCategoryName: tcReq.ChildEventCategoryName,
			}

			if err := tx.Model(&childCategory).Where("child_event_category_name = ?", tcReq.ChildEventCategoryName).First(&childCategory).Error; err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Duplicate child event category: " + err.Error(),
				})
			}

			if err := tx.Model(&childCategory).Create(&childCategory).Error; err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to create child event category: " + err.Error(),
				})
			}

		}

	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to commit transaction: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Success create event categories",
	})

}

func DeleteSubCategoryEvent(c *fiber.Ctx) error {
	var deleteReq map[string]interface{}

	if err := c.BodyParser(&deleteReq); err != nil {
		log.Printf("Error parsing : %v", err)
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	deleteName, _ := deleteReq["child_category_event"].(string)

	if err := config.DB.Model(&models.ChildEventCategory{}).Delete(&models.ChildEventCategory{}, "child_event_category_name = ?", deleteName).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete category: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Success delete event categories",
	})

}

func DeleteCategoryEvent(c *fiber.Ctx) error {
	var deleteReq map[string]interface{}

	if err := c.BodyParser(&deleteReq); err != nil {
		log.Printf("Error parsing : %v", err)
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	deleteName, _ := deleteReq["category_event"].(string)

	var categoryNameDelete models.EventCategory
	if err := config.DB.Model(&categoryNameDelete).Preload("ChildEventCategory").Where("event_category_name = ?", deleteName).Find(&categoryNameDelete).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Category event not found",
		})
	}

	if len(categoryNameDelete.ChildEventCategory) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "You have to delete the child categories first ",
		})
	}

	if err := config.DB.Model(&models.EventCategory{}).Delete(&categoryNameDelete, "event_category_id = ?", categoryNameDelete.EventCategoryID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete category: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Success delete event categories",
		"Deleted": categoryNameDelete.EventCategoryName,
	})

}

func InitializeDefaultCategories() error {
	categories := map[string][]string{
		"Hiburan":              {"Musik", "Konser", "Festival", "Stand Up Comedy", "Film", "Teater", "K-Pop", "Dance Performance"},
		"Teknologi":            {"Konferensi Teknologi", "Workshop IT", "Startup", "Software Development", "Artificial Intelligence", "Data Science", "Cybersecurity", "Gaming & Esports"},
		"Edukasi":              {"Seminar", "Workshop", "Pelatihan", "Webinar", "Bootcamp", "Kelas Online", "Literasi Digital", "Kelas Bisnis"},
		"Olahraga":             {"Marathon", "Fun Run", "Sepak Bola", "Badminton", "Gym & Fitness", "Yoga", "Esport", "Cycling Event", "Horse Race"},
		"Bisnis & Profesional": {"Konferensi Bisnis", "Networking", "Karir", "Entrepreneurship", "Leadership", "Startup Meetup", "Investor & Pitching"},
		"Seni & Budaya":        {"Pameran Seni", "Pentas Budaya", "Fotografi", "Seni Rupa", "Crafting", "Pameran Museum", "Fashion Show"},
		"Komunitas":            {"Kegiatan Relawan", "Kegiatan Sosial", "Gathering Komunitas", "Komunitas Hobi", "Meetup", "Charity Event"},
		"Kuliner":              {"Festival Kuliner", "Food Tasting", "Workshop Memasak", "Street Food Event"},
		"Kesehatan":            {"Seminar Kesehatan", "Medical Check Event", "Workshop Kesehatan Mental", "Donor Darah"},
		"Agama & Spiritual":    {"Kajian", "Retreat", "Pengajian", "Event Keagamaan", "Meditasi"},
		"Travel & Outdoor":     {"Camping", "Hiking", "Trip Wisata", "Outdoor Gathering", "Photography Trip"},
		"Keluarga & Anak":      {"Family Gathering", "Event Anak", "Workshop Parenting", "Pentas Anak"},
		"Fashion & Beauty":     {"Fashion Expo", "Beauty Class", "Makeup Workshop", "Brand Launching"},
	}

	var count int64
	if err := config.DB.Model(&models.EventCategory{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check existing categories: %w", err)
	}

	if count > 0 {
		log.Println("Default event categories already exist")
		return nil
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for categoryName, subcategories := range categories {

		var existingCategory models.EventCategory
		err := tx.Where("event_category_name = ?", categoryName).First(&existingCategory).Error

		var categoryID string

		if err != nil {

			categoryID = utils.GenerateEventCategoryID()
			eventCategory := models.EventCategory{
				EventCategoryID:   categoryID,
				EventCategoryName: categoryName,
			}

			if err := tx.Create(&eventCategory).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to create category %s: %w", categoryName, err)
			}
			log.Printf("Created main category: %s", categoryName)
		} else {

			categoryID = existingCategory.EventCategoryID
			log.Printf("Category already exists: %s", categoryName)
		}

		for _, subcategoryName := range subcategories {

			var existingSubcategory models.ChildEventCategory
			err := tx.Where("child_event_category_name = ? AND parent_category_id = ?", subcategoryName, categoryID).First(&existingSubcategory).Error

			if err != nil {

				childEventCategory := models.ChildEventCategory{
					ChildEventCategoryID:   utils.GenerateChildEventCategoryID(),
					ParentCategoryID:       categoryID,
					ParentCategoryName:     categoryName,
					ChildEventCategoryName: subcategoryName,
				}

				if err := tx.Create(&childEventCategory).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("failed to create subcategory %s for category %s: %w", subcategoryName, categoryName, err)
				}
				log.Printf("Created subcategory: %s -> %s", categoryName, subcategoryName)
			} else {
				log.Printf("Subcategory already exists: %s -> %s", categoryName, subcategoryName)
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Println("Successfully initialized all default categories and subcategories")
	return nil
}
