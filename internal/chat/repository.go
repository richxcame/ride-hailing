package chat

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles chat data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new chat repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// MESSAGES
// ========================================

// SaveMessage persists a chat message
func (r *Repository) SaveMessage(ctx context.Context, msg *ChatMessage) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO chat_messages (
			id, ride_id, sender_id, sender_role, message_type,
			content, image_url, latitude, longitude,
			status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		msg.ID, msg.RideID, msg.SenderID, msg.SenderRole, msg.MessageType,
		msg.Content, msg.ImageURL, msg.Latitude, msg.Longitude,
		msg.Status, msg.CreatedAt,
	)
	return err
}

// GetMessagesByRide retrieves all messages for a ride
func (r *Repository) GetMessagesByRide(ctx context.Context, rideID uuid.UUID, limit, offset int) ([]ChatMessage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, ride_id, sender_id, sender_role, message_type,
			content, image_url, latitude, longitude,
			status, delivered_at, read_at, created_at
		FROM chat_messages
		WHERE ride_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3`,
		rideID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		m := ChatMessage{}
		if err := rows.Scan(
			&m.ID, &m.RideID, &m.SenderID, &m.SenderRole, &m.MessageType,
			&m.Content, &m.ImageURL, &m.Latitude, &m.Longitude,
			&m.Status, &m.DeliveredAt, &m.ReadAt, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

// GetMessageByID retrieves a single message
func (r *Repository) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*ChatMessage, error) {
	m := &ChatMessage{}
	err := r.db.QueryRow(ctx, `
		SELECT id, ride_id, sender_id, sender_role, message_type,
			content, image_url, latitude, longitude,
			status, delivered_at, read_at, created_at
		FROM chat_messages
		WHERE id = $1`, messageID,
	).Scan(
		&m.ID, &m.RideID, &m.SenderID, &m.SenderRole, &m.MessageType,
		&m.Content, &m.ImageURL, &m.Latitude, &m.Longitude,
		&m.Status, &m.DeliveredAt, &m.ReadAt, &m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// MarkMessagesDelivered marks all undelivered messages for a user as delivered
func (r *Repository) MarkMessagesDelivered(ctx context.Context, rideID, recipientID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE chat_messages
		SET status = $1, delivered_at = NOW()
		WHERE ride_id = $2 AND sender_id != $3 AND status = $4`,
		MessageStatusDelivered, rideID, recipientID, MessageStatusSent,
	)
	return err
}

// MarkMessagesRead marks messages as read up to a given message ID
func (r *Repository) MarkMessagesRead(ctx context.Context, rideID, recipientID, lastReadID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE chat_messages
		SET status = $1, read_at = NOW()
		WHERE ride_id = $2
			AND sender_id != $3
			AND status IN ($4, $5)
			AND created_at <= (SELECT created_at FROM chat_messages WHERE id = $6)`,
		MessageStatusRead, rideID, recipientID,
		MessageStatusSent, MessageStatusDelivered, lastReadID,
	)
	return err
}

// GetUnreadCount returns the count of unread messages for a user in a ride
func (r *Repository) GetUnreadCount(ctx context.Context, rideID, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM chat_messages
		WHERE ride_id = $1 AND sender_id != $2 AND status != $3`,
		rideID, userID, MessageStatusRead,
	).Scan(&count)
	return count, err
}

// GetLastMessage returns the most recent message in a ride
func (r *Repository) GetLastMessage(ctx context.Context, rideID uuid.UUID) (*ChatMessage, error) {
	m := &ChatMessage{}
	err := r.db.QueryRow(ctx, `
		SELECT id, ride_id, sender_id, sender_role, message_type,
			content, image_url, latitude, longitude,
			status, delivered_at, read_at, created_at
		FROM chat_messages
		WHERE ride_id = $1
		ORDER BY created_at DESC
		LIMIT 1`, rideID,
	).Scan(
		&m.ID, &m.RideID, &m.SenderID, &m.SenderRole, &m.MessageType,
		&m.Content, &m.ImageURL, &m.Latitude, &m.Longitude,
		&m.Status, &m.DeliveredAt, &m.ReadAt, &m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// DeleteMessagesByRide deletes all messages for a ride (cleanup after ride ends + grace period)
func (r *Repository) DeleteMessagesByRide(ctx context.Context, rideID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		"DELETE FROM chat_messages WHERE ride_id = $1", rideID,
	)
	return err
}

// GetMessageCount returns total messages in a ride
func (r *Repository) GetMessageCount(ctx context.Context, rideID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM chat_messages WHERE ride_id = $1", rideID,
	).Scan(&count)
	return count, err
}

// ========================================
// QUICK REPLIES
// ========================================

// GetQuickReplies returns predefined quick reply options for a role
func (r *Repository) GetQuickReplies(ctx context.Context, role string) ([]QuickReply, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, role, category, text, sort_order, is_active
		FROM chat_quick_replies
		WHERE (role = $1 OR role = 'both') AND is_active = true
		ORDER BY category, sort_order`, role,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var replies []QuickReply
	for rows.Next() {
		qr := QuickReply{}
		if err := rows.Scan(&qr.ID, &qr.Role, &qr.Category, &qr.Text, &qr.SortOrder, &qr.IsActive); err != nil {
			return nil, err
		}
		replies = append(replies, qr)
	}
	return replies, nil
}

// CreateQuickReply adds a new quick reply option (admin)
func (r *Repository) CreateQuickReply(ctx context.Context, qr *QuickReply) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO chat_quick_replies (id, role, category, text, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		qr.ID, qr.Role, qr.Category, qr.Text, qr.SortOrder, qr.IsActive,
	)
	return err
}

// ========================================
// ACTIVE CONVERSATIONS
// ========================================

// GetActiveConversations returns rides with recent chat activity for a user
func (r *Repository) GetActiveConversations(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT cm.ride_id
		FROM chat_messages cm
		JOIN rides r ON r.id = cm.ride_id
		WHERE (r.rider_id = $1 OR r.driver_id = $1)
			AND r.status IN ('accepted', 'in_progress')
		ORDER BY cm.ride_id`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query active conversations: %w", err)
	}
	defer rows.Close()

	var rideIDs []uuid.UUID
	for rows.Next() {
		var rideID uuid.UUID
		if err := rows.Scan(&rideID); err != nil {
			return nil, err
		}
		rideIDs = append(rideIDs, rideID)
	}
	return rideIDs, nil
}
