package user

import (
	"context"
	"strings"

	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/subscribeprofile"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/random"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	maxSubscribePresetCount = 10
	maxSubscribePresetItems = 200
)

var subscribePresetUpdateReadColumns = []string{
	"id",
	"user_id",
	"user_subscribe_id",
	"preset_key",
	"enabled",
	"is_default",
	"created_at",
	"updated_at",
}

type QueryUserSubscribePresetListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryUserSubscribePresetListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryUserSubscribePresetListLogic {
	return &QueryUserSubscribePresetListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryUserSubscribePresetListLogic) QueryUserSubscribePresetList(req *types.QueryUserSubscribePresetListRequest) (*types.QueryUserSubscribePresetListResponse, error) {
	currentUser, err := l.currentUser()
	if err != nil {
		return nil, err
	}
	userSub, err := l.findOwnedSubscribe(currentUser.Id, req.UserSubscribeId)
	if err != nil {
		return nil, err
	}
	presets, err := subscribeprofile.QueryPresets(l.ctx, l.svcCtx, currentUser.Id, userSub.Id)
	if err != nil {
		return nil, err
	}
	return &types.QueryUserSubscribePresetListResponse{List: presets}, nil
}

func (l *QueryUserSubscribePresetListLogic) currentUser() (*modelUser.User, error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*modelUser.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	return u, nil
}

func (l *QueryUserSubscribePresetListLogic) findOwnedSubscribe(userID, userSubscribeID int64) (*modelUser.Subscribe, error) {
	userSub, err := l.svcCtx.UserModel.FindOneSubscribeForPreset(l.ctx, userSubscribeID)
	if err != nil {
		l.Errorw("FindOneSubscribe failed", logger.Field("error", err.Error()), logger.Field("userSubscribeId", userSubscribeID))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "FindOneSubscribe failed: %v", err.Error())
	}
	if userSub.UserId != userID {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "UserSubscribeId does not belong to the current user")
	}
	return userSub, nil
}

func validatePresetName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "preset name cannot be empty")
	}
	if len([]rune(name)) > 100 {
		return "", errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "preset name is too long")
	}
	return name, nil
}

func (l *QueryUserSubscribePresetListLogic) validatePresetItems(userSub *modelUser.Subscribe, items []types.UserSubscribePresetItem) ([]types.UserSubscribePresetItem, error) {
	if len(items) > maxSubscribePresetItems {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "too many subscription items")
	}
	nodes, err := subscribeprofile.QueryAvailableNodes(l.ctx, l.svcCtx, userSub)
	if err != nil {
		return nil, err
	}
	return subscribeprofile.ValidateSelectedItems(nodes, items)
}

type CreateUserSubscribePresetLogic struct {
	*QueryUserSubscribePresetListLogic
}

func NewCreateUserSubscribePresetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateUserSubscribePresetLogic {
	return &CreateUserSubscribePresetLogic{
		QueryUserSubscribePresetListLogic: NewQueryUserSubscribePresetListLogic(ctx, svcCtx),
	}
}

