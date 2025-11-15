package handlers

import (
	"fmt"
	"time"

	"github.com/Tsaniii18/Ticketing-Backend/config"
	"github.com/Tsaniii18/Ticketing-Backend/models"
	"github.com/Tsaniii18/Ticketing-Backend/utils"
	"github.com/gofiber/fiber/v2"
)

func AddToCart(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	var cartData struct {
		TicketCategoryID string `json:"ticket_category_id"`
		Quantity         uint   `json:"quantity"`
	}

	if err := c.BodyParser(&cartData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	if cartData.TicketCategoryID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Ticket category ID is required",
		})
	}

	if cartData.Quantity == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Quantity must be at least 1",
		})
	}

	var ticketCategory models.TicketCategory
	if err := config.DB.First(&ticketCategory, "ticket_category_id = ?", cartData.TicketCategoryID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Ticket category not found",
		})
	}

	if ticketCategory.Sold+cartData.Quantity > ticketCategory.Quota {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Not enough quota available",
		})
	}

	priceTotal := float64(cartData.Quantity) * ticketCategory.Price

	cart := models.Cart{
		CartID:           utils.GenerateCartID(),
		TicketCategoryID: ticketCategory.TicketCategoryID,
		OwnerID:          user.UserID,
		Quantity:         cartData.Quantity,
		PriceTotal:       priceTotal,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := config.DB.Create(&cart).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add to cart",
		})
	}

	// Load event data for this ticket category
	var event models.Event
	if err := config.DB.First(&event, "event_id = ?", ticketCategory.EventID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load event data",
		})
	}

	// Build enriched response
	cartResponse := CartResponse{
		CartID:     cart.CartID,
		OwnerID:    cart.OwnerID,
		Quantity:   cart.Quantity,
		PriceTotal: cart.PriceTotal,
		CreatedAt:  cart.CreatedAt,
		UpdatedAt:  cart.UpdatedAt,
		TicketCategory: &TicketCategoryResponse{
			TicketCategoryID: ticketCategory.TicketCategoryID,
			Name:            ticketCategory.Name,
			EventID:         ticketCategory.EventID,
			Price:           ticketCategory.Price,
			Quota:           ticketCategory.Quota,
			Sold:           ticketCategory.Sold,
			Description:     ticketCategory.Description,
			DateTimeStart:   ticketCategory.DateTimeStart,
			DateTimeEnd:     ticketCategory.DateTimeEnd,
			CreatedAt:       ticketCategory.CreatedAt,
			UpdatedAt:       ticketCategory.UpdatedAt,
		},
		Event: &EventResponse{
			EventID:          event.EventID,
			Name:            event.Name,
			OwnerID:         event.OwnerID,
			Status:          event.Status,
			DateStart:       event.DateStart,
			DateEnd:         event.DateEnd,
			Location:        event.Location,
			City:           event.City,
			Description:     event.Description,
			Image:          event.Image,
			Flyer:          event.Flyer,
			Category:       event.Category,
			TotalTicketsSold: event.TotalTicketsSold,
			CreatedAt:      event.CreatedAt,
			UpdatedAt:      event.UpdatedAt,
		},
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Item added to cart successfully",
		"cart":    cartResponse,
	})
}

