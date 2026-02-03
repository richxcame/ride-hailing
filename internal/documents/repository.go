package documents

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for documents
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new documents repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// DOCUMENT TYPES
// ========================================

// GetDocumentTypes gets all active document types
func (r *Repository) GetDocumentTypes(ctx context.Context) ([]*DocumentType, error) {
	query := `
		SELECT id, code, name, description, is_required, requires_expiry, requires_front_back,
			   default_validity_months, renewal_reminder_days, requires_manual_review,
			   auto_ocr_enabled, country_codes, display_order, is_active, created_at, updated_at
		FROM document_types
		WHERE is_active = true
		ORDER BY display_order, name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get document types: %w", err)
	}
	defer rows.Close()

	var types []*DocumentType
	for rows.Next() {
		dt := &DocumentType{}
		if err := rows.Scan(
			&dt.ID, &dt.Code, &dt.Name, &dt.Description, &dt.IsRequired, &dt.RequiresExpiry,
			&dt.RequiresFrontBack, &dt.DefaultValidityMonths, &dt.RenewalReminderDays,
			&dt.RequiresManualReview, &dt.AutoOCREnabled, &dt.CountryCodes, &dt.DisplayOrder,
			&dt.IsActive, &dt.CreatedAt, &dt.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan document type: %w", err)
		}
		types = append(types, dt)
	}

	return types, nil
}

// GetDocumentTypeByCode gets a document type by its code
func (r *Repository) GetDocumentTypeByCode(ctx context.Context, code string) (*DocumentType, error) {
	query := `
		SELECT id, code, name, description, is_required, requires_expiry, requires_front_back,
			   default_validity_months, renewal_reminder_days, requires_manual_review,
			   auto_ocr_enabled, country_codes, display_order, is_active, created_at, updated_at
		FROM document_types
		WHERE code = $1 AND is_active = true
	`

	dt := &DocumentType{}
	err := r.db.QueryRow(ctx, query, code).Scan(
		&dt.ID, &dt.Code, &dt.Name, &dt.Description, &dt.IsRequired, &dt.RequiresExpiry,
		&dt.RequiresFrontBack, &dt.DefaultValidityMonths, &dt.RenewalReminderDays,
		&dt.RequiresManualReview, &dt.AutoOCREnabled, &dt.CountryCodes, &dt.DisplayOrder,
		&dt.IsActive, &dt.CreatedAt, &dt.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get document type: %w", err)
	}

	return dt, nil
}

// GetRequiredDocumentTypes gets all required document types
func (r *Repository) GetRequiredDocumentTypes(ctx context.Context) ([]*DocumentType, error) {
	query := `
		SELECT id, code, name, description, is_required, requires_expiry, requires_front_back,
			   default_validity_months, renewal_reminder_days, requires_manual_review,
			   auto_ocr_enabled, country_codes, display_order, is_active, created_at, updated_at
		FROM document_types
		WHERE is_required = true AND is_active = true
		ORDER BY display_order, name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get required document types: %w", err)
	}
	defer rows.Close()

	var types []*DocumentType
	for rows.Next() {
		dt := &DocumentType{}
		if err := rows.Scan(
			&dt.ID, &dt.Code, &dt.Name, &dt.Description, &dt.IsRequired, &dt.RequiresExpiry,
			&dt.RequiresFrontBack, &dt.DefaultValidityMonths, &dt.RenewalReminderDays,
			&dt.RequiresManualReview, &dt.AutoOCREnabled, &dt.CountryCodes, &dt.DisplayOrder,
			&dt.IsActive, &dt.CreatedAt, &dt.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan document type: %w", err)
		}
		types = append(types, dt)
	}

	return types, nil
}

// ========================================
// DRIVER DOCUMENTS
// ========================================

