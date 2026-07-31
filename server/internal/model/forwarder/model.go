package forwarder

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Instance struct {
	ID             int64     `gorm:"primaryKey"`
	InstanceID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	BootID         uuid.UUID `gorm:"type:uuid;not null"`
	Name           string    `gorm:"type:varchar(100);not null;default:''"`
	Hostname       string    `gorm:"type:varchar(255);not null;default:''"`
	PublicIP       string    `gorm:"type:varchar(64);not null;default:'';index"`
	Version        string    `gorm:"type:varchar(64);not null;default:''"`
	StartedAt      time.Time
	FirstSeenAt    time.Time
	LastSeenAt     time.Time `gorm:"index"`
	DesiredVersion string    `gorm:"type:varchar(64);not null;default:''"`
	DesiredHash    string    `gorm:"type:varchar(128);not null;default:''"`
	AppliedVersion string    `gorm:"type:varchar(64);not null;default:''"`
	AppliedHash    string    `gorm:"type:varchar(128);not null;default:''"`
	ApplyStatus    string    `gorm:"type:varchar(32);not null;default:'pending'"`
	ApplyError     string    `gorm:"type:text;not null;default:''"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (*Instance) TableName() string {
	return "forwarder_instances"
}

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Register(ctx context.Context, instance *Instance) error {
	now := time.Now().UTC()
	instance.FirstSeenAt = now
	instance.LastSeenAt = now
	instance.CreatedAt = now
	instance.UpdatedAt = now
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "instance_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"boot_id":      instance.BootID,
			"name":         instance.Name,
			"hostname":     instance.Hostname,
			"public_ip":    instance.PublicIP,
			"version":      instance.Version,
			"started_at":   instance.StartedAt,
			"last_seen_at": now,
			"updated_at":   now,
		}),
	}).Create(instance).Error
}

func (s *Store) FindByInstanceID(ctx context.Context, instanceID uuid.UUID) (*Instance, error) {
	var instance Instance
	err := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&instance).Error
	return &instance, err
}

func (s *Store) List(ctx context.Context, search string) ([]Instance, error) {
	rows := make([]Instance, 0)
	query := s.db.WithContext(ctx).Model(&Instance{})
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name ILIKE ? OR hostname ILIKE ? OR public_ip ILIKE ? OR instance_id::text ILIKE ?", like, like, like, like)
	}
	err := query.Order("last_seen_at DESC, id DESC").Find(&rows).Error
	return rows, err
}

func (s *Store) UpdateDesired(ctx context.Context, instanceID uuid.UUID, version, hash string) error {
	return s.db.WithContext(ctx).Model(&Instance{}).
		Where(
			"instance_id = ? AND (desired_version <> ? OR desired_hash <> ?)",
			instanceID, version, hash,
		).
		Updates(map[string]any{
			"desired_version": version,
			"desired_hash":    hash,
			"updated_at":      time.Now().UTC(),
		}).Error
}

func (s *Store) TouchRuntime(
	ctx context.Context,
	instanceID, bootID uuid.UUID,
	version, appliedVersion, appliedHash, applyStatus, applyError string,
) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&Instance{}).
		Where(
			`instance_id = ? AND (
				last_seen_at < ? OR boot_id <> ? OR version <> ? OR
				applied_version <> ? OR applied_hash <> ? OR
				apply_status <> ? OR apply_error <> ?
			)`,
			instanceID, now.Add(-30*time.Second), bootID, version,
			appliedVersion, appliedHash, applyStatus, applyError,
		).
		Updates(map[string]any{
			"boot_id":         bootID,
			"version":         version,
			"last_seen_at":    now,
			"applied_version": appliedVersion,
			"applied_hash":    appliedHash,
			"apply_status":    applyStatus,
			"apply_error":     applyError,
			"updated_at":      now,
		}).Error
}
