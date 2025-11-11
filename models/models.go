package models

import (
	"time"
)

type User struct {
	UserID                  string    `gorm:"primaryKey;type:char(60)" json:"user_id"`
	Username                string    `gorm:"uniqueIndex;size:50" json:"username"`
	Name                    string    `gorm:"size:100" json:"name"`
	Email                   string    `gorm:"uniqueIndex;size:100" json:"email"`
	Password                string    `gorm:"size:255" json:"-"`
	Role                    string    `gorm:"size:20;default:user" json:"role"`
	ProfilePict             string    `gorm:"size:255" json:"profile_pict"`
	Organization            string    `gorm:"size:100" json:"organization"`
	OrganizationType        string    `gorm:"size:50" json:"organization_type"`
	OrganizationDescription string    `gorm:"type:text" json:"organization_description"`
	KTP                     string    `gorm:"size:255" json:"ktp"`
	RegisterStatus          string    `gorm:"size:20;default:pending" json:"register_status"`
	RegisterComment         string    `gorm:"type:text" json:"register_comment"`
	AccessToken             string    `gorm:"size:500" json:"-"`
	RefreshToken            string    `gorm:"size:500" json:"-"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`

	// Relationships
	Events               []Event              `gorm:"foreignKey:OwnerID" json:"events,omitempty"`
	Tickets              []Ticket             `gorm:"foreignKey:OwnerID" json:"tickets,omitempty"`
	Carts                []Cart               `gorm:"foreignKey:OwnerID" json:"carts,omitempty"`
	TransactionHistories []TransactionHistory `gorm:"foreignKey:OwnerID" json:"transaction_histories,omitempty"`
}

type Event struct {
	EventID         string    `gorm:"primaryKey;type:char(60)" json:"event_id"`
	Name            string    `gorm:"size:100" json:"name"`
	OwnerID         string    `gorm:"type:char(60);not null" json:"owner_id"`
	Status          string    `gorm:"size:20;default:pending" json:"status"`
	ApprovalComment string    `gorm:"type:text" json:"approval_comment"`
	DateStart       time.Time `json:"date_start"`
	DateEnd         time.Time `json:"date_end"`
	Location        string    `gorm:"size:255" json:"location"`
	Description     string    `gorm:"type:text" json:"description"`
	Image           string    `gorm:"size:255" json:"image"`
	Flyer           string    `gorm:"size:255" json:"flyer"`
	Category        string    `gorm:"size:50" json:"category"`
	TotalAttendant  uint      `gorm:"default:0" json:"total_attendant"`
	TotalSales      float64   `gorm:"type:decimal(10,2);default:0" json:"total_sales"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relationships
	Owner            User             `gorm:"foreignKey:OwnerID;references:UserID" json:"owner"`
	TicketCategories []TicketCategory `gorm:"foreignKey:EventID" json:"ticket_categories,omitempty"`
	Tickets          []Ticket         `gorm:"foreignKey:EventID" json:"tickets,omitempty"`
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

	// Relationships
	TicketCategory TicketCategory `gorm:"foreignKey:TicketCategoryID" json:"ticket_category"`
	Owner          User           `gorm:"foreignKey:OwnerID" json:"owner"`
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
	TicketCategory TicketCategory `gorm:"foreignKey:TicketCategoryID" json:"ticket_category"`
	Owner          User           `gorm:"foreignKey:OwnerID" json:"owner"`
}

type TransactionHistory struct {
	TransactionID     string    `gorm:"primaryKey;type:char(60)" json:"transaction_id"`
	OwnerID           string    `gorm:"type:char(60);not null" json:"owner_id"`
	TransactionTime   time.Time `json:"transaction_time"`
	PriceTotal        float64   `gorm:"type:decimal(10,2)" json:"price_total"`
	CreatedAt         time.Time `json:"created_at"`
	TransactionStatus string    `gorm:"size:20;default:pending" json:"transaction_status"`

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
	TicketCategory     TicketCategory     `gorm:"foreignKey:TicketCategoryID" json:"ticket_category"`
	TransactionHistory TransactionHistory `gorm:"foreignKey:TransactionID" json:"transaction_history"`
	Owner              User               `gorm:"foreignKey:OwnerID" json:"owner"`
}
