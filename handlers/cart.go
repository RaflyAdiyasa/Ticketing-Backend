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

	return c.Status(fiber.StatusCreated).JSON(cart)
}

func GetCart(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	type EnrichedCartItem struct {

		// carts
		CartID     string    `json:"cart_id"`
		OwnerID    string    `json:"owner_id"`
		Quantity   uint      `json:"quantity"`
		PriceTotal float64   `json:"price_total"`
		CreatedAt  time.Time `json:"created_at"`
		UpdatedAt  time.Time `json:"updated_at"`

		// ticket_categories
		TicketCategoryID string    `json:"ticket_category_id"`
		TCName           string    `json:"name"`
		TCEventID        string    `json:"event_id"`
		TCPrice          float64   `json:"price"`
		TCQuota          uint      `json:"quota"`
		TCSold           uint      `json:"sold"`
		TCDescription    string    `json:"description"`
		TCDateTimeStart  time.Time `json:"date_time_start"`
		TCDateTimeEnd    time.Time `json:"date_time_end"`

		// events
		EventName             string    `json:"event_name"`
		EventCity             string    `json:"event_city"`
		EventStart            time.Time `json:"event_start"`
		EventEnd              time.Time `json:"event_end"`
		EventTotalTicketsSold uint      `json:"total_tickets_sold"`
	}

	var enrichedCart []EnrichedCartItem
	if err := config.DB.
		Table("carts").
		Joins("JOIN ticket_categories ON ticket_categories.ticket_category_id = carts.ticket_category_id").
		Joins("JOIN events ON events.event_id = ticket_categories.event_id").
		Where("carts.owner_id = ?", user.UserID).
		Select(`
        carts.*,

        ticket_categories.ticket_category_id AS ticket_category_id,
        ticket_categories.name AS tc_name,
        ticket_categories.event_id AS tc_event_id,
        ticket_categories.price AS tc_price,
        ticket_categories.quota AS tc_quota,
        ticket_categories.sold AS tc_sold,
        ticket_categories.description AS tc_description,
        ticket_categories.date_time_start AS tc_date_time_start,
        ticket_categories.date_time_end AS tc_date_time_end,

        events.event_id AS event_id,
        events.name AS event_name,
        events.city AS event_city,
        events.date_start AS event_start,
        events.date_end AS event_end,
        events.total_tickets_sold AS event_total_tickets_sold
    `).
		Scan(&enrichedCart).Error; err != nil {

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch cart: " + err.Error(),
		})
	}

	return c.JSON(enrichedCart)

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

	return c.JSON(cart)
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

	if err := config.DB.Where("cart_id = ? AND owner_id = ?", deleteData.CartID, user.UserID).Delete(&models.Cart{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete cart item: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Cart item deleted successfully",
	})
}
