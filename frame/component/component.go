package component

import (
	"errors"
	"fmt"

	"github.com/jinzhu/gorm"
	"wataru.com/gogo/frame/db"
	"wataru.com/gogo/logger"
)

type Bean interface {
	Initialize()
	Destroy()
}

type Component struct {
	Bean
}

type Dao struct {
	Component
	Db *gorm.DB
}

type Controller struct {
	Component
}

type Service struct {
	Component
	Db *gorm.DB
}

func (s *Service) DoTransaction(fn func(*gorm.DB) interface{}) interface{} {
	var result interface{}
	var pnc interface{}
	f := func(tx0 *gorm.DB) (err error) {
		defer func() {
			if r := recover(); r != nil {
				msg := fmt.Sprintf("%s", r)
				logger.Error("Transaction rollback for error: %s", msg)
				err = errors.New(msg)
				pnc = r
			}
		}()
		result = fn(tx0)
		return err
	}
	err := s.Db.Transaction(f)
	if err != nil {
		panic(pnc)
	}
	return result
}

func (s *Component) Initialize() {
}

func (s *Component) Destroy() {
}

func NewBaseService() *Service {
	return &Service{
		Db: db.Db,
	}
}

func NewBaseController() *Controller {
	return &Controller{}
}
