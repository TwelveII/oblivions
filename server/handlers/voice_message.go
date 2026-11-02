// Package handlers — VoiceMessageHandler: ephemeral voice channel chat.
// Endpoints are membership-gated by VoiceMessageService (must be in the
// target voice channel). Attachments use the existing UploadPipeline with
// KindVoiceMsg so the channel-scoped directory can be wiped on cleanup.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/akinalp/mqvi/models"
	"github.com/akinalp/mqvi/pkg"
	"github.com/akinalp/mqvi/pkg/ctxkeys"
	"github.com/akinalp/mqvi/pkg/files"
	"github.com/akinalp/mqvi/pkg/ratelimit"
	"github.com/akinalp/mqvi/services"
)

const maxVoiceMessageUploadFiles = 5

type VoiceMessageHandler struct {
	voiceMessageService services.VoiceMessageService
	pipeline            services.UploadPipeline
	storage             services.StorageService
	urlSigner           services.FileURLSigner
	messageLimiter      *ratelimit.MessageRateLimiter
	maxUploadSize       int64
}

func NewVoiceMessageHandler(
	voiceMessageService services.VoiceMessageService,
	pipeline services.UploadPipeline,
	storage services.StorageService,
	urlSigner services.FileURLSigner,
	messageLimiter *ratelimit.MessageRateLimiter,
	maxUploadSize int64,
) *VoiceMessageHandler {
	return &VoiceMessageHandler{
		voiceMessageService: voiceMessageService,
		pipeline:            pipeline,
		storage:             storage,
		urlSigner:           urlSigner,
		messageLimiter:      messageLimiter,
		maxUploadSize:       maxUploadSize,
	}
}

// List handles GET /api/voice-channels/{channelId}/messages
func (h *VoiceMessageHandler) List(w http.ResponseWriter, r *http.Request) {
	channelID := r.PathValue("channelId")
	user, ok := r.Context().Value(ctxkeys.User).(*models.User)
	if !ok {
		pkg.ErrorWithMessage(w, http.StatusUnauthorized, "user not found in context")
		return
	}

	msgs, err := h.voiceMessageService.List(r.Context(), user.ID, channelID, 200)
	if err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, http.StatusOK, msgs)
}

