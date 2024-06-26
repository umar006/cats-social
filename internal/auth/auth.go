package auth

import (
	"cats-social/internal/domain"
	"cats-social/internal/repository"
	"database/sql"

	"github.com/gin-gonic/gin"
)

type AuthService interface {
	Authentication(db *sql.DB) gin.HandlerFunc
}

type authServiceImpl struct {
	ur repository.UserRepository
}

func NewAuth(ur repository.UserRepository) AuthService {
	return &authServiceImpl{
		ur: ur,
	}
}

func (a *authServiceImpl) Authentication(db *sql.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		invalidTokenErr := domain.NewUnauthenticatedError("invalid token")
		bearerToken := ctx.GetHeader("Authorization")
		user := domain.NewUser()

		err := user.ValidateToken(bearerToken)
		if err != nil {
			ctx.AbortWithStatusJSON(invalidTokenErr.Status(), invalidTokenErr)
			return
		}

		_, err = a.ur.GetByEmail(db, user.Email)
		if err != nil {
			ctx.AbortWithStatusJSON(invalidTokenErr.Status(), invalidTokenErr)
			return
		}

		ctx.Set("userData", user)
		ctx.Next()
	}
}
