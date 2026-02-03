package safety

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for safety features
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new safety repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// EMERGENCY ALERTS
// ========================================

// CreateEmergencyAlert creates a new emergency alert
func (r *Repository) CreateEmergencyAlert(ctx context.Context, alert *EmergencyAlert) error {
	query := `
		INSERT INTO emergency_alerts (
			id, user_id, ride_id, type, status,
			latitude, longitude, address, description,
			police_notified, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.Exec(ctx, query,
		alert.ID, alert.UserID, alert.RideID, alert.Type, alert.Status,
		alert.Latitude, alert.Longitude, alert.Address, alert.Description,
		alert.PoliceNotified, alert.CreatedAt, alert.UpdatedAt,
	)
	return err
}

// GetEmergencyAlert retrieves an emergency alert by ID
func (r *Repository) GetEmergencyAlert(ctx context.Context, id uuid.UUID) (*EmergencyAlert, error) {
	query := `
		SELECT id, user_id, ride_id, type, status,
			latitude, longitude, address, description,
			audio_recording_url, video_recording_url,
			responded_at, responded_by, resolved_at, resolution,
			police_notified, police_ref_number,
			created_at, updated_at
		FROM emergency_alerts
		WHERE id = $1
	`
	var alert EmergencyAlert
	err := r.db.QueryRow(ctx, query, id).Scan(
		&alert.ID, &alert.UserID, &alert.RideID, &alert.Type, &alert.Status,
		&alert.Latitude, &alert.Longitude, &alert.Address, &alert.Description,
		&alert.AudioRecordingURL, &alert.VideoRecordingURL,
		&alert.RespondedAt, &alert.RespondedBy, &alert.ResolvedAt, &alert.Resolution,
		&alert.PoliceNotified, &alert.PoliceRefNumber,
		&alert.CreatedAt, &alert.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &alert, err
}

// UpdateEmergencyAlertStatus updates the status of an emergency alert
func (r *Repository) UpdateEmergencyAlertStatus(ctx context.Context, id uuid.UUID, status EmergencyStatus, respondedBy *uuid.UUID, resolution string) error {
	query := `
		UPDATE emergency_alerts
		SET status = $2,
			responded_at = CASE WHEN $3::uuid IS NOT NULL THEN NOW() ELSE responded_at END,
			responded_by = COALESCE($3, responded_by),
			resolved_at = CASE WHEN $2 IN ('resolved', 'false_alarm', 'cancelled') THEN NOW() ELSE resolved_at END,
			resolution = COALESCE(NULLIF($4, ''), resolution),
			updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, status, respondedBy, resolution)
	return err
}

// GetActiveEmergencyAlerts retrieves all active emergency alerts
func (r *Repository) GetActiveEmergencyAlerts(ctx context.Context) ([]*EmergencyAlert, error) {
	query := `
		SELECT id, user_id, ride_id, type, status,
			latitude, longitude, address, description,
			audio_recording_url, video_recording_url,
			responded_at, responded_by, resolved_at, resolution,
			police_notified, police_ref_number,
			created_at, updated_at
		FROM emergency_alerts
		WHERE status IN ('active', 'responded')
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*EmergencyAlert
	for rows.Next() {
		var alert EmergencyAlert
		if err := rows.Scan(
			&alert.ID, &alert.UserID, &alert.RideID, &alert.Type, &alert.Status,
			&alert.Latitude, &alert.Longitude, &alert.Address, &alert.Description,
			&alert.AudioRecordingURL, &alert.VideoRecordingURL,
			&alert.RespondedAt, &alert.RespondedBy, &alert.ResolvedAt, &alert.Resolution,
			&alert.PoliceNotified, &alert.PoliceRefNumber,
			&alert.CreatedAt, &alert.UpdatedAt,
		); err != nil {
			return nil, err
		}
		alerts = append(alerts, &alert)
	}
	return alerts, nil
}

// GetUserEmergencyAlerts retrieves emergency alerts for a user
func (r *Repository) GetUserEmergencyAlerts(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*EmergencyAlert, error) {
	query := `
		SELECT id, user_id, ride_id, type, status,
			latitude, longitude, address, description,
			audio_recording_url, video_recording_url,
			responded_at, responded_by, resolved_at, resolution,
			police_notified, police_ref_number,
			created_at, updated_at
		FROM emergency_alerts
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*EmergencyAlert
	for rows.Next() {
		var alert EmergencyAlert
		if err := rows.Scan(
			&alert.ID, &alert.UserID, &alert.RideID, &alert.Type, &alert.Status,
			&alert.Latitude, &alert.Longitude, &alert.Address, &alert.Description,
			&alert.AudioRecordingURL, &alert.VideoRecordingURL,
			&alert.RespondedAt, &alert.RespondedBy, &alert.ResolvedAt, &alert.Resolution,
			&alert.PoliceNotified, &alert.PoliceRefNumber,
			&alert.CreatedAt, &alert.UpdatedAt,
		); err != nil {
			return nil, err
		}
		alerts = append(alerts, &alert)
	}
	return alerts, nil
}

