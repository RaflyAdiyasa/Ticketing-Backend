package models

import (
	"time"
)

type User struct {
	UserID                  string `gorm:"primaryKey;type:char(60)" json:"user_id"`
	Username                string `gorm:"uniqueIndex;size:50" json:"username"`
	Name                    string `gorm:"size:100" json:"name"`
	Email                   string `gorm:"uniqueIndex;size:100" json:"email"`
	Password                string `gorm:"size:255" json:"-"`
	Role                    string `gorm:"size:20;default:user" json:"role"`
	ProfilePict             string `gorm:"size:255" json:"profile_pict"`
	Organization            string `gorm:"size:100" json:"organization"`
	OrganizationType        string `gorm:"size:50" json:"organization_type"`
	OrganizationDescription string `gorm:"type:text" json:"organization_description"`
	KTP                     string `gorm:"size:255" json:"ktp"`
	RegisterStatus          string `gorm:"size:20;default:pending" json:"register_status"`
	RegisterComment         string `gorm:"type:text" json:"register_comment"`
	// TotalLikes              uint      `gorm:"default:0" json:"total_likes"`
	AccessToken  string    `gorm:"size:500" json:"-"`
	RefreshToken string    `gorm:"size:500" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relationships
	Events               []Event              `gorm:"foreignKey:OwnerID" json:"events,omitempty"`
	Tickets              []Ticket             `gorm:"foreignKey:OwnerID" json:"tickets,omitempty"`
	Carts                []Cart               `gorm:"foreignKey:OwnerID" json:"carts,omitempty"`
	TransactionHistories []TransactionHistory `gorm:"foreignKey:OwnerID" json:"transaction_histories,omitempty"`
	LikedEvents          []Event              `gorm:"many2many:event_likes;foreignKey:UserID;joinForeignKey:user_id;references:EventID;joinReferences:event_id" json:"liked_events,omitempty"`
}

type Event struct {
	EventID          string    `gorm:"primaryKey;type:char(60)" json:"event_id"`
	Name             string    `gorm:"size:100" json:"name"`
	OwnerID          string    `gorm:"type:char(60);not null" json:"owner_id"`
	Status           string    `gorm:"size:20;default:pending" json:"status"`
	ApprovalComment  string    `gorm:"type:text" json:"approval_comment"`
	DateStart        time.Time `json:"date_start"`
	DateEnd          time.Time `json:"date_end"`
	Location         string    `gorm:"size:255" json:"location"`
	Venue            string    `gorm:"size:100" json:"venue"`
	District         string    `gorm:"size:100" json:"district"`
	Description      string    `gorm:"type:text" json:"description"`
	Rules            string    `gorm:"type:text" json:"rules"`
	Image            string    `gorm:"size:255" json:"image"`
	Flyer            string    `gorm:"size:255" json:"flyer"`
	Category         string    `gorm:"size:50" json:"category"`
	ChildCategory    string    `gorm:"size:50" json:"child_category"`
	TotalAttendant   uint      `gorm:"default:0" json:"total_attendant"`
	TotalLikes       uint      `gorm:"default:0" json:"total_likes"`
	TotalSales       float64   `gorm:"type:decimal(10,2);default:0" json:"total_sales"`
	TotalTicketsSold uint      `gorm:"default:0" json:"total_tickets_sold"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// Relationships
	Owner            User             `gorm:"foreignKey:OwnerID;references:UserID" json:"owner"`
	TicketCategories []TicketCategory `gorm:"foreignKey:EventID" json:"ticket_categories,omitempty"`
	Tickets          []Ticket         `gorm:"foreignKey:EventID" json:"tickets,omitempty"`
	LikedBy          []User           `gorm:"many2many:event_likes;foreignKey:EventID;joinForeignKey:event_id;references:UserID;joinReferences:user_id" json:"liked_by,omitempty"`
}

