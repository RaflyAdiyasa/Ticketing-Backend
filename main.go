package main

import (
    "log"
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "github.com/gofiber/fiber/v2/middleware/logger"
    "gorm.io/gorm"
    "github.com/Tsaniii18/Ticketing-Backend/config"
    "github.com/Tsaniii18/Ticketing-Backend/models"
    "github.com/Tsaniii18/Ticketing-Backend/routes"
)

func main() {
    // Connect to database
    config.ConnectDatabase()

    err := migrateDatabase(config.DB)
    if err != nil {
        log.Fatal("Failed to migrate database:", err)
    }
    
    app := fiber.New()
    
    // Middleware
    app.Use(cors.New(cors.Config{
        AllowOrigins: "*",
        AllowHeaders: "Origin, Content-Type, Accept, Authorization",
    }))
    app.Use(logger.New())
    
    // Setup routes
    routes.SetupRoutes(app)
    
    log.Println("Server running on port 3000")
    log.Fatal(app.Listen(":3000"))
}

func migrateDatabase(db *gorm.DB) error {
    db.Exec("SET FOREIGN_KEY_CHECKS = 0")
    
    err := db.AutoMigrate(&models.User{})
    if err != nil {
        return err
    }

    err = db.AutoMigrate(&models.Event{})
    if err != nil {
        return err
    }
    
    err = db.AutoMigrate(&models.TicketCategory{})
    if err != nil {
        return err
    }
	
    err = db.AutoMigrate(&models.Cart{})
    if err != nil {
        return err
    }

    err = db.AutoMigrate(&models.TransactionHistory{})
    if err != nil {
        return err
    }

    err = db.AutoMigrate(&models.Ticket{})
    if err != nil {
        return err
    }
    
    err = db.AutoMigrate(&models.TransactionDetail{})
    if err != nil {
        return err
    }
    
    err = db.AutoMigrate(&models.Report{})
    if err != nil {
        return err
    }

    db.Exec("SET FOREIGN_KEY_CHECKS = 1")
    
    log.Println("Database migrated successfully")
    return nil
}