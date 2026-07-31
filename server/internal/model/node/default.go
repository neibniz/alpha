package node

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var _ Model = (*customServerModel)(nil)

//goland:noinspection GoNameStartsWithPackageName
type (
	Model interface {
		serverModel
		NodeModel
		XrayTemplateModel
		customCacheLogicModel
		customServerLogicModel
	}
	serverModel interface {
		InsertServer(ctx context.Context, data *Server, tx ...*gorm.DB) error
		FindOneServer(ctx context.Context, id int64) (*Server, error)
		FindOneServerForRuntime(ctx context.Context, id int64) (*Server, error)
		ServerExists(ctx context.Context, id int64) (bool, error)
		UpdateServer(ctx context.Context, data *Server, tx ...*gorm.DB) error
		DeleteServer(ctx context.Context, id int64, tx ...*gorm.DB) error
		Transaction(ctx context.Context, fn func(db *gorm.DB) error) error
		QueryServerList(ctx context.Context, ids []int64) (servers []*Server, err error)
		QueryServerIPList(ctx context.Context) ([]string, error)
		QueryDeployServerList(ctx context.Context) ([]*Server, error)
	}

	NodeModel interface {
		InsertNode(ctx context.Context, data *Node, tx ...*gorm.DB) error
		FindOneNode(ctx context.Context, id int64) (*Node, error)
		UpdateNode(ctx context.Context, data *Node, tx ...*gorm.DB) error
		DeleteNode(ctx context.Context, id int64, tx ...*gorm.DB) error
	}
	XrayTemplateModel interface {
		InsertXrayTemplate(ctx context.Context, data *XrayTemplate, tx ...*gorm.DB) error
		FindOneXrayTemplate(ctx context.Context, id int64) (*XrayTemplate, error)
		UpdateXrayTemplate(ctx context.Context, data *XrayTemplate, tx ...*gorm.DB) error
		DeleteXrayTemplate(ctx context.Context, id int64, tx ...*gorm.DB) error
		BindXrayTemplates(ctx context.Context, serverId int64, bindings []*ServerXrayTemplate, tx ...*gorm.DB) error
		QueryServerXrayTemplates(ctx context.Context, serverId int64, enabledOnly bool) ([]*ServerXrayTemplate, error)
		QueryServerXrayTemplatesByServerIDs(ctx context.Context, serverIDs []int64, enabledOnly bool) (map[int64][]*ServerXrayTemplate, error)
		QueryServerXrayTemplatesWithOptions(ctx context.Context, serverId int64, opts QueryServerXrayTemplateOptions) ([]*ServerXrayTemplate, error)
	}

	customServerModel struct {
		*defaultServerModel
	}
	defaultServerModel struct {
		*gorm.DB
		Cache *redis.Client
	}
)

func newServerModel(db *gorm.DB, cache *redis.Client) *defaultServerModel {
	return &defaultServerModel{
		DB:    db,
		Cache: cache,
	}
}

// NewModel returns a model for the database table.
func NewModel(conn *gorm.DB, cache *redis.Client) Model {
	return &customServerModel{
		defaultServerModel: newServerModel(conn, cache),
	}
}

func (m *defaultServerModel) InsertServer(ctx context.Context, data *Server, tx ...*gorm.DB) error {
	db := m.DB
	if len(tx) > 0 {
		db = tx[0]
	}
	return db.WithContext(ctx).Create(data).Error
}

func (m *defaultServerModel) FindOneServer(ctx context.Context, id int64) (*Server, error) {
	var server Server
	err := m.WithContext(ctx).Model(&Server{}).Where("id = ?", id).First(&server).Error
	return &server, err
}

func (m *defaultServerModel) FindOneServerForRuntime(ctx context.Context, id int64) (*Server, error) {
	var server Server
	err := m.WithContext(ctx).
		Model(&Server{}).
		Select(serverRuntimeColumns).
		Where("id = ?", id).
		First(&server).Error
	return &server, err
}