type TicketCategory struct {
	Name             string    `gorm:"size:100" json:"name"`
	TicketCategoryID string    `gorm:"primaryKey;type:char(60)" json:"ticket_category_id"`
	EventID          string    `gorm:"type:char(60);not null" json:"event_id"`
	Price            float64   `gorm:"type:decimal(10,2)" json:"price"`
	Quota            uint      `json:"quota"`
	Sold             uint      `gorm:"default:0" json:"sold"`
	Description      string    `gorm:"type:text" json:"description"`
	DateTimeStart    time.Time `json:"date_time_start"`
	DateTimeEnd      time.Time `json:"date_time_end"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Attendant        uint      `gorm:"default:0" json:"attendant"`

	// Relationships
	Tickets            []Ticket            `gorm:"foreignKey:TicketCategoryID" json:"tickets,omitempty"`
	Carts              []Cart              `gorm:"foreignKey:TicketCategoryID" json:"carts,omitempty"`
	TransactionDetails []TransactionDetail `gorm:"foreignKey:TicketCategoryID" json:"transaction_details,omitempty"`
}

type Ticket struct {
	TicketID         string    `gorm:"primaryKey;type:char(60)" json:"ticket_id"`
	EventID          string    `gorm:"type:char(60);not null" json:"event_id"`
	TicketCategoryID string    `gorm:"type:char(60);not null" json:"ticket_category_id"`
	OwnerID          string    `gorm:"type:char(60);not null" json:"owner_id"`
	Status           string    `gorm:"size:20;default:active" json:"status"`
	Code             string    `gorm:"size:100;uniqueIndex" json:"code"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	ExpiresAt        time.Time `json:"expires_at"`
	Tag              string    `gorm:"size:100" json:"tag" default:"My Ticket"`

	// Relationships
	Owner User `gorm:"foreignKey:OwnerID" json:"owner"`
}

type Cart struct {
	CartID           string    `gorm:"primaryKey;type:char(60)" json:"cart_id"`
	TicketCategoryID string    `gorm:"type:char(60);not null" json:"ticket_category_id"`
	OwnerID          string    `gorm:"type:char(60);not null" json:"owner_id"`
	Quantity         uint      `gorm:"default:1" json:"quantity"`
	PriceTotal       float64   `gorm:"type:decimal(10,2)" json:"price_total"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// Relationships
	Owner User `gorm:"foreignKey:OwnerID" json:"owner"`
}

type TransactionHistory struct {
	TransactionID     string    `gorm:"primaryKey;type:char(60)" json:"transaction_id"`
	OwnerID           string    `gorm:"type:char(60);not null" json:"owner_id"`
	TransactionTime   time.Time `json:"transaction_time"`
	PriceTotal        float64   `gorm:"type:decimal(10,2)" json:"price_total"`
	CreatedAt         time.Time `json:"created_at"`
	TransactionStatus string    `gorm:"size:20;default:pending" json:"transaction_status"`
	LinkPayment       string    `gorm:"size:255" json:"link_payment"`

	// Relationships
	Owner              User                `gorm:"foreignKey:OwnerID" json:"owner"`
	TransactionDetails []TransactionDetail `gorm:"foreignKey:TransactionID" json:"transaction_details,omitempty"`
}

type TransactionDetail struct {
	TransactionDetailID string  `gorm:"primaryKey;type:char(60)" json:"transaction_detail_id"`
	TicketCategoryID    string  `gorm:"type:char(60);not null" json:"ticket_category_id"`
	TransactionID       string  `gorm:"type:char(60);not null" json:"transaction_id"`
	OwnerID             string  `gorm:"type:char(60);not null" json:"owner_id"`
	Quantity            uint    `json:"quantity"`
	Subtotal            float64 `gorm:"type:decimal(10,2)" json:"subtotal"`

	// Relationships
	Owner User `gorm:"foreignKey:OwnerID" json:"owner"`
}

type EventLike struct {
	UserID  string `gorm:"primaryKey;type:char(60);not null" json:"user_id"`
	EventID string `gorm:"primaryKey;type:char(60);not null" json:"event_id"`

	User  User  `gorm:"foreignKey:UserID;references:UserID" json:"user"`
	Event Event `gorm:"foreignKey:EventID;references:EventID" json:"event"`
}
