package echoapi

import (
	"sort"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/trezcool/masomo/backend/core/user"
)

var (
	applicationName        string
	expirationDelta        time.Duration
	refreshExpirationDelta time.Duration

	// appJWTConfig is the default JWT auth middleware config.
	appJWTConfig   middleware.JWTConfig
	contextKey     = "userToken"
	contextUserKey = "user"
)

func configureAuth(appName string, secretKey []byte, expDelta, refreshExpDelta time.Duration) echo.MiddlewareFunc {
	applicationName = appName
	expirationDelta = expDelta
	refreshExpirationDelta = refreshExpDelta

	appJWTConfig = middleware.JWTConfig{
		SigningKey:    secretKey,
		SigningMethod: middleware.AlgorithmHS256,
		ContextKey:    contextKey,
		Claims:        new(Claims),
	}
	return middleware.JWTWithConfig(appJWTConfig)
}

// Claims represents the authorization claims transmitted via a JWT.
type Claims struct {
	jwt.StandardClaims
	OriginalIssuedAt int64    `json:"oriat,omitempty"`
	IsStudent        bool     `json:"is_student,omitempty"` // -> STUDENT PORTAL
	IsTeacher        bool     `json:"is_teacher,omitempty"` // -> TEACHER PORTAL
	IsAdmin          bool     `json:"is_admin,omitempty"`   // -> ADMIN PORTAL
	Roles            []string `json:"roles,omitempty"`
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
			Issuer:    applicationName,
			Subject:   strconv.Itoa(usr.ID),
			Audience:  "Academia",
			ExpiresAt: now.Add(expirationDelta).Unix(),
			IssuedAt:  nownix,
		},
		OriginalIssuedAt: oriat,
		IsStudent:        usr.IsStudent(),
		IsTeacher:        usr.IsTeacher(),
		IsAdmin:          usr.IsAdmin(),
		Roles:            usr.Roles,
	}
	return claims
}

func authenticate(uname, pwd string, svc user.Service) (*Claims, error) {
	if usr, err := svc.GetByUsernameOrEmail(uname); err == nil {
		if err := usr.CheckPassword(pwd); err == nil {
			if !usr.IsActive {
				return nil, errAccountDeactivated
			}
			usr, err := svc.SetLastLogin(usr)
			if err != nil {
				return nil, err
			}
			return GetUserClaims(usr), nil
		}
	}
	return nil, errAuthenticationFailed
}

// GenerateToken generates a signed JWT token string representing the user Claims.
func GenerateToken(claims *Claims) (string, error) {
	method := jwt.GetSigningMethod(appJWTConfig.SigningMethod)
	token := jwt.NewWithClaims(method, claims)

	ss, err := token.SignedString(appJWTConfig.SigningKey)
	if err != nil {
		return "", errTokenSigningFailed // todo: wrap err
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
			return user.User{}, err
		}
	}

	uid, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return user.User{}, err
	}

	usr, err := svc.GetByID(uid)
	if err != nil {
		return user.User{}, err
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
		return "", err
	}

	usr, err := getContextUser(ctx, svc, claims)
	if err != nil {
		return "", err
	}

	// check if user is still active
	if !usr.IsActive {
		return "", errAccountDeactivated
	}

	// check if refresh has not expired
	expTime := time.Unix(claims.OriginalIssuedAt, 0).Add(refreshExpirationDelta)
	if time.Now().After(expTime) {
		return "", errRefreshExpired
	}

	newClaims := GetUserClaims(usr, claims.OriginalIssuedAt)
	return GenerateToken(newClaims)
}
