package sundayaccount

import (
	"io"
	"net/http"
	"strings"

	pb "github.com/studyverse/ems-backend/gen/registrationforms/v1"
	"github.com/studyverse/ems-backend/internal/registrationfiles"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func decodeProto(w http.ResponseWriter, r *http.Request, target proto.Message, allowEmpty bool) bool {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 16384))
	if err != nil {
		fail(w, 400, "Invalid request")
		return false
	}
	if allowEmpty && len(strings.TrimSpace(string(raw))) == 0 {
		return true
	}
	if err := protojson.Unmarshal(raw, target); err != nil {
		fail(w, 400, "Invalid request")
		return false
	}
	return true
}

func writeProto(w http.ResponseWriter, message proto.Message) {
	raw, err := protojson.Marshal(message)
	if err != nil {
		fail(w, 503, "Could not load upload details")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (h *Handler) beginFormUpload(w http.ResponseWriter, r *http.Request, p *principal) {
	input := new(pb.BeginFileUploadRequest)
	if !decodeProto(w, r, input, false) {
		return
	}
	if h.forms == nil {
		failCode(w, 409, "failed_precondition", "Private file storage is unavailable")
		return
	}
	response, err := h.forms.BeginUploadForOwner(r.Context(), registrationfiles.Owner{AccountID: p.ID}, input)
	if err != nil {
		registrationError(w, err)
		return
	}
	writeProto(w, response)
}

func (h *Handler) formUpload(w http.ResponseWriter, r *http.Request, p *principal, path string) {
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" {
		fail(w, 404, "Not found")
		return
	}
	owner := registrationfiles.Owner{AccountID: p.ID}
	switch {
	case parts[1] == "complete" && r.Method == "POST":
		input := new(pb.CompleteFileUploadRequest)
		if !decodeProto(w, r, input, true) {
			return
		}
		if input.UploadId != "" && input.UploadId != parts[0] {
			fail(w, 400, "Upload ID does not match the request path")
			return
		}
		input.UploadId = parts[0]
		if h.forms == nil {
			failCode(w, 409, "failed_precondition", "Private file storage is unavailable")
			return
		}
		response, err := h.forms.CompleteUploadForOwner(r.Context(), owner, input)
		if err != nil {
			registrationError(w, err)
			return
		}
		writeProto(w, response)
	case parts[1] == "download" && r.Method == "GET":
		if h.forms == nil {
			failCode(w, 409, "failed_precondition", "Private file storage is unavailable")
			return
		}
		response, err := h.forms.DownloadForOwner(r.Context(), owner, parts[0])
		if err != nil {
			registrationError(w, err)
			return
		}
		writeProto(w, response)
	default:
		fail(w, 404, "Not found")
	}
}
