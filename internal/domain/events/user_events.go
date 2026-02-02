package events

import (
	"time"

	"turtle/internal/domain/valueobjects"
)

// DomainEvent is the base interface for all domain events
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
	AggregateID() string
}

// BaseEvent provides common fields for all events
type BaseEvent struct {
	eventType   string
	aggregateID string
	occurredAt  time.Time
}

func (e BaseEvent) EventType() string     { return e.eventType }
func (e BaseEvent) OccurredAt() time.Time { return e.occurredAt }
func (e BaseEvent) AggregateID() string   { return e.aggregateID }

// User Events

type UserCreatedEvent struct {
	BaseEvent
	Role  string
	Email *string
	Phone *string
}

func NewUserCreatedEvent(userID, role string, email, phone *string) *UserCreatedEvent {
	return &UserCreatedEvent{
		BaseEvent: BaseEvent{
			eventType:   "user.created",
			aggregateID: userID,
			occurredAt:  time.Now(),
		},
		Role:  role,
		Email: email,
		Phone: phone,
	}
}

type UserProfileUpdatedEvent struct {
	BaseEvent
}

func NewUserProfileUpdatedEvent(userID string) *UserProfileUpdatedEvent {
	return &UserProfileUpdatedEvent{
		BaseEvent: BaseEvent{
			eventType:   "user.profile.updated",
			aggregateID: userID,
			occurredAt:  time.Now(),
		},
	}
}

type UserEmailVerifiedEvent struct {
	BaseEvent
}

func NewUserEmailVerifiedEvent(userID string) *UserEmailVerifiedEvent {
	return &UserEmailVerifiedEvent{
		BaseEvent: BaseEvent{
			eventType:   "user.email.verified",
			aggregateID: userID,
			occurredAt:  time.Now(),
		},
	}
}

type UserPhoneVerifiedEvent struct {
	BaseEvent
}

func NewUserPhoneVerifiedEvent(userID string) *UserPhoneVerifiedEvent {
	return &UserPhoneVerifiedEvent{
		BaseEvent: BaseEvent{
			eventType:   "user.phone.verified",
			aggregateID: userID,
			occurredAt:  time.Now(),
		},
	}
}

type UserPhoneAddedEvent struct {
	BaseEvent
	Phone string
}

func NewUserPhoneAddedEvent(userID, phone string) *UserPhoneAddedEvent {
	return &UserPhoneAddedEvent{
		BaseEvent: BaseEvent{
			eventType:   "user.phone.added",
			aggregateID: userID,
			occurredAt:  time.Now(),
		},
		Phone: phone,
	}
}

type UserRoleAddedEvent struct {
	BaseEvent
	Role string
}

func NewUserRoleAddedEvent(userID, role string) *UserRoleAddedEvent {
	return &UserRoleAddedEvent{
		BaseEvent: BaseEvent{
			eventType:   "user.role.added",
			aggregateID: userID,
			occurredAt:  time.Now(),
		},
		Role: role,
	}
}

type UserBlockedEvent struct {
	BaseEvent
	Reason string
}

func NewUserBlockedEvent(userID, reason string) *UserBlockedEvent {
	return &UserBlockedEvent{
		BaseEvent: BaseEvent{
			eventType:   "user.blocked",
			aggregateID: userID,
			occurredAt:  time.Now(),
		},
		Reason: reason,
	}
}

type UserUnblockedEvent struct {
	BaseEvent
}

func NewUserUnblockedEvent(userID string) *UserUnblockedEvent {
	return &UserUnblockedEvent{
		BaseEvent: BaseEvent{
			eventType:   "user.unblocked",
			aggregateID: userID,
			occurredAt:  time.Now(),
		},
	}
}

// Wallet Events

type WalletCreditedEvent struct {
	BaseEvent
	Amount     *valueobjects.Money
	OldBalance *valueobjects.Money
	NewBalance *valueobjects.Money
}

func NewWalletCreditedEvent(
	userID string,
	amount, oldBalance, newBalance *valueobjects.Money,
) *WalletCreditedEvent {
	return &WalletCreditedEvent{
		BaseEvent: BaseEvent{
			eventType:   "wallet.credited",
			aggregateID: userID,
			occurredAt:  time.Now(),
		},
		Amount:     amount,
		OldBalance: oldBalance,
		NewBalance: newBalance,
	}
}

type WalletDebitedEvent struct {
	BaseEvent
	Amount     *valueobjects.Money
	OldBalance *valueobjects.Money
	NewBalance *valueobjects.Money
}

func NewWalletDebitedEvent(
	userID string,
	amount, oldBalance, newBalance *valueobjects.Money,
) *WalletDebitedEvent {
	return &WalletDebitedEvent{
		BaseEvent: BaseEvent{
			eventType:   "wallet.debited",
			aggregateID: userID,
			occurredAt:  time.Now(),
		},
		Amount:     amount,
		OldBalance: oldBalance,
		NewBalance: newBalance,
	}
}

// Captain Events

type CaptainKYCSubmittedEvent struct {
	BaseEvent
}

func NewCaptainKYCSubmittedEvent(captainID string) *CaptainKYCSubmittedEvent {
	return &CaptainKYCSubmittedEvent{
		BaseEvent: BaseEvent{
			eventType:   "captain.kyc.submitted",
			aggregateID: captainID,
			occurredAt:  time.Now(),
		},
	}
}

type CaptainKYCApprovedEvent struct {
	BaseEvent
}

func NewCaptainKYCApprovedEvent(captainID string) *CaptainKYCApprovedEvent {
	return &CaptainKYCApprovedEvent{
		BaseEvent: BaseEvent{
			eventType:   "captain.kyc.approved",
			aggregateID: captainID,
			occurredAt:  time.Now(),
		},
	}
}

type CaptainKYCRejectedEvent struct {
	BaseEvent
	Reason string
}

func NewCaptainKYCRejectedEvent(captainID, reason string) *CaptainKYCRejectedEvent {
	return &CaptainKYCRejectedEvent{
		BaseEvent: BaseEvent{
			eventType:   "captain.kyc.rejected",
			aggregateID: captainID,
			occurredAt:  time.Now(),
		},
		Reason: reason,
	}
}

type CaptainWentOnlineEvent struct {
	BaseEvent
	Location *valueobjects.Location
}

func NewCaptainWentOnlineEvent(captainID string, location *valueobjects.Location) *CaptainWentOnlineEvent {
	return &CaptainWentOnlineEvent{
		BaseEvent: BaseEvent{
			eventType:   "captain.went.online",
			aggregateID: captainID,
			occurredAt:  time.Now(),
		},
		Location: location,
	}
}

type CaptainWentOfflineEvent struct {
	BaseEvent
}

func NewCaptainWentOfflineEvent(captainID string) *CaptainWentOfflineEvent {
	return &CaptainWentOfflineEvent{
		BaseEvent: BaseEvent{
			eventType:   "captain.went.offline",
			aggregateID: captainID,
			occurredAt:  time.Now(),
		},
	}
}
