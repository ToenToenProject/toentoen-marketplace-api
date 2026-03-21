package services

import (
	"errors"
	"toentoen-backend/internal/models"
	"toentoen-backend/internal/repositories"
)

type AssignmentService interface {
	AssignBusinessAdminToBusiness(ownerID int64, input models.AssignBusinessAdminInput)	(*models.BusinessAdminAssignment, error)
	GetMyAdmins(ownerID int64) ([]models.BusinessAdmin, error)
	UpdateAssignment(ownerID, adminID, businessID int64, input models.UpdateAssignmentInput) (*models.BusinessAdminAssignment, error)
	DeleteBusinessAdmin(ownerID, adminID int64) error
}

type assignmentService struct {
	repo repositories.AssignmentRepository
}

func NewAssignmentService(repo repositories.AssignmentRepository) AssignmentService {
	return &assignmentService{repo}
}

func (s *assignmentService) AssignBusinessAdminToBusiness(ownerID int64, input models.AssignBusinessAdminInput) (*models.BusinessAdminAssignment, error) {
	if !s.repo.CheckBusinessOwnership(ownerID, input.BusinessID){
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
		BusinessID: input.BusinessID,
		Role: input.Role,
		ShiftStart: input.ShiftStart,
		ShiftEnd: input.ShiftEnd,
		IsActive: true,
	}

	if err := s.repo.AssignBusinessAdmin(&assignment); err != nil {
		return nil, err
	}

	assignmentDetails, _ := s.repo.GetAssignmentWithDetails(input.BusinessAdminID, input.BusinessID)

	return assignmentDetails, nil
}

func (s *assignmentService) GetMyAdmins(ownerID int64) ([]models.BusinessAdmin, error) {
	return s.repo.GetAllAdminsByOwner(ownerID)
}

func (s *assignmentService) UpdateAssignment(ownerID, adminID, businessID int64, input models.UpdateAssignmentInput) (*models.BusinessAdminAssignment, error) {
	if !s.repo.CheckBusinessAdminOwnership(ownerID, adminID) {
		return nil, errors.New("business admin not found or not yours")
	}

	if !s.repo.CheckBusinessOwnership(ownerID, businessID) {
		return nil, errors.New("business not found or not yours")
	}

	assignment, err := s.repo.GetAssignment(adminID, businessID)
	if err != nil {
		return nil, errors.New("this business admin has not been placed in this business")
	}

	if input.Role != nil {assignment.Role = *input.Role}
	if input.ShiftStart != nil {assignment.ShiftStart = input.ShiftStart}
	if input.ShiftEnd != nil {assignment.ShiftEnd = input.ShiftEnd}
	if input.IsActive != nil {assignment.IsActive = *input.IsActive}

	if err := s.repo.AssignBusinessAdmin(assignment); err != nil {
		return nil, errors.New("failed to save placement changes")
	}

	return assignment, nil
}

func (s *assignmentService) DeleteBusinessAdmin(ownerID, adminID int64) error {
	if !s.repo.CheckBusinessAdminOwnership(ownerID, adminID) {
		return errors.New("business admin not found or not yours")
	}

	return s.repo.DeleteBusinessAdminAccount(adminID)
}