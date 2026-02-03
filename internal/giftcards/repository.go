package giftcards

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles gift card data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new gift cards repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// GIFT CARDS
// ========================================

// CreateCard inserts a new gift card
func (r *Repository) CreateCard(ctx context.Context, card *GiftCard) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO gift_cards (
			id, code, card_type, status, original_amount, remaining_amount, currency,
			purchaser_id, recipient_id, recipient_email, recipient_name,
			personal_message, design_template, expires_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		card.ID, card.Code, card.CardType, card.Status,
		card.OriginalAmount, card.RemainingAmount, card.Currency,
		card.PurchaserID, card.RecipientID, card.RecipientEmail, card.RecipientName,
		card.PersonalMessage, card.DesignTemplate, card.ExpiresAt,
		card.CreatedAt, card.UpdatedAt,
	)
	return err
}

// GetCardByCode retrieves a gift card by its redemption code
func (r *Repository) GetCardByCode(ctx context.Context, code string) (*GiftCard, error) {
	card := &GiftCard{}
	err := r.db.QueryRow(ctx, `
		SELECT id, code, card_type, status, original_amount, remaining_amount, currency,
			purchaser_id, recipient_id, recipient_email, recipient_name,
			personal_message, design_template, expires_at, redeemed_at,
			created_at, updated_at
		FROM gift_cards WHERE code = $1`, code,
	).Scan(
		&card.ID, &card.Code, &card.CardType, &card.Status,
		&card.OriginalAmount, &card.RemainingAmount, &card.Currency,
		&card.PurchaserID, &card.RecipientID, &card.RecipientEmail, &card.RecipientName,
		&card.PersonalMessage, &card.DesignTemplate, &card.ExpiresAt, &card.RedeemedAt,
		&card.CreatedAt, &card.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return card, nil
}

// GetCardByID retrieves a gift card by ID
func (r *Repository) GetCardByID(ctx context.Context, id uuid.UUID) (*GiftCard, error) {
	card := &GiftCard{}
	err := r.db.QueryRow(ctx, `
		SELECT id, code, card_type, status, original_amount, remaining_amount, currency,
			purchaser_id, recipient_id, recipient_email, recipient_name,
			personal_message, design_template, expires_at, redeemed_at,
			created_at, updated_at
		FROM gift_cards WHERE id = $1`, id,
	).Scan(
		&card.ID, &card.Code, &card.CardType, &card.Status,
		&card.OriginalAmount, &card.RemainingAmount, &card.Currency,
		&card.PurchaserID, &card.RecipientID, &card.RecipientEmail, &card.RecipientName,
		&card.PersonalMessage, &card.DesignTemplate, &card.ExpiresAt, &card.RedeemedAt,
		&card.CreatedAt, &card.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return card, nil
}

// RedeemCard assigns a card to a user
func (r *Repository) RedeemCard(ctx context.Context, cardID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE gift_cards
		SET recipient_id = $2, redeemed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = $3 AND recipient_id IS NULL`,
		cardID, userID, CardStatusActive,
	)
	return err
}

// DeductBalance atomically deducts from a card balance
func (r *Repository) DeductBalance(ctx context.Context, cardID uuid.UUID, amount float64) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE gift_cards
		SET remaining_amount = remaining_amount - $2,
			status = CASE WHEN remaining_amount - $2 <= 0 THEN 'redeemed' ELSE status END,
			updated_at = NOW()
		WHERE id = $1 AND status = $3 AND remaining_amount >= $2`,
		cardID, amount, CardStatusActive,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// GetActiveCardsByUser retrieves all active cards for a user
func (r *Repository) GetActiveCardsByUser(ctx context.Context, userID uuid.UUID) ([]GiftCard, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, code, card_type, status, original_amount, remaining_amount, currency,
			purchaser_id, recipient_id, recipient_email, recipient_name,
			personal_message, design_template, expires_at, redeemed_at,
			created_at, updated_at
		FROM gift_cards
		WHERE recipient_id = $1 AND status = $2 AND remaining_amount > 0
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at ASC`,
		userID, CardStatusActive,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []GiftCard
	for rows.Next() {
		card := GiftCard{}
		if err := rows.Scan(
			&card.ID, &card.Code, &card.CardType, &card.Status,
			&card.OriginalAmount, &card.RemainingAmount, &card.Currency,
			&card.PurchaserID, &card.RecipientID, &card.RecipientEmail, &card.RecipientName,
			&card.PersonalMessage, &card.DesignTemplate, &card.ExpiresAt, &card.RedeemedAt,
			&card.CreatedAt, &card.UpdatedAt,
		); err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	return cards, nil
}

// GetPurchasedCardsByUser retrieves cards a user purchased
func (r *Repository) GetPurchasedCardsByUser(ctx context.Context, userID uuid.UUID) ([]GiftCard, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, code, card_type, status, original_amount, remaining_amount, currency,
			purchaser_id, recipient_id, recipient_email, recipient_name,
			personal_message, design_template, expires_at, redeemed_at,
			created_at, updated_at
		FROM gift_cards
		WHERE purchaser_id = $1
		ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []GiftCard
	for rows.Next() {
		card := GiftCard{}
		if err := rows.Scan(
			&card.ID, &card.Code, &card.CardType, &card.Status,
			&card.OriginalAmount, &card.RemainingAmount, &card.Currency,
			&card.PurchaserID, &card.RecipientID, &card.RecipientEmail, &card.RecipientName,
			&card.PersonalMessage, &card.DesignTemplate, &card.ExpiresAt, &card.RedeemedAt,
			&card.CreatedAt, &card.UpdatedAt,
		); err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	return cards, nil
}

// ========================================
// TRANSACTIONS
// ========================================

// CreateTransaction records a gift card usage
func (r *Repository) CreateTransaction(ctx context.Context, tx *GiftCardTransaction) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO gift_card_transactions (
			id, card_id, user_id, ride_id, amount,
			balance_before, balance_after, description, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		tx.ID, tx.CardID, tx.UserID, tx.RideID,
		tx.Amount, tx.BalanceBefore, tx.BalanceAfter,
		tx.Description, tx.CreatedAt,
	)
	return err
}

// GetTransactionsByUser retrieves recent transactions for a user
func (r *Repository) GetTransactionsByUser(ctx context.Context, userID uuid.UUID, limit int) ([]GiftCardTransaction, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, card_id, user_id, ride_id, amount,
			balance_before, balance_after, description, created_at
		FROM gift_card_transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txns []GiftCardTransaction
	for rows.Next() {
		tx := GiftCardTransaction{}
		if err := rows.Scan(
			&tx.ID, &tx.CardID, &tx.UserID, &tx.RideID,
			&tx.Amount, &tx.BalanceBefore, &tx.BalanceAfter,
			&tx.Description, &tx.CreatedAt,
		); err != nil {
			return nil, err
		}
		txns = append(txns, tx)
	}
	return txns, nil
}

// GetTotalBalance returns the total gift card balance for a user
func (r *Repository) GetTotalBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	var total float64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(remaining_amount), 0)
		FROM gift_cards
		WHERE recipient_id = $1 AND status = $2 AND remaining_amount > 0
			AND (expires_at IS NULL OR expires_at > NOW())`,
		userID, CardStatusActive,
	).Scan(&total)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return total, nil
}

// ExpireCards marks expired cards
func (r *Repository) ExpireCards(ctx context.Context) (int64, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE gift_cards
		SET status = 'expired', updated_at = NOW()
		WHERE status = 'active' AND expires_at IS NOT NULL AND expires_at < NOW()`,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
