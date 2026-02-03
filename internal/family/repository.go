package family

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles family account data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new family repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// FAMILY ACCOUNTS
// ========================================

// CreateFamily creates a new family account
func (r *Repository) CreateFamily(ctx context.Context, f *FamilyAccount) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO family_accounts (
			id, owner_id, name, monthly_budget, current_spend,
			budget_reset_day, currency, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		f.ID, f.OwnerID, f.Name, f.MonthlyBudget, f.CurrentSpend,
		f.BudgetResetDay, f.Currency, f.IsActive, f.CreatedAt, f.UpdatedAt,
	)
	return err
}

// GetFamilyByID retrieves a family account
func (r *Repository) GetFamilyByID(ctx context.Context, id uuid.UUID) (*FamilyAccount, error) {
	f := &FamilyAccount{}
	err := r.db.QueryRow(ctx, `
		SELECT id, owner_id, name, monthly_budget, current_spend,
			budget_reset_day, currency, is_active, created_at, updated_at
		FROM family_accounts WHERE id = $1`, id,
	).Scan(
		&f.ID, &f.OwnerID, &f.Name, &f.MonthlyBudget, &f.CurrentSpend,
		&f.BudgetResetDay, &f.Currency, &f.IsActive, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// GetFamilyByOwner retrieves a family account by owner
func (r *Repository) GetFamilyByOwner(ctx context.Context, ownerID uuid.UUID) (*FamilyAccount, error) {
	f := &FamilyAccount{}
	err := r.db.QueryRow(ctx, `
		SELECT id, owner_id, name, monthly_budget, current_spend,
			budget_reset_day, currency, is_active, created_at, updated_at
		FROM family_accounts WHERE owner_id = $1 AND is_active = true`, ownerID,
	).Scan(
		&f.ID, &f.OwnerID, &f.Name, &f.MonthlyBudget, &f.CurrentSpend,
		&f.BudgetResetDay, &f.Currency, &f.IsActive, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// UpdateFamily updates a family account
func (r *Repository) UpdateFamily(ctx context.Context, f *FamilyAccount) error {
	_, err := r.db.Exec(ctx, `
		UPDATE family_accounts
		SET name = $2, monthly_budget = $3, updated_at = $4
		WHERE id = $1`,
		f.ID, f.Name, f.MonthlyBudget, f.UpdatedAt,
	)
	return err
}

// IncrementSpend adds to the family's current monthly spend
func (r *Repository) IncrementSpend(ctx context.Context, familyID uuid.UUID, amount float64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE family_accounts
		SET current_spend = current_spend + $2, updated_at = NOW()
		WHERE id = $1`,
		familyID, amount,
	)
	return err
}

// ResetMonthlySpend resets monthly spend for all families with matching reset day
func (r *Repository) ResetMonthlySpend(ctx context.Context, dayOfMonth int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE family_accounts
		SET current_spend = 0, updated_at = NOW()
		WHERE budget_reset_day = $1 AND is_active = true`,
		dayOfMonth,
	)
	return err
}

// ========================================
// MEMBERS
// ========================================

// AddMember adds a member to a family
func (r *Repository) AddMember(ctx context.Context, m *FamilyMember) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO family_members (
			id, family_id, user_id, role, display_name,
			use_shared_payment, ride_approval_required, max_fare_per_ride,
			allowed_hours_start, allowed_hours_end, monthly_limit, monthly_spend,
			is_active, joined_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		m.ID, m.FamilyID, m.UserID, m.Role, m.DisplayName,
		m.UseSharedPayment, m.RideApprovalReq, m.MaxFarePerRide,
		m.AllowedHoursStart, m.AllowedHoursEnd, m.MonthlyLimit, m.MonthlySpend,
		m.IsActive, m.JoinedAt, m.CreatedAt, m.UpdatedAt,
	)
	return err
}

// GetMembersByFamily retrieves all members of a family
func (r *Repository) GetMembersByFamily(ctx context.Context, familyID uuid.UUID) ([]FamilyMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, family_id, user_id, role, display_name,
			use_shared_payment, ride_approval_required, max_fare_per_ride,
			allowed_hours_start, allowed_hours_end, monthly_limit, monthly_spend,
			is_active, joined_at, created_at, updated_at
		FROM family_members
		WHERE family_id = $1 AND is_active = true
		ORDER BY role, display_name`, familyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []FamilyMember
	for rows.Next() {
		m := FamilyMember{}
		if err := rows.Scan(
			&m.ID, &m.FamilyID, &m.UserID, &m.Role, &m.DisplayName,
			&m.UseSharedPayment, &m.RideApprovalReq, &m.MaxFarePerRide,
			&m.AllowedHoursStart, &m.AllowedHoursEnd, &m.MonthlyLimit, &m.MonthlySpend,
			&m.IsActive, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

// GetMemberByUserID retrieves a member by user ID
func (r *Repository) GetMemberByUserID(ctx context.Context, familyID, userID uuid.UUID) (*FamilyMember, error) {
	m := &FamilyMember{}
	err := r.db.QueryRow(ctx, `
		SELECT id, family_id, user_id, role, display_name,
			use_shared_payment, ride_approval_required, max_fare_per_ride,
			allowed_hours_start, allowed_hours_end, monthly_limit, monthly_spend,
			is_active, joined_at, created_at, updated_at
		FROM family_members
		WHERE family_id = $1 AND user_id = $2 AND is_active = true`, familyID, userID,
	).Scan(
		&m.ID, &m.FamilyID, &m.UserID, &m.Role, &m.DisplayName,
		&m.UseSharedPayment, &m.RideApprovalReq, &m.MaxFarePerRide,
		&m.AllowedHoursStart, &m.AllowedHoursEnd, &m.MonthlyLimit, &m.MonthlySpend,
		&m.IsActive, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// GetFamilyForUser finds which family a user belongs to
func (r *Repository) GetFamilyForUser(ctx context.Context, userID uuid.UUID) (*FamilyAccount, *FamilyMember, error) {
	m := &FamilyMember{}
	err := r.db.QueryRow(ctx, `
		SELECT id, family_id, user_id, role, display_name,
			use_shared_payment, ride_approval_required, max_fare_per_ride,
			allowed_hours_start, allowed_hours_end, monthly_limit, monthly_spend,
			is_active, joined_at, created_at, updated_at
		FROM family_members
		WHERE user_id = $1 AND is_active = true
		LIMIT 1`, userID,
	).Scan(
		&m.ID, &m.FamilyID, &m.UserID, &m.Role, &m.DisplayName,
		&m.UseSharedPayment, &m.RideApprovalReq, &m.MaxFarePerRide,
		&m.AllowedHoursStart, &m.AllowedHoursEnd, &m.MonthlyLimit, &m.MonthlySpend,
		&m.IsActive, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	f, err := r.GetFamilyByID(ctx, m.FamilyID)
	if err != nil {
		return nil, nil, err
	}
	return f, m, nil
}

// UpdateMember updates a member's settings
func (r *Repository) UpdateMember(ctx context.Context, m *FamilyMember) error {
	_, err := r.db.Exec(ctx, `
		UPDATE family_members
		SET role = $2, use_shared_payment = $3, ride_approval_required = $4,
			max_fare_per_ride = $5, allowed_hours_start = $6, allowed_hours_end = $7,
			monthly_limit = $8, updated_at = $9
		WHERE id = $1`,
		m.ID, m.Role, m.UseSharedPayment, m.RideApprovalReq,
		m.MaxFarePerRide, m.AllowedHoursStart, m.AllowedHoursEnd,
		m.MonthlyLimit, m.UpdatedAt,
	)
	return err
}

// RemoveMember deactivates a member
func (r *Repository) RemoveMember(ctx context.Context, memberID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE family_members SET is_active = false, updated_at = NOW()
		WHERE id = $1`, memberID,
	)
	return err
}

// IncrementMemberSpend adds to a member's monthly spend
func (r *Repository) IncrementMemberSpend(ctx context.Context, memberID uuid.UUID, amount float64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE family_members
		SET monthly_spend = monthly_spend + $2, updated_at = NOW()
		WHERE id = $1`,
		memberID, amount,
	)
	return err
}

// ========================================
// INVITES
// ========================================

// CreateInvite creates a family invite
func (r *Repository) CreateInvite(ctx context.Context, inv *FamilyInvite) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO family_invites (
			id, family_id, inviter_id, invitee_id, email, phone,
			role, status, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		inv.ID, inv.FamilyID, inv.InviterID, inv.InviteeID,
		inv.Email, inv.Phone, inv.Role, inv.Status, inv.ExpiresAt, inv.CreatedAt,
	)
	return err
}

// GetInviteByID retrieves an invite
func (r *Repository) GetInviteByID(ctx context.Context, id uuid.UUID) (*FamilyInvite, error) {
	inv := &FamilyInvite{}
	err := r.db.QueryRow(ctx, `
		SELECT id, family_id, inviter_id, invitee_id, email, phone,
			role, status, expires_at, responded_at, created_at
		FROM family_invites WHERE id = $1`, id,
	).Scan(
		&inv.ID, &inv.FamilyID, &inv.InviterID, &inv.InviteeID,
		&inv.Email, &inv.Phone, &inv.Role, &inv.Status,
		&inv.ExpiresAt, &inv.RespondedAt, &inv.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// GetPendingInvitesForUser retrieves pending invites for a user
func (r *Repository) GetPendingInvitesForUser(ctx context.Context, userID uuid.UUID, email *string) ([]FamilyInvite, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, family_id, inviter_id, invitee_id, email, phone,
			role, status, expires_at, responded_at, created_at
		FROM family_invites
		WHERE status = $1 AND expires_at > NOW()
			AND (invitee_id = $2 OR email = $3)
		ORDER BY created_at DESC`,
		InviteStatusPending, userID, email,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []FamilyInvite
	for rows.Next() {
		inv := FamilyInvite{}
		if err := rows.Scan(
			&inv.ID, &inv.FamilyID, &inv.InviterID, &inv.InviteeID,
			&inv.Email, &inv.Phone, &inv.Role, &inv.Status,
			&inv.ExpiresAt, &inv.RespondedAt, &inv.CreatedAt,
		); err != nil {
			return nil, err
		}
		invites = append(invites, inv)
	}
	return invites, nil
}

// UpdateInviteStatus updates an invite status
func (r *Repository) UpdateInviteStatus(ctx context.Context, inviteID uuid.UUID, status InviteStatus) error {
	_, err := r.db.Exec(ctx, `
		UPDATE family_invites
		SET status = $2, responded_at = NOW()
		WHERE id = $1`,
		inviteID, status,
	)
	return err
}

// ========================================
// RIDE LOGS
// ========================================

// LogRide records a family ride
func (r *Repository) LogRide(ctx context.Context, log *FamilyRideLog) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO family_ride_logs (id, family_id, member_id, ride_id, fare_amount, paid_by_owner, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		log.ID, log.FamilyID, log.MemberID, log.RideID,
		log.FareAmount, log.PaidByOwner, log.CreatedAt,
	)
	return err
}

