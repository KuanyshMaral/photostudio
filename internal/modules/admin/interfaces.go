package admin

import "context"

// Заглушки под будущую реализацию (пока можно оставить пустыми/минимальными).
// Потом сюда добавим методы репозиториев: studios/users/bookings.

type StudioRepository interface {
	// TODO
}

type UserRepository interface {
	// TODO
}

type BookingRepository interface {
	// TODO
}

type ServiceDeps struct {
	Studios  StudioRepository
	Users    UserRepository
	Bookings BookingRepository
}

var _ = context.Background
