package models

import (
    "time"
    "github.com/google/uuid"
)

type User struct {
    UserID                  uuid.UUID `gorm:"primaryKey;type:char(36)" json:"user_id"`
    Username                string    `gorm:"uniqueIndex;size:50" json:"username"`
    Name                    string    `gorm:"size:100" json:"name"`
    Email                   string    `gorm:"uniqueIndex;size:100" json:"email"`
    Password                string    `gorm:"size:255" json:"-"`
    Role                    string    `gorm:"size:20;default:user" json:"role"` // user, admin, organizer
    ProfilePict             string    `gorm:"size:255" json:"profile_pict"`
    Organization            string    `gorm:"size:100" json:"organization"`
    OrganizationType        string    `gorm:"size:50" json:"organization_type"`
    OrganizationDescription string    `gorm:"type:text" json:"organization_description"`
    KTP                     string    `gorm:"size:255" json:"ktp"`
    RegisterStatus          string    `gorm:"size:20;default:pending" json:"register_status"` // pending, approved, rejected
    RegisterComment         string    `gorm:"type:text" json:"register_comment"` // Comment dari admin saat verifikasi
    AccessToken             string    `gorm:"size:500" json:"-"`
    RefreshToken            string    `gorm:"size:500" json:"-"`
    CreatedAt               time.Time `json:"created_at"`
    UpdatedAt               time.Time `json:"updated_at"`
    
    // Relationships
    Events                  []Event   `gorm:"foreignKey:OwnerID" json:"events,omitempty"`
    Tickets                 []Ticket  `gorm:"foreignKey:OwnerID" json:"tickets,omitempty"`
    Carts                   []Cart    `gorm:"foreignKey:OwnerID" json:"carts,omitempty"`
    TransactionHistories    []TransactionHistory `gorm:"foreignKey:OwnerID" json:"transaction_histories,omitempty"`
    Reports                 []Report  `gorm:"foreignKey:OwnerID" json:"reports,omitempty"`
}

type Event struct {
    EventID           uint      `gorm:"primaryKey" json:"event_id"`
    Name              string    `gorm:"size:100" json:"name"`
    OwnerID           uuid.UUID `gorm:"type:char(36)" json:"owner_id"`
    Status            string    `gorm:"size:20;default:pending" json:"status"` // pending, approved, rejected
    ApprovalComment   string    `gorm:"type:text" json:"approval_comment"`
    DateStart         time.Time `json:"date_start"`
    DateEnd           time.Time `json:"date_end"`
    Location          string    `gorm:"size:255" json:"location"`
    Description       string    `gorm:"type:text" json:"description"`
    Image             string    `gorm:"size:255" json:"image"`
    Flyer             string    `gorm:"size:255" json:"flyer"`
    Category          string    `gorm:"size:50" json:"category"`
    CreatedAt         time.Time `json:"created_at"`
    UpdatedAt         time.Time `json:"updated_at"`
    
    // Relationships
    Owner             User             `gorm:"foreignKey:OwnerID" json:"owner"`
    TicketCategories  []TicketCategory `gorm:"foreignKey:EventID" json:"ticket_categories,omitempty"`
    Tickets           []Ticket         `gorm:"foreignKey:EventID" json:"tickets,omitempty"`
    Reports           []Report         `gorm:"foreignKey:EventID" json:"reports,omitempty"`
}

type TicketCategory struct {
    TicketCategoryID uint      `gorm:"primaryKey" json:"ticket_category_id"`
    EventID          uint      `json:"event_id"`
    Price            float64   `gorm:"type:decimal(10,2)" json:"price"`
    Quota            uint      `json:"quota"`
    Sold             uint      `gorm:"default:0" json:"sold"`
    Description      string    `gorm:"type:text" json:"description"`
    DateTimeStart    time.Time `json:"date_time_start"`
    DateTimeEnd      time.Time `json:"date_time_end"`
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
    
    // Relationships
    Event            Event    `gorm:"foreignKey:EventID" json:"event"`
    Tickets          []Ticket `gorm:"foreignKey:TicketCategoryID" json:"tickets,omitempty"`
    Carts            []Cart   `gorm:"foreignKey:TicketCategoryID" json:"carts,omitempty"`
    TransactionDetails []TransactionDetail `gorm:"foreignKey:TicketCategoryID" json:"transaction_details,omitempty"`
}

