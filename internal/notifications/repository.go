package notifications

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/models"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateNotification creates a new notification record
func (r *Repository) CreateNotification(ctx context.Context, notification *models.Notification) error {
	query := `
		INSERT INTO notifications (id, user_id, type, channel, title, body,
			data, status, scheduled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		notification.ID,
		notification.UserID,
		notification.Type,
		notification.Channel,
		notification.Title,
		notification.Body,
		notification.Data,
		notification.Status,
		notification.ScheduledAt,
	).Scan(&notification.CreatedAt, &notification.UpdatedAt)

	if err != nil {
		return common.NewInternalError("failed to create notification", err)
	}

	return nil
}

// GetNotificationByID retrieves a notification by ID
func (r *Repository) GetNotificationByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	notification := &models.Notification{}
	query := `
		SELECT id, user_id, type, channel, title, body, data, status,
			scheduled_at, sent_at, read_at, error_message, created_at, updated_at
		FROM notifications
		WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&notification.ID,
		&notification.UserID,
		&notification.Type,
		&notification.Channel,
		&notification.Title,
		&notification.Body,
		&notification.Data,
		&notification.Status,
		&notification.ScheduledAt,
		&notification.SentAt,
		&notification.ReadAt,
		&notification.ErrorMessage,
		&notification.CreatedAt,
		&notification.UpdatedAt,
	)

	if err != nil {
		return nil, common.NewNotFoundError("notification not found", err)
	}

	return notification, nil
}

// UpdateNotificationStatus updates the notification status
func (r *Repository) UpdateNotificationStatus(ctx context.Context, id uuid.UUID, status string, errorMessage *string) error {
	var sentAt *time.Time
	if status == "sent" {
		now := time.Now()
		sentAt = &now
	}

	query := `
		UPDATE notifications
		SET status = $1, sent_at = COALESCE($2, sent_at),
		    error_message = $3, updated_at = NOW()
		WHERE id = $4`

	_, err := r.db.Exec(ctx, query, status, sentAt, errorMessage, id)
	if err != nil {
		return common.NewInternalError("failed to update notification status", err)
	}

	return nil
}

// MarkNotificationAsRead marks a notification as read
func (r *Repository) MarkNotificationAsRead(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE notifications
		SET read_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND read_at IS NULL`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return common.NewInternalError("failed to mark notification as read", err)
	}

	return nil
}

// GetUserNotifications retrieves notifications for a user
func (r *Repository) GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Notification, error) {
	query := `
		SELECT id, user_id, type, channel, title, body, data, status,
			scheduled_at, sent_at, read_at, error_message, created_at, updated_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, common.NewInternalError("failed to get notifications", err)
	}
	defer rows.Close()

	var notifications []*models.Notification
	for rows.Next() {
		notification := &models.Notification{}
		err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.Type,
			&notification.Channel,
			&notification.Title,
			&notification.Body,
			&notification.Data,
			&notification.Status,
			&notification.ScheduledAt,
			&notification.SentAt,
			&notification.ReadAt,
			&notification.ErrorMessage,
			&notification.CreatedAt,
			&notification.UpdatedAt,
		)
		if err != nil {
			return nil, common.NewInternalError("failed to scan notification", err)
		}
		notifications = append(notifications, notification)
	}

	return notifications, nil
}

// GetUnreadNotificationCount gets count of unread notifications
func (r *Repository) GetUnreadNotificationCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM notifications
		WHERE user_id = $1 AND read_at IS NULL AND status = 'sent'`

	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, common.NewInternalError("failed to get unread count", err)
	}

	return count, nil
}

// GetPendingNotifications retrieves notifications that need to be sent
func (r *Repository) GetPendingNotifications(ctx context.Context, limit int) ([]*models.Notification, error) {
	query := `
		SELECT id, user_id, type, channel, title, body, data, status,
			scheduled_at, sent_at, read_at, error_message, created_at, updated_at
		FROM notifications
		WHERE status = 'pending' AND (scheduled_at IS NULL OR scheduled_at <= NOW())
		ORDER BY created_at ASC
		LIMIT $1`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, common.NewInternalError("failed to get pending notifications", err)
	}
	defer rows.Close()

	var notifications []*models.Notification
	for rows.Next() {
		notification := &models.Notification{}
		err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.Type,
			&notification.Channel,
			&notification.Title,
			&notification.Body,
			&notification.Data,
			&notification.Status,
			&notification.ScheduledAt,
			&notification.SentAt,
			&notification.ReadAt,
			&notification.ErrorMessage,
			&notification.CreatedAt,
			&notification.UpdatedAt,
		)
		if err != nil {
			return nil, common.NewInternalError("failed to scan notification", err)
		}
		notifications = append(notifications, notification)
	}

	return notifications, nil
}

// GetUserDeviceTokens retrieves FCM device tokens for a user
func (r *Repository) GetUserDeviceTokens(ctx context.Context, userID uuid.UUID) ([]string, error) {
	// This would typically come from a device_tokens table
	// For now, we'll return an empty slice and implement this later
	// when we add a proper device registration system
	return []string{}, nil
}

// GetUserPhoneNumber retrieves user's phone number for SMS
func (r *Repository) GetUserPhoneNumber(ctx context.Context, userID uuid.UUID) (string, error) {
	var phoneNumber string
	query := `SELECT phone_number FROM users WHERE id = $1`

	err := r.db.QueryRow(ctx, query, userID).Scan(&phoneNumber)
	if err != nil {
		return "", common.NewNotFoundError("user not found", err)
	}

	return phoneNumber, nil
}

// GetUserEmail retrieves user's email for email notifications
func (r *Repository) GetUserEmail(ctx context.Context, userID uuid.UUID) (string, error) {
	var email string
	query := `SELECT email FROM users WHERE id = $1`

	err := r.db.QueryRow(ctx, query, userID).Scan(&email)
	if err != nil {
		return "", common.NewNotFoundError("user not found", err)
	}

	return email, nil
}
