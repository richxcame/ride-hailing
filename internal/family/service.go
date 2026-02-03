package family

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
)

const (
	maxFamilyMembers = 10
	inviteExpiryDays = 7
)

// Service handles family account business logic
type Service struct {
	repo *Repository
}

// NewService creates a new family service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ========================================
// FAMILY MANAGEMENT
// ========================================

// CreateFamily creates a new family account and adds the owner as a member
func (s *Service) CreateFamily(ctx context.Context, ownerID uuid.UUID, req *CreateFamilyRequest) (*FamilyResponse, error) {
	// Check if user already owns a family
	existing, err := s.repo.GetFamilyByOwner(ctx, ownerID)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("check existing family: %w", err)
	}
	if existing != nil {
		return nil, common.NewConflictError("you already have a family account")
	}

	// Check if user is already a member of another family
	existingFamily, _, err := s.repo.GetFamilyForUser(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("check family membership: %w", err)
	}
	if existingFamily != nil {
		return nil, common.NewConflictError("you are already a member of a family account")
	}

	currency := "USD"
	if req.Currency != "" {
		currency = req.Currency
	}

	budgetResetDay := 1
	now := time.Now()

	family := &FamilyAccount{
		ID:             uuid.New(),
		OwnerID:        ownerID,
		Name:           req.Name,
		MonthlyBudget:  req.MonthlyBudget,
		CurrentSpend:   0,
		BudgetResetDay: budgetResetDay,
		Currency:       currency,
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.CreateFamily(ctx, family); err != nil {
		return nil, fmt.Errorf("create family: %w", err)
	}

	// Add owner as first member
	ownerMember := &FamilyMember{
		ID:               uuid.New(),
		FamilyID:         family.ID,
		UserID:           ownerID,
		Role:             MemberRoleOwner,
		DisplayName:      "Owner",
		UseSharedPayment: true,
		RideApprovalReq:  false,
		IsActive:         true,
		JoinedAt:         now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.repo.AddMember(ctx, ownerMember); err != nil {
		return nil, fmt.Errorf("add owner member: %w", err)
	}

	return &FamilyResponse{
		FamilyAccount: family,
		Members:       []FamilyMember{*ownerMember},
	}, nil
}

// GetFamily returns the family account with members
func (s *Service) GetFamily(ctx context.Context, familyID uuid.UUID, requesterID uuid.UUID) (*FamilyResponse, error) {
	family, err := s.repo.GetFamilyByID(ctx, familyID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("family not found", nil)
		}
		return nil, err
	}

	// Verify requester is a member
	_, err = s.repo.GetMemberByUserID(ctx, familyID, requesterID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewForbiddenError("not a member of this family")
		}
		return nil, err
	}

	members, err := s.repo.GetMembersByFamily(ctx, familyID)
	if err != nil {
		return nil, err
	}
	if members == nil {
		members = []FamilyMember{}
	}

	return &FamilyResponse{
		FamilyAccount: family,
		Members:       members,
	}, nil
}

// GetMyFamily returns the family the user belongs to
func (s *Service) GetMyFamily(ctx context.Context, userID uuid.UUID) (*FamilyResponse, error) {
	family, _, err := s.repo.GetFamilyForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if family == nil {
		return nil, common.NewNotFoundError("you are not in any family", nil)
	}

	members, err := s.repo.GetMembersByFamily(ctx, family.ID)
	if err != nil {
		return nil, err
	}
	if members == nil {
		members = []FamilyMember{}
	}

	return &FamilyResponse{
		FamilyAccount: family,
		Members:       members,
	}, nil
}

// UpdateFamily updates family settings (owner only)
func (s *Service) UpdateFamily(ctx context.Context, familyID uuid.UUID, ownerID uuid.UUID, req *UpdateFamilyRequest) (*FamilyAccount, error) {
	family, err := s.repo.GetFamilyByID(ctx, familyID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("family not found", nil)
		}
		return nil, err
	}

	if family.OwnerID != ownerID {
		return nil, common.NewForbiddenError("only the family owner can update settings")
	}

	if req.Name != nil {
		family.Name = *req.Name
	}
	if req.MonthlyBudget != nil {
		family.MonthlyBudget = req.MonthlyBudget
	}
	family.UpdatedAt = time.Now()

	if err := s.repo.UpdateFamily(ctx, family); err != nil {
		return nil, fmt.Errorf("update family: %w", err)
	}

	return family, nil
}

// ========================================
// INVITATIONS
// ========================================

