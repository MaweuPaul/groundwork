package service

import (
	"errors"
	"strings"

	"groundwork/internal/models"
	"groundwork/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrMissingFields      = errors.New("missing required fields")
	ErrEmailTaken         = errors.New("email already registered")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidRole        = errors.New("role must be auditor or asset_holder")
)

type AuthService interface {
	Register(email, password, firstName, lastName, username string, role models.UserRole) (*models.User, *TokenPair, error)
	Refresh(rawRefreshToken string) (*TokenPair, error)
	Login(email, password string) (*models.User, *TokenPair, error)
}

type authService struct {
	users  repository.UserRepository
	tokens TokenService
}

func NewAuthService(users repository.UserRepository, tokens TokenService) AuthService {
	return &authService{users: users, tokens: tokens}
}

// Register creates a new user and logs them in straight away.
func (s *authService) Register(email, password, firstName, lastName, username string, role models.UserRole) (*models.User, *TokenPair, error) {
	user, err := s.createUser(email, password, firstName, lastName, username, role)
	if err != nil {
		return nil, nil, err
	}

	pair, err := s.tokens.Issue(user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, pair, nil
}

// Login returns ErrInvalidCredentials for both an unknown email and a wrong
// password, so callers can't use it to find out which emails are registered.
func (s *authService) Login(email, password string) (*models.User, *TokenPair, error) {
	user, err := s.users.GetByEmail(normalizeEmail(email))
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	pair, err := s.tokens.Issue(user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, pair, nil
}

func (s *authService) Refresh(rawRefreshToken string) (*TokenPair, error) {
	return s.tokens.Rotate(rawRefreshToken)
}

func (s *authService) createUser(email, password, firstName, lastName, username string, role models.UserRole) (*models.User, error) {
	email = normalizeEmail(email)

	// check if all required fields are provided
	if email == "" || password == "" || firstName == "" || lastName == "" || username == "" {
		return nil, ErrMissingFields
	}

	if !role.Valid() {
		return nil, ErrInvalidRole
	}

	// check if user exists
	existing, err := s.users.GetByEmail(email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	existing, err = s.users.GetByUsername(username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUsernameTaken
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:     email,
		Password:  hash,
		FirstName: firstName,
		LastName:  lastName,
		Username:  username,
		Role:      role,
	}
	if err := s.users.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// normalizeEmail makes "Juma@Mail.com " and "juma@mail.com" the same account.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
