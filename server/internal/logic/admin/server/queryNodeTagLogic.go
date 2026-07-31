package server

import (
	"context"
	"sort"
	"strings"

	modelNode "github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type QueryNodeTagLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewQueryNodeTagLogic Query all node tags
func NewQueryNodeTagLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryNodeTagLogic {
	return &QueryNodeTagLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryNodeTagLogic) QueryNodeTag() (resp *types.QueryNodeTagResponse, err error) {
	tags, err := queryDistinctNodeTags(l.ctx, l.svcCtx.DB)
	if err != nil {
		l.Errorw("[QueryNodeTag] Query Database Error: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "[QueryNodeTag] Query Database Error")
	}

	return &types.QueryNodeTagResponse{
		Tags: tags,
	}, nil
}

func queryDistinctNodeTags(ctx context.Context, db *gorm.DB) ([]string, error) {
	if db.Dialector != nil && db.Dialector.Name() == "postgres" {
		return queryDistinctNodeTagsPostgres(ctx, db)
	}
	return queryDistinctNodeTagsFallback(ctx, db)
}

func queryDistinctNodeTagsPostgres(ctx context.Context, db *gorm.DB) ([]string, error) {
	var rows []nodeTagRow
	err := db.WithContext(ctx).Raw(`
SELECT DISTINCT btrim(tag) AS tag
FROM nodes
CROSS JOIN LATERAL unnest(string_to_array(COALESCE(tags, ''), ',')) AS tag
WHERE btrim(tag) <> ''
ORDER BY tag ASC
`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	tags := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Tag != "" {
			tags = append(tags, row.Tag)
		}
	}
	return tags, nil
}

func queryDistinctNodeTagsFallback(ctx context.Context, db *gorm.DB) ([]string, error) {
	var tagValues []string
	if err := db.WithContext(ctx).
		Model(&modelNode.Node{}).
		Where("tags <> ''").
		Pluck("tags", &tagValues).Error; err != nil {
		return nil, err
	}
	collector := newNodeTagCollector(len(tagValues))
	for _, value := range tagValues {
		collector.AddCSV(value)
	}
	return collector.Tags(), nil
}

type nodeTagCollector struct {
	seen    map[string]struct{}
	tags    []string
	capHint int
}

const smallNodeTagCollectorThreshold = 8

func newNodeTagCollector(capHint int) *nodeTagCollector {
	if capHint <= 0 {
		return &nodeTagCollector{}
	}
	return &nodeTagCollector{capHint: capHint}
}

func (c *nodeTagCollector) AddCSV(value string) {
	for len(value) > 0 {
		part := value
		if index := strings.IndexByte(value, ','); index >= 0 {
			part = value[:index]
			value = value[index+1:]
		} else {
			value = ""
		}
		c.Add(part)
	}
}

func (c *nodeTagCollector) Add(tag string) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return
	}
	if c.seen != nil {
		if _, ok := c.seen[tag]; ok {
			return
		}
		c.seen[tag] = struct{}{}
		c.ensureTags()
		c.tags = append(c.tags, tag)
		return
	}
	for _, existing := range c.tags {
		if existing == tag {
			return
		}
	}
	c.ensureTags()
	c.tags = append(c.tags, tag)
	if len(c.tags) > smallNodeTagCollectorThreshold {
		nextCap := adminServerLookupCap(len(c.tags), c.capHint)
		c.seen = make(map[string]struct{}, nextCap)
		if nextCap > cap(c.tags) {
			tags := make([]string, len(c.tags), nextCap)
			copy(tags, c.tags)
			c.tags = tags
		}
		for _, existing := range c.tags {
			c.seen[existing] = struct{}{}
		}
	}
}

func (c *nodeTagCollector) ensureTags() {
	if c.tags != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = smallNodeTagCollectorThreshold
	} else if capHint > smallNodeTagCollectorThreshold+1 {
		capHint = smallNodeTagCollectorThreshold + 1
	}
	c.tags = make([]string, 0, capHint)
}

func (c *nodeTagCollector) Tags() []string {
	if len(c.tags) == 0 {
		return []string{}
	}
	sort.Strings(c.tags)
	return c.tags
}

type nodeTagRow struct {
	Tag string
}
