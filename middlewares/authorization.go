package middlewares

import (
	"errors"
	"github.com/labstack/echo"
	"goappuser/models"
	"log"
	"net/http"
)

var (
	ErrUserNotAuthorized = errors.New("User not authorize")
	ErrUserHasNoRoles    = errors.New("User doesn't have any roles set")
	ErrUserRolesNotMatch = errors.New("User doesn't have corrects roles")
	ErrNoSessionUser     = errors.New("No user session found")
)

func NewAuthorizationMiddleware(roles models.AuthorizationLevel) *AuthorizationMiddlewareHandler {
	return &AuthorizationMiddlewareHandler{roles: roles}
}

type AuthorizationMiddlewareHandler struct {
	roles models.AuthorizationLevel
}

func (a *AuthorizationMiddlewareHandler) Process(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, isOk := c.Get("Session").(models.Sessionizer)
		if isOk == false {
			log.Println("In Authorization Session has not been found")
			return c.JSON(http.StatusUnauthorized, models.RequestError{Title: "Authorization Error", Description: ErrUserNotAuthorized.Error(), Code: 0})
			return ErrUserNotAuthorized
		}
		// log.Println("session found in context ", session)
		usr, isOk := session.GetUser().(models.Authorizer)
		if isOk == false {
			return c.JSON(http.StatusUnauthorized, models.RequestError{Title: "Authorization Error", Description: ErrUserHasNoRoles.Error(), Code: 0})
			return ErrUserHasNoRoles
		}
		log.Println("user, ", usr.(models.User).GetEmail(), "  role", usr.GetAuthorization().Description())
		if usr.GetAuthorization()&a.roles != 0 {
			next(c)
			return nil
		}
		c.JSON(http.StatusUnauthorized, models.RequestError{Title: "Authorization Error", Description: ErrUserRolesNotMatch.Error(), Code: 0})
		return ErrUserRolesNotMatch
	}
}
