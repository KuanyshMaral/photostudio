package chat

import "errors"

var (
	ErrRoomNotFound   = errors.New("room not found")
	ErrNotRoomMember  = errors.New("you are not a member of this room")
	ErrNotRoomAdmin   = errors.New("only room admin can perform this action")
	ErrAlreadyMember  = errors.New("user is already a member")
	ErrCannotChatSelf = errors.New("cannot start chat with yourself")
	ErrUserBlocked    = errors.New("cannot chat — user is blocked")
	ErrUserNotFound   = errors.New("user not found")
)