// CartResponse represents the enriched cart response structure
type CartResponse struct {
	CartID         string                  `json:"cart_id"`
	OwnerID        string                  `json:"owner_id"`
	Quantity       uint                    `json:"quantity"`
	PriceTotal     float64                 `json:"price_total"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
	TicketCategory *TicketCategoryResponse `json:"ticket_category"`
	Event          *EventResponse          `json:"event"`
}

type TicketCategoryResponse struct {
	TicketCategoryID string    `json:"ticket_category_id"`
	Name             string    `json:"name"`
	EventID          string    `json:"event_id"`
	Price            float64   `json:"price"`
	Quota            uint      `json:"quota"`
	Sold             uint      `json:"sold"`
	Description      string    `json:"description"`
	DateTimeStart    time.Time `json:"date_time_start"`
	DateTimeEnd      time.Time `json:"date_time_end"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type EventResponse struct {
	EventID          string    `json:"event_id"`
	Name             string    `json:"name"`
	OwnerID          string    `json:"owner_id"`
	Status           string    `json:"status"`
	DateStart        time.Time `json:"date_start"`
	DateEnd          time.Time `json:"date_end"`
	Location         string    `json:"location"`
	City             string    `json:"city"`
	Description      string    `json:"description"`
	Image            string    `json:"image"`
	Flyer            string    `json:"flyer"`
	Category         string    `json:"category"`
	TotalTicketsSold uint      `json:"total_tickets_sold"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func GetCart(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	var carts []models.Cart
	if err := config.DB.
		Where("owner_id = ?", user.UserID).
		Find(&carts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch cart: " + err.Error(),
		})
	}

	// Enrich cart data with ticket category and event information
	var cartResponses []CartResponse
	for _, cart := range carts {
		// Get ticket category for this cart item
		var ticketCategory models.TicketCategory
		if err := config.DB.First(&ticketCategory, "ticket_category_id = ?", cart.TicketCategoryID).Error; err != nil {
			continue // Skip this cart item if ticket category not found
		}

		// Get event for this ticket category
		var event models.Event
		if err := config.DB.First(&event, "event_id = ?", ticketCategory.EventID).Error; err != nil {
			continue // Skip this cart item if event not found
		}

		cartResponse := CartResponse{
			CartID:     cart.CartID,
			OwnerID:    cart.OwnerID,
			Quantity:   cart.Quantity,
			PriceTotal: cart.PriceTotal,
			CreatedAt:  cart.CreatedAt,
			UpdatedAt:  cart.UpdatedAt,
			TicketCategory: &TicketCategoryResponse{
				TicketCategoryID: ticketCategory.TicketCategoryID,
				Name:            ticketCategory.Name,
				EventID:         ticketCategory.EventID,
				Price:           ticketCategory.Price,
				Quota:           ticketCategory.Quota,
				Sold:           ticketCategory.Sold,
				Description:     ticketCategory.Description,
				DateTimeStart:   ticketCategory.DateTimeStart,
				DateTimeEnd:     ticketCategory.DateTimeEnd,
				CreatedAt:       ticketCategory.CreatedAt,
				UpdatedAt:       ticketCategory.UpdatedAt,
			},
			Event: &EventResponse{
				EventID:          event.EventID,
				Name:            event.Name,
				OwnerID:         event.OwnerID,
				Status:          event.Status,
				DateStart:       event.DateStart,
				DateEnd:         event.DateEnd,
				Location:        event.Location,
				City:           event.City,
				Description:     event.Description,
				Image:          event.Image,
				Flyer:          event.Flyer,
				Category:       event.Category,
				TotalTicketsSold: event.TotalTicketsSold,
				CreatedAt:      event.CreatedAt,
				UpdatedAt:      event.UpdatedAt,
			},
		}

		cartResponses = append(cartResponses, cartResponse)
	}

	return c.JSON(fiber.Map{
		"message": "Cart retrieved successfully",
		"carts":   cartResponses,
	})
}

func UpdateCart(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	var updateData struct {
		CartID   string `json:"cart_id"`
		Quantity uint   `json:"quantity"`
	}

	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request: " + err.Error(),
		})
	}

	if updateData.CartID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cart ID is required",
		})
	}

	if updateData.Quantity == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Quantity must be at least 1",
		})
	}

	var cart models.Cart
	if err := config.DB.
		Where("cart_id = ? AND owner_id = ?", updateData.CartID, user.UserID).
		First(&cart).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Cart item not found",
		})
	}

	var ticketCategory models.TicketCategory
	if err := config.DB.
		First(&ticketCategory, "ticket_category_id = ?", cart.TicketCategoryID).
		Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Ticket category not found",
		})
	}

	// Cek ketersediaan kuota
	availableQuota := ticketCategory.Quota - ticketCategory.Sold
	if updateData.Quantity > availableQuota {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Not enough quota available. Available: %d, Requested: %d", availableQuota, updateData.Quantity),
		})
	}

	// Update cart
	cart.Quantity = updateData.Quantity
	cart.PriceTotal = float64(updateData.Quantity) * ticketCategory.Price
	cart.UpdatedAt = time.Now()

	if err := config.DB.Save(&cart).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update cart: " + err.Error(),
		})
	}

	// Get event data for response
	var event models.Event
	if err := config.DB.First(&event, "event_id = ?", ticketCategory.EventID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load event data",
		})
	}

	// Build enriched response
	cartResponse := CartResponse{
		CartID:     cart.CartID,
		OwnerID:    cart.OwnerID,
		Quantity:   cart.Quantity,
		PriceTotal: cart.PriceTotal,
		CreatedAt:  cart.CreatedAt,
		UpdatedAt:  cart.UpdatedAt,
		TicketCategory: &TicketCategoryResponse{
			TicketCategoryID: ticketCategory.TicketCategoryID,
			Name:            ticketCategory.Name,
			EventID:         ticketCategory.EventID,
			Price:           ticketCategory.Price,
			Quota:           ticketCategory.Quota,
			Sold:           ticketCategory.Sold,
			Description:     ticketCategory.Description,
			DateTimeStart:   ticketCategory.DateTimeStart,
			DateTimeEnd:     ticketCategory.DateTimeEnd,
			CreatedAt:       ticketCategory.CreatedAt,
			UpdatedAt:       ticketCategory.UpdatedAt,
		},
		Event: &EventResponse{
			EventID:          event.EventID,
			Name:            event.Name,
			OwnerID:         event.OwnerID,
			Status:          event.Status,
			DateStart:       event.DateStart,
			DateEnd:         event.DateEnd,
			Location:        event.Location,
			City:           event.City,
			Description:     event.Description,
			Image:          event.Image,
			Flyer:          event.Flyer,
			Category:       event.Category,
			TotalTicketsSold: event.TotalTicketsSold,
			CreatedAt:      event.CreatedAt,
			UpdatedAt:      event.UpdatedAt,
		},
	}

	return c.JSON(fiber.Map{
		"message": "Cart updated successfully",
		"cart":    cartResponse,
	})
}

func DeleteCart(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	var deleteData struct {
		CartID string `json:"cart_id"`
	}

	if err := c.BodyParser(&deleteData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request: " + err.Error(),
		})
	}

	if deleteData.CartID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cart ID is required",
		})
	}

	// Check if cart exists and belongs to user
	var cart models.Cart
	if err := config.DB.
		Where("cart_id = ? AND owner_id = ?", deleteData.CartID, user.UserID).
		First(&cart).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Cart item not found",
		})
	}

	if err := config.DB.Delete(&cart).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete cart item: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Cart item deleted successfully",
		"deleted_cart_id": deleteData.CartID,
	})
}