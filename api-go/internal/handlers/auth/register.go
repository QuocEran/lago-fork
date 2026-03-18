package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/getlago/lago/api-go/internal/services/users"
)

type registerRequest struct {
	Email            string `json:"email"             binding:"required"`
	Password         string `json:"password"          binding:"required"`
	OrganizationName string `json:"organization_name" binding:"required"`
}

type registerResponse struct {
	Token        string         `json:"token"`
	User         userView       `json:"user"`
	Organization orgView        `json:"organization"`
	Membership   membershipView `json:"membership"`
}

type orgView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type membershipView struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

// Register handles POST /users/register.
func Register(svc users.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req registerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorBody("validation_error", err.Error()))
			return
		}

		result, err := svc.Register(c.Request.Context(), req.Email, req.Password, req.OrganizationName)
		if err != nil {
			switch {
			case errors.Is(err, users.ErrSignupDisabled):
				c.JSON(http.StatusForbidden, errorBody("signup_disabled", ""))
			case errors.Is(err, users.ErrUserAlreadyExists):
				c.JSON(http.StatusUnprocessableEntity, errorBody("user_already_exists", ""))
			default:
				c.JSON(http.StatusInternalServerError, errorBody("internal_error", ""))
			}
			return
		}

		c.JSON(http.StatusCreated, registerResponse{
			Token:        result.Token,
			User:         userView{ID: result.User.ID, Email: result.User.Email},
			Organization: orgView{ID: result.Organization.ID, Name: result.Organization.Name},
			Membership:   membershipView{ID: result.Membership.ID, Role: "admin"},
		})
	}
}