// Create handles POST /api/voice-channels/{channelId}/messages
// Accepts JSON or multipart/form-data (for file attachments, field name "files").
func (h *VoiceMessageHandler) Create(w http.ResponseWriter, r *http.Request) {
	channelID := r.PathValue("channelId")
	user, ok := r.Context().Value(ctxkeys.User).(*models.User)
	if !ok {
		pkg.ErrorWithMessage(w, http.StatusUnauthorized, "user not found in context")
		return
	}

	if h.messageLimiter != nil && !h.messageLimiter.Allow(user.ID) {
		retryAfter := h.messageLimiter.CooldownSeconds(user.ID)
		w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
		pkg.ErrorWithMessage(w, http.StatusTooManyRequests,
			fmt.Sprintf("too many messages, please wait %s",
				ratelimit.FormatRetryMessage(retryAfter)))
		return
	}

	contentType := r.Header.Get("Content-Type")
	var req models.CreateVoiceMessageRequest

	if isMultipart(contentType) {
		limitMultipartBody(w, r, h.maxUploadSize, maxVoiceMessageUploadFiles)
		if err := r.ParseMultipartForm(h.maxUploadSize); err != nil {
			pkg.ErrorWithMessage(w, http.StatusBadRequest, "failed to parse multipart form")
			return
		}
		if r.MultipartForm != nil && len(r.MultipartForm.File["files"]) > maxVoiceMessageUploadFiles {
			pkg.ErrorWithMessage(w, http.StatusBadRequest, "too many files")
			return
		}
		req.Content = r.FormValue("content")
		if r.FormValue("encryption_version") == "1" {
			req.EncryptionVersion = 1
			if value := r.FormValue("ciphertext"); value != "" { req.Ciphertext = &value }
			if value := r.FormValue("sender_device_id"); value != "" { req.SenderDeviceID = &value }
			if value := r.FormValue("e2ee_metadata"); value != "" { req.E2EEMetadata = &value }
		}
		if r.MultipartForm != nil && len(r.MultipartForm.File["files"]) > 0 {
			req.HasFiles = true
		}
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			pkg.ErrorWithMessage(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	var reservedBytes int64
	var fileHeaders []*multipart.FileHeader
	if isMultipart(contentType) && r.MultipartForm != nil {
		fileHeaders = r.MultipartForm.File["files"]
		for _, fh := range fileHeaders {
			if fh.Size < 0 || (h.maxUploadSize > 0 && fh.Size > h.maxUploadSize) {
				pkg.ErrorWithMessage(w, http.StatusBadRequest, "file too large")
				return
			}
			reservedBytes += fh.Size
		}
	}

	if reservedBytes > 0 {
		if h.storage == nil {
			pkg.ErrorWithMessage(w, http.StatusInternalServerError, "storage service unavailable")
			return
		}
		if err := h.storage.Reserve(r.Context(), user.ID, reservedBytes); err != nil {
			pkg.Error(w, err)
			return
		}
	}

	message, err := h.voiceMessageService.Create(r.Context(), user.ID, channelID, &req)
	if err != nil {
		if reservedBytes > 0 {
			if e := h.storage.Release(context.WithoutCancel(r.Context()), user.ID, reservedBytes); e != nil {
				log.Printf("[voice-message] quota refund failed: %v", e)
			}
		}
		pkg.Error(w, err)
		return
	}

	var attachedBytes int64
	rollback := func() {
		cleanupCtx := context.WithoutCancel(r.Context())
		if err := h.voiceMessageService.Delete(cleanupCtx, user.ID, message.ID); err != nil {
			log.Printf("[voice-message] rollback message deletion failed: %v", err)
		}
		unused := reservedBytes - attachedBytes
		if unused > 0 {
			if err := h.storage.Release(cleanupCtx, user.ID, unused); err != nil {
				log.Printf("[voice-message] quota refund failed: %v", err)
			}
		}
	}

	for _, fh := range fileHeaders {
		f, err := fh.Open()
		if err != nil {
			rollback()
			pkg.ErrorWithMessage(w, http.StatusBadRequest, "failed to open uploaded file")
			return
		}
		stored, storeErr := h.pipeline.Store(r.Context(), files.KindVoiceMsg, channelID, f, fh, h.maxUploadSize)
		_ = f.Close()
		if storeErr != nil {
			rollback()
			pkg.Error(w, storeErr)
			return
		}

		mime := fh.Header.Get("Content-Type")
		var mimePtr *string
		if mime != "" {
			mimePtr = &mime
		}
		att, attErr := h.voiceMessageService.AttachFile(r.Context(), message.ID, fh.Filename, stored.RelativeURL, stored.Size, mimePtr)
		if attErr != nil {
			h.pipeline.DeleteFromURL(stored.RelativeURL)
			rollback()
			pkg.Error(w, attErr)
			return
		}
		attachedBytes += stored.Size
		att.FileURL = h.urlSigner.SignURL(att.FileURL)
		message.Attachments = append(message.Attachments, *att)
	}

	unused := reservedBytes - attachedBytes
	if unused > 0 {
		if err := h.storage.Release(context.WithoutCancel(r.Context()), user.ID, unused); err != nil {
			log.Printf("[voice-message] unused quota refund failed: %v", err)
		}
	}

	h.voiceMessageService.BroadcastCreate(r.Context(), message)
	pkg.JSON(w, http.StatusCreated, message)
}

// Update handles PATCH /api/voice-channels/{channelId}/messages/{messageId}
func (h *VoiceMessageHandler) Update(w http.ResponseWriter, r *http.Request) {
	messageID := r.PathValue("messageId")
	user, ok := r.Context().Value(ctxkeys.User).(*models.User)
	if !ok {
		pkg.ErrorWithMessage(w, http.StatusUnauthorized, "user not found in context")
		return
	}

	var req models.UpdateVoiceMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pkg.ErrorWithMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	msg, err := h.voiceMessageService.Update(r.Context(), user.ID, messageID, &req)
	if err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, http.StatusOK, msg)
}

// Delete handles DELETE /api/voice-channels/{channelId}/messages/{messageId}
func (h *VoiceMessageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	messageID := r.PathValue("messageId")
	user, ok := r.Context().Value(ctxkeys.User).(*models.User)
	if !ok {
		pkg.ErrorWithMessage(w, http.StatusUnauthorized, "user not found in context")
		return
	}
	if err := h.voiceMessageService.Delete(r.Context(), user.ID, messageID); err != nil {
		pkg.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
