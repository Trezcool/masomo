package echoapi

import (
	"sort"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"

	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
)

var (
	// appJWTConfig is the default JWT auth middleware config.
	appJWTConfig = middleware.JWTConfig{
		SigningKey:    []byte(core.Conf.SecretKey),
		SigningMethod: middleware.AlgorithmHS256,
		ContextKey:    "userToken",
		Claims:        new(Claims),
	}
	contextUserKey = "user"
)

// Claims represents the authorization claims transmitted via a JWT.
type Claims struct {
	jwt.StandardClaims
	OrigIssuedAt int64    `json:"oriat,omitempty"`
	Username     string   `json:"username,omitempty"`
	Email        string   `json:"email,omitempty"`
	IsStudent    bool     `json:"is_student,omitempty"` // -> STUDENT PORTAL
	IsTeacher    bool     `json:"is_teacher,omitempty"` // -> TEACHER PORTAL
	IsAdmin      bool     `json:"is_admin,omitempty"`   // -> ADMIN PORTAL
	Roles        []string `json:"roles,omitempty"`
}

func GetUserClaims(usr user.User, origIat ...int64) *Claims {
	now := time.Now()
	nownix := now.Unix()

	var oriat int64
	if len(origIat) > 0 {
		oriat = origIat[0]
	} else {
		oriat = nownix
	}

	claims := &Claims{
		StandardClaims: jwt.StandardClaims{
			Issuer:    core.Conf.AppName,
			Subject:   usr.ID,
			Audience:  "Academia",
			ExpiresAt: now.Add(core.Conf.Server.JWTExpirationDelta).Unix(),
			IssuedAt:  nownix,
		},
		OrigIssuedAt: oriat,
		Username:     usr.Username,
		Email:        usr.Email,
		IsStudent:    usr.IsStudent(),
		IsTeacher:    usr.IsTeacher(),
		IsAdmin:      usr.IsAdmin(),
		Roles:        usr.Roles,
	}
	return claims
}

func authenticate(uname, pwd string, svc user.Service) (*Claims, error) {
	usr, err := svc.GetByUsernameOrEmail(uname)
	if err != nil {
		if err == user.ErrNotFound {
			return nil, errAuthenticationFailed
		}
		return nil, errors.Wrap(err, "finding user by username or email")
	}
	if err = usr.CheckPassword(pwd); err != nil {
		return nil, errAuthenticationFailed
	}
	if usr.IsActive != nil && !*usr.IsActive {
		return nil, errAccountDeactivated
	}
	usr, err = svc.SetLastLogin(usr)
	if err != nil {
		return nil, errors.Wrap(err, "setting lastLogin")
	}
	return GetUserClaims(usr), nil
}

// GenerateToken generates a signed JWT token string representing the user Claims.
func GenerateToken(claims *Claims) (string, error) {
	method := jwt.GetSigningMethod(appJWTConfig.SigningMethod)
	token := jwt.NewWithClaims(method, claims)

	ss, err := token.SignedString(appJWTConfig.SigningKey)
	if err != nil {
		return "", errors.New("signing token")
	}
	return ss, nil
}

func getContextClaims(ctx echo.Context) (Claims, error) {
	if token, ok := ctx.Get(appJWTConfig.ContextKey).(*jwt.Token); ok {
		if claims, ok := token.Claims.(*Claims); ok {
			return *claims, nil
		}
	}
	return Claims{}, errUnauthorized
}

func getContextUser(ctx echo.Context, svc user.Service, clms ...Claims) (user.User, error) {
	if usr, ok := ctx.Get(contextUserKey).(user.User); ok {
		return usr, nil
	}

	var claims Claims
	var err error
	if len(clms) > 0 {
		claims = clms[0]
	} else {
		claims, err = getContextClaims(ctx)
		if err != nil {
			return user.User{}, errors.Wrap(err, "getting context claims")
		}
	}

	usr, err := svc.GetByID(claims.Subject)
	if err != nil {
		return user.User{}, errors.Wrap(err, "finding user by ID")
	}
	ctx.Set(contextUserKey, usr)
	return usr, nil
}

func contextHasAnyRole(ctx echo.Context, roles []string) bool {
	if len(roles) == 0 {
		return true
	}
	if claims, err := getContextClaims(ctx); err == nil {
		sort.Strings(claims.Roles)
		for _, role := range roles {
			if i := sort.SearchStrings(claims.Roles, role); i < len(claims.Roles) {
				if match := claims.Roles[i]; role == match {
					return true
				}
			}
		}
	}
	return false
}

func refreshToken(ctx echo.Context, svc user.Service) (string, error) {
	claims, err := getContextClaims(ctx)
	if err != nil {
		return "", errors.Wrap(err, "getting context claims")
	}

	usr, err := getContextUser(ctx, svc, claims)
	if err != nil {
		return "", errors.Wrap(err, "getting context user")
	}

	// check if user is still active
	if usr.IsActive != nil && !*usr.IsActive {
		return "", errAccountDeactivated
	}

	// check if refresh has not expired
	expTime := time.Unix(claims.OrigIssuedAt, 0).Add(core.Conf.Server.JWTRefreshExpirationDelta)
	if time.Now().After(expTime) {
		return "", errRefreshExpired
	}

	newClaims := GetUserClaims(usr, claims.OrigIssuedAt)
	token, err := GenerateToken(newClaims)
	return token, errors.Wrap(err, "generating token")
}