// InviteMember sends an invitation to join the family
func (s *Service) InviteMember(ctx context.Context, familyID uuid.UUID, inviterID uuid.UUID, req *InviteMemberRequest) (*FamilyInvite, error) {
	family, err := s.repo.GetFamilyByID(ctx, familyID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("family not found", nil)
		}
		return nil, err
	}

	// Verify inviter is the owner
	if family.OwnerID != inviterID {
		return nil, common.NewForbiddenError("only the family owner can invite members")
	}

	// Check member count
	members, err := s.repo.GetMembersByFamily(ctx, familyID)
	if err != nil {
		return nil, err
	}
	if len(members) >= maxFamilyMembers {
		return nil, common.NewBadRequestError(fmt.Sprintf("family can have at most %d members", maxFamilyMembers), nil)
	}

	// Validate invite target
	if req.Email == nil && req.Phone == nil && req.UserID == nil {
		return nil, common.NewBadRequestError("must provide email, phone, or user_id", nil)
	}

	// Cannot invite owner role
	if req.Role == MemberRoleOwner {
		return nil, common.NewBadRequestError("cannot invite someone as owner", nil)
	}

	// If user_id provided, check they're not already a member
	if req.UserID != nil {
		_, err := s.repo.GetMemberByUserID(ctx, familyID, *req.UserID)
		if err == nil {
			return nil, common.NewConflictError("user is already a member of this family")
		}
		if err != pgx.ErrNoRows {
			return nil, err
		}
	}

	now := time.Now()
	invite := &FamilyInvite{
		ID:        uuid.New(),
		FamilyID:  familyID,
		InviterID: inviterID,
		InviteeID: req.UserID,
		Email:     req.Email,
		Phone:     req.Phone,
		Role:      req.Role,
		Status:    InviteStatusPending,
		ExpiresAt: now.AddDate(0, 0, inviteExpiryDays),
		CreatedAt: now,
	}

	if err := s.repo.CreateInvite(ctx, invite); err != nil {
		return nil, fmt.Errorf("create invite: %w", err)
	}

	return invite, nil
}

// RespondToInvite accepts or declines a family invite
func (s *Service) RespondToInvite(ctx context.Context, inviteID uuid.UUID, userID uuid.UUID, req *RespondToInviteRequest) error {
	invite, err := s.repo.GetInviteByID(ctx, inviteID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return common.NewNotFoundError("invite not found", nil)
		}
		return err
	}

	if invite.Status != InviteStatusPending {
		return common.NewBadRequestError("invite is no longer pending", nil)
	}

	if invite.ExpiresAt.Before(time.Now()) {
		_ = s.repo.UpdateInviteStatus(ctx, inviteID, InviteStatusExpired)
		return common.NewBadRequestError("invite has expired", nil)
	}

	// Verify the invite is for this user
	if invite.InviteeID != nil && *invite.InviteeID != userID {
		return common.NewForbiddenError("this invite is not for you")
	}

	if !req.Accept {
		return s.repo.UpdateInviteStatus(ctx, inviteID, InviteStatusDeclined)
	}

	// Accept: check user is not already in a family
	existingFamily, _, err := s.repo.GetFamilyForUser(ctx, userID)
	if err != nil {
		return err
	}
	if existingFamily != nil {
		return common.NewConflictError("you are already a member of a family")
	}

	// Check member count
	members, err := s.repo.GetMembersByFamily(ctx, invite.FamilyID)
	if err != nil {
		return err
	}
	if len(members) >= maxFamilyMembers {
		return common.NewBadRequestError("family has reached maximum members", nil)
	}

	now := time.Now()
	member := &FamilyMember{
		ID:               uuid.New(),
		FamilyID:         invite.FamilyID,
		UserID:           userID,
		Role:             invite.Role,
		DisplayName:      "Member",
		UseSharedPayment: true,
		RideApprovalReq:  invite.Role == MemberRoleTeen || invite.Role == MemberRoleChild,
		IsActive:         true,
		JoinedAt:         now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.repo.AddMember(ctx, member); err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	return s.repo.UpdateInviteStatus(ctx, inviteID, InviteStatusAccepted)
}

// GetPendingInvites returns pending invites for the user
func (s *Service) GetPendingInvites(ctx context.Context, userID uuid.UUID) ([]FamilyInvite, error) {
	invites, err := s.repo.GetPendingInvitesForUser(ctx, userID, nil)
	if err != nil {
		return nil, err
	}
	if invites == nil {
		invites = []FamilyInvite{}
	}
	return invites, nil
}

// ========================================
// MEMBER MANAGEMENT
// ========================================

