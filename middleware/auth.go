package middleware

import (
    "github.com/gofiber/fiber/v2"
    "github.com/golang-jwt/jwt/v4"
    "github.com/google/uuid"
    "github.com/Tsaniii18/Ticketing-Backend/config"
    "github.com/Tsaniii18/Ticketing-Backend/models"
)

func AuthMiddleware(c *fiber.Ctx) error {
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
    token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
        return []byte("goofygoobers"), nil
    })

    if err != nil || !token.Valid {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Invalid token",
        })
    }

    userIDStr := (*claims)["user_id"].(string)
    userID, err := uuid.Parse(userIDStr)
    if err != nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Invalid user ID in token",
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