// GetSpendingReport returns spending breakdown for a family
func (r *Repository) GetSpendingReport(ctx context.Context, familyID uuid.UUID) ([]MemberSpend, int, float64, error) {
	rows, err := r.db.Query(ctx, `
		SELECT fm.id, fm.display_name,
			COALESCE(SUM(frl.fare_amount), 0) as total_spend,
			COUNT(frl.id) as ride_count,
			fm.monthly_limit
		FROM family_members fm
		LEFT JOIN family_ride_logs frl ON frl.member_id = fm.id
			AND frl.created_at >= date_trunc('month', NOW())
		WHERE fm.family_id = $1 AND fm.is_active = true
		GROUP BY fm.id, fm.display_name, fm.monthly_limit
		ORDER BY total_spend DESC`, familyID,
	)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()

	var members []MemberSpend
	totalRides := 0
	totalSpend := 0.0
	for rows.Next() {
		ms := MemberSpend{}
		if err := rows.Scan(&ms.MemberID, &ms.DisplayName, &ms.Amount, &ms.RideCount, &ms.Limit); err != nil {
			return nil, 0, 0, err
		}
		totalRides += ms.RideCount
		totalSpend += ms.Amount
		members = append(members, ms)
	}
	return members, totalRides, totalSpend, nil
}
