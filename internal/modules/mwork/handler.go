package mwork

import (
	"errors"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(internal *gin.RouterGroup) {
	mworkGroup := internal.Group("/mwork")
	{
		mworkGroup.POST("/users/sync", h.SyncUser)
	}
}

// SyncUser синхронизирует пользователя из Mwork платформы.
// @Summary		Синхронизация пользователя из Mwork
// @Description	Внутренний API endpoint для синхронизации пользователя из платформы Mwork. Создаёт или обновляет пользователя в PhotoStudio на основе данных из Mwork. Доступно только для внутренних сервисов.
// @Tags		Интеграции - Mwork
// @Security	BearerAuth
// @Param		request	body	SyncUserRequest	true	"Данные пользователя из Mwork (mwork_user_id, email, full_name, role, phone)"
// @Success		200	{object}	gin.H{user_id=int,status=string} "Пользователь синхронизирован успешно"
// @Success		201	{object}	gin.H{user_id=int,status=string} "Новый пользователь создан"
// @Failure		400	{object}	gin.H "Ошибка валидации: неверные данные (invalid UUID, invalid role, etc.)"
// @Failure		409	{object}	gin.H "Конфликт: пользователь с этим email уже существует"
// @Failure		500	{object}	gin.H "Ошибка сервера при синхронизации пользователя"
// @Router		/internal/mwork/users/sync [POST]
func (h *Handler) SyncUser(c *gin.Context) {
	start := time.Now()

	var req SyncUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", validationDetails(err))
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
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request fields", map[string]any{
			"field_errors": fieldErrors,
		})
		return
	}

	user, result, err := h.service.SyncUser(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrConflict) {
			logSync(req, "conflict", start)
			writeError(c, http.StatusConflict, "CONFLICT", "User conflict", nil)
			return
		}
		logSync(req, "error", start)
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to sync user", nil)
		return
	}

	status := http.StatusOK
	if result == ResultCreated {
		status = http.StatusCreated
	}

	logSync(req, string(result), start)
	c.JSON(status, gin.H{
		"data": SyncUserResponse{
			ID:          user.ID,
			MworkUserID: req.MworkUserID,
			Email:       user.Email,
			Role:        req.Role,
		},
	})
}

func writeError(c *gin.Context, status int, code, message string, details map[string]any) {
	payload := gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	}
	if details != nil {
		payload["error"].(gin.H)["details"] = details
	}
	c.JSON(status, payload)
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
	return map[string]any{
		"field_errors": fieldErrors,
	}
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
