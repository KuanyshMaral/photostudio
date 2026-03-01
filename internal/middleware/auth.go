package middleware

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"photostudio/internal/domain/catalog"
	"photostudio/internal/pkg/chicontext"
)

const maxWebSocketTokenLength = 8192

// OwnershipChecker provides middleware to verify resource ownership
type OwnershipChecker struct {
	studioRepo *catalog.StudioRepository
	roomRepo   *catalog.RoomRepository
}

// NewOwnershipChecker creates a new ownership checker
func NewOwnershipChecker(
	studioRepo *catalog.StudioRepository,
	roomRepo *catalog.RoomRepository,
) *OwnershipChecker {
	return &OwnershipChecker{
		studioRepo: studioRepo,
		roomRepo:   roomRepo,
	}
}

// CheckStudioOwnership verifies the user owns the studio (chi-native http.Handler version).
// Satisfies catalog.OwnershipMiddleware and booking.OwnershipMiddleware.
func (oc *OwnershipChecker) CheckStudioOwnership() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := chicontext.UserIDFromCtx(r.Context())
			if userID == 0 {
				chiWriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
				return
			}

			studioIDStr := chi.URLParam(r, "id")
			studioID, err := strconv.ParseInt(studioIDStr, 10, 64)
			if err != nil {
				chiWriteError(w, http.StatusBadRequest, "INVALID_ID", "Invalid studio ID")
				return
			}

			studio, err := oc.studioRepo.GetByID(r.Context(), studioID)
			if err != nil {
				chiWriteError(w, http.StatusNotFound, "NOT_FOUND", "Studio not found")
				return
			}

			if studio.OwnerID != userID {
				chiWriteError(w, http.StatusForbidden, "FORBIDDEN", "You don't own this studio")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CheckRoomOwnership verifies the user owns the studio for the given room (chi-native version).
func (oc *OwnershipChecker) CheckRoomOwnership() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := chicontext.UserIDFromCtx(r.Context())
			if userID == 0 {
				chiWriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
				return
			}

			roomIDStr := chi.URLParam(r, "id")
			roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
			if err != nil {
				chiWriteError(w, http.StatusBadRequest, "INVALID_ID", "Invalid room ID")
				return
			}

			room, err := oc.roomRepo.GetByID(r.Context(), roomID)
			if err != nil {
				chiWriteError(w, http.StatusNotFound, "NOT_FOUND", "Room not found")
				return
			}

			studio, err := oc.studioRepo.GetByID(r.Context(), room.StudioID)
			if err != nil {
				chiWriteError(w, http.StatusNotFound, "NOT_FOUND", "Studio not found")
				return
			}

			if studio.OwnerID != userID {
				chiWriteError(w, http.StatusForbidden, "FORBIDDEN", "You don't own this resource")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