type Ticket struct {
    TicketID         uint      `gorm:"primaryKey" json:"ticket_id"`
    EventID          uint      `json:"event_id"`
    TicketCategoryID uint      `json:"ticket_category_id"`
    OwnerID          uuid.UUID `gorm:"type:char(36)" json:"owner_id"`
    Status           string    `gorm:"size:20;default:active" json:"status"` // active, used, cancelled
    Code             string    `gorm:"size:100;uniqueIndex" json:"code"`
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
    
    // Relationships
    Event            Event         `gorm:"foreignKey:EventID" json:"event"`
    TicketCategory   TicketCategory `gorm:"foreignKey:TicketCategoryID" json:"ticket_category"`
    Owner            User          `gorm:"foreignKey:OwnerID" json:"owner"`
}

type Cart struct {
    CartID           uint      `gorm:"primaryKey" json:"cart_id"`
    TicketCategoryID uint      `json:"ticket_category_id"`
    OwnerID          uuid.UUID `gorm:"type:char(36)" json:"owner_id"`
    Quantity         uint      `gorm:"default:1" json:"quantity"`
    PriceTotal       float64   `gorm:"type:decimal(10,2)" json:"price_total"`
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
    
    // Relationships
    TicketCategory   TicketCategory `gorm:"foreignKey:TicketCategoryID" json:"ticket_category"`
    Owner            User          `gorm:"foreignKey:OwnerID" json:"owner"`
}

type TransactionHistory struct {
    TransactionID    uint      `gorm:"primaryKey" json:"transaction_id"`
    OwnerID          uuid.UUID `gorm:"type:char(36)" json:"owner_id"`
    TransactionTime  time.Time `json:"transaction_time"`
    PriceTotal       float64   `gorm:"type:decimal(10,2)" json:"price_total"`
    CreatedAt        time.Time `json:"created_at"`
    
    // Relationships
    Owner            User               `gorm:"foreignKey:OwnerID" json:"owner"`
    TransactionDetails []TransactionDetail `gorm:"foreignKey:TransactionID" json:"transaction_details,omitempty"`
}

type TransactionDetail struct {
    TransactionDetailID uint    `gorm:"primaryKey" json:"transaction_detail_id"`
    TicketCategoryID    uint    `json:"ticket_category_id"`
    TransactionID       uint    `json:"transaction_id"`
    OwnerID             uuid.UUID `gorm:"type:char(36)" json:"owner_id"`
    Quantity            uint    `json:"quantity"`
    Subtotal            float64 `gorm:"type:decimal(10,2)" json:"subtotal"`
    
    // Relationships
    TicketCategory      TicketCategory    `gorm:"foreignKey:TicketCategoryID" json:"ticket_category"`
    TransactionHistory  TransactionHistory `gorm:"foreignKey:TransactionID" json:"transaction_history"`
    Owner               User              `gorm:"foreignKey:OwnerID" json:"owner"`
}

type Report struct {
    ReportID        uint    `gorm:"primaryKey" json:"report_id"`
    EventID         uint    `json:"event_id"`
    OwnerID         uuid.UUID `gorm:"type:char(36)" json:"owner_id"`
    TotalAttendant  uint    `json:"total_attendant"`
    TotalSales      float64 `gorm:"type:decimal(10,2)" json:"total_sales"`
    
    // Relationships
    Event           Event `gorm:"foreignKey:EventID" json:"event"`
    Owner           User  `gorm:"foreignKey:OwnerID" json:"owner"`
}