package models

import (
	"errors"
	"fmt"
	"github.com/labstack/echo"
	"goappuser/database"
	"goappuser/security"

	"gopkg.in/mgo.v2"
	"gotools"
	"log"
	"time"
)

var (
	ErrAlreadyRegister    = errors.New("mail already register")
	ErrAlreadyAuth        = errors.New("Already authenticate")
	ErrUserNotFound       = errors.New("User not found")
	ErrInvalidCredentials = errors.New("Invalid credentials")
	ErrNoSession          = errors.New("No session found")
)

//Manager interface to implements all feature to manage user
type UserManager interface {
	//Register register as a new user
	Register(User) error
	//IsExist check existence of the user
	IsExist(User) bool
	//ResetPassword user with specifics credentials
	ResetPassword(User, string) bool
	//GetByEmail retrieve a user using its email
	GetByEmail(email string, user User) error
	//Authenticate
	Authenticate(c *echo.Context, user User) (User, error)
}

//NewUser create a basic user with the mandatory parameters for each users
func NewUserDefaultExtended(email, password string) *UserDefaultExtended {
	return &UserDefaultExtended{UserDefault: &UserDefault{email: email, password: []byte(password), role: "user"}}
}

//User Represent a basic user

//TODO: Change User to an interface
type User interface {
	Email() string
	SetEmail(email string)
	Password() []byte
	SetPassword(pass []byte)
	Role() string
	SetRole(role string)
}

//User Represent a basic user

type UserDefault struct {
	password []byte `bson:"password" json:"-"`
	email    string `bson:"email" json:"email"`
	role     string `bson:"role" json:"-"`
}

type UserDefaultExtended struct {
	*UserDefault       `bson:"credentials" json:"credentials"`
	Name               string    `bson:"name" json:"name"`
	Surname            string    `bson:"surname" json:"surname"`
	Pseudo             string    `bson:"pseudo" json:"pseudo"`
	DateCreate         time.Time `bson:"created" json:"created"`
	DateLastConnection time.Time `bson:"lastconnection" json:"lastconnection,omitempty"`
	BirthDate          time.Time `bson:"birthdate" json:"birthdate,omitempty"`
}

func (u *UserDefault) Email() string {
	return u.email
}

func (u *UserDefault) SetEmail(email string) {
	u.email = email

}
func (u *UserDefault) Password() []byte {
	return u.password
}

func (u *UserDefault) SetPassword(pass []byte) {
	u.password = pass
}

func (u *UserDefault) Role() string {
	return u.role
}

func (u *UserDefault) SetRole(role string) {
	u.role = role
}

//NewDBUserManage create a db manager user
func NewDBUserManage(db dbm.DatabaseQuerier, auth security.AuthenticationProcesser, sm *SessionManager) *DBUserManage {
	return &DBUserManage{db: db, auth: auth, sessionManager: sm}
}

//DBUserManage respect Manager Interface using MGO (MongoDB driver)
type DBUserManage struct {
	db             dbm.DatabaseQuerier
	auth           security.AuthenticationProcesser
	sessionManager *SessionManager
}

//Register register as a new user
func (m *DBUserManage) Register(user User) error {
	if m.IsExist(user) {
		return ErrAlreadyRegister
	}

	pass, errHash := m.auth.Hash(user.Password())
	if errHash != nil {
		return errHash
	}
	user.SetPassword(pass)
	log.Println("insert user", user)
	if errInsert := m.db.InsertModel(user); errInsert != nil {
		log.Println("error insert", errInsert, " user: ", user.Email())
		return errInsert
	}
	log.Println("insert OK")
	return nil
}

//IsExist check existence of the user
func (m *DBUserManage) IsExist(user User) bool {
	u := &UserDefault{}
	if err := m.GetByEmail(user.Email(), u); err == nil {
		log.Println("IsExist user ", u)
		return tools.NotEmpty(u)
	} else if err == mgo.ErrNotFound {
		return false
	}
	return false
}

//ResetPassword user with specifics credentials
func (m *DBUserManage) ResetPassword(user User, newPassword string) bool {
	return false
}

//GetByEmail retrieve a user using its email
func (m *DBUserManage) GetByEmail(email string, user User) error {
	if err := m.db.GetOneModel(dbm.M{"email": email}, user); err != nil {
		return err
	}
	return nil
}

//Authenticate log the user
func (m *DBUserManage) Authenticate(c *echo.Context, user User) (User, error) {
	if session, isOk := (*c).Get("Session").(Session); isOk {
		return session.User, ErrAlreadyAuth
	}
	username, password, err := m.auth.GetCredentials(*c)
	if err != nil {
		return nil, errors.New(fmt.Sprint("Failed to retrieve credentials from request: ", err))
	}

	err = m.GetByEmail(username, user)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if ok := m.auth.Compare([]byte(password), user.Password()); ok == true {

		if _, cookie, err := m.sessionManager.CreateSession(user); err == nil {
			(*c).SetCookie(cookie)
		}
		return user, nil
	}
	return nil, ErrInvalidCredentials
}

func (m *DBUserManage) cleanSession(c echo.Context) error {
	if _, isOk := c.Get("Session").(Session); isOk {
		return ErrNoSession
	}
	return nil
	//TODO: use m.db.Remove Model to remove the session
	//	m.db.
}
