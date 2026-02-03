package family

import (
	"time"

	"github.com/google/uuid"
)

// MemberRole defines the member's role in the family account
type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"   // Account owner, full control
	MemberRoleAdult  MemberRole = "adult"   // Full access, shared payment
	MemberRoleTeen   MemberRole = "teen"    // Restricted, ride approval required
	MemberRoleChild  MemberRole = "child"   // Most restricted, always needs approval
)

// InviteStatus tracks family invitation state
type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"
	InviteStatusAccepted InviteStatus = "accepted"
	InviteStatusDeclined InviteStatus = "declined"
	InviteStatusExpired  InviteStatus = "expired"
)

// FamilyAccount represents a family group
type FamilyAccount struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OwnerID        uuid.UUID  `json:"owner_id" db:"owner_id"`
	Name           string     `json:"name" db:"name"`              // e.g., "The Smiths"
	MonthlyBudget  *float64   `json:"monthly_budget,omitempty" db:"monthly_budget"` // Spending cap
	CurrentSpend   float64    `json:"current_spend" db:"current_spend"`
	BudgetResetDay int        `json:"budget_reset_day" db:"budget_reset_day"` // Day of month to reset
	Currency       string     `json:"currency" db:"currency"`
	IsActive       bool       `json:"is_active" db:"is_active"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// FamilyMember represents a member in a family account
type FamilyMember struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	FamilyID        uuid.UUID  `json:"family_id" db:"family_id"`
	UserID          uuid.UUID  `json:"user_id" db:"user_id"`
	Role            MemberRole `json:"role" db:"role"`
	DisplayName     string     `json:"display_name" db:"display_name"`
	UseSharedPayment bool     `json:"use_shared_payment" db:"use_shared_payment"`
	RideApprovalReq  bool     `json:"ride_approval_required" db:"ride_approval_required"` // Need owner approval per ride
	MaxFarePerRide  *float64   `json:"max_fare_per_ride,omitempty" db:"max_fare_per_ride"`
	AllowedHoursStart *int     `json:"allowed_hours_start,omitempty" db:"allowed_hours_start"` // Hour 0-23
	AllowedHoursEnd   *int     `json:"allowed_hours_end,omitempty" db:"allowed_hours_end"`
	MonthlyLimit    *float64   `json:"monthly_limit,omitempty" db:"monthly_limit"` // Per-member spending limit
	MonthlySpend    float64    `json:"monthly_spend" db:"monthly_spend"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	JoinedAt        time.Time  `json:"joined_at" db:"joined_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// FamilyInvite represents an invitation to join a family account
type FamilyInvite struct {
	ID          uuid.UUID    `json:"id" db:"id"`
	FamilyID    uuid.UUID    `json:"family_id" db:"family_id"`
	InviterID   uuid.UUID    `json:"inviter_id" db:"inviter_id"`
	InviteeID   *uuid.UUID   `json:"invitee_id,omitempty" db:"invitee_id"`
	Email       *string      `json:"email,omitempty" db:"email"`
	Phone       *string      `json:"phone,omitempty" db:"phone"`
	Role        MemberRole   `json:"role" db:"role"`
	Status      InviteStatus `json:"status" db:"status"`
	ExpiresAt   time.Time    `json:"expires_at" db:"expires_at"`
	RespondedAt *time.Time   `json:"responded_at,omitempty" db:"responded_at"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
}

// FamilyRideLog tracks rides taken on the family account
type FamilyRideLog struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	FamilyID    uuid.UUID  `json:"family_id" db:"family_id"`
	MemberID    uuid.UUID  `json:"member_id" db:"member_id"`
	RideID      uuid.UUID  `json:"ride_id" db:"ride_id"`
	FareAmount  float64    `json:"fare_amount" db:"fare_amount"`
	PaidByOwner bool       `json:"paid_by_owner" db:"paid_by_owner"` // Charged to family payment
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// CreateFamilyRequest creates a new family account
type CreateFamilyRequest struct {
	Name          string   `json:"name" binding:"required"`
	MonthlyBudget *float64 `json:"monthly_budget,omitempty"`
	Currency      string   `json:"currency"`
}

// InviteMemberRequest invites someone to the family
type InviteMemberRequest struct {
	Email       *string    `json:"email,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	Role        MemberRole `json:"role" binding:"required"`
	DisplayName string     `json:"display_name" binding:"required"`
}

// RespondToInviteRequest accepts or declines an invite
type RespondToInviteRequest struct {
	Accept bool `json:"accept"`
}

// UpdateMemberRequest updates a member's settings
type UpdateMemberRequest struct {
	Role              *MemberRole `json:"role,omitempty"`
	UseSharedPayment  *bool       `json:"use_shared_payment,omitempty"`
	RideApprovalReq   *bool       `json:"ride_approval_required,omitempty"`
	MaxFarePerRide    *float64    `json:"max_fare_per_ride,omitempty"`
	AllowedHoursStart *int        `json:"allowed_hours_start,omitempty"`
	AllowedHoursEnd   *int        `json:"allowed_hours_end,omitempty"`
	MonthlyLimit      *float64    `json:"monthly_limit,omitempty"`
}

// UpdateFamilyRequest updates family settings
type UpdateFamilyRequest struct {
	Name          *string  `json:"name,omitempty"`
	MonthlyBudget *float64 `json:"monthly_budget,omitempty"`
}

// FamilyResponse returns family with members
type FamilyResponse struct {
	*FamilyAccount
	Members []FamilyMember `json:"members"`
}

// FamilySpendReport shows spending for a period
type FamilySpendReport struct {
	FamilyID       uuid.UUID       `json:"family_id"`
	Period         string          `json:"period"`
	TotalSpend     float64         `json:"total_spend"`
	Budget         *float64        `json:"budget,omitempty"`
	BudgetUsedPct  *float64        `json:"budget_used_pct,omitempty"`
	MemberSpending []MemberSpend   `json:"member_spending"`
	RideCount      int             `json:"ride_count"`
}

// MemberSpend shows per-member spending
type MemberSpend struct {
	MemberID    uuid.UUID `json:"member_id"`
	DisplayName string    `json:"display_name"`
	Amount      float64   `json:"amount"`
	RideCount   int       `json:"ride_count"`
	Limit       *float64  `json:"limit,omitempty"`
}

// RideAuthorization is the result of checking if a family member can take a ride
type RideAuthorization struct {
	Authorized       bool       `json:"authorized"`
	UseSharedPayment bool       `json:"use_shared_payment"`
	NeedsApproval    bool       `json:"needs_approval,omitempty"`
	FamilyID         *uuid.UUID `json:"family_id,omitempty"`
	MemberID         *uuid.UUID `json:"member_id,omitempty"`
	Reason           string     `json:"reason,omitempty"`
}
