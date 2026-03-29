package services

import (
	"errors"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/repositories"

	"golang.org/x/crypto/bcrypt"
)

type AssignmentService interface {
	AssignBusinessAdminToBusiness(ownerID string, input models.AssignBusinessAdminInput) (*models.BusinessAdminAssignment, error)
	GetMyAdmins(ownerID string) ([]models.BusinessAdmin, error)
	UpdateAssignment(requesterID, requesterRole, adminID, businessID string, input models.UpdateAssignmentInput) (*models.BusinessAdminAssignment, error)
	DeleteBusinessAdmin(ownerID, adminID string) error
}

type assignmentService struct {
	repo repositories.AssignmentRepository
}

func NewAssignmentService(repo repositories.AssignmentRepository) AssignmentService {
	return &assignmentService{repo}
}

func (s *assignmentService) AssignBusinessAdminToBusiness(ownerID string, input models.AssignBusinessAdminInput) (*models.BusinessAdminAssignment, error) {
	if !s.repo.CheckBusinessOwnership(ownerID, input.BusinessID) {
		return nil, errors.New("Your busines not found or this business is not yours")
	}

	if !s.repo.CheckBusinessAdminOwnership(ownerID, input.BusinessAdminID) {
		return nil, errors.New("Your business admin is not found or this business admin is not yours")
	}

	existing, err := s.repo.GetAssignment(input.BusinessAdminID, input.BusinessID)
	if err == nil && existing != nil {
		return nil, errors.New("admin has been placed in this business, please use the update feature to change working hours or role")
	}

	assignment := models.BusinessAdminAssignment{
		BusinessAdminID: input.BusinessAdminID,
		BusinessID:      input.BusinessID,
		Role:            input.Role,
		ShiftStart:      input.ShiftStart,
		ShiftEnd:        input.ShiftEnd,
		IsActive:        true,
	}

	if err := s.repo.AssignBusinessAdmin(&assignment); err != nil {
		return nil, err
	}

	assignmentDetails, _ := s.repo.GetAssignmentWithDetails(input.BusinessAdminID, input.BusinessID)

	return assignmentDetails, nil
}

func (s *assignmentService) GetMyAdmins(ownerID string) ([]models.BusinessAdmin, error) {
	return s.repo.GetAllAdminsByOwner(ownerID)
}

func (s *assignmentService) UpdateAssignment(requesterID, requesterRole, adminID, businessID string, input models.UpdateAssignmentInput) (*models.BusinessAdminAssignment, error) {

	if requesterRole == "owner" {
		if !s.repo.CheckBusinessAdminOwnership(requesterID, adminID) {
			return nil, errors.New("business admin not found or not yours")
		}
		if !s.repo.CheckBusinessOwnership(requesterID, businessID) {
			return nil, errors.New("business not found or not yours")
		}

	} else if requesterRole == "business_admin" {
		if requesterID != adminID {
			return nil, errors.New("forbidden: you can only update your own profile")
		}

		input.Role = nil
		input.ShiftStart = nil
		input.ShiftEnd = nil
		input.IsActive = nil

	} else {
		return nil, errors.New("forbidden: your role is not authorized to do this action")
	}

	accountUpdates := make(map[string]interface{})

	if input.Fullname != nil { accountUpdates["fullname"] = *input.Fullname }
	if input.Username != nil { accountUpdates["username"] = *input.Username }
	if input.Email != nil { accountUpdates["email"] = *input.Email }
	if input.PhoneNumber != nil { accountUpdates["phone_number"] = *input.PhoneNumber }
	if input.Gender != nil { accountUpdates["gender"] = *input.Gender }
	if input.Password != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
		if err == nil {
			accountUpdates["password_hash"] = string(hash)
		}
	}

	if len(accountUpdates) > 0 {
		if err := s.repo.UpdateAdminAccount(adminID, accountUpdates); err != nil {
			return nil, errors.New("failed to update admin account details")
		}
	}

	assignment, err := s.repo.GetAssignment(adminID, businessID)
	if err != nil {
		return nil, errors.New("this business admin has not been placed in this business")
	}

	if input.Role != nil { assignment.Role = *input.Role }
	if input.ShiftStart != nil { assignment.ShiftStart = input.ShiftStart }
	if input.ShiftEnd != nil { assignment.ShiftEnd = input.ShiftEnd }
	if input.IsActive != nil { assignment.IsActive = *input.IsActive }

	if err := s.repo.AssignBusinessAdmin(assignment); err != nil {
		return nil, errors.New("failed to save placement changes")
	}

	updatedAssignment, err := s.repo.GetAssignmentWithDetails(adminID, businessID)
	if err == nil {
		return updatedAssignment, nil
	}

	return assignment, nil
}

func (s *assignmentService) DeleteBusinessAdmin(ownerID, adminID string) error {
	if !s.repo.CheckBusinessAdminOwnership(ownerID, adminID) {
		return errors.New("business admin not found or not yours")
	}

	return s.repo.DeleteBusinessAdminAccount(adminID)
}