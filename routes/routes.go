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

	// Upload routes
	upload := app.Group("/api/upload", middleware.AuthMiddleware)
	upload.Post("/image", handlers.UploadImage)
	upload.Post("/images", handlers.UploadMultipleImages)

	// User routes
	user := app.Group("/api/users", middleware.AuthMiddleware)
	user.Get("/profile", handlers.GetProfile)
	user.Put("/profile", handlers.UpdateProfile)
	user.Get("/:id", middleware.AdminMiddleware, handlers.GetUserByID)
	user.Get("/", middleware.AdminMiddleware, handlers.GetUsers)
	user.Post("/:id/verify", middleware.AdminMiddleware, handlers.VerifyUser)

	// Event routes
	app.Get("/api/events", handlers.GetApprovedEvents)
	app.Get("/api/event/:id", handlers.GetEvent)
	app.Get("/api/events/popular", handlers.GetEventsPopular)
	event := app.Group("/api/events", middleware.AuthMiddleware)
	event.Get("/all", handlers.GetEvents)
	event.Get("/my-events", handlers.GetMyEvents)
	event.Post("/", middleware.OrganizerApprovalMiddleware, handlers.CreateEvent)
	event.Put("/:id", handlers.UpdateEvent)
	event.Delete("/:id", handlers.DeleteEvent)
	event.Get("/:id/report", handlers.GetEventReport)
	event.Get("/:id/report/download", handlers.DownloadEventReport)
	event.Patch("/:id/verify", middleware.AdminMiddleware, handlers.VerifyEvent)
	event.Post("/:id/like", handlers.AddLike)
	event.Get("/like", handlers.MyLikedEvent)

	// Ticket routes
	ticket := app.Group("/api/tickets", middleware.AuthMiddleware)
	ticket.Get("/", handlers.GetTickets)
	ticket.Get("/:id", handlers.GetEvent)
	ticket.Patch("/:event_id/:id/checkin", handlers.CheckInTicket)
	ticket.Get("/:id/code", handlers.GetTicketCode)
	ticket.Patch("/:id/tag", handlers.UpdateTagTicket)

	// Cart routes
	cart := app.Group("/api/cart", middleware.AuthMiddleware)
	cart.Post("/", handlers.AddToCart)
	cart.Get("/", handlers.GetCart)
	cart.Patch("/", handlers.UpdateCart)
	cart.Delete("/", handlers.DeleteCart)

	// Payment routes
	payment := app.Group("/api/payment", middleware.AuthMiddleware)
	payment.Post("/midtrans", handlers.PaymentMidtrans)
	app.Post("/midtrans/callback", handlers.PaymentNotificationHandler)

	// Transaction routes
	transaction := app.Group("/api/transactions", middleware.AuthMiddleware)
	transaction.Get("/", handlers.GetTransactionHistory)
	transaction.Get("/:id", handlers.GetTransactionDetail)
}
