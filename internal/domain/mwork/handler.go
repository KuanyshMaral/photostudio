package mwork

import (
	"errors"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"photostudio/internal/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) SyncUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req SyncUserRequest
	if err := response.BindJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", validationDetails(err))
		return
	}

	fieldErrors := map[string]string{}
	if _, err := uuid.Parse(req.MworkUserID); err != nil {
		fieldErrors["mwork_user_id"] = "must be a valid UUID"
	}
	if !isValidRole(req.Role) {
		fieldErrors["role"] = "must be one of model, employer, agency, admin"
	}
	if len(fieldErrors) > 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request fields", map[string]any{
			"field_errors": fieldErrors,
		})
		return
	}

	user, result, err := h.service.SyncUser(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrConflict) {
			logSync(req, "conflict", start)
			writeError(w, http.StatusConflict, "CONFLICT", "User conflict", nil)
			return
		}
		logSync(req, "error", start)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to sync user", nil)
		return
	}

	status := http.StatusOK
	if result == ResultCreated {
		status = http.StatusCreated
	}

	logSync(req, string(result), start)
	response.JSON(w, status, response.H{
		"data": SyncUserResponse{
			ID:          user.ID,
			MworkUserID: req.MworkUserID,
			Email:       user.Email,
			Role:        req.Role,
		},
	})
}

func writeError(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	payload := response.H{
		"error": response.H{
			"code":    code,
			"message": message,
		},
	}
	if details != nil {
		payload["error"].(response.H)["details"] = details
	}
	response.JSON(w, status, payload)
}

func validationDetails(err error) map[string]any {
	fieldErrors := map[string]string{}
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		requestType := reflect.TypeOf(SyncUserRequest{})
		for _, fieldError := range validationErrors {
			fieldName := fieldError.Field()
			if requestType.Kind() == reflect.Struct {
				if field, ok := requestType.FieldByName(fieldError.StructField()); ok {
					jsonTag := field.Tag.Get("json")
					if jsonTag != "" {
						fieldName = strings.Split(jsonTag, ",")[0]
					}
				}
			}
			fieldErrors[fieldName] = validationErrorMessage(fieldError)
		}
	}
	if len(fieldErrors) == 0 {
		return nil
	}
	return map[string]any{"field_errors": fieldErrors}
}

func validationErrorMessage(fieldError validator.FieldError) string {
	switch fieldError.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	default:
		return "is invalid"
	}
}

func isValidRole(role string) bool {
	switch role {
	case "model", "employer", "agency", "admin":
		return true
	default:
		return false
	}
}

func logSync(req SyncUserRequest, result string, start time.Time) {
	latency := time.Since(start).Milliseconds()
	log.Printf(
		"mwork_sync mwork_user_id=%s email=%s role=%s result=%s latency_ms=%d",
		req.MworkUserID,
		req.Email,
		req.Role,
		result,
		latency,
	)
}
