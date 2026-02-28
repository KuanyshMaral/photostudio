package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type addMemberTestRepo struct {
	room       *Room
	members    map[int64]bool
	addedUsers []int64
}

func (r *addMemberTestRepo) CreateRoom(ctx context.Context, room *Room) error { return nil }
func (r *addMemberTestRepo) GetRoomByID(ctx context.Context, id string) (*Room, error) {
	if r.room == nil || r.room.ID != id {
		return nil, ErrRoomNotFound
	}
	return r.room, nil
}
func (r *addMemberTestRepo) GetDirectRoomByUsers(ctx context.Context, userA, userB int64) (*Room, error) {
	return nil, nil
}
func (r *addMemberTestRepo) ListRoomsByUser(ctx context.Context, userID int64) ([]*RoomWithUnread, error) {
	return nil, nil
}
func (r *addMemberTestRepo) AddMember(ctx context.Context, m *RoomMember) error {
	r.members[m.UserID] = true
	r.addedUsers = append(r.addedUsers, m.UserID)
	return nil
}
func (r *addMemberTestRepo) RemoveMember(ctx context.Context, roomID string, userID int64) error {
	return nil
}
func (r *addMemberTestRepo) GetMember(ctx context.Context, roomID string, userID int64) (*RoomMember, error) {
	return nil, nil
}
func (r *addMemberTestRepo) GetMembers(ctx context.Context, roomID string) ([]*RoomMember, error) {
	return nil, nil
}
func (r *addMemberTestRepo) IsMember(ctx context.Context, roomID string, userID int64) (bool, error) {
	return r.members[userID], nil
}
func (r *addMemberTestRepo) UpdateLastRead(ctx context.Context, roomID string, userID int64) error {
	return nil
}
func (r *addMemberTestRepo) CreateMessage(ctx context.Context, msg *Message) error { return nil }
func (r *addMemberTestRepo) GetMessages(ctx context.Context, roomID string, limit, offset int) ([]*Message, error) {
	return nil, nil
}
func (r *addMemberTestRepo) CountUnread(ctx context.Context, roomID string, userID int64) (int, error) {
	return 0, nil
}
func (r *addMemberTestRepo) MarkRoomAsRead(ctx context.Context, roomID string, userID int64) error {
	return nil
}
func (r *addMemberTestRepo) CountTotalUnread(ctx context.Context, userID int64) (int, error) {
	return 0, nil
}

func TestAddMember_WithAdminTokenWithoutUserID_Succeeds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &addMemberTestRepo{
		room:    &Room{ID: "7ae6034b-f7c7-4505-9c13-f457a7f47561", Type: RoomTypeGroup, CreatedAt: time.Now()},
		members: map[int64]bool{},
	}
	h := NewHandler(NewService(repo, nil), nil)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("is_admin_token", true)
		c.Next()
	})
	r.POST("/chats/:id/members", h.AddMember)

	body, _ := json.Marshal(gin.H{"user_id": int64(9)})
	req := httptest.NewRequest(http.MethodPost, "/chats/7ae6034b-f7c7-4505-9c13-f457a7f47561/members", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []int64{9}, repo.addedUsers)
	assert.Contains(t, w.Body.String(), "member added")
}