// ========================================
// EMERGENCY CONTACTS
// ========================================

// CreateEmergencyContact creates a new emergency contact
func (r *Repository) CreateEmergencyContact(ctx context.Context, contact *EmergencyContact) error {
	// If this is primary, unset other primaries
	if contact.IsPrimary {
		_, _ = r.db.Exec(ctx, `UPDATE emergency_contacts SET is_primary = false WHERE user_id = $1`, contact.UserID)
	}

	query := `
		INSERT INTO emergency_contacts (
			id, user_id, name, phone, email, relationship,
			is_primary, notify_on_ride, notify_on_sos, verified,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.Exec(ctx, query,
		contact.ID, contact.UserID, contact.Name, contact.Phone, contact.Email, contact.Relationship,
		contact.IsPrimary, contact.NotifyOnRide, contact.NotifyOnSOS, contact.Verified,
		contact.CreatedAt, contact.UpdatedAt,
	)
	return err
}

// GetEmergencyContact retrieves an emergency contact by ID
func (r *Repository) GetEmergencyContact(ctx context.Context, id uuid.UUID) (*EmergencyContact, error) {
	query := `
		SELECT id, user_id, name, phone, email, relationship,
			is_primary, notify_on_ride, notify_on_sos, verified, verified_at,
			created_at, updated_at
		FROM emergency_contacts
		WHERE id = $1
	`
	var contact EmergencyContact
	err := r.db.QueryRow(ctx, query, id).Scan(
		&contact.ID, &contact.UserID, &contact.Name, &contact.Phone, &contact.Email, &contact.Relationship,
		&contact.IsPrimary, &contact.NotifyOnRide, &contact.NotifyOnSOS, &contact.Verified, &contact.VerifiedAt,
		&contact.CreatedAt, &contact.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &contact, err
}

// GetUserEmergencyContacts retrieves all emergency contacts for a user
func (r *Repository) GetUserEmergencyContacts(ctx context.Context, userID uuid.UUID) ([]*EmergencyContact, error) {
	query := `
		SELECT id, user_id, name, phone, email, relationship,
			is_primary, notify_on_ride, notify_on_sos, verified, verified_at,
			created_at, updated_at
		FROM emergency_contacts
		WHERE user_id = $1
		ORDER BY is_primary DESC, created_at ASC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*EmergencyContact
	for rows.Next() {
		var contact EmergencyContact
		if err := rows.Scan(
			&contact.ID, &contact.UserID, &contact.Name, &contact.Phone, &contact.Email, &contact.Relationship,
			&contact.IsPrimary, &contact.NotifyOnRide, &contact.NotifyOnSOS, &contact.Verified, &contact.VerifiedAt,
			&contact.CreatedAt, &contact.UpdatedAt,
		); err != nil {
			return nil, err
		}
		contacts = append(contacts, &contact)
	}
	return contacts, nil
}

