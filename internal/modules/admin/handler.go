package admin

import (
	"log"
	"net/http"
	"strconv"

	"photostudio/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(admin *gin.RouterGroup) {
	// studios moderation
	admin.GET("/studios/pending", h.GetPendingStudios)
	admin.POST("/studios/:id/approve", h.ApproveStudio)
	admin.POST("/studios/:id/reject", h.RejectStudio)

	// statistics
	admin.GET("/stats", h.GetStats)

	// users moderation
	admin.GET("/users", h.GetUsers)
	admin.PATCH("/users/:id/ban", h.BanUser)
	admin.PATCH("/users/:id/unban", h.UnbanUser)

	// reviews moderation
	admin.GET("/reviews", h.GetReviews)
	admin.POST("/reviews/:id/hide", h.HideReview)
	admin.POST("/reviews/:id/show", h.ShowReview)

	// Aliases для обратной совместимости
	admin.POST("/studios/:id/verify", h.ApproveStudio)
	admin.GET("/statistics", h.GetStats)
	admin.POST("/users/:id/block", h.BanUser)
	admin.POST("/users/:id/unblock", h.UnbanUser)

	// analytics
	admin.GET("/analytics", h.GetPlatformAnalytics)

	// vip/gold/promo
	admin.PATCH("/studios/:id/vip", h.SetStudioVIP)
	admin.PATCH("/studios/:id/gold", h.SetStudioGold)
	admin.PATCH("/studios/:id/promo", h.SetStudioPromo)

	// ads
	admin.GET("/ads", h.GetAds)
	admin.POST("/ads", h.CreateAd)
	admin.PATCH("/ads/:id", h.UpdateAd)
	admin.DELETE("/ads/:id", h.DeleteAd)

	// reviews new style (keep old POST routes too)
	admin.PATCH("/reviews/:id/hide", h.HideReview)
	admin.DELETE("/reviews/:id", h.DeleteReview)

}

// GetPendingStudios получает список студий ожидающих одобрения администратором.
// @Summary		Получить список ожидающих студий
// @Description	Возвращает постраничный список студий владельцев, которые ждут одобрения от администратора. Доступно только для администраторов.
// @Tags		Admin - Модерация студий
// @Security	BearerAuth
// @Param		page	query	int		false	"Номер страницы (по умолчанию 1)"	default(1)
// @Param		limit	query	int		false	"Количество записей на странице (по умолчанию 20)"	default(20)
// @Success		200	{object}		map[string]interface{} "Список ожидающих студий"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
	// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при получении данных"
// @Router		/admin/studios/pending [GET]
func (h *Handler) GetPendingStudios(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	page := parseIntDefault(c.Query("page"), 1)
	limit := parseIntDefault(c.Query("limit"), 20)

	owners, total, err := h.service.GetPendingStudioOwners(c.Request.Context(), page, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"pending_studios": owners,
		"count":           total,
	})
}

// ApproveStudio одобряет заявку на регистрацию студии владельцем.
// @Summary		Одобрить студию владельца
// @Description	Одобряет заявку на открытие студии от владельца. Запись указывает, что администратор подтвердил право владельца на управление студией. После одобрения студия появится в каталоге.
// @Tags		Admin - Модерация студий
// @Security	BearerAuth
// @Param		id	path	int	true	"ID владельца студии"
// @Success		200	{object}		map[string]interface{} "Студия успешно одобрена"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный ID владельца или студия уже одобрена"
	// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
	// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Router		/admin/studios/:id/approve [POST]
func (h *Handler) ApproveStudio(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	adminID := c.GetInt64("user_id")
	if adminID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	studioOwnerID, err := parseIDParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid studio owner ID")
		return
	}

	if err := h.service.ApproveStudioOwner(c.Request.Context(), studioOwnerID, adminID); err != nil {
		response.Error(c, http.StatusBadRequest, "APPROVE_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Studio verified successfully"})
}