// UpdateMember updates a member's settings (owner only)
func (s *Service) UpdateMember(ctx context.Context, familyID uuid.UUID, memberID uuid.UUID, ownerID uuid.UUID, req *UpdateMemberRequest) (*FamilyMember, error) {
	family, err := s.repo.GetFamilyByID(ctx, familyID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("family not found", nil)
		}
		return nil, err
	}

	if family.OwnerID != ownerID {
		return nil, common.NewForbiddenError("only the family owner can update members")
	}

	// Find the member in the family
	members, err := s.repo.GetMembersByFamily(ctx, familyID)
	if err != nil {
		return nil, err
	}

	var member *FamilyMember
	for i := range members {
		if members[i].ID == memberID {
			member = &members[i]
			break
		}
	}
	if member == nil {
		return nil, common.NewNotFoundError("member not found", nil)
	}

	// Can't change owner role
	if member.Role == MemberRoleOwner {
		return nil, common.NewBadRequestError("cannot modify the owner's settings this way", nil)
	}

	// Apply updates
	if req.Role != nil && *req.Role != MemberRoleOwner {
		member.Role = *req.Role
	}
	if req.UseSharedPayment != nil {
		member.UseSharedPayment = *req.UseSharedPayment
	}
	if req.RideApprovalReq != nil {
		member.RideApprovalReq = *req.RideApprovalReq
	}
	if req.MaxFarePerRide != nil {
		member.MaxFarePerRide = req.MaxFarePerRide
	}
	if req.AllowedHoursStart != nil {
		if *req.AllowedHoursStart < 0 || *req.AllowedHoursStart > 23 {
			return nil, common.NewBadRequestError("allowed_hours_start must be 0-23", nil)
		}
		member.AllowedHoursStart = req.AllowedHoursStart
	}
	if req.AllowedHoursEnd != nil {
		if *req.AllowedHoursEnd < 0 || *req.AllowedHoursEnd > 23 {
			return nil, common.NewBadRequestError("allowed_hours_end must be 0-23", nil)
		}
		member.AllowedHoursEnd = req.AllowedHoursEnd
	}
	if req.MonthlyLimit != nil {
		member.MonthlyLimit = req.MonthlyLimit
	}
	member.UpdatedAt = time.Now()

	if err := s.repo.UpdateMember(ctx, member); err != nil {
		return nil, fmt.Errorf("update member: %w", err)
	}

	return member, nil
}

// RemoveMember removes a member from the family (owner only)
func (s *Service) RemoveMember(ctx context.Context, familyID uuid.UUID, memberID uuid.UUID, ownerID uuid.UUID) error {
	family, err := s.repo.GetFamilyByID(ctx, familyID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return common.NewNotFoundError("family not found", nil)
		}
		return err
	}

	if family.OwnerID != ownerID {
		return common.NewForbiddenError("only the family owner can remove members")
	}

	// Find the member
	members, err := s.repo.GetMembersByFamily(ctx, familyID)
	if err != nil {
		return err
	}

	var member *FamilyMember
	for i := range members {
		if members[i].ID == memberID {
			member = &members[i]
			break
		}
	}
	if member == nil {
		return common.NewNotFoundError("member not found", nil)
	}

	if member.Role == MemberRoleOwner {
		return common.NewBadRequestError("cannot remove the family owner", nil)
	}

	return s.repo.RemoveMember(ctx, memberID)
}

// LeaveFamily allows a member to leave a family voluntarily
func (s *Service) LeaveFamily(ctx context.Context, userID uuid.UUID) error {
	family, member, err := s.repo.GetFamilyForUser(ctx, userID)
	if err != nil {
		return err
	}
	if family == nil {
		return common.NewNotFoundError("you are not in any family", nil)
	}

	if member.Role == MemberRoleOwner {
		return common.NewBadRequestError("owner cannot leave the family; transfer ownership or delete the family", nil)
	}

	return s.repo.RemoveMember(ctx, member.ID)
}

// ========================================
// RIDE AUTHORIZATION
// ========================================

