package users

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/models"
)

const (
	jwtTTL         = 3 * time.Hour
	bcryptCost     = 12
	loginMethod    = "email_password"
	hmacKeyByteLen = 32
)

var (
	ErrInvalidCredentials       = errors.New("incorrect_login_or_password")
	ErrLoginMethodNotAuthorized = errors.New("login_method_not_authorized")
	ErrSignupDisabled           = errors.New("signup_disabled")
	ErrUserAlreadyExists        = errors.New("user_already_exists")
)

type LoginResult struct {
	Token string
	User  models.User
}

type RegisterResult struct {
	Token        string
	User         models.User
	Organization models.Organization
	Membership   models.Membership
}

type AuthService interface {
	Login(ctx context.Context, email, password string) (*LoginResult, error)
	Register(ctx context.Context, email, password, orgName string) (*RegisterResult, error)
}

type authService struct {
	db        *gorm.DB
	jwtSecret string
}

func NewAuthService(db *gorm.DB, jwtSecret string) AuthService {
	return &authService{db: db, jwtSecret: jwtSecret}
}

func (s *authService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	if strings.ContainsRune(email, 0) || strings.ContainsRune(password, 0) {
		return nil, ErrInvalidCredentials
	}

	var user models.User
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordDigest), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	var memberships []models.Membership
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", user.ID, models.MembershipStatusActive).
		Preload("Organization").
		Find(&memberships).Error; err != nil {
		return nil, ErrInvalidCredentials
	}

	if len(memberships) == 0 {
		return nil, ErrInvalidCredentials
	}

	authorized := false
	for _, membership := range memberships {
		for _, method := range membership.Organization.AuthenticationMethods {
			if method == loginMethod {
				authorized = true
				break
			}
		}
		if authorized {
			break
		}
	}

	if !authorized {
		return nil, ErrLoginMethodNotAuthorized
	}

	token, err := s.signJWT(user.ID)
	if err != nil {
		return nil, err
	}

	return &LoginResult{Token: token, User: user}, nil
}

func (s *authService) Register(ctx context.Context, email, password, orgName string) (*RegisterResult, error) {
	if os.Getenv("LAGO_DISABLE_SIGNUP") == "true" {
		return nil, ErrSignupDisabled
	}

	var existing models.User
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&existing).Error; err == nil {
		return nil, ErrUserAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, err
	}

	hmacKey, err := generateHmacKey()
	if err != nil {
		return nil, err
	}

	var result RegisterResult
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user := models.User{
			Email:          email,
			PasswordDigest: string(hash),
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		org := models.Organization{
			Name:              orgName,
			HmacKey:           hmacKey,
			DocumentNumbering: 0,
		}
		if err := tx.Create(&org).Error; err != nil {
			return err
		}

		membership := models.Membership{
			UserID:         user.ID,
			OrganizationID: org.ID,
			Status:         models.MembershipStatusActive,
		}
		if err := tx.Create(&membership).Error; err != nil {
			return err
		}

		var adminRole models.Role
		err := tx.Where("admin = ? AND (organization_id IS NULL OR organization_id = ?)", true, org.ID).
			First(&adminRole).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			adminRole = models.Role{
				OrganizationID: &org.ID,
				Code:           "admin",
				Name:           "Admin",
				Admin:          true,
			}
			if err := tx.Create(&adminRole).Error; err != nil {
				return err
			}
		}

		membershipRole := models.MembershipRole{
			MembershipID:   membership.ID,
			RoleID:         adminRole.ID,
			OrganizationID: org.ID,
		}
		if err := tx.Create(&membershipRole).Error; err != nil {
			return err
		}

		result.User = user
		result.Organization = org
		result.Membership = membership
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	token, err := s.signJWT(result.User.ID)
	if err != nil {
		return nil, err
	}
	result.Token = token

	return &result, nil
}

func (s *authService) signJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub":          userID,
		"exp":          time.Now().Add(jwtTTL).Unix(),
		"login_method": loginMethod,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func generateHmacKey() (string, error) {
	b := make([]byte, hmacKeyByteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
