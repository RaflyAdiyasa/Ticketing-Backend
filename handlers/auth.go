package handlers

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Tsaniii18/Ticketing-Backend/config"
	"github.com/Tsaniii18/Ticketing-Backend/models"
	"github.com/Tsaniii18/Ticketing-Backend/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/joho/godotenv"
	"gorm.io/gorm/clause"
)

type LoginRequest struct {
	UsernameOrEmail string `json:"username_or_email"` // Bisa username atau email
	Password        string `json:"password"`
}

type RegisterRequest struct {
	Username                string `json:"username"`
	Name                    string `json:"name"`
	Email                   string `json:"email"`
	Password                string `json:"password"`
	Role                    string `json:"role"`                               // user, organizer
	Organization            string `json:"organization,omitempty"`             // Wajib untuk organizer
	OrganizationType        string `json:"organization_type,omitempty"`        // Wajib untuk organizer
	OrganizationDescription string `json:"organization_description,omitempty"` // Wajib untuk organizer
}

func Register(c *fiber.Ctx) error {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, using system environment")
	}

	// Parse form data
	username := c.FormValue("username")
	name := c.FormValue("name")
	email := c.FormValue("email")
	password := c.FormValue("password")
	role := c.FormValue("role")
	organization := c.FormValue("organization")
	organizationType := c.FormValue("organization_type")
	organizationDescription := c.FormValue("organization_description")

	// Validate role
	if role != "user" && role != "organizer" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Role must be either 'user' or 'organizer'",
		})
	}

	// Validasi Organization untuk organizer
	if role == "organizer" {
		if organization == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Organization is required for organizer registration",
			})
		}
	}

	// Check if user already exists
	var existingUser models.User
	if err := config.DB.Where("username = ? OR email = ?", username, email).First(&existingUser).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Username or email already exists",
		})
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}

	// Set register status based on role
	registerStatus := "approved" // user langsung approved
	if role == "organizer" {
		registerStatus = "pending" // organizer perlu approval admin
	}

	// Handle KTP upload untuk organizer
	var ktpURL string
	if role == "organizer" {
		ktpFile, err := c.FormFile("ktp")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "KTP image is required for organizer registration",
			})
		}

		// Validate file size (max 5MB)
		if ktpFile.Size > 5*1024*1024 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "KTP file size too large. Maximum size is 5MB",
			})
		}

		// Open file
		file, err := ktpFile.Open()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to open KTP file",
			})
		}
		defer file.Close()

		// Upload to Cloudinary
		folder := "ticketing-app/ktp"
		ktpURL, err = config.UploadImage(context.Background(), file, folder)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to upload KTP to Cloudinary",
			})
		}
	}

	user := models.User{
		UserID:                  utils.GenerateUserID(role),
		Username:                username,
		Name:                    name,
		Email:                   email,
		Password:                hashedPassword,
		Role:                    role,
		Organization:            organization,
		OrganizationType:        organizationType,
		OrganizationDescription: organizationDescription,
		KTP:                     ktpURL,
		RegisterStatus:          registerStatus,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	if err := config.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully",
		"user": fiber.Map{
			"user_id":         user.UserID,
			"username":        user.Username,
			"name":            user.Name,
			"email":           user.Email,
			"role":            user.Role,
			"register_status": user.RegisterStatus,
		},
	})
}

func DefaultAdminSetup() error {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, using system environment")

	}

	df_admin_username := os.Getenv("DEFAULT_ADMIN_USERNAME")
	df_admin_email := os.Getenv("DEFAULT_ADMIN_EMAIL")
	df_admin_pass := os.Getenv("DEFAULT_ADMIN_PASS")
	df_admin_name := os.Getenv("DEFAULT_ADMIN_NAME")

	// Cek apakah admin sudah ada
	var adminCheck models.User
	if err := config.DB.Where("username = ? && email = ?", df_admin_username, df_admin_email).First(&adminCheck).Error; err == nil {
		log.Println("Default admin user already exists")
		return nil
	}

	var admin models.User

	hashedPassword, err := utils.HashPassword(df_admin_pass)
	if err != nil {
		log.Fatal("Failed to hash default admin password:", err)
	}

	admin = models.User{
		UserID:    utils.GenerateUserID("admin"),
		Username:  df_admin_username,
		Name:      df_admin_name,
		Email:     df_admin_email,
		Password:  hashedPassword,
		Role:      "admin",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := config.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&admin).Error; err != nil {
		log.Fatal("Failed to create default admin user:", err)
	}
	log.Println("Default admin user created successfully, username:", df_admin_username)
	return nil
}

// Login function - MODIFIKASI: Allow organizer dengan status pending untuk login
func Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	var user models.User
	// Cari user berdasarkan username ATAU email
	if err := config.DB.Where("username = ? OR email = ?", req.UsernameOrEmail, req.UsernameOrEmail).First(&user).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	// Generate JWT token dengan claims yang benar
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = user.UserID
	claims["username"] = user.Username
	claims["role"] = user.Role
	claims["register_status"] = user.RegisterStatus // Tambahkan register_status ke claims
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
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
		UpdatedAt:   time.Now(),
	})

	return c.JSON(fiber.Map{
		"message": "Login successful",
		"token":   tokenString,
		"user": fiber.Map{
			"user_id":         user.UserID,
			"username":        user.Username,
			"name":            user.Name,
			"email":           user.Email,
			"role":            user.Role,
			"register_status": user.RegisterStatus,
		},
	})
}