// CreateDocument creates a new driver document
func (r *Repository) CreateDocument(ctx context.Context, doc *DriverDocument) error {
	ocrDataJSON, _ := json.Marshal(doc.OCRData)

	query := `
		INSERT INTO driver_documents (
			id, driver_id, document_type_id, status, file_url, file_key, file_name,
			file_size_bytes, file_mime_type, back_file_url, back_file_key,
			document_number, issue_date, expiry_date, issuing_authority,
			ocr_data, version, previous_document_id, submitted_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		doc.ID, doc.DriverID, doc.DocumentTypeID, doc.Status, doc.FileURL, doc.FileKey,
		doc.FileName, doc.FileSizeBytes, doc.FileMimeType, doc.BackFileURL, doc.BackFileKey,
		doc.DocumentNumber, doc.IssueDate, doc.ExpiryDate, doc.IssuingAuthority,
		ocrDataJSON, doc.Version, doc.PreviousDocumentID, doc.SubmittedAt,
	).Scan(&doc.CreatedAt, &doc.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create document: %w", err)
	}

	return nil
}

// GetDocument gets a document by ID
func (r *Repository) GetDocument(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
	query := `
		SELECT dd.id, dd.driver_id, dd.document_type_id, dd.status, dd.file_url, dd.file_key,
			   dd.file_name, dd.file_size_bytes, dd.file_mime_type, dd.back_file_url, dd.back_file_key,
			   dd.document_number, dd.issue_date, dd.expiry_date, dd.issuing_authority,
			   dd.ocr_data, dd.ocr_confidence, dd.ocr_processed_at, dd.reviewed_by, dd.reviewed_at,
			   dd.review_notes, dd.rejection_reason, dd.version, dd.previous_document_id,
			   dd.submitted_at, dd.created_at, dd.updated_at,
			   dt.id, dt.code, dt.name, dt.requires_expiry, dt.requires_front_back
		FROM driver_documents dd
		JOIN document_types dt ON dd.document_type_id = dt.id
		WHERE dd.id = $1
	`

	doc := &DriverDocument{}
	dt := &DocumentType{}
	var ocrDataJSON []byte

	err := r.db.QueryRow(ctx, query, documentID).Scan(
		&doc.ID, &doc.DriverID, &doc.DocumentTypeID, &doc.Status, &doc.FileURL, &doc.FileKey,
		&doc.FileName, &doc.FileSizeBytes, &doc.FileMimeType, &doc.BackFileURL, &doc.BackFileKey,
		&doc.DocumentNumber, &doc.IssueDate, &doc.ExpiryDate, &doc.IssuingAuthority,
		&ocrDataJSON, &doc.OCRConfidence, &doc.OCRProcessedAt, &doc.ReviewedBy, &doc.ReviewedAt,
		&doc.ReviewNotes, &doc.RejectionReason, &doc.Version, &doc.PreviousDocumentID,
		&doc.SubmittedAt, &doc.CreatedAt, &doc.UpdatedAt,
		&dt.ID, &dt.Code, &dt.Name, &dt.RequiresExpiry, &dt.RequiresFrontBack,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if len(ocrDataJSON) > 0 {
		json.Unmarshal(ocrDataJSON, &doc.OCRData)
	}
	doc.DocumentType = dt

	return doc, nil
}

// GetDriverDocuments gets all documents for a driver
func (r *Repository) GetDriverDocuments(ctx context.Context, driverID uuid.UUID) ([]*DriverDocument, error) {
	query := `
		SELECT dd.id, dd.driver_id, dd.document_type_id, dd.status, dd.file_url, dd.file_key,
			   dd.file_name, dd.file_size_bytes, dd.file_mime_type, dd.back_file_url, dd.back_file_key,
			   dd.document_number, dd.issue_date, dd.expiry_date, dd.issuing_authority,
			   dd.ocr_data, dd.ocr_confidence, dd.ocr_processed_at, dd.reviewed_by, dd.reviewed_at,
			   dd.review_notes, dd.rejection_reason, dd.version, dd.previous_document_id,
			   dd.submitted_at, dd.created_at, dd.updated_at,
			   dt.id, dt.code, dt.name, dt.requires_expiry, dt.requires_front_back
		FROM driver_documents dd
		JOIN document_types dt ON dd.document_type_id = dt.id
		WHERE dd.driver_id = $1 AND dd.status != 'superseded'
		ORDER BY dt.display_order, dd.submitted_at DESC
	`

	rows, err := r.db.Query(ctx, query, driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver documents: %w", err)
	}
	defer rows.Close()

	var docs []*DriverDocument
	for rows.Next() {
		doc := &DriverDocument{}
		dt := &DocumentType{}
		var ocrDataJSON []byte

		if err := rows.Scan(
			&doc.ID, &doc.DriverID, &doc.DocumentTypeID, &doc.Status, &doc.FileURL, &doc.FileKey,
			&doc.FileName, &doc.FileSizeBytes, &doc.FileMimeType, &doc.BackFileURL, &doc.BackFileKey,
			&doc.DocumentNumber, &doc.IssueDate, &doc.ExpiryDate, &doc.IssuingAuthority,
			&ocrDataJSON, &doc.OCRConfidence, &doc.OCRProcessedAt, &doc.ReviewedBy, &doc.ReviewedAt,
			&doc.ReviewNotes, &doc.RejectionReason, &doc.Version, &doc.PreviousDocumentID,
			&doc.SubmittedAt, &doc.CreatedAt, &doc.UpdatedAt,
			&dt.ID, &dt.Code, &dt.Name, &dt.RequiresExpiry, &dt.RequiresFrontBack,
		); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		if len(ocrDataJSON) > 0 {
			json.Unmarshal(ocrDataJSON, &doc.OCRData)
		}
		doc.DocumentType = dt
		docs = append(docs, doc)
	}

	return docs, nil
}

// GetLatestDocumentByType gets the latest document of a specific type for a driver
func (r *Repository) GetLatestDocumentByType(ctx context.Context, driverID, documentTypeID uuid.UUID) (*DriverDocument, error) {
	query := `
		SELECT id, driver_id, document_type_id, status, file_url, file_key,
			   file_name, file_size_bytes, file_mime_type, back_file_url, back_file_key,
			   document_number, issue_date, expiry_date, issuing_authority,
			   ocr_data, ocr_confidence, ocr_processed_at, reviewed_by, reviewed_at,
			   review_notes, rejection_reason, version, previous_document_id,
			   submitted_at, created_at, updated_at
		FROM driver_documents
		WHERE driver_id = $1 AND document_type_id = $2 AND status != 'superseded'
		ORDER BY submitted_at DESC
		LIMIT 1
	`

	doc := &DriverDocument{}
	var ocrDataJSON []byte

	err := r.db.QueryRow(ctx, query, driverID, documentTypeID).Scan(
		&doc.ID, &doc.DriverID, &doc.DocumentTypeID, &doc.Status, &doc.FileURL, &doc.FileKey,
		&doc.FileName, &doc.FileSizeBytes, &doc.FileMimeType, &doc.BackFileURL, &doc.BackFileKey,
		&doc.DocumentNumber, &doc.IssueDate, &doc.ExpiryDate, &doc.IssuingAuthority,
		&ocrDataJSON, &doc.OCRConfidence, &doc.OCRProcessedAt, &doc.ReviewedBy, &doc.ReviewedAt,
		&doc.ReviewNotes, &doc.RejectionReason, &doc.Version, &doc.PreviousDocumentID,
		&doc.SubmittedAt, &doc.CreatedAt, &doc.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if len(ocrDataJSON) > 0 {
		json.Unmarshal(ocrDataJSON, &doc.OCRData)
	}

	return doc, nil
}

// UpdateDocumentStatus updates a document's status
func (r *Repository) UpdateDocumentStatus(ctx context.Context, documentID uuid.UUID, status DocumentStatus, reviewedBy *uuid.UUID, reviewNotes, rejectionReason *string) error {
	query := `
		UPDATE driver_documents
		SET status = $1, reviewed_by = $2, reviewed_at = $3, review_notes = $4, rejection_reason = $5, updated_at = NOW()
		WHERE id = $6
	`

	var reviewedAt *time.Time
	if reviewedBy != nil {
		now := time.Now()
		reviewedAt = &now
	}

	_, err := r.db.Exec(ctx, query, status, reviewedBy, reviewedAt, reviewNotes, rejectionReason, documentID)
	if err != nil {
		return fmt.Errorf("failed to update document status: %w", err)
	}

	return nil
}

// UpdateDocumentOCRData updates the OCR data for a document
func (r *Repository) UpdateDocumentOCRData(ctx context.Context, documentID uuid.UUID, ocrData map[string]interface{}, confidence float64) error {
	ocrDataJSON, _ := json.Marshal(ocrData)

	query := `
		UPDATE driver_documents
		SET ocr_data = $1, ocr_confidence = $2, ocr_processed_at = NOW(), updated_at = NOW()
		WHERE id = $3
	`

	_, err := r.db.Exec(ctx, query, ocrDataJSON, confidence, documentID)
	if err != nil {
		return fmt.Errorf("failed to update OCR data: %w", err)
	}

	return nil
}

// UpdateDocumentDetails updates document details (from OCR or manual)
func (r *Repository) UpdateDocumentDetails(ctx context.Context, documentID uuid.UUID, documentNumber *string, issueDate, expiryDate *time.Time, issuingAuthority *string) error {
	query := `
		UPDATE driver_documents
		SET document_number = COALESCE($1, document_number),
		    issue_date = COALESCE($2, issue_date),
		    expiry_date = COALESCE($3, expiry_date),
		    issuing_authority = COALESCE($4, issuing_authority),
		    updated_at = NOW()
		WHERE id = $5
	`

	_, err := r.db.Exec(ctx, query, documentNumber, issueDate, expiryDate, issuingAuthority, documentID)
	if err != nil {
		return fmt.Errorf("failed to update document details: %w", err)
	}

	return nil
}

// SupersedeDocument marks an existing document as superseded
func (r *Repository) SupersedeDocument(ctx context.Context, documentID uuid.UUID) error {
	query := `UPDATE driver_documents SET status = 'superseded', updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, documentID)
	return err
}