// VerifyStudio верифицирует студию владельца с дополнительными заметками администратора.
// @Summary		Верифицировать студию
// @Description	Проверяет и верифицирует студию с добавлением административных заметок. Альтернативный эндпоинт для одобрения студии с дополнительной информацией.
// @Tags		Admin - Модерация студий
// @Security	BearerAuth
// @Param		id		path	int				true	"ID студии"
// @Param		request	body	VerifyStudioRequest	true	"Данные верификации (admin_notes)"
// @Success		200	{object}	interface{} "Студия успешно верифицирована"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный ID студии"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при верификации студии"
// @Router		/admin/studios/:id/verify [POST]
func (h *Handler) VerifyStudio(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	adminID := c.GetInt64("user_id")
	if adminID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	studioID, err := parseIDParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid studio ID")
		return
	}

	var req VerifyStudioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	log.Printf("admin action: VerifyStudio admin_id=%d studio_id=%d notes=%q", adminID, studioID, req.AdminNotes)

	studio, err := h.service.VerifyStudio(c.Request.Context(), studioID, adminID, req.AdminNotes)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, studio)
}

// RejectStudio отклоняет заявку на регистрацию студии владельцем.
// @Summary		Отклонить заявку студии
// @Description	Отклоняет заявку на открытие студии от владельца с указанием причины. Владелец сможет подать новую заявку и исправить ошибки.
// @Tags		Admin - Модерация студий
// @Security	BearerAuth
// @Param		id		path	int						true	"ID владельца студии"
// @Param		request	body	RejectStudioRequest		true	"Причина отклонения заявки"
// @Success		200	{object}		map[string]interface{} "Заявка успешно отклонена"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный ID владельца или отсутствует причина"
// @Failure		401	{object}		map[string]interface{} "Ошибка аутентификации"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Router		/admin/studios/:id/reject [POST]
func (h *Handler) RejectStudio(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	adminID := c.GetInt64("user_id")
	if adminID == 0 {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	studioOwnerID, err := parseIDParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid studio owner ID")
		return
	}

	var req RejectStudioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Reason is required")
		return
	}

	if err := h.service.RejectStudioOwner(c.Request.Context(), studioOwnerID, adminID, req.Reason); err != nil {
		response.Error(c, http.StatusBadRequest, "REJECT_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Application rejected"})
}
// GetStatistics получает расширённую статистику платформы.
// @Summary		Получить расширённую статистику
// @Description	Возвращает детальную статистику платформы. Алиас для GetStats с альтернативным путём (для обратной совместимости).
// @Tags		Admin - Статистика и аналитика
// @Security	BearerAuth
// @Success		200	{object}	interface{} "Объект со статистикой платформы"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при получении статистики"
// @Router		/admin/statistics [GET]
func (h *Handler) GetStatistics(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	log.Printf("admin action: GetStatistics")

	stats, err := h.service.GetStatistics(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, stats)
}

// GetStats получает основную статистику платформы.
// @Summary		Получить статистику платформы
// @Description	Возвращает ключевые показатели платформы: количество пользователей, студий, бронирований, доход и другие метрики. Доступно только администраторам.
// @Tags		Admin - Статистика и аналитика
// @Security	BearerAuth
// @Success		200	{object}		map[string]interface{} "Объект со статистикой платформы"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при получении статистики"
// @Router		/admin/stats [GET]
func (h *Handler) GetStats(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	stats, err := h.service.GetPlatformStats(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, stats)
}

// -------------------- Users --------------------

// BlockUser блокирует пользователя с указанной причиной (алиас для BanUser).
// @Summary		Заблокировать пользователя (алиас)
// @Description	Блокирует пользователя с указанной причиной. Алиас для BanUser для обратной совместимости.
// @Tags		Admin - Управление пользователями
// @Security	BearerAuth
// @Param		id		path	int				true	"ID пользователя"
// @Param		request	body	BlockUserRequest	true	"Причина блокировки"
// @Success		200	{object}	interface{} "Пользователь успешно заблокирован"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный ID пользователя"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при блокировке пользователя"
// @Router		/admin/users/:id/block [POST]
func (h *Handler) BlockUser(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	userID, err := parseIDParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	var req BlockUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	log.Printf("admin action: BlockUser user_id=%d reason=%q", userID, req.Reason)

	u, err := h.service.BlockUser(c.Request.Context(), userID, req.Reason)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, u)
}