// GetSOSContacts retrieves contacts that should be notified on SOS
func (r *Repository) GetSOSContacts(ctx context.Context, userID uuid.UUID) ([]*EmergencyContact, error) {
	query := `
		SELECT id, user_id, name, phone, email, relationship,
			is_primary, notify_on_ride, notify_on_sos, verified, verified_at,
			created_at, updated_at
		FROM emergency_contacts
		WHERE user_id = $1 AND notify_on_sos = true
		ORDER BY is_primary DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*EmergencyContact
	for rows.Next() {
		var contact EmergencyContact
		if err := rows.Scan(
			&contact.ID, &contact.UserID, &contact.Name, &contact.Phone, &contact.Email, &contact.Relationship,
			&contact.IsPrimary, &contact.NotifyOnRide, &contact.NotifyOnSOS, &contact.Verified, &contact.VerifiedAt,
			&contact.CreatedAt, &contact.UpdatedAt,
		); err != nil {
			return nil, err
		}
		contacts = append(contacts, &contact)
	}
	return contacts, nil
}

// UpdateEmergencyContact updates an emergency contact
func (r *Repository) UpdateEmergencyContact(ctx context.Context, contact *EmergencyContact) error {
	// If this is primary, unset other primaries
	if contact.IsPrimary {
		_, _ = r.db.Exec(ctx, `UPDATE emergency_contacts SET is_primary = false WHERE user_id = $1 AND id != $2`, contact.UserID, contact.ID)
	}

	query := `
		UPDATE emergency_contacts
		SET name = $2, phone = $3, email = $4, relationship = $5,
			is_primary = $6, notify_on_ride = $7, notify_on_sos = $8,
			updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query,
		contact.ID, contact.Name, contact.Phone, contact.Email, contact.Relationship,
		contact.IsPrimary, contact.NotifyOnRide, contact.NotifyOnSOS,
	)
	return err
}

