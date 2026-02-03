package support

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
)

// Service handles support business logic
type Service struct {
	repo *Repository
}

// NewService creates a new support service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ========================================
// USER METHODS
// ========================================

// CreateTicket creates a new support ticket
func (s *Service) CreateTicket(ctx context.Context, userID uuid.UUID, req *CreateTicketRequest) (*Ticket, error) {
	priority := PriorityMedium
	if req.Priority != nil {
		priority = *req.Priority
	}

	// Auto-escalate safety issues
	if req.Category == CategorySafety {
		priority = PriorityUrgent
	}

	now := time.Now()
	ticket := &Ticket{
		ID:           uuid.New(),
		UserID:       userID,
		RideID:       req.RideID,
		TicketNumber: generateTicketNumber(),
		Category:     req.Category,
		Priority:     priority,
		Status:       TicketStatusOpen,
		Subject:      req.Subject,
		Description:  req.Description,
		Tags:         []string{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateTicket(ctx, ticket); err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}

	// Add initial description as first message
	msg := &TicketMessage{
		ID:          uuid.New(),
		TicketID:    ticket.ID,
		SenderID:    userID,
		SenderType:  SenderUser,
		Message:     req.Description,
		Attachments: []string{},
		IsInternal:  false,
		CreatedAt:   now,
	}
	s.repo.CreateMessage(ctx, msg)

	return ticket, nil
}

// GetMyTickets returns the current user's tickets
func (s *Service) GetMyTickets(ctx context.Context, userID uuid.UUID, status *TicketStatus, page, pageSize int) ([]TicketSummary, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.repo.GetUserTickets(ctx, userID, status, pageSize, offset)
}

// GetTicket returns a ticket with access check
func (s *Service) GetTicket(ctx context.Context, ticketID, userID uuid.UUID) (*Ticket, error) {
	ticket, err := s.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("ticket not found", nil)
		}
		return nil, err
	}

	if ticket.UserID != userID {
		return nil, common.NewForbiddenError("not authorized to view this ticket")
	}

	return ticket, nil
}

// GetTicketMessages returns messages for a ticket
func (s *Service) GetTicketMessages(ctx context.Context, ticketID, userID uuid.UUID) ([]TicketMessage, error) {
	ticket, err := s.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("ticket not found", nil)
		}
		return nil, err
	}

	if ticket.UserID != userID {
		return nil, common.NewForbiddenError("not authorized to view this ticket")
	}

	return s.repo.GetMessagesByTicket(ctx, ticketID, false)
}

// AddMessage adds a message to a ticket
func (s *Service) AddMessage(ctx context.Context, ticketID, userID uuid.UUID, req *AddMessageRequest) (*TicketMessage, error) {
	ticket, err := s.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("ticket not found", nil)
		}
		return nil, err
	}

	if ticket.UserID != userID {
		return nil, common.NewForbiddenError("not authorized to reply to this ticket")
	}

	if ticket.Status == TicketStatusClosed {
		return nil, common.NewBadRequestError("cannot reply to a closed ticket", nil)
	}

	attachments := req.Attachments
	if attachments == nil {
		attachments = []string{}
	}

	now := time.Now()
	msg := &TicketMessage{
		ID:          uuid.New(),
		TicketID:    ticketID,
		SenderID:    userID,
		SenderType:  SenderUser,
		Message:     req.Message,
		Attachments: attachments,
		IsInternal:  false,
		CreatedAt:   now,
	}

	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	// Reopen ticket if it was waiting on user
	if ticket.Status == TicketStatusWaiting || ticket.Status == TicketStatusResolved {
		open := TicketStatusOpen
		s.repo.UpdateTicket(ctx, ticketID, &open, nil, nil, nil)
	}

	return msg, nil
}

