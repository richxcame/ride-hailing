package documents

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/storage"
	"go.uber.org/zap"
)

// OCRProvider represents the OCR service provider
type OCRProvider string

const (
	OCRProviderGoogleVision OCRProvider = "google_vision"
	OCRProviderAWSTextract  OCRProvider = "aws_textract"
	OCRProviderMock         OCRProvider = "mock" // For testing
)

// OCRWorkerConfig holds worker configuration
type OCRWorkerConfig struct {
	Provider         OCRProvider
	BatchSize        int
	PollInterval     time.Duration
	MaxRetries       int
	GoogleProjectID  string
	GoogleLocation   string
	AWSRegion        string
	MinConfidence    float64
	ProcessorTimeout time.Duration
}

// OCRWorker processes documents from the OCR queue
type OCRWorker struct {
	repo      *Repository
	storage   storage.Storage
	config    OCRWorkerConfig
	processor OCRProcessor
	stopCh    chan struct{}
}

// OCRProcessor interface for different OCR implementations
type OCRProcessor interface {
	ProcessDocument(ctx context.Context, imageData []byte, mimeType string) (*OCRResult, error)
	Name() string
}

// NewOCRWorker creates a new OCR worker
func NewOCRWorker(repo *Repository, storage storage.Storage, config OCRWorkerConfig) *OCRWorker {
	if config.BatchSize == 0 {
		config.BatchSize = 10
	}
	if config.PollInterval == 0 {
		config.PollInterval = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.MinConfidence == 0 {
		config.MinConfidence = 0.7
	}
	if config.ProcessorTimeout == 0 {
		config.ProcessorTimeout = 60 * time.Second
	}

	worker := &OCRWorker{
		repo:    repo,
		storage: storage,
		config:  config,
		stopCh:  make(chan struct{}),
	}

	// Initialize processor based on provider
	switch config.Provider {
	case OCRProviderGoogleVision:
		worker.processor = NewGoogleVisionProcessor(config.GoogleProjectID, config.GoogleLocation)
	case OCRProviderAWSTextract:
		worker.processor = NewAWSTextractProcessor(config.AWSRegion)
	case OCRProviderMock:
		worker.processor = NewMockOCRProcessor()
	default:
		worker.processor = NewMockOCRProcessor()
	}

	return worker
}

// Start begins processing OCR jobs
func (w *OCRWorker) Start(ctx context.Context) {
	logger.Info("OCR Worker started", zap.String("provider", w.processor.Name()))

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	// Process immediately on start
	w.processBatch(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Info("OCR Worker stopping due to context cancellation")
			return
		case <-w.stopCh:
			logger.Info("OCR Worker stopped")
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

// Stop stops the worker
func (w *OCRWorker) Stop() {
	close(w.stopCh)
}

// processBatch processes a batch of OCR jobs
func (w *OCRWorker) processBatch(ctx context.Context) {
	jobs, err := w.repo.GetPendingOCRJobs(ctx, w.config.BatchSize)
	if err != nil {
		logger.Error("Failed to get pending OCR jobs", zap.Error(err))
		return
	}

	if len(jobs) == 0 {
		return
	}

	logger.Info("Processing OCR batch", zap.Int("count", len(jobs)))

	for _, job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
			w.processJob(ctx, job)
		}
	}
}

// processJob processes a single OCR job
func (w *OCRWorker) processJob(ctx context.Context, job *OCRProcessingQueue) {
	jobCtx, cancel := context.WithTimeout(ctx, w.config.ProcessorTimeout)
	defer cancel()

	// Mark as processing
	if err := w.repo.UpdateOCRJobStatus(ctx, job.ID, "processing", nil, nil); err != nil {
		logger.Error("Failed to mark OCR job as processing", zap.Error(err))
		return
	}

	// Get document
	doc, err := w.repo.GetDocument(jobCtx, job.DocumentID)
	if err != nil {
		w.failJob(ctx, job, fmt.Sprintf("document not found: %v", err))
		return
	}

	// Download document from storage
	imageData, err := w.downloadDocument(jobCtx, doc.FileKey)
	if err != nil {
		w.failJob(ctx, job, fmt.Sprintf("failed to download document: %v", err))
		return
	}

	// Process with OCR
	mimeType := "image/jpeg"
	if doc.FileMimeType != nil {
		mimeType = *doc.FileMimeType
	}

	result, err := w.processor.ProcessDocument(jobCtx, imageData, mimeType)
	if err != nil {
		w.handleProcessingError(ctx, job, err)
		return
	}

	// Validate confidence
	if result.Confidence < w.config.MinConfidence {
		logger.Warn("OCR result below confidence threshold",
			zap.String("document_id", doc.ID.String()),
			zap.Float64("confidence", result.Confidence),
			zap.Float64("threshold", w.config.MinConfidence),
		)
	}

	// Save result
	ocrData := w.buildOCRData(result)
	if err := w.repo.UpdateDocumentOCRData(ctx, doc.ID, ocrData, result.Confidence); err != nil {
		w.failJob(ctx, job, fmt.Sprintf("failed to save OCR data: %v", err))
		return
	}

	// Update document details from OCR
	w.updateDocumentFromOCR(ctx, doc.ID, result)

	// Mark job as completed
	resultJSON, _ := json.Marshal(result)
	resultStr := string(resultJSON)
	if err := w.repo.UpdateOCRJobStatus(ctx, job.ID, "completed", &resultStr, nil); err != nil {
		logger.Error("Failed to mark OCR job as completed", zap.Error(err))
	}

	// Log history
	w.logOCRHistory(ctx, doc.ID, result)

	logger.Info("OCR job completed",
		zap.String("document_id", doc.ID.String()),
		zap.Float64("confidence", result.Confidence),
	)
}

func (w *OCRWorker) downloadDocument(ctx context.Context, fileKey string) ([]byte, error) {
	reader, err := w.storage.Download(ctx, fileKey)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func (w *OCRWorker) buildOCRData(result *OCRResult) map[string]interface{} {
	data := map[string]interface{}{
		"raw_text":   result.RawText,
		"confidence": result.Confidence,
		"metadata":   result.Metadata,
	}

	if result.DocumentNumber != "" {
		data["document_number"] = result.DocumentNumber
	}
	if result.FullName != "" {
		data["full_name"] = result.FullName
	}
	if result.DateOfBirth != nil {
		data["date_of_birth"] = result.DateOfBirth.Format("2006-01-02")
	}
	if result.IssueDate != nil {
		data["issue_date"] = result.IssueDate.Format("2006-01-02")
	}
	if result.ExpiryDate != nil {
		data["expiry_date"] = result.ExpiryDate.Format("2006-01-02")
	}
	if result.IssuingAuthority != "" {
		data["issuing_authority"] = result.IssuingAuthority
	}
	if result.Address != "" {
		data["address"] = result.Address
	}
	if result.VehiclePlate != "" {
		data["vehicle_plate"] = result.VehiclePlate
	}
	if result.VehicleVIN != "" {
		data["vehicle_vin"] = result.VehicleVIN
	}

	return data
}

func (w *OCRWorker) updateDocumentFromOCR(ctx context.Context, documentID uuid.UUID, result *OCRResult) {
	var docNum, authority *string
	if result.DocumentNumber != "" {
		docNum = &result.DocumentNumber
	}
	if result.IssuingAuthority != "" {
		authority = &result.IssuingAuthority
	}

	if err := w.repo.UpdateDocumentDetails(ctx, documentID, docNum, result.IssueDate, result.ExpiryDate, authority); err != nil {
		logger.Warn("Failed to update document details from OCR", zap.Error(err))
	}
}

func (w *OCRWorker) logOCRHistory(ctx context.Context, documentID uuid.UUID, result *OCRResult) {
	notes := fmt.Sprintf("OCR processed with %.0f%% confidence", result.Confidence*100)
	history := &DocumentVerificationHistory{
		ID:             uuid.New(),
		DocumentID:     documentID,
		Action:         "ocr_processed",
		IsSystemAction: true,
		Notes:          &notes,
	}

	if err := w.repo.CreateHistory(ctx, history); err != nil {
		logger.Warn("Failed to create OCR history entry", zap.Error(err))
	}
}

func (w *OCRWorker) failJob(ctx context.Context, job *OCRProcessingQueue, errMsg string) {
	if err := w.repo.UpdateOCRJobStatus(ctx, job.ID, "failed", nil, &errMsg); err != nil {
		logger.Error("Failed to mark OCR job as failed", zap.Error(err))
	}
	logger.Error("OCR job failed",
		zap.String("job_id", job.ID.String()),
		zap.String("document_id", job.DocumentID.String()),
		zap.String("error", errMsg),
	)
}

func (w *OCRWorker) handleProcessingError(ctx context.Context, job *OCRProcessingQueue, err error) {
	job.RetryCount++
	if job.RetryCount >= w.config.MaxRetries {
		w.failJob(ctx, job, fmt.Sprintf("max retries exceeded: %v", err))
		return
	}

	// Schedule retry with exponential backoff
	nextRetry := time.Now().Add(time.Duration(job.RetryCount*job.RetryCount) * time.Minute)
	if updateErr := w.repo.UpdateOCRJobRetry(ctx, job.ID, job.RetryCount, nextRetry); updateErr != nil {
		logger.Error("Failed to update OCR job for retry", zap.Error(updateErr))
	}

	logger.Warn("OCR job will be retried",
		zap.String("job_id", job.ID.String()),
		zap.Int("retry_count", job.RetryCount),
		zap.Time("next_retry", nextRetry),
		zap.Error(err),
	)
}

// ========================================
// MOCK OCR PROCESSOR (for development/testing)
// ========================================

// MockOCRProcessor simulates OCR processing
type MockOCRProcessor struct{}

func NewMockOCRProcessor() *MockOCRProcessor {
	return &MockOCRProcessor{}
}

func (p *MockOCRProcessor) Name() string {
	return "mock"
}

func (p *MockOCRProcessor) ProcessDocument(ctx context.Context, imageData []byte, mimeType string) (*OCRResult, error) {
	// Simulate processing time
	time.Sleep(500 * time.Millisecond)

	// Return mock data
	issueDate := time.Now().AddDate(-1, 0, 0)
	expiryDate := time.Now().AddDate(2, 0, 0)

	return &OCRResult{
		DocumentNumber:   fmt.Sprintf("DL%d", time.Now().Unix()%1000000),
		FullName:         "Mock Driver Name",
		IssueDate:        &issueDate,
		ExpiryDate:       &expiryDate,
		IssuingAuthority: "Mock Authority",
		Confidence:       0.95,
		RawText:          "Mock OCR extracted text",
		Metadata: map[string]interface{}{
			"processor":    "mock",
			"processed_at": time.Now().Format(time.RFC3339),
		},
	}, nil
}

// ========================================
// GOOGLE VISION PROCESSOR
// ========================================

// GoogleVisionProcessor uses Google Cloud Vision API
type GoogleVisionProcessor struct {
	projectID string
	location  string
}

func NewGoogleVisionProcessor(projectID, location string) *GoogleVisionProcessor {
	return &GoogleVisionProcessor{
		projectID: projectID,
		location:  location,
	}
}

func (p *GoogleVisionProcessor) Name() string {
	return "google_vision"
}

func (p *GoogleVisionProcessor) ProcessDocument(ctx context.Context, imageData []byte, mimeType string) (*OCRResult, error) {
	// Note: This is a placeholder implementation
	// In production, you would use the actual Google Cloud Vision API
	// Example: cloud.google.com/go/vision/apiv1

	// For now, return an error indicating the API needs to be configured
	if p.projectID == "" {
		return nil, fmt.Errorf("Google Vision API not configured: missing project ID")
	}

	// Simulate the API call structure
	// In production:
	// client, err := vision.NewImageAnnotatorClient(ctx)
	// image := &vision.Image{Content: imageData}
	// resp, err := client.DocumentTextDetection(ctx, image, nil)

	// Parse the response and extract relevant fields
	result := &OCRResult{
		Confidence: 0.85,
		RawText:    string(imageData[:min(100, len(imageData))]),
		Metadata: map[string]interface{}{
			"processor":  "google_vision",
			"project_id": p.projectID,
		},
	}

	// Extract document fields from raw text
	result.DocumentNumber = p.extractDocumentNumber(result.RawText)
	result.ExpiryDate = p.extractExpiryDate(result.RawText)

	return result, nil
}

func (p *GoogleVisionProcessor) extractDocumentNumber(text string) string {
	// Common patterns for document numbers
	patterns := []string{
		`(?i)(?:license|licence|dl|no|number)[:\s]*([A-Z0-9-]+)`,
		`(?i)([A-Z]{2}[0-9]{6,})`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(text); len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}
	return ""
}

func (p *GoogleVisionProcessor) extractExpiryDate(text string) *time.Time {
	// Common date patterns
	patterns := []string{
		`(?i)exp(?:iry|ires)?[:\s]*(\d{2}[/-]\d{2}[/-]\d{4})`,
		`(?i)valid\s*(?:until|till)[:\s]*(\d{2}[/-]\d{2}[/-]\d{4})`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(text); len(matches) > 1 {
			dateStr := matches[1]
			for _, layout := range []string{"02/01/2006", "02-01-2006", "01/02/2006", "01-02-2006"} {
				if t, err := time.Parse(layout, dateStr); err == nil {
					return &t
				}
			}
		}
	}
	return nil
}

// ========================================
// AWS TEXTRACT PROCESSOR
// ========================================

// AWSTextractProcessor uses AWS Textract
type AWSTextractProcessor struct {
	region string
}

func NewAWSTextractProcessor(region string) *AWSTextractProcessor {
	return &AWSTextractProcessor{region: region}
}

func (p *AWSTextractProcessor) Name() string {
	return "aws_textract"
}

func (p *AWSTextractProcessor) ProcessDocument(ctx context.Context, imageData []byte, mimeType string) (*OCRResult, error) {
	// Note: This is a placeholder implementation
	// In production, you would use the actual AWS Textract API
	// Example: github.com/aws/aws-sdk-go-v2/service/textract

	if p.region == "" {
		return nil, fmt.Errorf("AWS Textract not configured: missing region")
	}

	// Simulate the API call structure
	// In production:
	// cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(p.region))
	// client := textract.NewFromConfig(cfg)
	// output, err := client.AnalyzeDocument(ctx, &textract.AnalyzeDocumentInput{
	//     Document: &types.Document{Bytes: imageData},
	//     FeatureTypes: []types.FeatureType{types.FeatureTypeQueries},
	// })

	result := &OCRResult{
		Confidence: 0.88,
		RawText:    base64.StdEncoding.EncodeToString(imageData[:min(50, len(imageData))]),
		Metadata: map[string]interface{}{
			"processor": "aws_textract",
			"region":    p.region,
		},
	}

	return result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
