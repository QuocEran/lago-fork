package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/getlago/lago/api-go/internal/services/users"
)

type loginRequest struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	Token string   `json:"token"`
	User  userView `json:"user"`
}

type userView struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// Login handles POST /users/login.
func Login(svc users.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorBody("validation_error", err.Error()))
			return
		}

		result, err := svc.Login(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			switch {
			case errors.Is(err, users.ErrLoginMethodNotAuthorized):
				c.JSON(http.StatusUnprocessableEntity, errorBody("login_method_not_authorized", ""))
			case errors.Is(err, users.ErrInvalidCredentials):
				c.JSON(http.StatusUnauthorized, errorBody("incorrect_login_or_password", ""))
			default:
				c.JSON(http.StatusInternalServerError, errorBody("internal_error", ""))
			}
			return
		}

		c.JSON(http.StatusOK, loginResponse{
			Token: result.Token,
			User:  userView{ID: result.User.ID, Email: result.User.Email},
		})
	}
}

func errorBody(code, detail string) gin.H {
	body := gin.H{"status": "error", "error_code": code, "error_details": gin.H{}}
	if detail != "" {
		body["error_details"] = gin.H{"message": detail}
	}
	return body
}
