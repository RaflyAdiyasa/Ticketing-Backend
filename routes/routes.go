package routes

import (
	"github.com/Tsaniii18/Ticketing-Backend/handlers"
	"github.com/Tsaniii18/Ticketing-Backend/middleware"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	// Auth routes
	auth := app.Group("/api/auth")
	auth.Post("/register", handlers.Register)
	auth.Post("/login", handlers.Login)

	// User routes
	user := app.Group("/api/users", middleware.AuthMiddleware)
	user.Get("/profile", handlers.GetProfile)
	user.Put("/profile", handlers.UpdateProfile)
	user.Get("/", middleware.AdminMiddleware, handlers.GetUsers)
	user.Post("/:id/verify", middleware.AdminMiddleware, handlers.VerifyUser)

	// Event routes
	app.Get("/api/events", handlers.GetEvents)
	event := app.Group("/api/events", middleware.AuthMiddleware)
	event.Post("/", handlers.CreateEvent)
	event.Get("/:id", handlers.GetEvent)
	event.Put("/:id", handlers.UpdateEvent)
	event.Patch("/:id/verify", middleware.AdminMiddleware, handlers.VerifyEvent)
	event.Delete("/:id", handlers.DeleteEvent)

	// Ticket routes
	ticket := app.Group("/api/tickets", middleware.AuthMiddleware)
	ticket.Post("/", handlers.CreateTicketCategory)
	ticket.Get("/", handlers.GetTickets)
	ticket.Get("/:id", handlers.GetEvent)
	ticket.Patch("/:id/checkin", handlers.CheckInTicket)

	// Cart routes
	cart := app.Group("/api/cart", middleware.AuthMiddleware)
	cart.Post("/", handlers.AddToCart)
	cart.Get("/", handlers.GetCart)
	cart.Patch("/", handlers.UpdateCart)
	cart.Delete("/", handlers.DeleteCart)

	// Transaction routes
	transaction := app.Group("/api", middleware.AuthMiddleware)
	transaction.Post("/checkout", handlers.Checkout)
	transaction.Get("/transactions", handlers.GetTransactions)
	transaction.Get("/transactions/:id", handlers.GetTransaction)
}
