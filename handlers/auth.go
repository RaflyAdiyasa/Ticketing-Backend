package handlers

import (
    "time"
    "github.com/gofiber/fiber/v2"
    "github.com/golang-jwt/jwt/v4"
    "github.com/google/uuid"
    "github.com/Tsaniii18/Ticketing-Backend/config"
    "github.com/Tsaniii18/Ticketing-Backend/models"
    "github.com/Tsaniii18/Ticketing-Backend/utils"
)

type LoginRequest struct {
    UsernameOrEmail string `json:"username_or_email"` // Bisa username atau email
    Password        string `json:"password"`
}

type RegisterRequest struct {
    Username               string `json:"username"`
    Name                   string `json:"name"`
    Email                  string `json:"email"`
    Password               string `json:"password"`
    Role                   string `json:"role"` // user, organizer
    Organization           string `json:"organization,omitempty"` // Wajib untuk organizer
    OrganizationType       string `json:"organization_type,omitempty"` // Wajib untuk organizer
    OrganizationDescription string `json:"organization_description,omitempty"` // Wajib untuk organizer
    KTP                    string `json:"ktp,omitempty"` // Wajib untuk organizer
}

func Register(c *fiber.Ctx) error {
    var req RegisterRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request",
        })
    }

    // Validate role
    if req.Role != "user" && req.Role != "organizer" {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Role must be either 'user' or 'organizer'",
        })
    }

    // Validasi Organization & KTP untuk organizer
    if req.Role == "organizer" {
        if req.KTP == "" {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "KTP is required for organizer registration",
            })
        }
        if req.Organization == "" {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Organization is required for organizer registration",
            })
        }
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

    // Set register status based on role
    registerStatus := "approved" // user langsung approved
    if req.Role == "organizer" {
        registerStatus = "pending" // organizer perlu approval admin
    }

    user := models.User{
        UserID:                uuid.New(),
        Username:              req.Username,
        Name:                  req.Name,
        Email:                 req.Email,
        Password:              hashedPassword,
        Role:                  req.Role,
        Organization:          req.Organization,
        OrganizationType:      req.OrganizationType,
        OrganizationDescription: req.OrganizationDescription,
        KTP:                   req.KTP,
        RegisterStatus:        registerStatus,
        CreatedAt:             time.Now(),
        UpdatedAt:             time.Now(),
    }

    if err := config.DB.Create(&user).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to create user",
        })
    }

    return c.Status(fiber.StatusCreated).JSON(fiber.Map{
        "message": "User registered successfully",
        "user": fiber.Map{
            "user_id":        user.UserID,
            "username":       user.Username,
            "name":           user.Name,
            "email":          user.Email,
            "role":           user.Role,
            "register_status": user.RegisterStatus,
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

    // Check approval status for organizer
    if user.Role == "organizer" && user.RegisterStatus != "approved" {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "error": "Organizer account not approved yet",
        })
    }

    // Generate JWT token
    token := jwt.New(jwt.SigningMethodHS256)
    claims := token.Claims.(jwt.MapClaims)
    claims["user_id"] = user.UserID
    claims["username"] = user.Username
    claims["role"] = user.Role
    claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

    tokenString, err := token.SignedString([]byte("goofygoobers"))
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
            "user_id":        user.UserID,
            "username":       user.Username,
            "name":           user.Name,
            "email":          user.Email,
            "role":           user.Role,
            "register_status": user.RegisterStatus,
        },
    })
}