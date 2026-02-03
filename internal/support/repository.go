package support

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles support data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new support repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// TICKETS
// ========================================

// CreateTicket creates a new support ticket
func (r *Repository) CreateTicket(ctx context.Context, ticket *Ticket) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO support_tickets (
			id, user_id, ride_id, ticket_number, category, priority,
			status, subject, description, tags, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		ticket.ID, ticket.UserID, ticket.RideID, ticket.TicketNumber,
		ticket.Category, ticket.Priority, ticket.Status,
		ticket.Subject, ticket.Description, ticket.Tags,
		ticket.CreatedAt, ticket.UpdatedAt,
	)
	return err
}

// GetTicketByID retrieves a ticket by ID
func (r *Repository) GetTicketByID(ctx context.Context, id uuid.UUID) (*Ticket, error) {
	ticket := &Ticket{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, ride_id, assigned_to, ticket_number, category,
			priority, status, subject, description, tags,
			resolved_at, closed_at, created_at, updated_at
		FROM support_tickets WHERE id = $1`, id,
	).Scan(
		&ticket.ID, &ticket.UserID, &ticket.RideID, &ticket.AssignedTo,
		&ticket.TicketNumber, &ticket.Category, &ticket.Priority, &ticket.Status,
		&ticket.Subject, &ticket.Description, &ticket.Tags,
		&ticket.ResolvedAt, &ticket.ClosedAt, &ticket.CreatedAt, &ticket.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return ticket, nil
}

// GetUserTickets returns tickets for a user
func (r *Repository) GetUserTickets(ctx context.Context, userID uuid.UUID, status *TicketStatus, limit, offset int) ([]TicketSummary, int, error) {
	countQuery := `SELECT COUNT(*) FROM support_tickets WHERE user_id = $1`
	query := `
		SELECT t.id, t.ticket_number, t.category, t.priority, t.status,
			t.subject, t.created_at, t.updated_at,
			(SELECT COUNT(*) FROM support_ticket_messages WHERE ticket_id = t.id AND is_internal = false),
			(SELECT message FROM support_ticket_messages WHERE ticket_id = t.id AND is_internal = false ORDER BY created_at DESC LIMIT 1)
		FROM support_tickets t
		WHERE t.user_id = $1`

	args := []interface{}{userID}
	argIdx := 2

	if status != nil {
		filter := fmt.Sprintf(" AND t.status = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		query += filter
		args = append(args, *status)
		argIdx++
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total)

	query += fmt.Sprintf(" ORDER BY t.updated_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tickets []TicketSummary
	for rows.Next() {
		ts := TicketSummary{}
		if err := rows.Scan(
			&ts.ID, &ts.TicketNumber, &ts.Category, &ts.Priority, &ts.Status,
			&ts.Subject, &ts.CreatedAt, &ts.UpdatedAt,
			&ts.MessageCount, &ts.LastMessage,
		); err != nil {
			return nil, 0, err
		}
		tickets = append(tickets, ts)
	}
	if tickets == nil {
		tickets = []TicketSummary{}
	}
	return tickets, total, nil
}

// UpdateTicket updates ticket fields
func (r *Repository) UpdateTicket(ctx context.Context, id uuid.UUID, status *TicketStatus, priority *TicketPriority, assignedTo *uuid.UUID, tags []string) error {
	now := time.Now()

	query := "UPDATE support_tickets SET updated_at = $2"
	args := []interface{}{id, now}
	argIdx := 3

	if status != nil {
		query += fmt.Sprintf(", status = $%d", argIdx)
		args = append(args, *status)
		argIdx++

		if *status == TicketStatusResolved {
			query += fmt.Sprintf(", resolved_at = $%d", argIdx)
			args = append(args, now)
			argIdx++
		}
		if *status == TicketStatusClosed {
			query += fmt.Sprintf(", closed_at = $%d", argIdx)
			args = append(args, now)
			argIdx++
		}
	}

	if priority != nil {
		query += fmt.Sprintf(", priority = $%d", argIdx)
		args = append(args, *priority)
		argIdx++
	}

	if assignedTo != nil {
		query += fmt.Sprintf(", assigned_to = $%d", argIdx)
		args = append(args, *assignedTo)
		argIdx++
	}

	if tags != nil {
		query += fmt.Sprintf(", tags = $%d", argIdx)
		args = append(args, tags)
		argIdx++
	}

	query += " WHERE id = $1"
	_, err := r.db.Exec(ctx, query, args...)
	return err
}

// ========================================
// MESSAGES
// ========================================

// CreateMessage creates a new message in a ticket
func (r *Repository) CreateMessage(ctx context.Context, msg *TicketMessage) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO support_ticket_messages (
			id, ticket_id, sender_id, sender_type, message, attachments, is_internal, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		msg.ID, msg.TicketID, msg.SenderID, msg.SenderType,
		msg.Message, msg.Attachments, msg.IsInternal, msg.CreatedAt,
	)
	return err
}

// GetMessagesByTicket returns messages for a ticket
func (r *Repository) GetMessagesByTicket(ctx context.Context, ticketID uuid.UUID, includeInternal bool) ([]TicketMessage, error) {
	query := `
		SELECT id, ticket_id, sender_id, sender_type, message, attachments, is_internal, created_at
		FROM support_ticket_messages
		WHERE ticket_id = $1`

	if !includeInternal {
		query += " AND is_internal = false"
	}
	query += " ORDER BY created_at ASC"

	rows, err := r.db.Query(ctx, query, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []TicketMessage
	for rows.Next() {
		msg := TicketMessage{}
		if err := rows.Scan(
			&msg.ID, &msg.TicketID, &msg.SenderID, &msg.SenderType,
			&msg.Message, &msg.Attachments, &msg.IsInternal, &msg.CreatedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	if messages == nil {
		messages = []TicketMessage{}
	}
	return messages, nil
}

// ========================================
// FAQ ARTICLES
// ========================================

// GetFAQArticles returns active FAQ articles
func (r *Repository) GetFAQArticles(ctx context.Context, category *string) ([]FAQArticle, error) {
	query := `
		SELECT id, category, title, content, sort_order, is_active, view_count, created_at, updated_at
		FROM faq_articles
		WHERE is_active = true`

	args := []interface{}{}
	if category != nil {
		query += " AND category = $1"
		args = append(args, *category)
	}
	query += " ORDER BY sort_order ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []FAQArticle
	for rows.Next() {
		a := FAQArticle{}
		if err := rows.Scan(
			&a.ID, &a.Category, &a.Title, &a.Content, &a.SortOrder,
			&a.IsActive, &a.ViewCount, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}
	if articles == nil {
		articles = []FAQArticle{}
	}
	return articles, nil
}

// GetFAQArticleByID retrieves a single FAQ article
func (r *Repository) GetFAQArticleByID(ctx context.Context, id uuid.UUID) (*FAQArticle, error) {
	article := &FAQArticle{}
	err := r.db.QueryRow(ctx, `
		SELECT id, category, title, content, sort_order, is_active, view_count, created_at, updated_at
		FROM faq_articles WHERE id = $1`, id,
	).Scan(
		&article.ID, &article.Category, &article.Title, &article.Content,
		&article.SortOrder, &article.IsActive, &article.ViewCount,
		&article.CreatedAt, &article.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return article, nil
}

// IncrementFAQViewCount increments the view count of an FAQ article
func (r *Repository) IncrementFAQViewCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE faq_articles SET view_count = view_count + 1 WHERE id = $1`, id,
	)
	return err
}