// UpdateDocumentBackFile updates the back file for a document
func (r *Repository) UpdateDocumentBackFile(ctx context.Context, documentID uuid.UUID, backFileURL, backFileKey string) error {
	query := `
		UPDATE driver_documents
		SET back_file_url = $1, back_file_key = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, backFileURL, backFileKey, documentID)
	return err
}

// ========================================
// VERIFICATION STATUS
// ========================================

// GetDriverVerificationStatus gets the verification status for a driver
func (r *Repository) GetDriverVerificationStatus(ctx context.Context, driverID uuid.UUID) (*DriverVerificationStatus, error) {
	query := `
		SELECT driver_id, verification_status, required_documents_count, submitted_documents_count,
			   approved_documents_count, documents_submitted_at, documents_approved_at,
			   background_check_completed_at, approved_by, approved_at, rejection_reason,
			   suspended_at, suspended_by, suspension_reason, suspension_end_date,
			   next_document_expiry, expiry_warning_sent_at, created_at, updated_at
		FROM driver_verification_status
		WHERE driver_id = $1
	`

	status := &DriverVerificationStatus{}
	err := r.db.QueryRow(ctx, query, driverID).Scan(
		&status.DriverID, &status.VerificationStatus, &status.RequiredDocumentsCount,
		&status.SubmittedDocumentsCount, &status.ApprovedDocumentsCount,
		&status.DocumentsSubmittedAt, &status.DocumentsApprovedAt, &status.BackgroundCheckCompletedAt,
		&status.ApprovedBy, &status.ApprovedAt, &status.RejectionReason,
		&status.SuspendedAt, &status.SuspendedBy, &status.SuspensionReason, &status.SuspensionEndDate,
		&status.NextDocumentExpiry, &status.ExpiryWarningSentAt, &status.CreatedAt, &status.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return status, nil
}

// ========================================
// PENDING REVIEWS (ADMIN)
// ========================================

// GetPendingReviews gets documents pending review
func (r *Repository) GetPendingReviews(ctx context.Context, limit, offset int) ([]*PendingReviewDocument, int, error) {
	countQuery := `
		SELECT COUNT(*) FROM driver_documents WHERE status IN ('pending', 'under_review')
	`
	var total int
	r.db.QueryRow(ctx, countQuery).Scan(&total)

	query := `
		SELECT dd.id, dd.driver_id, dd.document_type_id, dd.status, dd.file_url, dd.file_key,
			   dd.file_name, dd.document_number, dd.expiry_date, dd.ocr_confidence,
			   dd.submitted_at, dd.created_at, dd.updated_at,
			   u.first_name || ' ' || u.last_name AS driver_name,
			   u.phone_number AS driver_phone, u.email AS driver_email,
			   dt.name AS document_type_name,
			   EXTRACT(EPOCH FROM (NOW() - dd.submitted_at)) / 3600 AS hours_pending
		FROM driver_documents dd
		JOIN drivers d ON dd.driver_id = d.id
		JOIN users u ON d.user_id = u.id
		JOIN document_types dt ON dd.document_type_id = dt.id
		WHERE dd.status IN ('pending', 'under_review')
		ORDER BY dd.submitted_at ASC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get pending reviews: %w", err)
	}
	defer rows.Close()

	var reviews []*PendingReviewDocument
	for rows.Next() {
		doc := &DriverDocument{}
		review := &PendingReviewDocument{Document: doc}

		if err := rows.Scan(
			&doc.ID, &doc.DriverID, &doc.DocumentTypeID, &doc.Status, &doc.FileURL, &doc.FileKey,
			&doc.FileName, &doc.DocumentNumber, &doc.ExpiryDate, &doc.OCRConfidence,
			&doc.SubmittedAt, &doc.CreatedAt, &doc.UpdatedAt,
			&review.DriverName, &review.DriverPhone, &review.DriverEmail,
			&review.DocumentType, &review.HoursPending,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan pending review: %w", err)
		}

		review.OCRConfidence = doc.OCRConfidence
		reviews = append(reviews, review)
	}

	return reviews, total, nil
}