func (m *defaultServerModel) ServerExists(ctx context.Context, id int64) (bool, error) {
	if id <= 0 {
		return false, nil
	}
	err := m.ensureModelExistsByID(ctx, &Server{}, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}

func (m *defaultServerModel) ensureModelExistsByID(ctx context.Context, model interface{}, id int64) error {
	var row struct {
		Id int64
	}
	return m.WithContext(ctx).
		Model(model).
		Select("id").
		Where("id = ?", id).
		First(&row).Error
}

func (m *defaultServerModel) UpdateServer(ctx context.Context, data *Server, tx ...*gorm.DB) error {
	if data == nil || data.Id <= 0 {
		return gorm.ErrRecordNotFound
	}
	db := m.DB
	if len(tx) > 0 {
		db = tx[0]
	}
	updatedAt := time.Now()
	result := db.WithContext(ctx).
		Model(&Server{}).
		Where("id = ?", data.Id).
		Select("name", "country", "city", "address", "updated_at").
		UpdateColumns(&Server{
			Name:      data.Name,
			Country:   data.Country,
			City:      data.City,
			Address:   data.Address,
			UpdatedAt: updatedAt,
		})
	if err := existingUpdateError(result); err != nil {
		return err
	}
	data.UpdatedAt = updatedAt
	return nil
}

func (m *defaultServerModel) DeleteServer(ctx context.Context, id int64, tx ...*gorm.DB) error {
	db := m.DB
	if len(tx) > 0 {
		db = tx[0]
	}
	return db.WithContext(ctx).Where("id = ?", id).Delete(&Server{}).Error
}

func (m *defaultServerModel) InsertNode(ctx context.Context, data *Node, tx ...*gorm.DB) error {
	db := m.DB
	if len(tx) > 0 {
		db = tx[0]
	}
	return db.WithContext(ctx).Create(data).Error
}

func (m *defaultServerModel) FindOneNode(ctx context.Context, id int64) (*Node, error) {
	var node Node
	err := m.WithContext(ctx).Model(&Node{}).Where("id = ?", id).First(&node).Error
	return &node, err
}

func (m *defaultServerModel) UpdateNode(ctx context.Context, data *Node, tx ...*gorm.DB) error {
	if data == nil || data.Id <= 0 {
		return gorm.ErrRecordNotFound
	}
	db := m.DB
	if len(tx) > 0 {
		db = tx[0]
	}
	updatedAt := time.Now()
	result := db.WithContext(ctx).
		Model(&Node{}).
		Where("id = ?", data.Id).
		Select("name", "tags", "port", "address", "server_id", "protocol", "inbound_alias", "enabled", "updated_at").
		UpdateColumns(&Node{
			Name:         data.Name,
			Tags:         data.Tags,
			Port:         data.Port,
			Address:      data.Address,
			ServerId:     data.ServerId,
			Protocol:     data.Protocol,
			InboundAlias: data.InboundAlias,
			Enabled:      data.Enabled,
			UpdatedAt:    updatedAt,
		})
	if err := existingUpdateError(result); err != nil {
		return err
	}
	data.UpdatedAt = updatedAt
	return nil
}

func existingUpdateError(result *gorm.DB) error {
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (m *defaultServerModel) DeleteNode(ctx context.Context, id int64, tx ...*gorm.DB) error {
	db := m.DB
	if len(tx) > 0 {
		db = tx[0]
	}
	return db.WithContext(ctx).Where("id = ?", id).Delete(&Node{}).Error
}

func (m *defaultServerModel) InsertXrayTemplate(ctx context.Context, data *XrayTemplate, tx ...*gorm.DB) error {
	db := m.DB
	if len(tx) > 0 {
		db = tx[0]
	}
	return db.WithContext(ctx).Create(data).Error
}

func (m *defaultServerModel) FindOneXrayTemplate(ctx context.Context, id int64) (*XrayTemplate, error) {
	var template XrayTemplate
	err := m.WithContext(ctx).Model(&XrayTemplate{}).Where("id = ?", id).First(&template).Error
	return &template, err
}

func (m *defaultServerModel) UpdateXrayTemplate(ctx context.Context, data *XrayTemplate, tx ...*gorm.DB) error {
	if err := m.ensureModelExistsByID(ctx, &XrayTemplate{}, data.Id); err != nil {
		return err
	}
	db := m.DB
	if len(tx) > 0 {
		db = tx[0]
	}
	return db.WithContext(ctx).Where("id = ?", data.Id).Save(data).Error
}

func (m *defaultServerModel) DeleteXrayTemplate(ctx context.Context, id int64, tx ...*gorm.DB) error {
	db := m.DB
	if len(tx) > 0 {
		db = tx[0]
	}
	return db.WithContext(ctx).Where("id = ?", id).Delete(&XrayTemplate{}).Error
}

func (m *defaultServerModel) BindXrayTemplates(ctx context.Context, serverId int64, bindings []*ServerXrayTemplate, tx ...*gorm.DB) error {
	db := m.DB
	if len(tx) > 0 {
		db = tx[0]
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("server_id = ?", serverId).Delete(&ServerXrayTemplate{}).Error; err != nil {
			return err
		}
		if len(bindings) == 0 {
			return nil
		}
		return tx.Create(&bindings).Error
	})
}

func (m *defaultServerModel) QueryServerXrayTemplates(ctx context.Context, serverId int64, enabledOnly bool) ([]*ServerXrayTemplate, error) {
	templatePreload := XrayTemplatePreloadFull
	if enabledOnly {
		templatePreload = XrayTemplatePreloadRuntime
	}
	return m.QueryServerXrayTemplatesWithOptions(ctx, serverId, QueryServerXrayTemplateOptions{
		EnabledOnly:     enabledOnly,
		TemplatePreload: templatePreload,
	})
}

func (m *defaultServerModel) QueryServerXrayTemplatesByServerIDs(ctx context.Context, serverIDs []int64, enabledOnly bool) (map[int64][]*ServerXrayTemplate, error) {
	ids := normalizeServerIDs(serverIDs)
	if len(ids) == 0 {
		return map[int64][]*ServerXrayTemplate{}, nil
	}
	templatePreload := XrayTemplatePreloadFull
	if enabledOnly {
		templatePreload = XrayTemplatePreloadRuntime
	}
	list := make([]*ServerXrayTemplate, 0)
	query := m.WithContext(ctx).Model(&ServerXrayTemplate{}).
		Where("server_id IN ?", ids).
		Order(`"server_xray_templates"."server_id" ASC, "server_xray_templates"."sort" ASC, "server_xray_templates"."id" ASC`)
	if enabledOnly {
		query = query.Select(serverXrayTemplateRuntimeColumns).
			Where(`"server_xray_templates"."enabled" = ?`, true).
			Joins("JOIN xray_templates ON xray_templates.id = server_xray_templates.template_id AND xray_templates.enabled = ?", true)
	}
	query = applyXrayTemplatePreload(query, templatePreload)
	if err := query.Find(&list).Error; err != nil {
		return nil, err
	}
	return groupServerXrayTemplatesByServerID(list, ids), nil
}

const serverXrayTemplateGroupPrecountThreshold = 8

func groupServerXrayTemplatesByServerID(list []*ServerXrayTemplate, serverIDs []int64) map[int64][]*ServerXrayTemplate {
	result := make(map[int64][]*ServerXrayTemplate, len(serverIDs))
	for _, id := range serverIDs {
		result[id] = []*ServerXrayTemplate{}
	}
	if len(list) == 0 {
		return result
	}
	if len(list) <= serverXrayTemplateGroupPrecountThreshold {
		for _, item := range list {
			if item == nil {
				continue
			}
			if _, ok := result[item.ServerId]; ok {
				result[item.ServerId] = append(result[item.ServerId], item)
			}
		}
		return result
	}

	counts := make(map[int64]int, len(serverIDs))
	for _, item := range list {
		if item != nil {
			if _, ok := result[item.ServerId]; !ok {
				continue
			}
			counts[item.ServerId]++
		}
	}
	for serverID, count := range counts {
		result[serverID] = make([]*ServerXrayTemplate, 0, count)
	}
	for _, item := range list {
		if item == nil {
			continue
		}
		if _, ok := result[item.ServerId]; ok {
			result[item.ServerId] = append(result[item.ServerId], item)
		}
	}
	return result
}

func (m *defaultServerModel) QueryServerXrayTemplatesWithOptions(ctx context.Context, serverId int64, opts QueryServerXrayTemplateOptions) ([]*ServerXrayTemplate, error) {
	list := make([]*ServerXrayTemplate, 0)
	query := m.WithContext(ctx).Model(&ServerXrayTemplate{}).
		Where("server_id = ?", serverId).
		Order(`"server_xray_templates"."sort" ASC, "server_xray_templates"."id" ASC`)
	if opts.EnabledOnly {
		query = query.Select(serverXrayTemplateRuntimeColumns).
			Where(`"server_xray_templates"."enabled" = ?`, true).
			Joins("JOIN xray_templates ON xray_templates.id = server_xray_templates.template_id AND xray_templates.enabled = ?", true)
	}
	query = applyXrayTemplatePreload(query, opts.TemplatePreload)
	err := query.Find(&list).Error
	return list, err
}

func normalizeServerIDs(ids []int64) []int64 {
	return normalizeNodePositiveIDs(ids)
}

func applyXrayTemplatePreload(query *gorm.DB, preload XrayTemplatePreloadMode) *gorm.DB {
	switch preload {
	case XrayTemplatePreloadNone:
		return query
	case XrayTemplatePreloadSummary:
		return query.Preload("Template", func(db *gorm.DB) *gorm.DB {
			return db.Select(xrayTemplateBindingSummaryColumns)
		})
	case XrayTemplatePreloadRuntime:
		return query.Preload("Template", func(db *gorm.DB) *gorm.DB {
			return db.Select(xrayTemplateRuntimeColumns)
		})
	case XrayTemplatePreloadFull:
		return query.Preload("Template")
	default:
		return query
	}
}

func (m *defaultServerModel) Transaction(ctx context.Context, fn func(db *gorm.DB) error) error {
	return m.WithContext(ctx).Transaction(fn)
}
