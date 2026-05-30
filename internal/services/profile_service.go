package services

import (
	"errors"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/repositories"

	"gorm.io/gorm"
)

type ProfileService interface {
	UpdateCustomerProfile(accountID string, input models.UpdateProfileCustomerInput) (*models.Account, error)
	UpdateOwnerProfile(accountID string, input models.UpdateProfileOwnerInput) (*models.Account, error)
}

type profileService struct {
	repo repositories.ProfileRepository
}

func NewProfileService(repo repositories.ProfileRepository) ProfileService {
	return &profileService{repo}
}

func isAccountDataComplete(account *models.Account) string {
	if account == nil {
		return "inactive"
	}

	switch account.Role {
	case "owner":
		if account.Email != nil &&
			account.PhoneNumber != nil &&
			account.Fullname != "" &&
			account.Username != "" &&
			account.PasswordHash != nil &&
			account.Gender != nil &&
			account.Owner.KtpNumber != nil &&
			*account.Owner.KtpNumber != "" &&
			account.Owner.KtpPhotoUrl != nil &&
			*account.Owner.KtpPhotoUrl != "" {
			return "active"
		}
		return "inactive"

	case "business_admin":
		if account.Gender != nil && account.Email != nil {
			return "active"
		}
		return "inactive"

	default:
		return "inactive"
	}
}

func (s *profileService) UpdateCustomerProfile(accountID string, input models.UpdateProfileCustomerInput) (*models.Account, error) {
	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("account not found")
		}
		return nil, err
	}

	if input.Fullname != nil {
		account.Fullname = *input.Fullname
	}
	if input.Username != nil {
		account.Username = *input.Username
	}
	if input.Email != nil {
		account.Email = input.Email
	}
	if input.PhoneNumber != nil {
		account.PhoneNumber = input.PhoneNumber
	}
	if input.Gender != nil {
		account.Gender = input.Gender
	}

	customer, err := s.repo.GetCustomerByAccountID(accountID)
	if err != nil {
		return nil, errors.New("user specific profile data not found")
	}

	if input.Bio != nil {
		customer.Bio = input.Bio
	}
	if input.ProfilePictureUrl != nil {
		customer.ProfilePictureUrl = input.ProfilePictureUrl
	}
	if input.BirthDate != nil {
		if *input.BirthDate == "" {
			customer.BirthDate = nil
		} else {
			customer.BirthDate = input.BirthDate
		}
	}

	account.Customer = customer
	account.AccountStatus = isAccountDataComplete(account)

	if err := s.repo.SaveCustomerProfile(account, customer); err != nil {
		return nil, errors.New("failed to save user profile data")
	}

	return account, nil
}

func (s *profileService) UpdateOwnerProfile(accountID string, input models.UpdateProfileOwnerInput) (*models.Account, error) {
	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("account not found")
		}
		return nil, err
	}

	if input.Fullname != nil {
		account.Fullname = *input.Fullname
	}
	if input.Username != nil {
		account.Username = *input.Username
	}
	if input.Email != nil {
		account.Email = input.Email
	}
	if input.PhoneNumber != nil {
		account.PhoneNumber = input.PhoneNumber
	}
	if input.Gender != nil {
		account.Gender = input.Gender
	}

	owner, err := s.repo.GetOwnerByAccountID(accountID)
	if err != nil {
		return nil, errors.New("user specific profile data not found")
	}

	if input.KtpNumber != nil {
		if *input.KtpNumber == "" {
			owner.KtpNumber = nil
		} else {
			owner.KtpNumber = input.KtpNumber
		}
	}
	if input.KtpPhotoUrl != nil {
		if *input.KtpPhotoUrl == "" {
			owner.KtpPhotoUrl = nil
		} else {
			owner.KtpPhotoUrl = input.KtpPhotoUrl
		}
	}

	if input.ReferralCode != nil {
		if *input.ReferralCode == "" {
			owner.ReferralCode = nil
		} else {
			owner.ReferralCode = input.ReferralCode
		}
	}

	account.Owner = owner
	account.AccountStatus = isAccountDataComplete(account)

	if err := s.repo.SaveOwnerProfile(account, owner); err != nil {
		return nil, errors.New("failed to save owner profile data")
	}
	return account, nil
}