// GetExpiringDocuments gets documents expiring soon
func (r *Repository) GetExpiringDocuments(ctx context.Context, daysAhead int) ([]*ExpiringDocument, error) {
	query := `
		SELECT dd.id, dd.driver_id, dd.document_type_id, dd.status, dd.file_url,
			   dd.document_number, dd.expiry_date, dd.created_at, dd.updated_at,
			   u.first_name || ' ' || u.last_name AS driver_name,
			   u.email AS driver_email, u.phone_number AS driver_phone,
			   dt.name AS document_type_name,
			   (dd.expiry_date - CURRENT_DATE) AS days_until_expiry,
			   CASE
				   WHEN dd.expiry_date < CURRENT_DATE THEN 'expired'
				   WHEN dd.expiry_date <= CURRENT_DATE + INTERVAL '7 days' THEN 'critical'
				   WHEN dd.expiry_date <= CURRENT_DATE + INTERVAL '30 days' THEN 'warning'
				   ELSE 'ok'
			   END AS urgency
		FROM driver_documents dd
		JOIN drivers d ON dd.driver_id = d.id
		JOIN users u ON d.user_id = u.id
		JOIN document_types dt ON dd.document_type_id = dt.id
		WHERE dd.status = 'approved'
		  AND dd.expiry_date IS NOT NULL
		  AND dd.expiry_date <= CURRENT_DATE + ($1 || ' days')::INTERVAL
		ORDER BY dd.expiry_date ASC
	`

	rows, err := r.db.Query(ctx, query, daysAhead)
	if err != nil {
		return nil, fmt.Errorf("failed to get expiring documents: %w", err)
	}
	defer rows.Close()

	var expiring []*ExpiringDocument
	for rows.Next() {
		doc := &DriverDocument{}
		exp := &ExpiringDocument{Document: doc}

		if err := rows.Scan(
			&doc.ID, &doc.DriverID, &doc.DocumentTypeID, &doc.Status, &doc.FileURL,
			&doc.DocumentNumber, &doc.ExpiryDate, &doc.CreatedAt, &doc.UpdatedAt,
			&exp.DriverName, &exp.DriverEmail, &exp.DriverPhone,
			&exp.DocumentType, &exp.DaysUntilExpiry, &exp.Urgency,
		); err != nil {
			return nil, fmt.Errorf("failed to scan expiring document: %w", err)
		}

		expiring = append(expiring, exp)
	}

	return expiring, nil
}

