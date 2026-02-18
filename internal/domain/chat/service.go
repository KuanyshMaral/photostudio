package chat

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// BlockChecker is implemented by the relationship service
type BlockChecker interface {
	IsBlocked(ctx context.Context, userA, userB int64) (bool, error)
}

// Service handles chat business logic
type Service struct {
	repo         Repository
	blockChecker BlockChecker
}

func NewService(repo Repository, blockChecker BlockChecker) *Service {
	return &Service{repo: repo, blockChecker: blockChecker}
}

// ---- Direct Room ----

// GetOrCreateDirectRoom returns an existing 1-on-1 room or creates a new one.
// Checks that neither user has blocked the other.
func (s *Service) GetOrCreateDirectRoom(ctx context.Context, userID, recipientID int64) (*Room, error) {
	if userID == recipientID {
		return nil, ErrCannotChatSelf
	}

	if s.blockChecker != nil {
		blocked, err := s.blockChecker.IsBlocked(ctx, userID, recipientID)
		if err != nil {
			return nil, err
		}
		if blocked {
			return nil, ErrUserBlocked
		}
	}

	existing, err := s.repo.GetDirectRoomByUsers(ctx, userID, recipientID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	room := &Room{
		ID:        uuid.New().String(),
		Type:      RoomTypeDirect,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateRoom(ctx, room); err != nil {
		return nil, err
	}

	now := time.Now()
	for _, uid := range []int64{userID, recipientID} {
		if err := s.repo.AddMember(ctx, &RoomMember{
			RoomID:   room.ID,
			UserID:   uid,
			Role:     MemberRoleMember,
			JoinedAt: now,
		}); err != nil {
			return nil, err
		}
	}
	return room, nil
}

// ---- Group Room ----

// CreateGroupRoom creates a named group room. Creator becomes admin.
func (s *Service) CreateGroupRoom(ctx context.Context, creatorID int64, name string, memberIDs []int64) (*Room, error) {
	room := &Room{
		ID:        uuid.New().String(),
		Type:      RoomTypeGroup,
		Name:      sql.NullString{String: name, Valid: name != ""},
		CreatorID: sql.NullInt64{Int64: creatorID, Valid: true},
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateRoom(ctx, room); err != nil {
		return nil, err
	}

	now := time.Now()
	// Creator is admin
	if err := s.repo.AddMember(ctx, &RoomMember{
		RoomID:   room.ID,
		UserID:   creatorID,
		Role:     MemberRoleAdmin,
		JoinedAt: now,
	}); err != nil {
		return nil, err
	}

	// Other members
	for _, uid := range memberIDs {
		if uid == creatorID {
			continue
		}
		_ = s.repo.AddMember(ctx, &RoomMember{
			RoomID:   room.ID,
			UserID:   uid,
			Role:     MemberRoleMember,
			JoinedAt: now,
		})
	}
	return room, nil
}

// ---- Member Management ----

// AddMember adds a user to a group room. Only admins can do this.
func (s *Service) AddMember(ctx context.Context, requesterID int64, roomID string, newMemberID int64) error {
	room, err := s.repo.GetRoomByID(ctx, roomID)
	if err != nil {
		return err
	}

	isMember, _ := s.repo.IsMember(ctx, roomID, requesterID)
	if !isMember {
		return ErrNotRoomMember
	}

	if room.Type == RoomTypeGroup {
		member, _ := s.repo.GetMember(ctx, roomID, requesterID)
		if member == nil || !member.IsAdmin() {
			return ErrNotRoomAdmin
		}
	}

	alreadyMember, _ := s.repo.IsMember(ctx, roomID, newMemberID)
	if alreadyMember {
		return ErrAlreadyMember
	}

	return s.repo.AddMember(ctx, &RoomMember{
		RoomID:   roomID,
		UserID:   newMemberID,
		Role:     MemberRoleMember,
		JoinedAt: time.Now(),
	})
}

// RemoveMember removes a user from a room. Admin can remove others; anyone can leave.
func (s *Service) RemoveMember(ctx context.Context, requesterID int64, roomID string, targetID int64) error {
	room, err := s.repo.GetRoomByID(ctx, roomID)
	if err != nil {
		return err
	}

	isMember, _ := s.repo.IsMember(ctx, roomID, requesterID)
	if !isMember {
		return ErrNotRoomMember
	}

	// Self-leave always allowed
	if requesterID == targetID {
		return s.repo.RemoveMember(ctx, roomID, targetID)
	}

	// Only admin can remove others in group rooms
	if room.Type == RoomTypeGroup {
		member, _ := s.repo.GetMember(ctx, roomID, requesterID)
		if member == nil || !member.IsAdmin() {
			return ErrNotRoomAdmin
		}
	}

	return s.repo.RemoveMember(ctx, roomID, targetID)
}

// GetMembers returns all members of a room (requester must be a member).
func (s *Service) GetMembers(ctx context.Context, requesterID int64, roomID string) ([]*RoomMember, error) {
	isMember, _ := s.repo.IsMember(ctx, roomID, requesterID)
	if !isMember {
		return nil, ErrNotRoomMember
	}
	return s.repo.GetMembers(ctx, roomID)
}

// ---- Messages ----

// SendMessage sends a message to a room. Validates membership and block status.
func (s *Service) SendMessage(ctx context.Context, senderID int64, roomID string, content string, uploadID *string) (*Message, error) {
	room, err := s.repo.GetRoomByID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	isMember, _ := s.repo.IsMember(ctx, roomID, senderID)
	if !isMember {
		return nil, ErrNotRoomMember
	}

	// For direct rooms, check block status
	if room.Type == RoomTypeDirect && s.blockChecker != nil {
		members, _ := s.repo.GetMembers(ctx, roomID)
		for _, m := range members {
			if m.UserID != senderID {
				blocked, _ := s.blockChecker.IsBlocked(ctx, senderID, m.UserID)
				if blocked {
					return nil, ErrUserBlocked
				}
			}
		}
	}

	msg := &Message{
		ID:        uuid.New().String(),
		RoomID:    roomID,
		SenderID:  senderID,
		Content:   content,
		CreatedAt: time.Now(),
	}
	if uploadID != nil && *uploadID != "" {
		msg.UploadID = sql.NullString{String: *uploadID, Valid: true}
	}

	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// GetMessages returns paginated messages for a room.
func (s *Service) GetMessages(ctx context.Context, userID int64, roomID string, limit, offset int) ([]*Message, error) {
	isMember, _ := s.repo.IsMember(ctx, roomID, userID)
	if !isMember {
		return nil, ErrNotRoomMember
	}
	return s.repo.GetMessages(ctx, roomID, limit, offset)
}

// MarkAsRead marks all messages in a room as read for the user.
func (s *Service) MarkAsRead(ctx context.Context, userID int64, roomID string) error {
	isMember, _ := s.repo.IsMember(ctx, roomID, userID)
	if !isMember {
		return ErrNotRoomMember
	}
	return s.repo.MarkRoomAsRead(ctx, roomID, userID)
}

// ListRooms returns all rooms the user is a member of.
func (s *Service) ListRooms(ctx context.Context, userID int64) ([]*RoomWithUnread, error) {
	return s.repo.ListRoomsByUser(ctx, userID)
}

// GetUnreadCount returns total unread messages across all rooms.
func (s *Service) GetUnreadCount(ctx context.Context, userID int64) (int, error) {
	return s.repo.CountTotalUnread(ctx, userID)
}
