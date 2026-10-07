package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"rtspviewer/internal/models"
	"rtspviewer/internal/services"
	"rtspviewer/pkg/response"
)

type StreamHandler struct{ streams *services.StreamManager }

func NewStreamHandler(streams *services.StreamManager) *StreamHandler {
	return &StreamHandler{streams: streams}
}
func (h *StreamHandler) List(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, h.streams.List())
}
func (h *StreamHandler) Get(w http.ResponseWriter, r *http.Request) {
	stream, err := h.streams.Get(r.PathValue("id"))
	if err != nil {
		streamError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, stream)
}
func (h *StreamHandler) Create(w http.ResponseWriter, r *http.Request) {
	contentType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if contentType != "application/json" {
		response.Error(w, http.StatusUnsupportedMediaType, "send application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var input struct {
		Name    string `json:"name"`
		RTSPURL string `json:"rtspUrl"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "request must contain name and rtspUrl as JSON strings")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		response.Error(w, http.StatusBadRequest, "send exactly one JSON object")
		return
	}
	stream, err := h.streams.Add(input.Name, input.RTSPURL)
	if err != nil {
		streamError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, stream)
}
func (h *StreamHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.streams.Remove(r.PathValue("id")); err != nil {
		streamError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *StreamHandler) Start(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, h.streams.Start)
}
func (h *StreamHandler) Stop(w http.ResponseWriter, r *http.Request) { h.action(w, r, h.streams.Stop) }
func (h *StreamHandler) Restart(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, h.streams.Restart)
}
func (h *StreamHandler) action(w http.ResponseWriter, r *http.Request, action func(string) (models.Stream, error)) {
	stream, err := action(r.PathValue("id"))
	if err != nil {
		streamError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, stream)
}

func streamError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, services.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, services.ErrDuplicate):
		status = http.StatusConflict
	case errors.Is(err, services.ErrLimit):
		status = http.StatusTooManyRequests
	case errors.Is(err, services.ErrFFmpegUnavailable), errors.Is(err, services.ErrShutdown):
		status = http.StatusServiceUnavailable
	}
	response.Error(w, status, err.Error())
}