func (l *CreateUserSubscribePresetLogic) CreateUserSubscribePreset(req *types.CreateUserSubscribePresetRequest) (*types.CreateUserSubscribePresetResponse, error) {
	currentUser, err := l.currentUser()
	if err != nil {
		return nil, err
	}
	name, err := validatePresetName(req.Name)
	if err != nil {
		return nil, err
	}
	userSub, err := l.findOwnedSubscribe(currentUser.Id, req.UserSubscribeId)
	if err != nil {
		return nil, err
	}
	items, err := l.validatePresetItems(userSub, req.Items)
	if err != nil {
		return nil, err
	}

	preset := modelUser.SubscribePreset{}
	if err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		var lockedSubscribe modelUser.Subscribe
		if lockErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "user_id").
			Where("id = ?", userSub.Id).
			First(&lockedSubscribe).Error; lockErr != nil {
			return lockErr
		}
		if lockedSubscribe.UserId != currentUser.Id {
			return errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "subscription does not belong to current user")
		}

		var count int64
		if countErr := tx.Model(&modelUser.SubscribePreset{}).
			Where("user_id = ? AND user_subscribe_id = ?", currentUser.Id, userSub.Id).
			Count(&count).Error; countErr != nil {
			return countErr
		}
		if count >= maxSubscribePresetCount {
			return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "too many subscribe presets")
		}
		isDefault := req.IsDefault || count == 0
		if isDefault && count > 0 {
			if updateErr := tx.Model(&modelUser.SubscribePreset{}).
				Where("user_id = ? AND user_subscribe_id = ? AND is_default = ?", currentUser.Id, userSub.Id, true).
				Update("is_default", false).Error; updateErr != nil {
				return updateErr
			}
		}

		enabled := true
		preset = modelUser.SubscribePreset{
			UserId:          currentUser.Id,
			UserSubscribeId: userSub.Id,
			Name:            name,
			PresetKey:       "ps_" + random.KeyNew(24, 1),
			Mode:            modelUser.SubscribePresetModeCustom,
			Items:           subscribeprofile.MarshalItems(items),
			Sort:            int(count) + 1,
			Enabled:         &enabled,
			IsDefault:       isDefault,
		}
		return tx.Create(&preset).Error
	}); err != nil {
		var codeErr *xerr.CodeError
		if errors.As(err, &codeErr) {
			return nil, err
		}
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "create subscribe preset error: %v", err.Error())
	}
	return &types.CreateUserSubscribePresetResponse{Preset: subscribeprofile.ToTypePreset(preset)}, nil
}

type UpdateUserSubscribePresetLogic struct {
	*QueryUserSubscribePresetListLogic
}

func NewUpdateUserSubscribePresetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserSubscribePresetLogic {
	return &UpdateUserSubscribePresetLogic{
		QueryUserSubscribePresetListLogic: NewQueryUserSubscribePresetListLogic(ctx, svcCtx),
	}
}

func (l *UpdateUserSubscribePresetLogic) UpdateUserSubscribePreset(req *types.UpdateUserSubscribePresetRequest) (*types.CreateUserSubscribePresetResponse, error) {
	currentUser, err := l.currentUser()
	if err != nil {
		return nil, err
	}
	name, err := validatePresetName(req.Name)
	if err != nil {
		return nil, err
	}

	var preset modelUser.SubscribePreset
	if err = l.svcCtx.DB.WithContext(l.ctx).
		Model(&modelUser.SubscribePreset{}).
		Select(subscribePresetUpdateReadColumns).
		Where("id = ? AND user_id = ?", req.Id, currentUser.Id).
		First(&preset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "subscribe preset not found")
		}
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query subscribe preset error: %v", err.Error())
	}
	userSub, err := l.findOwnedSubscribe(currentUser.Id, preset.UserSubscribeId)
	if err != nil {
		return nil, err
	}
	items, err := l.validatePresetItems(userSub, req.Items)
	if err != nil {
		return nil, err
	}
	if preset.IsDefault {
		if req.Enabled != nil && !*req.Enabled {
			return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "default preset cannot be disabled")
		}
		if req.IsDefault != nil && !*req.IsDefault {
			return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "set another preset as default first")
		}
	}

	updates := map[string]any{
		"name":        name,
		"mode":        modelUser.SubscribePresetModeCustom,
		"profile_key": "",
		"items":       subscribeprofile.MarshalItems(items),
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	makeDefault := req.IsDefault != nil && *req.IsDefault
	if makeDefault {
		updates["enabled"] = true
		updates["is_default"] = true
	}
	if err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		var lockedSubscribe modelUser.Subscribe
		if lockErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "user_id").
			Where("id = ?", userSub.Id).
			First(&lockedSubscribe).Error; lockErr != nil {
			return lockErr
		}
		if lockedSubscribe.UserId != currentUser.Id {
			return errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "subscription does not belong to current user")
		}
		if makeDefault {
			if updateErr := tx.Model(&modelUser.SubscribePreset{}).
				Where("user_id = ? AND user_subscribe_id = ? AND id <> ? AND is_default = ?", currentUser.Id, userSub.Id, preset.Id, true).
				Update("is_default", false).Error; updateErr != nil {
				return updateErr
			}
		}
		result := tx.Model(&modelUser.SubscribePreset{}).
			Where("id = ? AND user_id = ?", preset.Id, currentUser.Id).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "subscribe preset changed; refresh and retry")
		}
		return nil
	}); err != nil {
		var codeErr *xerr.CodeError
		if errors.As(err, &codeErr) {
			return nil, err
		}
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update subscribe preset error: %v", err.Error())
	}

	preset.Name = name
	preset.Mode = modelUser.SubscribePresetModeCustom
	preset.ProfileKey = ""
	preset.Items = updates["items"].(string)
	if req.Enabled != nil {
		preset.Enabled = req.Enabled
	}
	if makeDefault {
		enabled := true
		preset.Enabled = &enabled
		preset.IsDefault = true
	}
	return &types.CreateUserSubscribePresetResponse{Preset: subscribeprofile.ToTypePreset(preset)}, nil
}

