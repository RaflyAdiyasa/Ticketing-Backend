package handlers

import (
	"log"
	"os"
	"time"

	"github.com/Tsaniii18/Ticketing-Backend/config"
	"github.com/Tsaniii18/Ticketing-Backend/models"
	"github.com/Tsaniii18/Ticketing-Backend/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/joho/godotenv"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username         string `json:"username"`
	Name             string `json:"name"`
	Email            string `json:"email"`
	Password         string `json:"password"`
	Organization     string `json:"organization"`
	OrganizationType string `json:"organization_type"`
}

func Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	// Check if user already exists
	var existingUser models.User
	if err := config.DB.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Username or email already exists",
		})
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}

	user := models.User{
		Username:         req.Username,
		Name:             req.Name,
		Email:            req.Email,
		Password:         hashedPassword,
		Organization:     req.Organization,
		OrganizationType: req.OrganizationType,
		RegisterStatus:   "pending",
		Role:             "user",
	}

	if err := config.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully",
		"user": fiber.Map{
			"user_id":  user.UserID,
			"username": user.Username,
			"name":     user.Name,
			"email":    user.Email,
		},
	})
}

func Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	var user models.User
	if err := config.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	if user.RegisterStatus != "approved" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Account not approved yet",
		})
	}

	// Generate JWT token
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = user.UserID
	claims["username"] = user.Username
	claims["role"] = user.Role
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	err := godotenv.Load()
	if err != nil {
		log.Println(" -- .env file not found, using system environment")
	}
	seed := os.Getenv("JWT_SEED")

	tokenString, err := token.SignedString([]byte(seed))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	// Update user tokens
	config.DB.Model(&user).Updates(models.User{
		AccessToken: tokenString,
	})

	return c.JSON(fiber.Map{
		"message": "Login successful",
		"token":   tokenString,
		"user": fiber.Map{
			"user_id":  user.UserID,
			"username": user.Username,
			"name":     user.Name,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}
