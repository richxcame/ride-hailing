package giftcards

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
)

// Service handles gift card business logic
type Service struct {
	repo *Repository
}

// NewService creates a new gift cards service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// PurchaseCard creates a new gift card (purchased by a user)
func (s *Service) PurchaseCard(ctx context.Context, purchaserID uuid.UUID, req *PurchaseGiftCardRequest) (*GiftCard, error) {
	if req.Amount < 5 || req.Amount > 500 {
		return nil, common.NewBadRequestError("amount must be between 5 and 500", nil)
	}

	currency := "USD"
	if req.Currency != "" {
		currency = req.Currency
	}

	now := time.Now()
	expiresAt := now.AddDate(1, 0, 0) // 1 year expiry

	card := &GiftCard{
		ID:              uuid.New(),
		Code:            generateGiftCode(),
		CardType:        CardTypePurchased,
		Status:          CardStatusActive,
		OriginalAmount:  req.Amount,
		RemainingAmount: req.Amount,
		Currency:        currency,
		PurchaserID:     &purchaserID,
		RecipientEmail:  req.RecipientEmail,
		RecipientName:   req.RecipientName,
		PersonalMessage: req.PersonalMessage,
		DesignTemplate:  req.DesignTemplate,
		ExpiresAt:       &expiresAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.CreateCard(ctx, card); err != nil {
		return nil, fmt.Errorf("create card: %w", err)
	}

	return card, nil
}

// RedeemCard applies a gift card to a user's account
func (s *Service) RedeemCard(ctx context.Context, userID uuid.UUID, req *RedeemGiftCardRequest) (*GiftCard, error) {
	card, err := s.repo.GetCardByCode(ctx, req.Code)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("gift card not found", nil)
		}
		return nil, err
	}

	if card.Status != CardStatusActive {
		return nil, common.NewBadRequestError("gift card is not active", nil)
	}

	if card.RemainingAmount <= 0 {
		return nil, common.NewBadRequestError("gift card has no remaining balance", nil)
	}

	if card.ExpiresAt != nil && card.ExpiresAt.Before(time.Now()) {
		return nil, common.NewBadRequestError("gift card has expired", nil)
	}

	// Check if already redeemed by someone else
	if card.RecipientID != nil && *card.RecipientID != userID {
		return nil, common.NewBadRequestError("gift card already redeemed by another user", nil)
	}

	// Assign to user if not already assigned
	if card.RecipientID == nil {
		if err := s.repo.RedeemCard(ctx, card.ID, userID); err != nil {
			return nil, err
		}
		card.RecipientID = &userID
		now := time.Now()
		card.RedeemedAt = &now
	}

	return card, nil
}

// CheckBalance returns gift card balance by code (public lookup)
func (s *Service) CheckBalance(ctx context.Context, code string) (*CheckBalanceResponse, error) {
	card, err := s.repo.GetCardByCode(ctx, code)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("gift card not found", nil)
		}
		return nil, err
	}

	isValid := card.Status == CardStatusActive &&
		card.RemainingAmount > 0 &&
		(card.ExpiresAt == nil || card.ExpiresAt.After(time.Now()))

	return &CheckBalanceResponse{
		Code:            card.Code,
		Status:          card.Status,
		OriginalAmount:  card.OriginalAmount,
		RemainingAmount: card.RemainingAmount,
		Currency:        card.Currency,
		ExpiresAt:       card.ExpiresAt,
		IsValid:         isValid,
	}, nil
}

// GetMySummary returns all gift card info for a user
func (s *Service) GetMySummary(ctx context.Context, userID uuid.UUID) (*GiftCardSummary, error) {
	cards, err := s.repo.GetActiveCardsByUser(ctx, userID)
	if err != nil {
		cards = []GiftCard{}
	}

	totalBalance, _ := s.repo.GetTotalBalance(ctx, userID)
	txns, _ := s.repo.GetTransactionsByUser(ctx, userID, 20)

	if cards == nil {
		cards = []GiftCard{}
	}
	if txns == nil {
		txns = []GiftCardTransaction{}
	}

	return &GiftCardSummary{
		TotalBalance:       totalBalance,
		ActiveCards:        len(cards),
		Cards:              cards,
		RecentTransactions: txns,
	}, nil
}