// AuthorizeRide checks if a family member can take a ride with the given fare
func (s *Service) AuthorizeRide(ctx context.Context, userID uuid.UUID, fareAmount float64) (*RideAuthorization, error) {
	family, member, err := s.repo.GetFamilyForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if family == nil || member == nil {
		return &RideAuthorization{
			Authorized:       true,
			UseSharedPayment: false,
			Reason:           "user is not in a family account",
		}, nil
	}

	if !member.UseSharedPayment {
		return &RideAuthorization{
			Authorized:       true,
			UseSharedPayment: false,
			Reason:           "member uses personal payment",
		}, nil
	}

	// Check family budget
	if family.MonthlyBudget != nil {
		if family.CurrentSpend+fareAmount > *family.MonthlyBudget {
			return &RideAuthorization{
				Authorized:       false,
				UseSharedPayment: true,
				Reason:           "family monthly budget would be exceeded",
			}, nil
		}
	}

	// Check member fare cap
	if member.MaxFarePerRide != nil && fareAmount > *member.MaxFarePerRide {
		return &RideAuthorization{
			Authorized:       false,
			UseSharedPayment: true,
			Reason:           fmt.Sprintf("fare exceeds per-ride limit of %.2f", *member.MaxFarePerRide),
		}, nil
	}

	// Check member monthly limit
	if member.MonthlyLimit != nil && member.MonthlySpend+fareAmount > *member.MonthlyLimit {
		return &RideAuthorization{
			Authorized:       false,
			UseSharedPayment: true,
			Reason:           "member monthly limit would be exceeded",
		}, nil
	}

	// Check allowed hours
	if member.AllowedHoursStart != nil && member.AllowedHoursEnd != nil {
		currentHour := time.Now().Hour()
		start := *member.AllowedHoursStart
		end := *member.AllowedHoursEnd

		var inRange bool
		if start <= end {
			inRange = currentHour >= start && currentHour < end
		} else {
			// Overnight range, e.g., 22-06
			inRange = currentHour >= start || currentHour < end
		}

		if !inRange {
			return &RideAuthorization{
				Authorized:       false,
				UseSharedPayment: true,
				Reason:           fmt.Sprintf("rides allowed only between %d:00 and %d:00", start, end),
			}, nil
		}
	}

	// Check if approval is required
	if member.RideApprovalReq {
		return &RideAuthorization{
			Authorized:       false,
			UseSharedPayment: true,
			NeedsApproval:    true,
			Reason:           "ride requires owner approval",
		}, nil
	}

	return &RideAuthorization{
		Authorized:       true,
		UseSharedPayment: true,
		FamilyID:         &family.ID,
		MemberID:         &member.ID,
	}, nil
}

// RecordFamilyRide records a ride taken on the family account and updates spending
func (s *Service) RecordFamilyRide(ctx context.Context, familyID uuid.UUID, memberID uuid.UUID, rideID uuid.UUID, fareAmount float64) error {
	log := &FamilyRideLog{
		ID:          uuid.New(),
		FamilyID:    familyID,
		MemberID:    memberID,
		RideID:      rideID,
		FareAmount:  fareAmount,
		PaidByOwner: true,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.LogRide(ctx, log); err != nil {
		return fmt.Errorf("log ride: %w", err)
	}

	// Increment family spend
	if err := s.repo.IncrementSpend(ctx, familyID, fareAmount); err != nil {
		return fmt.Errorf("increment family spend: %w", err)
	}

	// Increment member spend
	if err := s.repo.IncrementMemberSpend(ctx, memberID, fareAmount); err != nil {
		return fmt.Errorf("increment member spend: %w", err)
	}

	return nil
}

// ========================================
// SPENDING REPORTS
// ========================================

// GetSpendingReport returns spending breakdown for the family
func (s *Service) GetSpendingReport(ctx context.Context, familyID uuid.UUID, requesterID uuid.UUID) (*FamilySpendReport, error) {
	family, err := s.repo.GetFamilyByID(ctx, familyID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("family not found", nil)
		}
		return nil, err
	}

	// Verify requester is a member
	_, err = s.repo.GetMemberByUserID(ctx, familyID, requesterID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewForbiddenError("not a member of this family")
		}
		return nil, err
	}

	memberSpending, rideCount, totalSpend, err := s.repo.GetSpendingReport(ctx, familyID)
	if err != nil {
		return nil, err
	}
	if memberSpending == nil {
		memberSpending = []MemberSpend{}
	}

	report := &FamilySpendReport{
		FamilyID:       familyID,
		Period:         "current_month",
		TotalSpend:     totalSpend,
		Budget:         family.MonthlyBudget,
		MemberSpending: memberSpending,
		RideCount:      rideCount,
	}

	if family.MonthlyBudget != nil && *family.MonthlyBudget > 0 {
		pct := (totalSpend / *family.MonthlyBudget) * 100
		report.BudgetUsedPct = &pct
	}

	return report, nil
}

// ResetMonthlySpend resets spending for families with matching budget reset day
func (s *Service) ResetMonthlySpend(ctx context.Context, dayOfMonth int) error {
	return s.repo.ResetMonthlySpend(ctx, dayOfMonth)
}