// ========================================
// ADMIN - TICKET LISTING & STATS
// ========================================

// GetAllTickets returns tickets for admin with filters
func (r *Repository) GetAllTickets(ctx context.Context, status *TicketStatus, priority *TicketPriority, category *TicketCategory, limit, offset int) ([]TicketSummary, int, error) {
	countQuery := `SELECT COUNT(*) FROM support_tickets WHERE 1=1`
	query := `
		SELECT t.id, t.ticket_number, t.category, t.priority, t.status,
			t.subject, t.created_at, t.updated_at,
			(SELECT COUNT(*) FROM support_ticket_messages WHERE ticket_id = t.id),
			(SELECT message FROM support_ticket_messages WHERE ticket_id = t.id ORDER BY created_at DESC LIMIT 1)
		FROM support_tickets t
		WHERE 1=1`

	args := []interface{}{}
	argIdx := 1

	if status != nil {
		filter := fmt.Sprintf(" AND t.status = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		query += filter
		args = append(args, *status)
		argIdx++
	}
	if priority != nil {
		filter := fmt.Sprintf(" AND t.priority = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND priority = $%d", argIdx)
		query += filter
		args = append(args, *priority)
		argIdx++
	}
	if category != nil {
		filter := fmt.Sprintf(" AND t.category = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND category = $%d", argIdx)
		query += filter
		args = append(args, *category)
		argIdx++
	}

	var total int
	r.db.QueryRow(ctx, countQuery, args...).Scan(&total)

	query += fmt.Sprintf(" ORDER BY t.updated_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tickets []TicketSummary
	for rows.Next() {
		ts := TicketSummary{}
		if err := rows.Scan(
			&ts.ID, &ts.TicketNumber, &ts.Category, &ts.Priority, &ts.Status,
			&ts.Subject, &ts.CreatedAt, &ts.UpdatedAt,
			&ts.MessageCount, &ts.LastMessage,
		); err != nil {
			return nil, 0, err
		}
		tickets = append(tickets, ts)
	}
	if tickets == nil {
		tickets = []TicketSummary{}
	}
	return tickets, total, nil
}

// GetTicketStats returns aggregate ticket statistics
func (r *Repository) GetTicketStats(ctx context.Context) (*TicketStats, error) {
	stats := &TicketStats{}

	r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'open'),
			COUNT(*) FILTER (WHERE status = 'in_progress'),
			COUNT(*) FILTER (WHERE status = 'waiting_on_user'),
			COUNT(*) FILTER (WHERE status = 'resolved'),
			COUNT(*) FILTER (WHERE status = 'escalated'),
			COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 60) FILTER (WHERE resolved_at IS NOT NULL), 0)
		FROM support_tickets`,
	).Scan(
		&stats.TotalOpen, &stats.TotalInProgress, &stats.TotalWaiting,
		&stats.TotalResolved, &stats.TotalEscalated, &stats.AvgResolutionMin,
	)

	// By category
	stats.ByCategory = map[string]int{}
	rows, err := r.db.Query(ctx, `
		SELECT category, COUNT(*) FROM support_tickets
		WHERE status NOT IN ('resolved', 'closed')
		GROUP BY category`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cat string
			var cnt int
			if rows.Scan(&cat, &cnt) == nil {
				stats.ByCategory[cat] = cnt
			}
		}
	}

	// By priority
	stats.ByPriority = map[string]int{}
	rows2, err := r.db.Query(ctx, `
		SELECT priority, COUNT(*) FROM support_tickets
		WHERE status NOT IN ('resolved', 'closed')
		GROUP BY priority`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var pri string
			var cnt int
			if rows2.Scan(&pri, &cnt) == nil {
				stats.ByPriority[pri] = cnt
			}
		}
	}

	return stats, nil
}