// ========================================
// HISTORY
// ========================================

// CreateHistory creates a document verification history entry
func (r *Repository) CreateHistory(ctx context.Context, history *DocumentVerificationHistory) error {
	metadataJSON, _ := json.Marshal(history.Metadata)

	query := `
		INSERT INTO document_verification_history (
			id, document_id, action, previous_status, new_status,
			performed_by, is_system_action, notes, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at
	`

	return r.db.QueryRow(ctx, query,
		history.ID, history.DocumentID, history.Action, history.PreviousStatus,
		history.NewStatus, history.PerformedBy, history.IsSystemAction, history.Notes, metadataJSON,
	).Scan(&history.CreatedAt)
}

// GetDocumentHistory gets the history for a document
func (r *Repository) GetDocumentHistory(ctx context.Context, documentID uuid.UUID) ([]*DocumentVerificationHistory, error) {
	query := `
		SELECT id, document_id, action, previous_status, new_status,
			   performed_by, is_system_action, notes, metadata, created_at
		FROM document_verification_history
		WHERE document_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document history: %w", err)
	}
	defer rows.Close()

	var history []*DocumentVerificationHistory
	for rows.Next() {
		h := &DocumentVerificationHistory{}
		var metadataJSON []byte

		if err := rows.Scan(
			&h.ID, &h.DocumentID, &h.Action, &h.PreviousStatus, &h.NewStatus,
			&h.PerformedBy, &h.IsSystemAction, &h.Notes, &metadataJSON, &h.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &h.Metadata)
		}
		history = append(history, h)
	}

	return history, nil
}

// ========================================
// OCR QUEUE
// ========================================

// CreateOCRJob creates an OCR processing job
func (r *Repository) CreateOCRJob(ctx context.Context, job *OCRProcessingQueue) error {
	query := `
		INSERT INTO ocr_processing_queue (id, document_id, status, priority, max_retries)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`

	return r.db.QueryRow(ctx, query,
		job.ID, job.DocumentID, job.Status, job.Priority, job.MaxRetries,
	).Scan(&job.CreatedAt, &job.UpdatedAt)
}

// GetPendingOCRJobs gets pending OCR jobs
func (r *Repository) GetPendingOCRJobs(ctx context.Context, limit int) ([]*OCRProcessingQueue, error) {
	query := `
		SELECT id, document_id, status, priority, provider, started_at, completed_at,
			   processing_time_ms, raw_response, extracted_data, confidence_score,
			   error_message, retry_count, max_retries, next_retry_at, created_at, updated_at
		FROM ocr_processing_queue
		WHERE status = 'pending'
		   OR (status = 'failed' AND retry_count < max_retries AND (next_retry_at IS NULL OR next_retry_at <= NOW()))
		ORDER BY priority DESC, created_at ASC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending OCR jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*OCRProcessingQueue
	for rows.Next() {
		job := &OCRProcessingQueue{}
		var rawResponseJSON, extractedDataJSON []byte

		if err := rows.Scan(
			&job.ID, &job.DocumentID, &job.Status, &job.Priority, &job.Provider,
			&job.StartedAt, &job.CompletedAt, &job.ProcessingTimeMs,
			&rawResponseJSON, &extractedDataJSON, &job.ConfidenceScore,
			&job.ErrorMessage, &job.RetryCount, &job.MaxRetries, &job.NextRetryAt,
			&job.CreatedAt, &job.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan OCR job: %w", err)
		}

		if len(rawResponseJSON) > 0 {
			json.Unmarshal(rawResponseJSON, &job.RawResponse)
		}
		if len(extractedDataJSON) > 0 {
			json.Unmarshal(extractedDataJSON, &job.ExtractedData)
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// UpdateOCRJobStatus updates an OCR job with full status info
func (r *Repository) UpdateOCRJobStatus(ctx context.Context, jobID uuid.UUID, status string, result, errorMsg *string) error {
	query := `
		UPDATE ocr_processing_queue
		SET status = $1,
		    extracted_data = CASE WHEN $2::text IS NOT NULL THEN $2::jsonb ELSE extracted_data END,
		    error_message = $3,
		    completed_at = CASE WHEN $1 IN ('completed', 'failed') THEN NOW() ELSE completed_at END,
		    started_at = CASE WHEN $1 = 'processing' AND started_at IS NULL THEN NOW() ELSE started_at END,
		    updated_at = NOW()
		WHERE id = $4
	`
	_, err := r.db.Exec(ctx, query, status, result, errorMsg, jobID)
	return err
}

// CompleteOCRJob marks an OCR job as completed
func (r *Repository) CompleteOCRJob(ctx context.Context, jobID uuid.UUID, extractedData map[string]interface{}, confidence float64, processingTimeMs int) error {
	extractedDataJSON, _ := json.Marshal(extractedData)

	query := `
		UPDATE ocr_processing_queue
		SET status = 'completed', completed_at = NOW(), extracted_data = $1,
		    confidence_score = $2, processing_time_ms = $3, updated_at = NOW()
		WHERE id = $4
	`
	_, err := r.db.Exec(ctx, query, extractedDataJSON, confidence, processingTimeMs, jobID)
	return err
}

// FailOCRJob marks an OCR job as failed
func (r *Repository) FailOCRJob(ctx context.Context, jobID uuid.UUID, errorMessage string) error {
	query := `
		UPDATE ocr_processing_queue
		SET status = 'failed', error_message = $1, retry_count = retry_count + 1,
		    next_retry_at = NOW() + INTERVAL '5 minutes' * retry_count, updated_at = NOW()
		WHERE id = $2
	`
	_, err := r.db.Exec(ctx, query, errorMessage, jobID)
	return err
}

// UpdateOCRJobRetry updates an OCR job for retry
func (r *Repository) UpdateOCRJobRetry(ctx context.Context, jobID uuid.UUID, retryCount int, nextRetry time.Time) error {
	query := `
		UPDATE ocr_processing_queue
		SET status = 'pending', retry_count = $1, next_retry_at = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, retryCount, nextRetry, jobID)
	return err
}