type DeleteUserSubscribePresetLogic struct {
	*QueryUserSubscribePresetListLogic
}

func NewDeleteUserSubscribePresetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteUserSubscribePresetLogic {
	return &DeleteUserSubscribePresetLogic{
		QueryUserSubscribePresetListLogic: NewQueryUserSubscribePresetListLogic(ctx, svcCtx),
	}
}

func (l *DeleteUserSubscribePresetLogic) DeleteUserSubscribePreset(req *types.DeleteUserSubscribePresetRequest) error {
	currentUser, err := l.currentUser()
	if err != nil {
		return err
	}
	var preset modelUser.SubscribePreset
	if err = l.svcCtx.DB.WithContext(l.ctx).
		Select("id", "user_id", "user_subscribe_id", "is_default").
		Where("id = ? AND user_id = ?", req.Id, currentUser.Id).
		First(&preset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "subscribe preset not found")
		}
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query subscribe preset error: %v", err.Error())
	}
	if err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		var lockedSubscribe modelUser.Subscribe
		if lockErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "user_id").
			Where("id = ?", preset.UserSubscribeId).
			First(&lockedSubscribe).Error; lockErr != nil {
			return lockErr
		}
		if lockedSubscribe.UserId != currentUser.Id {
			return errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "subscription does not belong to current user")
		}
		var lockedPreset modelUser.SubscribePreset
		if queryErr := tx.Select("id", "user_subscribe_id", "is_default").
			Where("id = ? AND user_id = ?", preset.Id, currentUser.Id).
			First(&lockedPreset).Error; queryErr != nil {
			if errors.Is(queryErr, gorm.ErrRecordNotFound) {
				return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "subscribe preset changed; refresh and retry")
			}
			return queryErr
		}
		result := tx.Where("id = ? AND user_id = ?", lockedPreset.Id, currentUser.Id).
			Delete(&modelUser.SubscribePreset{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "subscribe preset changed; refresh and retry")
		}
		if !lockedPreset.IsDefault {
			return nil
		}
		var replacement modelUser.SubscribePreset
		findErr := tx.Select("id").
			Where("user_id = ? AND user_subscribe_id = ? AND enabled = ?", currentUser.Id, lockedPreset.UserSubscribeId, true).
			Order("sort ASC, id ASC").
			First(&replacement).Error
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			return nil
		}
		if findErr != nil {
			return findErr
		}
		return tx.Model(&modelUser.SubscribePreset{}).
			Where("id = ?", replacement.Id).
			Update("is_default", true).Error
	}); err != nil {
		var codeErr *xerr.CodeError
		if errors.As(err, &codeErr) {
			return err
		}
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseDeletedError), "delete subscribe preset error: %v", err.Error())
	}
	return nil
}