// GetPurchasedCards returns cards a user has purchased
func (s *Service) GetPurchasedCards(ctx context.Context, userID uuid.UUID) ([]GiftCard, error) {
	cards, err := s.repo.GetPurchasedCardsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if cards == nil {
		cards = []GiftCard{}
	}
	return cards, nil
}

// UseBalance deducts from gift card balance for a ride payment
// Returns the amount actually deducted (may be less than requested if insufficient balance)
func (s *Service) UseBalance(ctx context.Context, userID uuid.UUID, rideID uuid.UUID, amount float64) (float64, error) {
	cards, err := s.repo.GetActiveCardsByUser(ctx, userID)
	if err != nil {
		return 0, err
	}

	remaining := amount
	totalDeducted := 0.0

	// Use cards in FIFO order (oldest first)
	for _, card := range cards {
		if remaining <= 0 {
			break
		}

		deductAmount := math.Min(remaining, card.RemainingAmount)
		deductAmount = math.Round(deductAmount*100) / 100

		success, err := s.repo.DeductBalance(ctx, card.ID, deductAmount)
		if err != nil || !success {
			continue
		}

		// Record transaction
		tx := &GiftCardTransaction{
			ID:            uuid.New(),
			CardID:        card.ID,
			UserID:        userID,
			RideID:        &rideID,
			Amount:        deductAmount,
			BalanceBefore: card.RemainingAmount,
			BalanceAfter:  card.RemainingAmount - deductAmount,
			Description:   "Ride payment",
			CreatedAt:     time.Now(),
		}
		s.repo.CreateTransaction(ctx, tx)

		totalDeducted += deductAmount
		remaining -= deductAmount
	}

	return totalDeducted, nil
}

// GetTotalBalance returns the user's total available gift card balance
func (s *Service) GetTotalBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	return s.repo.GetTotalBalance(ctx, userID)
}

// ========================================
// ADMIN
// ========================================

// CreateBulk creates multiple gift cards at once (admin/corporate)
func (s *Service) CreateBulk(ctx context.Context, req *CreateBulkRequest) (*BulkCreateResponse, error) {
	currency := "USD"
	if req.Currency != "" {
		currency = req.Currency
	}

	now := time.Now()
	var expiresAt *time.Time
	if req.ExpiresInDays != nil {
		exp := now.AddDate(0, 0, *req.ExpiresInDays)
		expiresAt = &exp
	}

	var cards []GiftCard
	for i := 0; i < req.Count; i++ {
		card := GiftCard{
			ID:              uuid.New(),
			Code:            generateGiftCode(),
			CardType:        req.CardType,
			Status:          CardStatusActive,
			OriginalAmount:  req.Amount,
			RemainingAmount: req.Amount,
			Currency:        currency,
			ExpiresAt:       expiresAt,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		if err := s.repo.CreateCard(ctx, &card); err != nil {
			return nil, fmt.Errorf("create card %d: %w", i+1, err)
		}
		cards = append(cards, card)
	}

	return &BulkCreateResponse{
		Cards: cards,
		Count: len(cards),
		Total: float64(len(cards)) * req.Amount,
	}, nil
}

// ExpireCards marks expired cards (called by scheduler)
func (s *Service) ExpireCards(ctx context.Context) (int64, error) {
	return s.repo.ExpireCards(ctx)
}

// ========================================
// HELPERS
// ========================================

// generateGiftCode creates a readable gift card code (XXXX-XXXX-XXXX-XXXX)
func generateGiftCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // No confusing chars (0/O, 1/I/L)
	code := make([]byte, 16)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		code[i] = chars[n.Int64()]
	}
	return fmt.Sprintf("%s-%s-%s-%s",
		string(code[0:4]), string(code[4:8]),
		string(code[8:12]), string(code[12:16]))
}