// CloseTicket allows a user to close their own ticket
func (s *Service) CloseTicket(ctx context.Context, ticketID, userID uuid.UUID) error {
	ticket, err := s.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return common.NewNotFoundError("ticket not found", nil)
		}
		return err
	}

	if ticket.UserID != userID {
		return common.NewForbiddenError("not authorized to close this ticket")
	}

	if ticket.Status == TicketStatusClosed {
		return common.NewBadRequestError("ticket is already closed", nil)
	}

	closed := TicketStatusClosed
	return s.repo.UpdateTicket(ctx, ticketID, &closed, nil, nil, nil)
}

// ========================================
// FAQ METHODS
// ========================================

// GetFAQArticles returns help center articles
func (s *Service) GetFAQArticles(ctx context.Context, category *string) ([]FAQArticle, error) {
	return s.repo.GetFAQArticles(ctx, category)
}

// GetFAQArticle returns a single article and increments view count
func (s *Service) GetFAQArticle(ctx context.Context, id uuid.UUID) (*FAQArticle, error) {
	article, err := s.repo.GetFAQArticleByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("article not found", nil)
		}
		return nil, err
	}

	s.repo.IncrementFAQViewCount(ctx, id)
	return article, nil
}

// ========================================
// ADMIN METHODS
// ========================================

// AdminGetTickets returns all tickets with filters
func (s *Service) AdminGetTickets(ctx context.Context, status *TicketStatus, priority *TicketPriority, category *TicketCategory, page, pageSize int) ([]TicketSummary, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.repo.GetAllTickets(ctx, status, priority, category, pageSize, offset)
}

// AdminGetTicket returns a ticket for admin (no ownership check)
func (s *Service) AdminGetTicket(ctx context.Context, ticketID uuid.UUID) (*Ticket, error) {
	ticket, err := s.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("ticket not found", nil)
		}
		return nil, err
	}
	return ticket, nil
}

// AdminGetMessages returns all messages including internal notes
func (s *Service) AdminGetMessages(ctx context.Context, ticketID uuid.UUID) ([]TicketMessage, error) {
	_, err := s.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("ticket not found", nil)
		}
		return nil, err
	}
	return s.repo.GetMessagesByTicket(ctx, ticketID, true)
}

// AdminReply adds an admin reply to a ticket
func (s *Service) AdminReply(ctx context.Context, ticketID, agentID uuid.UUID, req *AdminReplyRequest) (*TicketMessage, error) {
	ticket, err := s.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("ticket not found", nil)
		}
		return nil, err
	}

	if ticket.Status == TicketStatusClosed {
		return nil, common.NewBadRequestError("cannot reply to a closed ticket", nil)
	}

	now := time.Now()
	msg := &TicketMessage{
		ID:          uuid.New(),
		TicketID:    ticketID,
		SenderID:    agentID,
		SenderType:  SenderAgent,
		Message:     req.Message,
		Attachments: []string{},
		IsInternal:  req.IsInternal,
		CreatedAt:   now,
	}

	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("create admin reply: %w", err)
	}

	// Update status to in_progress or waiting_on_user
	if !req.IsInternal {
		waiting := TicketStatusWaiting
		s.repo.UpdateTicket(ctx, ticketID, &waiting, nil, &agentID, nil)
	} else if ticket.Status == TicketStatusOpen {
		inProgress := TicketStatusInProgress
		s.repo.UpdateTicket(ctx, ticketID, &inProgress, nil, &agentID, nil)
	}

	return msg, nil
}

// AdminUpdateTicket updates ticket metadata
func (s *Service) AdminUpdateTicket(ctx context.Context, ticketID uuid.UUID, req *UpdateTicketRequest) error {
	_, err := s.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return common.NewNotFoundError("ticket not found", nil)
		}
		return err
	}

	return s.repo.UpdateTicket(ctx, ticketID, req.Status, req.Priority, req.AssignedTo, req.Tags)
}

// AdminGetStats returns ticket analytics
func (s *Service) AdminGetStats(ctx context.Context) (*TicketStats, error) {
	return s.repo.GetTicketStats(ctx)
}

// ========================================
// HELPERS
// ========================================

func generateTicketNumber() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(999999))
	return fmt.Sprintf("TKT-%06d", n.Int64())
}
