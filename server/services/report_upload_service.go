// Package services — ReportUploadService: evidence file upload for reports.
// Only image files accepted.
package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/akinalp/mqvi/models"
	"github.com/akinalp/mqvi/pkg"
	"github.com/akinalp/mqvi/pkg/files"
	"github.com/akinalp/mqvi/repository"
)

// ReportUploadService handles evidence file uploads for reports.
type ReportUploadService interface {
	Upload(ctx context.Context, reportID string, file multipart.File, header *multipart.FileHeader) (*models.ReportAttachment, error)
}

type reportUploadService struct {
	reportRepo repository.ReportRepository
	pipeline   UploadPipeline
	maxSize    int64
}

func NewReportUploadService(
	reportRepo repository.ReportRepository,
	pipeline UploadPipeline,
	maxSize int64,
) ReportUploadService {
	return &reportUploadService{
		reportRepo: reportRepo,
		pipeline:   pipeline,
		maxSize:    maxSize,
	}
}

var allowedReportMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

func (s *reportUploadService) Upload(ctx context.Context, reportID string, file multipart.File, header *multipart.FileHeader) (*models.ReportAttachment, error) {
	if header.Size > s.maxSize {
		return nil, fmt.Errorf("%w: file too large (max %dMB)", pkg.ErrBadRequest, s.maxSize/(1024*1024))
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	mimeBase := strings.Split(contentType, ";")[0]
	mimeBase = strings.TrimSpace(mimeBase)

	if !allowedReportMimeTypes[mimeBase] {
		return nil, fmt.Errorf("%w: only images are allowed for report evidence (got: %s)", pkg.ErrBadRequest, mimeBase)
	}

	stored, err := s.pipeline.Store(ctx, files.KindReport, reportID, file, header, s.maxSize)
	if err != nil {
		return nil, err
	}

	fileSize := stored.Size
	att := &models.ReportAttachment{
		ReportID: reportID,
		Filename: header.Filename,
		FileURL:  stored.RelativeURL,
		FileSize: &fileSize,
		MimeType: &mimeBase,
	}

	if err := s.reportRepo.CreateAttachment(ctx, att); err != nil {
		s.pipeline.DeleteFromURL(stored.RelativeURL)
		return nil, fmt.Errorf("failed to create report attachment record: %w", err)
	}

	return att, nil
}
