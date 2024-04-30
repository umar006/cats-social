package models

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type TokenService interface {
	GetJWTSecret() string
	GetBcryptSalt() string
}

type tokenService struct {
	JWTSecret  string
	BcryptSalt string
}

func NewTokenService() *tokenService {
	return &tokenService{
		JWTSecret:  os.Getenv("JWT_SECRET"),
		BcryptSalt: os.Getenv("BCRYPT_SALT"),
	}
}

func (t *tokenService) GetJWTSecret() string {

	return t.JWTSecret
}

func (t *tokenService) GetBcryptSalt() string {

	return t.BcryptSalt
}

type User struct {
	Id           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	Name         string    `json:"name" db:"name"`
	Password     string    `json:"password" db:"password"`
	TokenService TokenService
}

func NewUser() *User {
	id := uuid.New()
	token := NewTokenService()

	return &User{
		Id:           id,
		TokenService: token,
	}
}

var invalidTokenErr = NewUnauthenticatedError("invalid token")

func (u *User) HashPassword() MessageErr {
	salt, err := strconv.Atoi(u.TokenService.GetBcryptSalt())

	if err != nil {
		return NewInternalServerError("SOMETHING WENT WRONG")
	}

	bs, err := bcrypt.GenerateFromPassword([]byte(u.Password), salt)

	if err != nil {
		return NewInternalServerError("SOMETHING WENT WRONG")
	}

	u.Password = string(bs)

	return nil
}
func (u *User) ComparePassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

func (u *User) GenerateToken() (string, error) {
	claims := jwt.MapClaims{
		"id":   u.Id,
		"name": u.Name,
		"exp":  time.Now().Add(time.Hour * 8).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(u.TokenService.GetJWTSecret()))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (u *User) parseToken(tokenString string) (*jwt.Token, MessageErr) {

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, invalidTokenErr
		}

		secretKey := u.TokenService.GetJWTSecret()

		return []byte(secretKey), nil
	})

	if err != nil {

		return nil, invalidTokenErr
	}

	return token, nil
}

func (u *User) bindTokenToUserEntity(claim jwt.MapClaims) MessageErr {

	if uuid, ok := claim["id"].(uuid.UUID); !ok {
		return invalidTokenErr
	} else {
		u.Id = uuid
	}

	if name, ok := claim["name"].(string); !ok {
		return invalidTokenErr
	} else {
		u.Name = name
	}

	return nil
}

func (u *User) ValidateToken(bearerToken string) MessageErr {
	isBearer := strings.HasPrefix(bearerToken, "Bearer")

	if !isBearer {
		return NewUnauthenticatedError("token should be Bearer")
	}

	splitToken := strings.Fields(bearerToken)

	if len(splitToken) != 2 {
		return NewUnauthenticatedError("invalid token")
	}

	tokenString := splitToken[1]

	token, err := u.parseToken(tokenString)

	if err != nil {
		return err
	}

	var mapClaims jwt.MapClaims

	if claims, ok := token.Claims.(jwt.MapClaims); !ok || !token.Valid {
		return invalidTokenErr
	} else {
		mapClaims = claims
	}

	err = u.bindTokenToUserEntity(mapClaims)

	return err
}