// UnblockUser разблокирует пользователя (алиас для UnbanUser).
// @Summary		Разблокировать пользователя (алиас)
// @Description	Восстанавливает доступ заблокированного пользователя. Алиас для UnbanUser для обратной совместимости.
// @Tags		Admin - Управление пользователями
// @Security	BearerAuth
// @Param		id	path	int	true	"ID пользователя"
// @Success		200	{object}	interface{} "Пользователь успешно разблокирован"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный ID пользователя"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при разблокировке пользователя"
// @Router		/admin/users/:id/unblock [POST]
func (h *Handler) UnblockUser(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	userID, err := parseIDParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	log.Printf("admin action: UnblockUser user_id=%d", userID)

	u, err := h.service.UnblockUser(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, u)
}

// BanUser блокирует пользователя на платформе.
// @Summary		Заблокировать пользователя
// @Description	Блокирует пользователя с указанной причиной. Заблокированный пользователь не сможет использовать платформу, но его данные сохраняются.
// @Tags		Admin - Управление пользователями
// @Security	BearerAuth
// @Param		id		path	int					true	"ID пользователя"
// @Param		request	body	BlockUserRequest	true	"Причина блокировки"
// @Success		200	{object}		map[string]interface{} "Пользователь успешно заблокирован"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный ID пользователя или отсутствует причина"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при блокировке пользователя"
// @Router		/admin/users/:id/ban [PATCH]
func (h *Handler) BanUser(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	userID, err := parseIDParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	var req BlockUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Reason is required")
		return
	}

	if err := h.service.BanUser(c.Request.Context(), userID, req.Reason); err != nil {
		response.Error(c, http.StatusBadRequest, "BAN_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "User banned"})
}

// UnbanUser разблокирует ранее заблокированного пользователя.
// @Summary		Разблокировать пользователя
// @Description	Восстанавливает доступ заблокированного пользователя к платформе.
// @Tags		Admin - Управление пользователями
// @Security	BearerAuth
// @Param		id	path	int	true	"ID пользователя"
// @Success		200	{object}		map[string]interface{} "Пользователь успешно разблокирован"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный ID пользователя"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при разблокировке пользователя"
// @Router		/admin/users/:id/unban [PATCH]
func (h *Handler) UnbanUser(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	userID, err := parseIDParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	if err := h.service.UnbanUser(c.Request.Context(), userID); err != nil {
		response.Error(c, http.StatusBadRequest, "UNBAN_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "User unbanned"})
}

// GetUsers получает список всех пользователей с фильтрацией.
// @Summary		Получить список пользователей
// @Description	Возвращает постраничный список всех пользователей платформы с возможностью фильтрации по статусу и другим параметрам. Доступно только администраторам.
// @Tags		Admin - Управление пользователями
// @Security	BearerAuth
// @Param		page	query	int		false	"Номер страницы (по умолчанию 1)"	default(1)
// @Param		limit	query	int		false	"Количество записей на странице (по умолчанию 20)"	default(20)
// @Param		status	query	string	false	"Фильтр по статусу пользователя (active, banned, etc.)"
// @Success		200	{object}	UserListResponse "Список пользователей с общим количеством и параметрами страницы"
// @Failure		400	{object}		map[string]interface{} "Ошибка валидации параметров запроса"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при получении списка пользователей"
// @Router		/admin/users [GET]
func (h *Handler) GetUsers(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	page := parseIntDefault(c.Query("page"), 1)
	limit := parseIntDefault(c.Query("limit"), 20)

	var filter UserListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	log.Printf("admin action: GetUsers page=%d limit=%d", page, limit)

	users, total, err := h.service.ListUsers(c.Request.Context(), filter, page, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, UserListResponse{
		Users: users,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// -------------------- Reviews --------------------

// GetReviews получает список отзывов с возможностью фильтрации.
// @Summary		Получить список отзывов
// @Description	Возвращает постраничный список всех отзывов на платформе с фильтром по скрытым/видимым отзывам и другим параметрам. Доступно только администраторам.
// @Tags		Admin - Модерация отзывов
// @Security	BearerAuth
// @Param		page	query	int		false	"Номер страницы (по умолчанию 1)"	default(1)
// @Param		limit	query	int		false	"Количество записей на странице (по умолчанию 20)"	default(20)
// @Param		hidden	query	boolean	false	"Показать только скрытые отзывы"
// @Success		200	{object}	ReviewListResponse "Список отзывов с общим количеством и параметрами страницы"
// @Failure		400	{object}		map[string]interface{} "Ошибка валидации параметров запроса"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при получении списка отзывов"
// @Router		/admin/reviews [GET]
func (h *Handler) GetReviews(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	page := parseIntDefault(c.Query("page"), 1)
	limit := parseIntDefault(c.Query("limit"), 20)

	var filter ReviewListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	log.Printf("admin action: GetReviews page=%d limit=%d", page, limit)

	reviews, total, err := h.service.ListReviews(c.Request.Context(), filter, page, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, ReviewListResponse{
		Reviews: reviews,
		Total:   total,
		Page:    page,
		Limit:   limit,
	})
}

// HideReview скрывает отзыв из публичного доступа на платформе.
// @Summary		Скрыть отзыв
// @Description	Скрывает отзыв, чтобы он не отображался в списке отзывов студии. Отзыв остаётся в БД, но не видит пользователям.
// @Tags		Admin - Модерация отзывов
// @Security	BearerAuth
// @Param		id	path	int	true	"ID отзыва"
// @Success		200	{object}	interface{} "Информация об отзыве после скрытия"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный ID отзыва"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при скрытии отзыва"
// @Router		/admin/reviews/:id/hide [PATCH]
func (h *Handler) HideReview(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	reviewID, err := parseIDParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid review ID")
		return
	}

	log.Printf("admin action: HideReview review_id=%d", reviewID)

	rv, err := h.service.HideReview(c.Request.Context(), reviewID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, rv)
}

// ShowReview восстанавливает видимость ранее скрытого отзыва.
// @Summary		Показать отзыв
// @Description	Восстанавливает видимость скрытого отзыва, чтобы он вновь отображался в списке отзывов студии.
// @Tags		Admin - Модерация отзывов
// @Security	BearerAuth
// @Param		id	path	int	true	"ID отзыва"
// @Success		200	{object}	interface{} "Информация об отзыве после восстановления"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный ID отзыва"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при восстановлении отзыва"
// @Router		/admin/reviews/:id/show [PATCH]
func (h *Handler) ShowReview(c *gin.Context) {
	if !isAdmin(c) {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	reviewID, err := parseIDParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid review ID")
		return
	}

	log.Printf("admin action: ShowReview review_id=%d", reviewID)

	rv, err := h.service.ShowReview(c.Request.Context(), reviewID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(c, http.StatusOK, rv)
}

// -------------------- helpers --------------------

func isAdmin(c *gin.Context) bool {
	role, ok := c.Get("role")
	if !ok {
		return false
	}
	rs, ok := role.(string)
	return ok && rs == "admin"
}

func parseIDParam(c *gin.Context, name string) (int64, error) {
	return strconv.ParseInt(c.Param(name), 10, 64)
}

func parseIntDefault(v string, def int) int {
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// GetPlatformAnalytics получает детальную аналитику платформы за период.
// @Summary		Получить аналитику платформы
// @Description	Возвращает подробную аналитику активности платформы за указанный период: данные о пользователях, бронированиях, доходах, популярных студиях и другие метрики.
// @Tags		Admin - Статистика и аналитика
// @Security	BearerAuth
// @Param		days	query	int	false	"Количество дней для аналитики (1-365, по умолчанию 30)"	default(30)
// @Success		200	{object}		map[string]interface{} "Аналитические данные платформы"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при получении аналитики"
// @Router		/admin/analytics [GET]
func (h *Handler) GetPlatformAnalytics(c *gin.Context) {
	daysBack := 30
	if d := c.Query("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 && v <= 365 {
			daysBack = v
		}
	}

	analytics, err := h.service.GetPlatformAnalytics(c.Request.Context(), daysBack)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "ANALYTICS_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"analytics": analytics})
}

// SetStudioVIP устанавливает или снимает VIP статус студии.
// @Summary		Установить VIP статус студии
// @Description	Назначает или отменяет VIP статус студии. VIP студии получают приоритет в выдаче и специальные выделения в каталоге.
// @Tags		Admin - Управление студиями
// @Security	BearerAuth
// @Param		id		path	int						true	"ID студии"
// @Param		request	body	object{is_vip=boolean}	true	"Значение VIP статуса (true/false)"
// @Success		200	{object}		map[string]interface{} "VIP статус успешно обновлён"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный запрос"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при обновлении VIP статуса"
// @Router		/admin/studios/:id/vip [PATCH]
func (h *Handler) SetStudioVIP(c *gin.Context) {
	studioID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		IsVIP bool `json:"is_vip"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.service.SetStudioVIP(c.Request.Context(), studioID, req.IsVIP); err != nil {
		response.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "VIP status updated"})
}

// SetStudioGold устанавливает или снимает Gold статус студии.
// @Summary		Установить Gold статус студии
// @Description	Назначает или отменяет Gold статус студии. Gold студии получают улучшенный рейтинг и видимость в поиске.
// @Tags		Admin - Управление студиями
// @Security	BearerAuth
// @Param		id		path	int						true	"ID студии"
// @Param		request	body	object{is_gold=boolean}	true	"Значение Gold статуса (true/false)"
// @Success		200	{object}		map[string]interface{} "Gold статус успешно обновлён"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный запрос"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при обновлении Gold статуса"
// @Router		/admin/studios/:id/gold [PATCH]
func (h *Handler) SetStudioGold(c *gin.Context) {
	studioID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		IsGold bool `json:"is_gold"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.service.SetStudioGold(c.Request.Context(), studioID, req.IsGold); err != nil {
		response.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Gold status updated"})
}

// SetStudioPromo добавляет или убирает студию из промо-слайдера главной страницы.
// @Summary		Установить промо статус студии
// @Description	Добавляет студию в ротирующийся слайдер продвигаемых студий на главной странице или убирает её оттуда. Студии в слайдере получают дополнительную видимость.
// @Tags		Admin - Управление студиями
// @Security	BearerAuth
// @Param		id		path	int									true	"ID студии"
// @Param		request	body	object{in_promo_slider=boolean}	true	"Значение промо статуса (true/false)"
// @Success		200	{object}		map[string]interface{} "Промо статус успешно обновлён"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный запрос"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при обновлении промо статуса"
// @Router		/admin/studios/:id/promo [PATCH]
func (h *Handler) SetStudioPromo(c *gin.Context) {
	studioID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		InPromo bool `json:"in_promo_slider"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.service.SetStudioPromo(c.Request.Context(), studioID, req.InPromo); err != nil {
		response.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Promo status updated"})
}

// GetAds получает список всех объявлений/баннеров на платформе.
// @Summary		Получить список объявлений
// @Description	Возвращает список баннеров и объявлений с фильтром по месту размещения и статусу активности. Используется для управления рекламными материалами на платформе.
// @Tags		Admin - Объявления и реклама
// @Security	BearerAuth
// @Param		placement	query	string	false	"Место размещения объявления (homepage, booking_page, etc.)"
// @Param		active_only	query	boolean	false	"Показать только активные объявления (true/false)"
// @Success		200	{object}		map[string]interface{} "Список объявлений"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при получении объявлений"
// @Router		/admin/ads [GET]
func (h *Handler) GetAds(c *gin.Context) {
	placement := c.Query("placement")
	activeOnly := c.Query("active_only") == "true"

	ads, err := h.service.GetAds(c.Request.Context(), placement, activeOnly)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"ads": ads})
}

// CreateAd создаёт новое объявление или баннер на платформе.
// @Summary		Создать объявление
// @Description	Создаёт новый баннер/объявление для размещения на платформе. Объявление может быть размещено в различных местах: на главной странице, странице бронирования и т.д.
// @Tags		Admin - Объявления и реклама
// @Security	BearerAuth
// @Param		request	body	Ad	true	"Данные объявления (placement, url, image_url, active, etc.)"
// @Success		201	{object}		map[string]interface{} "Объявление успешно создано"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный запрос или отсутствуют необходимые поля"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при создании объявления"
// @Router		/admin/ads [POST]
func (h *Handler) CreateAd(c *gin.Context) {
	var ad Ad
	if err := c.ShouldBindJSON(&ad); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.service.CreateAd(c.Request.Context(), &ad); err != nil {
		response.Error(c, http.StatusInternalServerError, "CREATE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"ad": ad})
}

// UpdateAd обновляет существующее объявление.
// @Summary		Обновить объявление
// @Description	Обновляет параметры существующего баннера/объявления: URL, изображение, место размещения, статус активности и другие поля.
// @Tags		Admin - Объявления и реклама
// @Security	BearerAuth
// @Param		id		path	int						true	"ID объявления"
// @Param		request	body	map[string]interface{}	true	"Поля для обновления (placement, url, image_url, active, etc.)"
// @Success		200	{object}		map[string]interface{} "Объявление успешно обновлено"
// @Failure		400	{object}		map[string]interface{} "Ошибка: неверный запрос"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при обновлении объявления"
// @Router		/admin/ads/:id [PATCH]
func (h *Handler) UpdateAd(c *gin.Context) {
	adID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.service.UpdateAd(c.Request.Context(), adID, updates); err != nil {
		response.Error(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Ad updated"})
}

// DeleteAd удаляет объявление с платформы.
// @Summary		Удалить объявление
// @Description	Удаляет и удаляет из БД баннер/объявление с платформы. После удаления объявление больше не будет отображаться.
// @Tags		Admin - Объявления и реклама
// @Security	BearerAuth
// @Param		id	path	int	true	"ID объявления"
// @Success		200	{object}		map[string]interface{} "Объявление успешно удалено"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при удалении объявления"
// @Router		/admin/ads/:id [DELETE]
func (h *Handler) DeleteAd(c *gin.Context) {
	adID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := h.service.DeleteAd(c.Request.Context(), adID); err != nil {
		response.Error(c, http.StatusInternalServerError, "DELETE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Ad deleted"})
}

// DeleteReview удаляет отзыв из системы (полное удаление).
// @Summary		Удалить отзыв
// @Description	Полностью удаляет отзыв из базы данных. Это необратимое действие. Отзыв указанного автора удаляется вместе со всеми его данными.
// @Tags		Admin - Модерация отзывов
// @Security	BearerAuth
// @Param		id	path	int	true	"ID отзыва"
// @Success		200	{object}		map[string]interface{} "Отзыв успешно удалён"
// @Failure		403	{object}		map[string]interface{} "Доступ запрещён (требуются права администратора)"
// @Failure		500	{object}		map[string]interface{} "Ошибка сервера при удалении отзыва"
// @Router		/admin/reviews/:id [DELETE]
func (h *Handler) DeleteReview(c *gin.Context) {
	reviewID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := h.service.DeleteReview(c.Request.Context(), reviewID); err != nil {
		response.Error(c, http.StatusInternalServerError, "DELETE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Review deleted"})
}


