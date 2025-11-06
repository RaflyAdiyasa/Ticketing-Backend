package middleware

import (
	"log"
	"os"

	"github.com/Tsaniii18/Ticketing-Backend/config"
	"github.com/Tsaniii18/Ticketing-Backend/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/joho/godotenv"
)

func AuthMiddleware(c *fiber.Ctx) error {

	err := godotenv.Load()
	if err != nil {
		log.Println("  .env file not found, using system environment")
	}

	tokenString := c.Get("Authorization")
	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization header required",
		})
	}

	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	claims := &jwt.MapClaims{}
	seed := os.Getenv("JWT_SEED")
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(seed), nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	// Extract user_id from claims - handle both string and direct UUID
	var userID string
	if userIDClaim, exists := (*claims)["user_id"]; exists {
		switch v := userIDClaim.(type) {
		case string:
			userID = v
		default:
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid user ID format in token",
			})
		}
	} else {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in token",
		})
	}

	var user models.User
	if err := config.DB.First(&user, "user_id = ?", userID).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	c.Locals("user", user)
	return c.Next()
}

func AdminMiddleware(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	if user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}
	return c.Next()
}

func OrganizerMiddleware(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	if user.Role != "organizer" && user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Organizer access required",
		})
	}
	return c.Next()
}
