package helpers

import (
	"sort"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/trezcool/masomo/backend/apps/user"
)

var (
	// settings TODO: load from config
	appName         = "Masomo"
	secretKey       = []byte("secret")
	expirationDelta = time.Hour

	// AppJWTConfig is the default JWT auth middleware config.
	AppJWTConfig = middleware.JWTConfig{
		SigningKey:    secretKey,
		SigningMethod: middleware.AlgorithmHS256,
		ContextKey:    "userToken",
		Claims:        new(Claims),
	}
	contextUserKey = "user"
)

// Claims represents the authorization claims transmitted via a JWT.
type Claims struct {
	jwt.StandardClaims
	IsStudent bool     `json:"is_student"` // -> STUDENT PORTAL
	IsTeacher bool     `json:"is_teacher"` // -> TEACHER PORTAL
	IsAdmin   bool     `json:"is_admin"`   // -> ADMIN PORTAL
	Roles     []string `json:"roles"`
}

func Authenticate(uname, pwd string, service *user.Service) (*Claims, error) {
	if usr, err := service.GetByUsernameOrEmail(uname); err == nil {
		if err := usr.CheckPassword(pwd); err == nil {
			if !usr.IsActive {
				return nil, accountDeactivatedErr
			}
			now := time.Now()
			claims := &Claims{
				StandardClaims: jwt.StandardClaims{
					Issuer:    appName,
					Subject:   strconv.Itoa(usr.ID),
					Audience:  "Academia",
					ExpiresAt: now.Add(expirationDelta).Unix(),
					IssuedAt:  now.Unix(),
				},
				IsStudent: usr.IsStudent(),
				IsTeacher: usr.IsTeacher(),
				IsAdmin:   usr.IsAdmin(),
				Roles:     usr.Roles,
			}
			return claims, nil
		}
	}
	return nil, authenticationFailedErr
}

// GenerateToken generates a signed JWT token string representing the user Claims.
func GenerateToken(claims *Claims) (string, error) {
	method := jwt.GetSigningMethod(AppJWTConfig.SigningMethod)
	token := jwt.NewWithClaims(method, claims)

	ss, err := token.SignedString(AppJWTConfig.SigningKey)
	if err != nil {
		return "", tokenSigningError
	}
	return ss, nil
}

func getContextClaims(c echo.Context) (*Claims, error) {
	if token, ok := c.Get(AppJWTConfig.ContextKey).(*jwt.Token); ok {
		if claims, ok := token.Claims.(*Claims); ok {
			return claims, nil
		}
	}
	return nil, unauthorizedErr
}

func GetContextUser(c echo.Context, service *user.Service) (user.User, error) {
	if usr, ok := c.Get(contextUserKey).(user.User); ok {
		return usr, nil
	}

	claims, err := getContextClaims(c)
	if err != nil {
		return user.User{}, err
	}

	uid, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return user.User{}, err
	}

	usr, err := service.GetByID(uid)
	c.Set(contextUserKey, usr)
	return usr, err
}

func contextHasAnyRole(c echo.Context, roles []string) bool {
	if len(roles) == 0 {
		return true
	}
	if claims, err := getContextClaims(c); err == nil {
		sort.Strings(claims.Roles)
		for _, role := range roles {
			if idx := sort.SearchStrings(claims.Roles, role); idx < len(claims.Roles) {
				if match := claims.Roles[idx]; role == match {
					return true
				}
			}
		}
	}
	return false
}

// TODO: token refresh !!!
