package domain

import "time"

// OwnerPIN хранит хэш PIN-кода владельца
type OwnerPIN struct {
	UserID    int64     `json:"user_id" gorm:"primaryKey"`
	PinHash   string    `json:"-" gorm:"column:pin_hash"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (OwnerPIN) TableName() string {
	return "owner_pins"
}

// ProcurementItem — элемент списка закупок
type ProcurementItem struct {
	ID          int64      `json:"id" gorm:"primaryKey"`
	OwnerID     int64      `json:"owner_id" gorm:"index"`
	Title       string     `json:"title" gorm:"not null"`
	Description string     `json:"description,omitempty"`
	Quantity    int        `json:"quantity" gorm:"default:1"`
	EstCost     float64    `json:"est_cost,omitempty"`
	Priority    string     `json:"priority" gorm:"default:'medium'"` // low, medium, high
	IsCompleted bool       `json:"is_completed" gorm:"default:false"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (ProcurementItem) TableName() string {
	return "procurement_items"
}

// MaintenanceItem — элемент списка обслуживания
type MaintenanceItem struct {
	ID          int64      `json:"id" gorm:"primaryKey"`
	OwnerID     int64      `json:"owner_id" gorm:"index"`
	Title       string     `json:"title" gorm:"not null"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status" gorm:"default:'pending'"` // pending, in_progress, completed
	Priority    string     `json:"priority" gorm:"default:'medium'"`
	AssignedTo  string     `json:"assigned_to,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (MaintenanceItem) TableName() string {
	return "maintenance_items"
}

// CompanyProfile — профиль компании владельца
type CompanyProfile struct {
	ID              int64             `json:"id" gorm:"primaryKey"`
	OwnerID         int64             `json:"owner_id" gorm:"uniqueIndex"`
	Logo            string            `json:"logo,omitempty"`
	CompanyName     string            `json:"company_name" gorm:"not null"`
	ContactPerson   string            `json:"contact_person"`
	Email           string            `json:"email"`
	Phone           string            `json:"phone"`
	Website         string            `json:"website,omitempty"`
	City            string            `json:"city"`
	CompanyType     string            `json:"company_type,omitempty"` // ИП, ТОО, АО
	Description     string            `json:"description,omitempty" gorm:"type:text"`
	Specialization  string            `json:"specialization,omitempty"`
	YearsExperience int               `json:"years_experience,omitempty"`
	TeamSize        int               `json:"team_size,omitempty"`
	WorkHours       string            `json:"work_hours,omitempty"`
	Services        []string          `json:"services,omitempty" gorm:"type:jsonb;serializer:json"`
	Socials         map[string]string `json:"socials,omitempty" gorm:"type:jsonb;serializer:json"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

func (CompanyProfile) TableName() string {
	return "company_profiles"
}

// PortfolioProject — проект в портфолио
type PortfolioProject struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	OwnerID   int64     `json:"owner_id" gorm:"index"`
	ImageURL  string    `json:"image_url" gorm:"not null"`
	Title     string    `json:"title"`
	Category  string    `json:"category,omitempty"`
	SortOrder int       `json:"sort_order" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
}

func (PortfolioProject) TableName() string {
	return "portfolio_projects"
}
