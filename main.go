package main

import (
	"log"
	"os"

	"github.com/Tsaniii18/Ticketing-Backend/config"
	"github.com/Tsaniii18/Ticketing-Backend/handlers"
	"github.com/Tsaniii18/Ticketing-Backend/models"
	"github.com/Tsaniii18/Ticketing-Backend/routes"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gorm.io/gorm"
)

func main() {
	// Connect to database
	config.ConnectDatabase()

	// Initialize Cloudinary
	config.InitCloudinary()

	err := migrateDatabase(config.DB)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))
	app.Use(logger.New())

	routes.SetupRoutes(app)

	if err := handlers.DefaultAdminSetup(); err != nil {
		log.Fatal("Failed to setup default admin:", err)
	}

	if err := handlers.InitializeDefaultCategories(); err != nil {
		log.Fatal("Failed to setup default category event:", err)
	}

	if err := handlers.InitialScheduleEventEnd(config.DB); err != nil {
		log.Fatal("Failed to start event_auto_status goroutine:", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	address := "0.0.0.0:" + port
	log.Println("Server running on", address)

	log.Fatal(app.Listen(address))

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

	err = db.AutoMigrate(&models.Feedback{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&models.EventLike{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&models.ChildEventCategory{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&models.EventCategory{})
	if err != nil {
		return err
	}

	db.Exec("SET FOREIGN_KEY_CHECKS = 1")

	log.Println("Database migrated successfully")
	return nil
}
