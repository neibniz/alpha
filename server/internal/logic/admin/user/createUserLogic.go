package user

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/uuidx"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type CreateUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logger.Logger
}

func NewCreateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateUserLogic {
	return &CreateUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logger.WithContext(ctx),
	}
}
func (l *CreateUserLogic) CreateUser(req *types.CreateUserRequest) error {
	if req.ReferCode == "" {
		// timestamp replaces user id
		req.ReferCode = uuidx.UserInviteCode(time.Now().UnixMicro())
	}
	pwd, err := tool.EncodePassWord(req.Password)
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "password does not meet policy")
	}
	newUser := &user.User{
		Password:           pwd,
		Algo:               tool.PasswordAlgo,
		ReferralPercentage: req.ReferralPercentage,
		OnlyFirstPurchase:  &req.OnlyFirstPurchase,
		ReferCode:          req.ReferCode,
		Balance:            req.Balance,
		Commission:         req.Commission,
		GiftAmount:         req.GiftAmount,
		IsAdmin:            &req.IsAdmin,
	}
	var ams []user.AuthMethods

	if req.Email != "" {
		_, err := l.svcCtx.UserModel.FindUserAuthMethodByOpenID(l.ctx, "email", req.Email)
		if err == nil {
			return errors.Wrapf(xerr.NewErrCode(xerr.EmailExist), "email exist")
		}
		ams = append(ams, user.AuthMethods{
			AuthType:       "email",
			AuthIdentifier: req.Email,
		})
	}

	newUser.AuthMethods = ams

	// todo: get product id and duration
	if req.RefererUser != "" {
		// get referer user id
		u, err := l.svcCtx.UserModel.FindOneByEmail(l.ctx, req.RefererUser)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.Wrapf(xerr.NewErrCode(xerr.UserNotExist), "referer user not found: %v", err.Error())
			}
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find referer user failed: %v", err.Error())
		}
		newUser.RefererId = u.Id
	}

	err = l.svcCtx.UserModel.Insert(l.ctx, newUser)
	if err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "insert user failed: %v", err.Error())
	}
	return nil
}
