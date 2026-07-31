package migrate

import (
	"time"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/uuidx"
	"gorm.io/gorm"
)

// CreateAdminUser create admin user
func CreateAdminUser(email, password string, tx *gorm.DB) error {
	enable := true
	return tx.Transaction(func(tx *gorm.DB) error {
		// Prevent duplicate creation
		if tx.Model(&user.User{}).Find(&user.User{}).RowsAffected != 0 {
			logger.Info("User already exists, skip creating administrator account")
			return nil
		}
		passwordHash, err := tool.EncodePassWord(password)
		if err != nil {
			return err
		}

		u := user.User{
			Password:  passwordHash,
			Algo:      tool.PasswordAlgo,
			IsAdmin:   &enable,
			ReferCode: uuidx.UserInviteCode(time.Now().Unix()),
		}
		if err := tx.Model(&user.User{}).Save(&u).Error; err != nil {
			return err
		}
		method := user.AuthMethods{
			UserId:         u.Id,
			AuthType:       "email",
			AuthIdentifier: email,
			Verified:       true,
		}
		if err := tx.Model(&user.AuthMethods{}).Save(&method).Error; err != nil {
			return err
		}
		return nil
	})
}