// DeleteEmergencyContact deletes an emergency contact
func (r *Repository) DeleteEmergencyContact(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM emergency_contacts WHERE id = $1`, id)
	return err
}

// VerifyEmergencyContact marks a contact as verified
func (r *Repository) VerifyEmergencyContact(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE emergency_contacts SET verified = true, verified_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// ========================================
// RIDE SHARE LINKS
// ========================================

// CreateRideShareLink creates a new ride share link
func (r *Repository) CreateRideShareLink(ctx context.Context, link *RideShareLink) error {
	query := `
		INSERT INTO ride_share_links (
			id, ride_id, user_id, token, expires_at, is_active,
			share_location, share_driver, share_eta, share_route,
			view_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.Exec(ctx, query,
		link.ID, link.RideID, link.UserID, link.Token, link.ExpiresAt, link.IsActive,
		link.ShareLocation, link.ShareDriver, link.ShareETA, link.ShareRoute,
		link.ViewCount, link.CreatedAt, link.UpdatedAt,
	)
	return err
}

// GetRideShareLinkByToken retrieves a share link by token
func (r *Repository) GetRideShareLinkByToken(ctx context.Context, token string) (*RideShareLink, error) {
	query := `
		SELECT id, ride_id, user_id, token, expires_at, is_active,
			share_location, share_driver, share_eta, share_route,
			view_count, last_viewed_at, created_at, updated_at
		FROM ride_share_links
		WHERE token = $1 AND is_active = true AND expires_at > NOW()
	`
	var link RideShareLink
	err := r.db.QueryRow(ctx, query, token).Scan(
		&link.ID, &link.RideID, &link.UserID, &link.Token, &link.ExpiresAt, &link.IsActive,
		&link.ShareLocation, &link.ShareDriver, &link.ShareETA, &link.ShareRoute,
		&link.ViewCount, &link.LastViewedAt, &link.CreatedAt, &link.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &link, err
}

// GetRideShareLinks retrieves all share links for a ride
func (r *Repository) GetRideShareLinks(ctx context.Context, rideID uuid.UUID) ([]*RideShareLink, error) {
	query := `
		SELECT id, ride_id, user_id, token, expires_at, is_active,
			share_location, share_driver, share_eta, share_route,
			view_count, last_viewed_at, created_at, updated_at
		FROM ride_share_links
		WHERE ride_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, rideID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*RideShareLink
	for rows.Next() {
		var link RideShareLink
		if err := rows.Scan(
			&link.ID, &link.RideID, &link.UserID, &link.Token, &link.ExpiresAt, &link.IsActive,
			&link.ShareLocation, &link.ShareDriver, &link.ShareETA, &link.ShareRoute,
			&link.ViewCount, &link.LastViewedAt, &link.CreatedAt, &link.UpdatedAt,
		); err != nil {
			return nil, err
		}
		links = append(links, &link)
	}
	return links, nil
}

// IncrementShareLinkViewCount increments the view count for a share link
func (r *Repository) IncrementShareLinkViewCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE ride_share_links SET view_count = view_count + 1, last_viewed_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// DeactivateShareLink deactivates a share link
func (r *Repository) DeactivateShareLink(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE ride_share_links SET is_active = false, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// DeactivateRideShareLinks deactivates all share links for a ride
func (r *Repository) DeactivateRideShareLinks(ctx context.Context, rideID uuid.UUID) error {
	query := `UPDATE ride_share_links SET is_active = false, updated_at = NOW() WHERE ride_id = $1`
	_, err := r.db.Exec(ctx, query, rideID)
	return err
}

// ========================================
// SAFETY SETTINGS
// ========================================

// GetSafetySettings retrieves safety settings for a user
func (r *Repository) GetSafetySettings(ctx context.Context, userID uuid.UUID) (*SafetySettings, error) {
	query := `
		SELECT user_id, sos_enabled, sos_gesture, sos_auto_call_911,
			auto_share_with_contacts, share_on_ride_start, share_on_emergency,
			periodic_check_in, check_in_interval_mins,
			route_deviation_alert, speed_alert_enabled, long_stop_alert_mins,
			allow_ride_recording, record_audio_on_sos,
			night_ride_protection, night_protection_start, night_protection_end,
			created_at, updated_at
		FROM safety_settings
		WHERE user_id = $1
	`
	var settings SafetySettings
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&settings.UserID, &settings.SOSEnabled, &settings.SOSGesture, &settings.SOSAutoCall911,
		&settings.AutoShareWithContacts, &settings.ShareOnRideStart, &settings.ShareOnEmergency,
		&settings.PeriodicCheckIn, &settings.CheckInIntervalMins,
		&settings.RouteDeviationAlert, &settings.SpeedAlertEnabled, &settings.LongStopAlertMins,
		&settings.AllowRideRecording, &settings.RecordAudioOnSOS,
		&settings.NightRideProtection, &settings.NightProtectionStart, &settings.NightProtectionEnd,
		&settings.CreatedAt, &settings.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		// Return default settings
		return r.createDefaultSafetySettings(ctx, userID)
	}
	return &settings, err
}

// createDefaultSafetySettings creates default safety settings for a user
func (r *Repository) createDefaultSafetySettings(ctx context.Context, userID uuid.UUID) (*SafetySettings, error) {
	settings := &SafetySettings{
		UserID:                userID,
		SOSEnabled:            true,
		SOSGesture:            "shake",
		SOSAutoCall911:        false,
		AutoShareWithContacts: false,
		ShareOnRideStart:      false,
		ShareOnEmergency:      true,
		PeriodicCheckIn:       false,
		CheckInIntervalMins:   0,
		RouteDeviationAlert:   true,
		SpeedAlertEnabled:     false,
		LongStopAlertMins:     10,
		AllowRideRecording:    false,
		RecordAudioOnSOS:      true,
		NightRideProtection:   false,
		NightProtectionStart:  "22:00",
		NightProtectionEnd:    "06:00",
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	query := `
		INSERT INTO safety_settings (
			user_id, sos_enabled, sos_gesture, sos_auto_call_911,
			auto_share_with_contacts, share_on_ride_start, share_on_emergency,
			periodic_check_in, check_in_interval_mins,
			route_deviation_alert, speed_alert_enabled, long_stop_alert_mins,
			allow_ride_recording, record_audio_on_sos,
			night_ride_protection, night_protection_start, night_protection_end,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`
	_, err := r.db.Exec(ctx, query,
		settings.UserID, settings.SOSEnabled, settings.SOSGesture, settings.SOSAutoCall911,
		settings.AutoShareWithContacts, settings.ShareOnRideStart, settings.ShareOnEmergency,
		settings.PeriodicCheckIn, settings.CheckInIntervalMins,
		settings.RouteDeviationAlert, settings.SpeedAlertEnabled, settings.LongStopAlertMins,
		settings.AllowRideRecording, settings.RecordAudioOnSOS,
		settings.NightRideProtection, settings.NightProtectionStart, settings.NightProtectionEnd,
		settings.CreatedAt, settings.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return settings, nil
}

// UpdateSafetySettings updates safety settings for a user
func (r *Repository) UpdateSafetySettings(ctx context.Context, settings *SafetySettings) error {
	query := `
		UPDATE safety_settings
		SET sos_enabled = $2, sos_gesture = $3, sos_auto_call_911 = $4,
			auto_share_with_contacts = $5, share_on_ride_start = $6, share_on_emergency = $7,
			periodic_check_in = $8, check_in_interval_mins = $9,
			route_deviation_alert = $10, speed_alert_enabled = $11, long_stop_alert_mins = $12,
			allow_ride_recording = $13, record_audio_on_sos = $14,
			night_ride_protection = $15, night_protection_start = $16, night_protection_end = $17,
			updated_at = NOW()
		WHERE user_id = $1
	`
	_, err := r.db.Exec(ctx, query,
		settings.UserID, settings.SOSEnabled, settings.SOSGesture, settings.SOSAutoCall911,
		settings.AutoShareWithContacts, settings.ShareOnRideStart, settings.ShareOnEmergency,
		settings.PeriodicCheckIn, settings.CheckInIntervalMins,
		settings.RouteDeviationAlert, settings.SpeedAlertEnabled, settings.LongStopAlertMins,
		settings.AllowRideRecording, settings.RecordAudioOnSOS,
		settings.NightRideProtection, settings.NightProtectionStart, settings.NightProtectionEnd,
	)
	return err
}

// ========================================
// SAFETY CHECKS
// ========================================

// CreateSafetyCheck creates a new safety check
func (r *Repository) CreateSafetyCheck(ctx context.Context, check *SafetyCheck) error {
	query := `
		INSERT INTO safety_checks (
			id, ride_id, user_id, type, status, sent_at, trigger_reason, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		check.ID, check.RideID, check.UserID, check.Type, check.Status,
		check.SentAt, check.TriggerReason, check.CreatedAt,
	)
	return err
}

// UpdateSafetyCheckResponse updates the response to a safety check
func (r *Repository) UpdateSafetyCheckResponse(ctx context.Context, id uuid.UUID, status SafetyCheckStatus, response string) error {
	query := `
		UPDATE safety_checks
		SET status = $2, responded_at = NOW(), response = $3
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, status, response)
	return err
}

// GetPendingSafetyChecks retrieves all pending safety checks for a user
func (r *Repository) GetPendingSafetyChecks(ctx context.Context, userID uuid.UUID) ([]*SafetyCheck, error) {
	query := `
		SELECT id, ride_id, user_id, type, status, sent_at, responded_at, response, trigger_reason, escalated, escalated_at, created_at
		FROM safety_checks
		WHERE user_id = $1 AND status = 'pending'
		ORDER BY sent_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []*SafetyCheck
	for rows.Next() {
		var check SafetyCheck
		if err := rows.Scan(
			&check.ID, &check.RideID, &check.UserID, &check.Type, &check.Status,
			&check.SentAt, &check.RespondedAt, &check.Response, &check.TriggerReason,
			&check.Escalated, &check.EscalatedAt, &check.CreatedAt,
		); err != nil {
			return nil, err
		}
		checks = append(checks, &check)
	}
	return checks, nil
}

// EscalateSafetyCheck marks a safety check as escalated
func (r *Repository) EscalateSafetyCheck(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE safety_checks SET escalated = true, escalated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// GetExpiredPendingSafetyChecks retrieves pending checks that have exceeded the timeout
func (r *Repository) GetExpiredPendingSafetyChecks(ctx context.Context, timeout time.Duration) ([]*SafetyCheck, error) {
	query := `
		SELECT id, ride_id, user_id, type, status, sent_at, responded_at, response, trigger_reason, escalated, escalated_at, created_at
		FROM safety_checks
		WHERE status = 'pending' AND sent_at < $1
		ORDER BY sent_at ASC
	`
	rows, err := r.db.Query(ctx, query, time.Now().Add(-timeout))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []*SafetyCheck
	for rows.Next() {
		var check SafetyCheck
		if err := rows.Scan(
			&check.ID, &check.RideID, &check.UserID, &check.Type, &check.Status,
			&check.SentAt, &check.RespondedAt, &check.Response, &check.TriggerReason,
			&check.Escalated, &check.EscalatedAt, &check.CreatedAt,
		); err != nil {
			return nil, err
		}
		checks = append(checks, &check)
	}
	return checks, nil
}

// ========================================
// ROUTE DEVIATION ALERTS
// ========================================

// CreateRouteDeviationAlert creates a new route deviation alert
func (r *Repository) CreateRouteDeviationAlert(ctx context.Context, alert *RouteDeviationAlert) error {
	query := `
		INSERT INTO route_deviation_alerts (
			id, ride_id, driver_id, expected_lat, expected_lng, actual_lat, actual_lng,
			deviation_meters, expected_route, actual_route, rider_notified, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.Exec(ctx, query,
		alert.ID, alert.RideID, alert.DriverID, alert.ExpectedLat, alert.ExpectedLng,
		alert.ActualLat, alert.ActualLng, alert.DeviationMeters,
		alert.ExpectedRoute, alert.ActualRoute, alert.RiderNotified, alert.CreatedAt,
	)
	return err
}

// UpdateRouteDeviationAcknowledgement updates the acknowledgement of a route deviation
func (r *Repository) UpdateRouteDeviationAcknowledgement(ctx context.Context, id uuid.UUID, response string) error {
	query := `UPDATE route_deviation_alerts SET driver_response = $2, acknowledged = true, acknowledged_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, response)
	return err
}

// GetRecentRouteDeviations retrieves recent route deviations for a ride
func (r *Repository) GetRecentRouteDeviations(ctx context.Context, rideID uuid.UUID, since time.Duration) ([]*RouteDeviationAlert, error) {
	query := `
		SELECT id, ride_id, driver_id, expected_lat, expected_lng, actual_lat, actual_lng,
			deviation_meters, expected_route, actual_route, rider_notified, driver_response,
			acknowledged, acknowledged_at, created_at
		FROM route_deviation_alerts
		WHERE ride_id = $1 AND created_at > $2
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, rideID, time.Now().Add(-since))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*RouteDeviationAlert
	for rows.Next() {
		var alert RouteDeviationAlert
		if err := rows.Scan(
			&alert.ID, &alert.RideID, &alert.DriverID, &alert.ExpectedLat, &alert.ExpectedLng,
			&alert.ActualLat, &alert.ActualLng, &alert.DeviationMeters,
			&alert.ExpectedRoute, &alert.ActualRoute, &alert.RiderNotified, &alert.DriverResponse,
			&alert.Acknowledged, &alert.AcknowledgedAt, &alert.CreatedAt,
		); err != nil {
			return nil, err
		}
		alerts = append(alerts, &alert)
	}
	return alerts, nil
}

// ========================================
// SAFETY INCIDENT REPORTS
// ========================================

// CreateSafetyIncidentReport creates a new safety incident report
func (r *Repository) CreateSafetyIncidentReport(ctx context.Context, report *SafetyIncidentReport) error {
	query := `
		INSERT INTO safety_incident_reports (
			id, reporter_id, reported_user_id, ride_id, category, severity,
			description, photos, audio_url, video_url, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.Exec(ctx, query,
		report.ID, report.ReporterID, report.ReportedUserID, report.RideID,
		report.Category, report.Severity, report.Description, report.Photos,
		report.AudioURL, report.VideoURL, report.Status, report.CreatedAt,
	)
	return err
}

// GetSafetyIncidentReport retrieves an incident report by ID
func (r *Repository) GetSafetyIncidentReport(ctx context.Context, id uuid.UUID) (*SafetyIncidentReport, error) {
	query := `
		SELECT id, reporter_id, reported_user_id, ride_id, category, severity,
			description, photos, audio_url, video_url, status, assigned_to,
			resolution, action_taken, created_at, investigated_at, resolved_at
		FROM safety_incident_reports
		WHERE id = $1
	`
	var report SafetyIncidentReport
	err := r.db.QueryRow(ctx, query, id).Scan(
		&report.ID, &report.ReporterID, &report.ReportedUserID, &report.RideID,
		&report.Category, &report.Severity, &report.Description, &report.Photos,
		&report.AudioURL, &report.VideoURL, &report.Status, &report.AssignedTo,
		&report.Resolution, &report.ActionTaken, &report.CreatedAt, &report.InvestigatedAt, &report.ResolvedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &report, err
}

// UpdateSafetyIncidentStatus updates the status of an incident report
func (r *Repository) UpdateSafetyIncidentStatus(ctx context.Context, id uuid.UUID, status string, resolution, actionTaken string) error {
	query := `
		UPDATE safety_incident_reports
		SET status = $2,
			resolution = COALESCE(NULLIF($3, ''), resolution),
			action_taken = COALESCE(NULLIF($4, ''), action_taken),
			investigated_at = CASE WHEN $2 = 'investigating' THEN NOW() ELSE investigated_at END,
			resolved_at = CASE WHEN $2 IN ('resolved', 'dismissed') THEN NOW() ELSE resolved_at END
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, status, resolution, actionTaken)
	return err
}

// ========================================
// TRUSTED/BLOCKED DRIVERS
// ========================================

// AddTrustedDriver adds a driver to a rider's trusted list
func (r *Repository) AddTrustedDriver(ctx context.Context, riderID, driverID uuid.UUID, note string) error {
	query := `
		INSERT INTO trusted_drivers (id, rider_id, driver_id, note, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (rider_id, driver_id) DO UPDATE SET note = $4
	`
	_, err := r.db.Exec(ctx, query, uuid.New(), riderID, driverID, note)
	return err
}

// RemoveTrustedDriver removes a driver from a rider's trusted list
func (r *Repository) RemoveTrustedDriver(ctx context.Context, riderID, driverID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM trusted_drivers WHERE rider_id = $1 AND driver_id = $2`, riderID, driverID)
	return err
}

// IsDriverTrusted checks if a driver is trusted by a rider
func (r *Repository) IsDriverTrusted(ctx context.Context, riderID, driverID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM trusted_drivers WHERE rider_id = $1 AND driver_id = $2)`, riderID, driverID).Scan(&exists)
	return exists, err
}

// AddBlockedDriver adds a driver to a rider's blocked list
func (r *Repository) AddBlockedDriver(ctx context.Context, riderID, driverID uuid.UUID, reason string) error {
	query := `
		INSERT INTO blocked_drivers (id, rider_id, driver_id, reason, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (rider_id, driver_id) DO UPDATE SET reason = $4
	`
	_, err := r.db.Exec(ctx, query, uuid.New(), riderID, driverID, reason)
	return err
}

// RemoveBlockedDriver removes a driver from a rider's blocked list
func (r *Repository) RemoveBlockedDriver(ctx context.Context, riderID, driverID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocked_drivers WHERE rider_id = $1 AND driver_id = $2`, riderID, driverID)
	return err
}

// IsDriverBlocked checks if a driver is blocked by a rider
func (r *Repository) IsDriverBlocked(ctx context.Context, riderID, driverID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM blocked_drivers WHERE rider_id = $1 AND driver_id = $2)`, riderID, driverID).Scan(&exists)
	return exists, err
}

// GetBlockedDrivers retrieves all blocked drivers for a rider
func (r *Repository) GetBlockedDrivers(ctx context.Context, riderID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `SELECT driver_id FROM blocked_drivers WHERE rider_id = $1`, riderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var driverIDs []uuid.UUID
	for rows.Next() {
		var driverID uuid.UUID
		if err := rows.Scan(&driverID); err != nil {
			return nil, err
		}
		driverIDs = append(driverIDs, driverID)
	}
	return driverIDs, nil
}

// CountUserEmergencyAlerts counts emergency alerts for a user (for admin)
func (r *Repository) CountUserEmergencyAlerts(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM emergency_alerts WHERE user_id = $1`, userID).Scan(&count)
	return count, err
}

// GetEmergencyAlertStats retrieves statistics about emergency alerts
func (r *Repository) GetEmergencyAlertStats(ctx context.Context, since time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total alerts
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM emergency_alerts WHERE created_at >= $1`, since).Scan(&total); err != nil {
		return nil, err
	}
	stats["total"] = total

	// By type
	rows, err := r.db.Query(ctx, `SELECT type, COUNT(*) FROM emergency_alerts WHERE created_at >= $1 GROUP BY type`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byType := make(map[string]int)
	for rows.Next() {
		var t string
		var count int
		if err := rows.Scan(&t, &count); err != nil {
			return nil, err
		}
		byType[t] = count
	}
	stats["by_type"] = byType

	// By status
	rows2, err := r.db.Query(ctx, `SELECT status, COUNT(*) FROM emergency_alerts WHERE created_at >= $1 GROUP BY status`, since)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	byStatus := make(map[string]int)
	for rows2.Next() {
		var s string
		var count int
		if err := rows2.Scan(&s, &count); err != nil {
			return nil, err
		}
		byStatus[s] = count
	}
	stats["by_status"] = byStatus

	// Average response time
	var avgResponseMinutes *float64
	if err := r.db.QueryRow(ctx, `
		SELECT AVG(EXTRACT(EPOCH FROM (responded_at - created_at)) / 60)
		FROM emergency_alerts
		WHERE responded_at IS NOT NULL AND created_at >= $1
	`, since).Scan(&avgResponseMinutes); err != nil {
		return nil, fmt.Errorf("failed to get avg response time: %w", err)
	}
	stats["avg_response_minutes"] = avgResponseMinutes

	return stats, nil
}
