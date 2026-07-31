package analytics

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/privacy"
)

const (
	defaultAccessRetentionDays        = 90
	defaultTrafficRawRetentionHours   = 72
	defaultTrafficRetentionDays       = 90
	defaultTrafficHourlyRetentionDays = 730
	defaultForwarderRetentionDays     = 90
	maxRetentionDays                  = 3650
	accessPrivacyVersion              = 1
	forwarderPrivacyVersion           = 1
	accessPrivacyMigrationBatch       = 1000
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type clickHouseStore struct {
	conn     driver.Conn
	database string
	privacy  *privacy.Codec
}

type protectedAccessEvent struct {
	AccessEvent
	PrivacyVersion            uint8
	SourceIPBlindIndex        string
	SourceNetworkBlindIndex   string
	SourceIPDisplay           string
	DestinationHostBlindIndex string
	DestinationHostDisplay    string
	DestinationIPBlindIndex   string
	DestinationIPDisplay      string
	ReasonDisplay             string
}

type protectedForwardConnectionEvent struct {
	ForwardConnectionEvent
	PrivacyVersion                 uint8
	SourceIPBlindIndex             string
	SourceIPDisplay                string
	ConfiguredTargetHostBlindIndex string
	ConfiguredTargetHostDisplay    string
	ResolvedTargetIPBlindIndex     string
	ResolvedTargetIPDisplay        string
	ForwarderOutboundIPDisplay     string
	ErrorDisplay                   string
}

func NewClickHouseStore(ctx context.Context, cfg config.ClickHouseConfig, codecs ...*privacy.Codec) (Store, error) {
	if !cfg.Enabled {
		return NewUnavailableStore(), nil
	}
	if !identifierPattern.MatchString(cfg.Database) {
		return nil, fmt.Errorf("invalid ClickHouse database name %q", cfg.Database)
	}
	if strings.TrimSpace(cfg.Addr) == "" {
		return nil, fmt.Errorf("ClickHouse address is required")
	}
	if cfg.AccessRetentionDays <= 0 {
		cfg.AccessRetentionDays = defaultAccessRetentionDays
	}
	if cfg.TrafficRetentionDays <= 0 {
		cfg.TrafficRetentionDays = defaultTrafficRetentionDays
	}
	if cfg.TrafficRawRetentionHours <= 0 {
		cfg.TrafficRawRetentionHours = defaultTrafficRawRetentionHours
	}
	if cfg.TrafficHourlyRetentionDays <= 0 {
		cfg.TrafficHourlyRetentionDays = defaultTrafficHourlyRetentionDays
	}
	if cfg.ForwarderRetentionDays <= 0 {
		cfg.ForwarderRetentionDays = defaultForwarderRetentionDays
	}
	if cfg.AccessRetentionDays > maxRetentionDays ||
		cfg.TrafficRetentionDays > maxRetentionDays ||
		cfg.TrafficHourlyRetentionDays > maxRetentionDays ||
		cfg.ForwarderRetentionDays > maxRetentionDays ||
		cfg.TrafficRawRetentionHours > maxRetentionDays*24 {
		return nil, fmt.Errorf("ClickHouse retention cannot exceed %d days", maxRetentionDays)
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.Addr},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: cfg.Username,
			Password: cfg.Password,
		},
		DialTimeout:     5 * time.Second,
		MaxOpenConns:    8,
		MaxIdleConns:    4,
		ConnMaxLifetime: 30 * time.Minute,
		Compression:     &clickhouse.Compression{Method: clickhouse.CompressionLZ4},
	})
	if err != nil {
		return nil, err
	}
	privacyCodec := privacy.DisabledCodec()
	if len(codecs) > 0 && codecs[0] != nil {
		privacyCodec = codecs[0]
	}
	store := &clickHouseStore{conn: conn, database: cfg.Database, privacy: privacyCodec}
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping ClickHouse: %w", err)
	}
	if err := store.initialize(
		ctx,
		cfg.AccessRetentionDays,
		cfg.TrafficRawRetentionHours,
		cfg.TrafficRetentionDays,
		cfg.TrafficHourlyRetentionDays,
		cfg.ForwarderRetentionDays,
	); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := store.migrateAccessPrivacy(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return store, nil
}

func (s *clickHouseStore) Available() bool { return true }
func (s *clickHouseStore) Close() error    { return s.conn.Close() }

func (s *clickHouseStore) initialize(
	ctx context.Context,
	accessRetentionDays int,
	trafficRawRetentionHours int,
	trafficRetentionDays int,
	trafficHourlyRetentionDays int,
	forwarderRetentionDays int,
) error {
	queries := []string{
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", s.database),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.xray_access_events (
			event_id UUID,
			event_time DateTime64(6, 'UTC'),
			ingested_at DateTime64(6, 'UTC') DEFAULT now64(6),
			server_id UInt64,
			user_id UInt64,
			user_subscribe_id UInt64,
			source_ip String,
			source_port UInt16,
			destination_host String,
			destination_ip String,
			destination_port UInt16,
			network LowCardinality(String),
			status LowCardinality(String),
			outbound_tag LowCardinality(String),
			reason String CODEC(ZSTD(1)),
			parser_version UInt16,
			privacy_version UInt8 DEFAULT 0,
			source_ip_bidx String DEFAULT '',
			source_network_bidx String DEFAULT '',
			source_ip_display String DEFAULT '',
			destination_host_bidx String DEFAULT '',
			destination_host_display String DEFAULT '',
			destination_ip_bidx String DEFAULT '',
			destination_ip_display String DEFAULT '',
			reason_display String DEFAULT ''
		) ENGINE = ReplacingMergeTree(ingested_at)
		PARTITION BY toYYYYMM(event_time)
		ORDER BY (user_subscribe_id, event_time, server_id, event_id)
		TTL event_time + INTERVAL %d DAY DELETE`, s.database, accessRetentionDays),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.traffic_events (
			event_id UUID,
			event_time DateTime64(3, 'UTC'),
			ingested_at DateTime64(3, 'UTC') DEFAULT now64(3),
			server_id UInt64,
			user_id UInt64,
			user_subscribe_id UInt64,
			download UInt64,
			upload UInt64
		) ENGINE = ReplacingMergeTree(ingested_at)
		PARTITION BY toYYYYMMDD(event_time)
		ORDER BY (user_subscribe_id, event_time, server_id, event_id)
		TTL event_time + INTERVAL %d HOUR DELETE`, s.database, trafficRawRetentionHours),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.traffic_minute (
			bucket_time DateTime64(3, 'UTC'),
			refreshed_at DateTime64(3, 'UTC') DEFAULT now64(3),
			server_id UInt64,
			user_id UInt64,
			user_subscribe_id UInt64,
			download UInt64,
			upload UInt64
		) ENGINE = ReplacingMergeTree(refreshed_at)
		PARTITION BY toYYYYMM(bucket_time)
		ORDER BY (user_subscribe_id, bucket_time, server_id)
		TTL bucket_time + INTERVAL %d DAY DELETE`, s.database, trafficRetentionDays),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.traffic_hourly (
			bucket_time DateTime64(3, 'UTC'),
			refreshed_at DateTime64(3, 'UTC') DEFAULT now64(3),
			server_id UInt64,
			user_id UInt64,
			user_subscribe_id UInt64,
			download UInt64,
			upload UInt64
		) ENGINE = ReplacingMergeTree(refreshed_at)
		PARTITION BY toYYYYMM(bucket_time)
		ORDER BY (user_subscribe_id, bucket_time, server_id)
			TTL bucket_time + INTERVAL %d DAY DELETE`, s.database, trafficHourlyRetentionDays),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.forward_connection_events (
			event_id UUID,
			connection_id UUID,
			instance_id UUID,
			boot_id UUID,
			event_time DateTime64(6, 'UTC'),
			ingested_at DateTime64(6, 'UTC') DEFAULT now64(6),
			event_type LowCardinality(String),
			source_ip String,
			source_port UInt16,
			listen_ip String,
			listen_port UInt16,
			configured_target_host String,
			configured_target_port UInt16,
			resolved_target_ip String,
			resolved_target_port UInt16,
			forwarder_outbound_ip String,
			forwarder_outbound_port UInt16,
			rule_id String,
			server_id UInt64,
			node_id UInt64,
			inbound LowCardinality(String),
			config_version String,
			connected_at Nullable(DateTime64(6, 'UTC')),
			closed_at Nullable(DateTime64(6, 'UTC')),
			duration_ms Int64,
			upload_bytes UInt64,
			download_bytes UInt64,
			close_initiator LowCardinality(String),
			close_reason LowCardinality(String),
			error String CODEC(ZSTD(1)),
			privacy_version UInt8 DEFAULT 0,
			source_ip_bidx String DEFAULT '',
			source_ip_display String DEFAULT '',
			configured_target_host_bidx String DEFAULT '',
			configured_target_host_display String DEFAULT '',
			resolved_target_ip_bidx String DEFAULT '',
			resolved_target_ip_display String DEFAULT '',
			forwarder_outbound_ip_display String DEFAULT '',
			error_display String DEFAULT ''
		) ENGINE = ReplacingMergeTree(ingested_at)
		PARTITION BY toYYYYMMDD(event_time)
		ORDER BY (instance_id, event_time, connection_id, event_type, event_id)
		TTL event_time + INTERVAL %d DAY DELETE`, s.database, forwarderRetentionDays),
		fmt.Sprintf("ALTER TABLE %s.xray_access_events ADD COLUMN IF NOT EXISTS privacy_version UInt8 DEFAULT 0", s.database),
		fmt.Sprintf("ALTER TABLE %s.xray_access_events ADD COLUMN IF NOT EXISTS source_ip_bidx String DEFAULT ''", s.database),
		fmt.Sprintf("ALTER TABLE %s.xray_access_events ADD COLUMN IF NOT EXISTS source_network_bidx String DEFAULT ''", s.database),
		fmt.Sprintf("ALTER TABLE %s.xray_access_events ADD COLUMN IF NOT EXISTS source_ip_display String DEFAULT ''", s.database),
		fmt.Sprintf("ALTER TABLE %s.xray_access_events ADD COLUMN IF NOT EXISTS destination_host_bidx String DEFAULT ''", s.database),
		fmt.Sprintf("ALTER TABLE %s.xray_access_events ADD COLUMN IF NOT EXISTS destination_host_display String DEFAULT ''", s.database),
		fmt.Sprintf("ALTER TABLE %s.xray_access_events ADD COLUMN IF NOT EXISTS destination_ip_bidx String DEFAULT ''", s.database),
		fmt.Sprintf("ALTER TABLE %s.xray_access_events ADD COLUMN IF NOT EXISTS destination_ip_display String DEFAULT ''", s.database),
		fmt.Sprintf("ALTER TABLE %s.xray_access_events ADD COLUMN IF NOT EXISTS reason_display String DEFAULT ''", s.database),
		fmt.Sprintf("ALTER TABLE %s.xray_access_events ADD INDEX IF NOT EXISTS idx_access_event_time event_time TYPE minmax GRANULARITY 1", s.database),
		fmt.Sprintf("ALTER TABLE %s.xray_access_events ADD INDEX IF NOT EXISTS idx_access_server server_id TYPE set(1024) GRANULARITY 4", s.database),
		fmt.Sprintf("ALTER TABLE %s.xray_access_events ADD INDEX IF NOT EXISTS idx_access_user user_id TYPE minmax GRANULARITY 4", s.database),
		fmt.Sprintf("ALTER TABLE %s.xray_access_events ADD INDEX IF NOT EXISTS idx_access_source_bidx source_ip_bidx TYPE bloom_filter(0.01) GRANULARITY 4", s.database),
		fmt.Sprintf("ALTER TABLE %s.xray_access_events ADD INDEX IF NOT EXISTS idx_access_source_network_bidx source_network_bidx TYPE bloom_filter(0.01) GRANULARITY 4", s.database),
		fmt.Sprintf("ALTER TABLE %s.xray_access_events ADD INDEX IF NOT EXISTS idx_access_destination_host_bidx destination_host_bidx TYPE bloom_filter(0.01) GRANULARITY 4", s.database),
		fmt.Sprintf("ALTER TABLE %s.xray_access_events ADD INDEX IF NOT EXISTS idx_access_destination_ip_bidx destination_ip_bidx TYPE bloom_filter(0.01) GRANULARITY 4", s.database),
		fmt.Sprintf("ALTER TABLE %s.traffic_events ADD INDEX IF NOT EXISTS idx_traffic_event_time event_time TYPE minmax GRANULARITY 1", s.database),
		fmt.Sprintf("ALTER TABLE %s.traffic_events ADD INDEX IF NOT EXISTS idx_traffic_server server_id TYPE bloom_filter(0.01) GRANULARITY 4", s.database),
		fmt.Sprintf("ALTER TABLE %s.traffic_events ADD INDEX IF NOT EXISTS idx_traffic_user user_id TYPE bloom_filter(0.01) GRANULARITY 4", s.database),
		fmt.Sprintf("ALTER TABLE %s.forward_connection_events ADD INDEX IF NOT EXISTS idx_forward_event_time event_time TYPE minmax GRANULARITY 1", s.database),
		fmt.Sprintf("ALTER TABLE %s.forward_connection_events ADD INDEX IF NOT EXISTS idx_forward_connection connection_id TYPE bloom_filter(0.01) GRANULARITY 4", s.database),
		fmt.Sprintf("ALTER TABLE %s.forward_connection_events ADD INDEX IF NOT EXISTS idx_forward_rule rule_id TYPE bloom_filter(0.01) GRANULARITY 4", s.database),
		fmt.Sprintf("ALTER TABLE %s.forward_connection_events ADD INDEX IF NOT EXISTS idx_forward_source source_ip_bidx TYPE bloom_filter(0.01) GRANULARITY 4", s.database),
		fmt.Sprintf("ALTER TABLE %s.forward_connection_events ADD INDEX IF NOT EXISTS idx_forward_target configured_target_host_bidx TYPE bloom_filter(0.01) GRANULARITY 4", s.database),
	}
	for _, query := range queries {
		if err := s.conn.Exec(ctx, query); err != nil {
			return fmt.Errorf("initialize ClickHouse schema: %w", err)
		}
	}
	if err := s.backfillTrafficRollups(ctx); err != nil {
		return fmt.Errorf("backfill ClickHouse traffic rollups: %w", err)
	}
	retentionQueries := []string{
		fmt.Sprintf("ALTER TABLE %s.traffic_events MODIFY TTL event_time + INTERVAL %d HOUR DELETE", s.database, trafficRawRetentionHours),
		fmt.Sprintf("ALTER TABLE %s.traffic_minute MODIFY TTL bucket_time + INTERVAL %d DAY DELETE", s.database, trafficRetentionDays),
		fmt.Sprintf("ALTER TABLE %s.traffic_hourly MODIFY TTL bucket_time + INTERVAL %d DAY DELETE", s.database, trafficHourlyRetentionDays),
		fmt.Sprintf("ALTER TABLE %s.forward_connection_events MODIFY TTL event_time + INTERVAL %d DAY DELETE", s.database, forwarderRetentionDays),
	}
	for _, query := range retentionQueries {
		if err := s.conn.Exec(ctx, query); err != nil {
			return fmt.Errorf("update ClickHouse retention: %w", err)
		}
	}
	return nil
}

func (s *clickHouseStore) backfillTrafficRollups(ctx context.Context) error {
	rollups := []struct {
		table      string
		bucketExpr string
	}{
		{table: "traffic_minute", bucketExpr: "toStartOfMinute(event_time)"},
		{table: "traffic_hourly", bucketExpr: "toStartOfHour(event_time)"},
	}
	for _, rollup := range rollups {
		var count uint64
		if err := s.conn.QueryRow(ctx, fmt.Sprintf("SELECT count() FROM %s.%s", s.database, rollup.table)).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		query := fmt.Sprintf(`INSERT INTO %s.%s
			(bucket_time, refreshed_at, server_id, user_id, user_subscribe_id, download, upload)
			SELECT %s AS bucket_time, now64(3), server_id, user_id, user_subscribe_id, sum(download), sum(upload)
			FROM %s.traffic_events FINAL
			GROUP BY bucket_time, server_id, user_id, user_subscribe_id`,
			s.database, rollup.table, rollup.bucketExpr, s.database,
		)
		if err := s.conn.Exec(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

func (s *clickHouseStore) InsertAccessEvents(ctx context.Context, events []AccessEvent) error {
	if len(events) == 0 {
		return nil
	}
	query := fmt.Sprintf(`INSERT INTO %s.xray_access_events (
		event_id, event_time, server_id, user_id, user_subscribe_id,
		source_ip, source_port, destination_host, destination_ip,
		destination_port, network, status, outbound_tag, reason, parser_version,
		privacy_version, source_ip_bidx, source_network_bidx, source_ip_display,
		destination_host_bidx, destination_host_display,
		destination_ip_bidx, destination_ip_display, reason_display
	)`, s.database)
	batch, err := s.conn.PrepareBatch(ctx, query)
	if err != nil {
		return err
	}
	defer batch.Close()
	for _, event := range events {
		serverID, err := analyticsUint64("server id", event.ServerID)
		if err != nil {
			return err
		}
		userID, err := analyticsUint64("user id", event.UserID)
		if err != nil {
			return err
		}
		userSubscribeID, err := analyticsUint64("user subscription id", event.UserSubscribeID)
		if err != nil {
			return err
		}
		protected, err := s.protectAccessEvent(event)
		if err != nil {
			return err
		}
		if err := batch.Append(
			protected.EventID, protected.EventTime.UTC(), serverID, userID,
			userSubscribeID, protected.SourceIP, protected.SourcePort, protected.DestinationHost,
			protected.DestinationIP, protected.DestinationPort, protected.Network, protected.Status,
			protected.OutboundTag, protected.Reason, protected.ParserVersion, protected.PrivacyVersion,
			protected.SourceIPBlindIndex, protected.SourceNetworkBlindIndex, protected.SourceIPDisplay,
			protected.DestinationHostBlindIndex, protected.DestinationHostDisplay,
			protected.DestinationIPBlindIndex, protected.DestinationIPDisplay, protected.ReasonDisplay,
		); err != nil {
			return err
		}
	}
	return batch.Send()
}

func (s *clickHouseStore) InsertForwardConnectionEvents(ctx context.Context, events []ForwardConnectionEvent) error {
	if len(events) == 0 {
		return nil
	}
	query := fmt.Sprintf(`INSERT INTO %s.forward_connection_events (
		event_id, connection_id, instance_id, boot_id, event_time, event_type,
		source_ip, source_port, listen_ip, listen_port,
		configured_target_host, configured_target_port, resolved_target_ip, resolved_target_port,
		forwarder_outbound_ip, forwarder_outbound_port, rule_id, server_id, node_id,
		inbound, config_version, connected_at, closed_at, duration_ms,
		upload_bytes, download_bytes, close_initiator, close_reason, error,
		privacy_version, source_ip_bidx, source_ip_display,
		configured_target_host_bidx, configured_target_host_display,
		resolved_target_ip_bidx, resolved_target_ip_display,
		forwarder_outbound_ip_display, error_display
	)`, s.database)
	batch, err := s.conn.PrepareBatch(ctx, query)
	if err != nil {
		return err
	}
	defer batch.Close()
	for _, event := range events {
		serverID, err := analyticsUint64("server id", event.ServerID)
		if err != nil {
			return err
		}
		nodeID, err := analyticsUint64("node id", event.NodeID)
		if err != nil {
			return err
		}
		protected, err := s.protectForwardConnectionEvent(event)
		if err != nil {
			return err
		}
		if err := batch.Append(
			protected.EventID, protected.ConnectionID, protected.InstanceID, protected.BootID,
			protected.EventTime.UTC(), protected.EventType, protected.SourceIP, protected.SourcePort,
			protected.ListenIP, protected.ListenPort, protected.ConfiguredTargetHost,
			protected.ConfiguredTargetPort, protected.ResolvedTargetIP, protected.ResolvedTargetPort,
			protected.ForwarderOutboundIP, protected.ForwarderOutboundPort, protected.RuleID,
			serverID, nodeID, protected.Inbound, protected.ConfigVersion,
			nullableAnalyticsTime(protected.ConnectedAt), nullableAnalyticsTime(protected.ClosedAt),
			protected.DurationMS, protected.UploadBytes, protected.DownloadBytes,
			protected.CloseInitiator, protected.CloseReason, protected.Error,
			protected.PrivacyVersion, protected.SourceIPBlindIndex, protected.SourceIPDisplay,
			protected.ConfiguredTargetHostBlindIndex, protected.ConfiguredTargetHostDisplay,
			protected.ResolvedTargetIPBlindIndex, protected.ResolvedTargetIPDisplay,
			protected.ForwarderOutboundIPDisplay, protected.ErrorDisplay,
		); err != nil {
			return err
		}
	}
	return batch.Send()
}

func (s *clickHouseStore) QueryForwardConnectionEvents(ctx context.Context, filter ForwardConnectionFilter) (ForwardConnectionPage, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	where := []string{"event_time >= fromUnixTimestamp64Milli(?)", "event_time < fromUnixTimestamp64Milli(?)"}
	args := []any{filter.Start.UnixMilli(), filter.End.UnixMilli()}
	if filter.InstanceID != uuid.Nil {
		where, args = append(where, "instance_id = ?"), append(args, filter.InstanceID)
	}
	if filter.ConnectionID != uuid.Nil {
		where, args = append(where, "connection_id = ?"), append(args, filter.ConnectionID)
	}
	if filter.ListenPort > 0 {
		where, args = append(where, "listen_port = ?"), append(args, filter.ListenPort)
	}
	if filter.EventType != "" {
		where, args = append(where, "event_type = ?"), append(args, filter.EventType)
	}
	if filter.SourceIP != "" {
		if s.privacy.Enabled() {
			where, args = append(where, "source_ip_bidx = ?"), append(args, s.privacy.BlindIndex("forwarder_source_ip", filter.SourceIP))
		} else {
			where, args = append(where, "source_ip = ?"), append(args, filter.SourceIP)
		}
	}
	if filter.Destination != "" {
		if s.privacy.Enabled() {
			index := s.privacy.BlindIndex("forwarder_destination", strings.ToLower(filter.Destination))
			where = append(where, "(configured_target_host_bidx = ? OR resolved_target_ip_bidx = ?)")
			args = append(args, index, index)
		} else {
			where = append(where, "(positionCaseInsensitive(configured_target_host, ?) > 0 OR resolved_target_ip = ?)")
			args = append(args, filter.Destination, filter.Destination)
		}
	}
	if !filter.CursorTime.IsZero() && filter.CursorID != uuid.Nil {
		where = append(where, "(event_time < fromUnixTimestamp64Milli(?) OR (event_time = fromUnixTimestamp64Milli(?) AND event_id < ?))")
		args = append(args, filter.CursorTime.UnixMilli(), filter.CursorTime.UnixMilli(), filter.CursorID)
	}
	query := fmt.Sprintf(`SELECT event_id, connection_id, instance_id, boot_id, event_time, event_type,
		if(privacy_version = 1, source_ip_display, source_ip), source_port, listen_ip, listen_port,
		if(privacy_version = 1, configured_target_host_display, configured_target_host), configured_target_port,
		if(privacy_version = 1, resolved_target_ip_display, resolved_target_ip), resolved_target_port,
		if(privacy_version = 1, forwarder_outbound_ip_display, forwarder_outbound_ip), forwarder_outbound_port,
		rule_id, server_id, node_id, inbound, config_version, connected_at, closed_at,
		duration_ms, upload_bytes, download_bytes, close_initiator, close_reason,
		if(privacy_version = 1, error_display, error)
		FROM %s.forward_connection_events FINAL
		WHERE %s
		ORDER BY event_time DESC, event_id DESC
		LIMIT ?`, s.database, strings.Join(where, " AND "))
	args = append(args, limit+1)
	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return ForwardConnectionPage{}, err
	}
	defer rows.Close()
	list := make([]ForwardConnectionEvent, 0, limit+1)
	for rows.Next() {
		var event ForwardConnectionEvent
		var serverID, nodeID uint64
		var connectedAt, closedAt *time.Time
		if err := rows.Scan(
			&event.EventID, &event.ConnectionID, &event.InstanceID, &event.BootID,
			&event.EventTime, &event.EventType, &event.SourceIP, &event.SourcePort,
			&event.ListenIP, &event.ListenPort, &event.ConfiguredTargetHost,
			&event.ConfiguredTargetPort, &event.ResolvedTargetIP, &event.ResolvedTargetPort,
			&event.ForwarderOutboundIP, &event.ForwarderOutboundPort, &event.RuleID,
			&serverID, &nodeID, &event.Inbound, &event.ConfigVersion, &connectedAt, &closedAt,
			&event.DurationMS, &event.UploadBytes, &event.DownloadBytes,
			&event.CloseInitiator, &event.CloseReason, &event.Error,
		); err != nil {
			return ForwardConnectionPage{}, err
		}
		event.ServerID, err = analyticsInt64("server id", serverID)
		if err != nil {
			return ForwardConnectionPage{}, err
		}
		event.NodeID, err = analyticsInt64("node id", nodeID)
		if err != nil {
			return ForwardConnectionPage{}, err
		}
		if connectedAt != nil {
			event.ConnectedAt = connectedAt.UTC()
		}
		if closedAt != nil {
			event.ClosedAt = closedAt.UTC()
		}
		list = append(list, event)
	}
	if err := rows.Err(); err != nil {
		return ForwardConnectionPage{}, err
	}
	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	return ForwardConnectionPage{List: list, HasMore: hasMore}, nil
}

func (s *clickHouseStore) protectForwardConnectionEvent(event ForwardConnectionEvent) (protectedForwardConnectionEvent, error) {
	protected := protectedForwardConnectionEvent{ForwardConnectionEvent: event}
	if !s.privacy.Enabled() {
		return protected, nil
	}
	protected.PrivacyVersion = forwarderPrivacyVersion
	var err error
	protected.SourceIP, err = s.privacy.Seal(forwarderPrivacyScope(event.EventID, "source_ip"), event.SourceIP)
	if err != nil {
		return protectedForwardConnectionEvent{}, fmt.Errorf("encrypt forwarder source IP: %w", err)
	}
	protected.ConfiguredTargetHost, err = s.privacy.Seal(forwarderPrivacyScope(event.EventID, "configured_target_host"), event.ConfiguredTargetHost)
	if err != nil {
		return protectedForwardConnectionEvent{}, fmt.Errorf("encrypt forwarder configured target: %w", err)
	}
	protected.ResolvedTargetIP, err = s.privacy.Seal(forwarderPrivacyScope(event.EventID, "resolved_target_ip"), event.ResolvedTargetIP)
	if err != nil {
		return protectedForwardConnectionEvent{}, fmt.Errorf("encrypt forwarder resolved target: %w", err)
	}
	protected.ForwarderOutboundIP, err = s.privacy.Seal(forwarderPrivacyScope(event.EventID, "forwarder_outbound_ip"), event.ForwarderOutboundIP)
	if err != nil {
		return protectedForwardConnectionEvent{}, fmt.Errorf("encrypt forwarder outbound IP: %w", err)
	}
	protected.Error, err = s.privacy.Seal(forwarderPrivacyScope(event.EventID, "error"), event.Error)
	if err != nil {
		return protectedForwardConnectionEvent{}, fmt.Errorf("encrypt forwarder error: %w", err)
	}
	protected.SourceIPBlindIndex = s.privacy.BlindIndex("forwarder_source_ip", event.SourceIP)
	protected.SourceIPDisplay = privacy.MaskIP(event.SourceIP)
	protected.ConfiguredTargetHostBlindIndex = s.privacy.BlindIndex("forwarder_destination", strings.ToLower(event.ConfiguredTargetHost))
	protected.ConfiguredTargetHostDisplay = maskAnalyticsHost(event.ConfiguredTargetHost)
	protected.ResolvedTargetIPBlindIndex = s.privacy.BlindIndex("forwarder_destination", strings.ToLower(event.ResolvedTargetIP))
	protected.ResolvedTargetIPDisplay = privacy.MaskIP(event.ResolvedTargetIP)
	protected.ForwarderOutboundIPDisplay = privacy.MaskIP(event.ForwarderOutboundIP)
	protected.ErrorDisplay = privacy.MaskIdentifier(event.Error)
	return protected, nil
}

func forwarderPrivacyScope(eventID uuid.UUID, field string) string {
	return "forward_connection_events." + field + ":" + eventID.String()
}

func maskAnalyticsHost(value string) string {
	if value == "" {
		return ""
	}
	if strings.Contains(value, ":") || strings.Count(value, ".") == 3 {
		return privacy.MaskIP(value)
	}
	return privacy.MaskIdentifier(value)
}

func nullableAnalyticsTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC()
}

func (s *clickHouseStore) refreshTrafficRollups(ctx context.Context, events []TrafficEvent) error {
	if len(events) == 0 {
		return nil
	}
	start := events[0].EventTime.UTC()
	end := start
	for index := 1; index < len(events); index++ {
		eventTime := events[index].EventTime.UTC()
		if eventTime.Before(start) {
			start = eventTime
		}
		if eventTime.After(end) {
			end = eventTime
		}
	}
	rollups := []struct {
		table      string
		bucketExpr string
		interval   time.Duration
	}{
		{table: "traffic_minute", bucketExpr: "toStartOfMinute(event_time)", interval: time.Minute},
		{table: "traffic_hourly", bucketExpr: "toStartOfHour(event_time)", interval: time.Hour},
	}
	for _, rollup := range rollups {
		windowStart := start.Truncate(rollup.interval)
		windowEnd := end.Truncate(rollup.interval).Add(rollup.interval)
		query := fmt.Sprintf(`INSERT INTO %s.%s
			(bucket_time, refreshed_at, server_id, user_id, user_subscribe_id, download, upload)
			SELECT %s AS bucket_time, now64(3), server_id, user_id, user_subscribe_id, sum(download), sum(upload)
			FROM %s.traffic_events FINAL
			WHERE event_time >= fromUnixTimestamp64Milli(?) AND event_time < fromUnixTimestamp64Milli(?)
			GROUP BY bucket_time, server_id, user_id, user_subscribe_id`,
			s.database, rollup.table, rollup.bucketExpr, s.database,
		)
		if err := s.conn.Exec(ctx, query, windowStart.UnixMilli(), windowEnd.UnixMilli()); err != nil {
			return err
		}
	}
	return nil
}

func (s *clickHouseStore) QueryAccessEvents(ctx context.Context, filter AccessFilter) (AccessPage, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	where := []string{"event_time >= fromUnixTimestamp64Milli(?)", "event_time < fromUnixTimestamp64Milli(?)"}
	args := []any{filter.Start.UnixMilli(), filter.End.UnixMilli()}
	where, args = s.appendAccessFilters(where, args, filter)
	sourceIPExpression := "if(privacy_version = 1, source_ip_display, source_ip)"
	destinationHostExpression := "if(privacy_version = 1, destination_host_display, destination_host)"
	destinationIPExpression := "if(privacy_version = 1, destination_ip_display, destination_ip)"
	reasonExpression := "if(privacy_version = 1, reason_display, reason)"
	if filter.RevealSensitive {
		sourceIPExpression = "source_ip"
		destinationHostExpression = "destination_host"
		destinationIPExpression = "destination_ip"
		reasonExpression = "reason"
	}
	query := fmt.Sprintf(`SELECT event_id, event_time, server_id, user_id, user_subscribe_id,
		%s, source_port, %s, %s, destination_port,
		network, status, outbound_tag, %s, parser_version, privacy_version
		FROM %s.xray_access_events FINAL
		WHERE %s
		ORDER BY event_time DESC, event_id DESC
		LIMIT ?`, sourceIPExpression, destinationHostExpression, destinationIPExpression,
		reasonExpression, s.database, strings.Join(where, " AND "))
	args = append(args, limit+1)
	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return AccessPage{}, err
	}
	defer rows.Close()
	list := make([]AccessEvent, 0, limit+1)
	for rows.Next() {
		var event AccessEvent
		var serverID, userID, userSubscribeID uint64
		var privacyVersion uint8
		if err := rows.Scan(
			&event.EventID, &event.EventTime, &serverID, &userID, &userSubscribeID,
			&event.SourceIP, &event.SourcePort, &event.DestinationHost, &event.DestinationIP,
			&event.DestinationPort, &event.Network, &event.Status, &event.OutboundTag,
			&event.Reason, &event.ParserVersion, &privacyVersion,
		); err != nil {
			return AccessPage{}, err
		}
		if err := assignAccessEventIDs(&event, serverID, userID, userSubscribeID); err != nil {
			return AccessPage{}, err
		}
		if filter.RevealSensitive {
			switch privacyVersion {
			case 0:
			case accessPrivacyVersion:
				if err := s.openAccessEvent(&event); err != nil {
					return AccessPage{}, err
				}
			default:
				return AccessPage{}, fmt.Errorf("unsupported Xray access privacy version %d", privacyVersion)
			}
			event.Revealed = true
		}
		list = append(list, event)
	}
	if err := rows.Err(); err != nil {
		return AccessPage{}, err
	}
	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	return AccessPage{List: list, HasMore: hasMore}, nil
}

func (s *clickHouseStore) openAccessEvent(event *AccessEvent) error {
	if event == nil {
		return nil
	}
	fields := []struct {
		name  string
		value *string
	}{
		{name: "source_ip", value: &event.SourceIP},
		{name: "destination_host", value: &event.DestinationHost},
		{name: "destination_ip", value: &event.DestinationIP},
		{name: "reason", value: &event.Reason},
	}
	for _, field := range fields {
		opened, err := s.privacy.Open(accessPrivacyScope(event.EventID, field.name), *field.value)
		if err != nil {
			return fmt.Errorf("decrypt Xray access %s: %w", field.name, err)
		}
		*field.value = opened
	}
	return nil
}

func (s *clickHouseStore) protectAccessEvent(event AccessEvent) (protectedAccessEvent, error) {
	protected := protectedAccessEvent{AccessEvent: event}
	if !s.privacy.Enabled() {
		return protected, nil
	}

	protected.PrivacyVersion = accessPrivacyVersion
	var err error
	protected.SourceIP, err = s.privacy.Seal(accessPrivacyScope(event.EventID, "source_ip"), event.SourceIP)
	if err != nil {
		return protectedAccessEvent{}, fmt.Errorf("encrypt Xray source IP: %w", err)
	}
	protected.DestinationHost, err = s.privacy.Seal(accessPrivacyScope(event.EventID, "destination_host"), event.DestinationHost)
	if err != nil {
		return protectedAccessEvent{}, fmt.Errorf("encrypt Xray destination host: %w", err)
	}
	protected.DestinationIP, err = s.privacy.Seal(accessPrivacyScope(event.EventID, "destination_ip"), event.DestinationIP)
	if err != nil {
		return protectedAccessEvent{}, fmt.Errorf("encrypt Xray destination IP: %w", err)
	}
	protected.Reason, err = s.privacy.Seal(accessPrivacyScope(event.EventID, "reason"), event.Reason)
	if err != nil {
		return protectedAccessEvent{}, fmt.Errorf("encrypt Xray reason: %w", err)
	}

	protected.SourceIPBlindIndex = s.privacy.BlindIndex("xray_source_ip", event.SourceIP)
	if sourceNetwork, ok := privacy.NormalizeIPNetwork(event.SourceIP); ok {
		protected.SourceNetworkBlindIndex = s.privacy.BlindIndex("xray_source_network", sourceNetwork)
	}
	protected.SourceIPDisplay = privacy.MaskIP(event.SourceIP)
	protected.DestinationHostBlindIndex = s.privacy.BlindIndex("xray_destination", strings.ToLower(event.DestinationHost))
	protected.DestinationHostDisplay = privacy.MaskIdentifier(event.DestinationHost)
	protected.DestinationIPBlindIndex = s.privacy.BlindIndex("xray_destination", strings.ToLower(event.DestinationIP))
	protected.DestinationIPDisplay = privacy.MaskIP(event.DestinationIP)
	protected.ReasonDisplay = privacy.MaskIdentifier(event.Reason)
	return protected, nil
}

func accessPrivacyScope(eventID uuid.UUID, field string) string {
	return "xray_access_events." + field + ":" + eventID.String()
}

func (s *clickHouseStore) migrateAccessPrivacy(ctx context.Context) error {
	if !s.privacy.Enabled() {
		return nil
	}
	for {
		query := fmt.Sprintf(`SELECT event_id, event_time, server_id, user_id, user_subscribe_id,
			source_ip, source_port, destination_host, destination_ip, destination_port,
			network, status, outbound_tag, reason, parser_version
			FROM %s.xray_access_events FINAL
			WHERE privacy_version = 0
			LIMIT %d`, s.database, accessPrivacyMigrationBatch)
		rows, err := s.conn.Query(ctx, query)
		if err != nil {
			return fmt.Errorf("query legacy Xray access events: %w", err)
		}
		events := make([]AccessEvent, 0, accessPrivacyMigrationBatch)
		for rows.Next() {
			var event AccessEvent
			var serverID, userID, userSubscribeID uint64
			if err := rows.Scan(
				&event.EventID, &event.EventTime, &serverID, &userID, &userSubscribeID,
				&event.SourceIP, &event.SourcePort, &event.DestinationHost, &event.DestinationIP,
				&event.DestinationPort, &event.Network, &event.Status, &event.OutboundTag,
				&event.Reason, &event.ParserVersion,
			); err != nil {
				_ = rows.Close()
				return fmt.Errorf("scan legacy Xray access event: %w", err)
			}
			if err := assignAccessEventIDs(&event, serverID, userID, userSubscribeID); err != nil {
				_ = rows.Close()
				return err
			}
			events = append(events, event)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return fmt.Errorf("iterate legacy Xray access events: %w", err)
		}
		_ = rows.Close()
		if len(events) == 0 {
			return nil
		}
		if err := s.InsertAccessEvents(ctx, events); err != nil {
			return fmt.Errorf("rewrite legacy Xray access events: %w", err)
		}
	}
}

func (s *clickHouseStore) appendAccessFilters(where []string, args []any, filter AccessFilter) ([]string, []any) {
	if filter.ServerID > 0 {
		where, args = append(where, "server_id = ?"), append(args, filter.ServerID)
	}
	if filter.UserID > 0 {
		where, args = append(where, "user_id = ?"), append(args, filter.UserID)
	}
	if filter.UserSubscribeID > 0 {
		where, args = append(where, "user_subscribe_id = ?"), append(args, filter.UserSubscribeID)
	}
	if filter.SourceIP != "" {
		if s.privacy.Enabled() {
			where, args = append(where, "source_ip_bidx = ?"), append(args, s.privacy.BlindIndex("xray_source_ip", filter.SourceIP))
		} else {
			where, args = append(where, "source_ip = ?"), append(args, filter.SourceIP)
		}
	}
	if filter.Destination != "" {
		if s.privacy.Enabled() {
			index := s.privacy.BlindIndex("xray_destination", strings.ToLower(filter.Destination))
			where, args = append(where, "(destination_host_bidx = ? OR destination_ip_bidx = ?)"), append(args, index, index)
		} else {
			where, args = append(where, "(lowerUTF8(destination_host) = lowerUTF8(?) OR destination_ip = ?)"), append(args, filter.Destination, filter.Destination)
		}
	}
	if filter.Status != "" {
		where, args = append(where, "status = ?"), append(args, filter.Status)
	}
	if !filter.CursorTime.IsZero() && filter.CursorID != uuid.Nil {
		where = append(where, "(event_time < fromUnixTimestamp64Milli(?) OR (event_time = fromUnixTimestamp64Milli(?) AND event_id < ?))")
		args = append(args, filter.CursorTime.UnixMilli(), filter.CursorTime.UnixMilli(), filter.CursorID)
	}
	return where, args
}

func (s *clickHouseStore) InsertTrafficEvents(ctx context.Context, events []TrafficEvent) error {
	if len(events) == 0 {
		return nil
	}
	query := fmt.Sprintf(`INSERT INTO %s.traffic_events (
		event_id, event_time, server_id, user_id, user_subscribe_id, download, upload
	)`, s.database)
	batch, err := s.conn.PrepareBatch(ctx, query)
	if err != nil {
		return err
	}
	defer batch.Close()
	for _, event := range events {
		serverID, err := analyticsUint64("server id", event.ServerID)
		if err != nil {
			return err
		}
		userID, err := analyticsUint64("user id", event.UserID)
		if err != nil {
			return err
		}
		userSubscribeID, err := analyticsUint64("user subscription id", event.UserSubscribeID)
		if err != nil {
			return err
		}
		download, err := analyticsUint64("download", event.Download)
		if err != nil {
			return err
		}
		upload, err := analyticsUint64("upload", event.Upload)
		if err != nil {
			return err
		}
		if err := batch.Append(
			event.EventID, event.EventTime.UTC(), serverID, userID,
			userSubscribeID, download, upload,
		); err != nil {
			return err
		}
	}
	if err := batch.Send(); err != nil {
		return err
	}
	return s.refreshTrafficRollups(ctx, events)
}

func (s *clickHouseStore) QueryTrafficEvents(ctx context.Context, filter TrafficFilter) (TrafficPage, error) {
	limit, offset := normalizeTrafficPage(filter.Limit, filter.Offset)
	where := []string{"event_time >= fromUnixTimestamp64Milli(?)", "event_time < fromUnixTimestamp64Milli(?)"}
	args := []any{filter.Start.UnixMilli(), filter.End.UnixMilli()}
	if filter.ServerID > 0 {
		where, args = append(where, "server_id = ?"), append(args, filter.ServerID)
	}
	if filter.UserID > 0 {
		where, args = append(where, "user_id = ?"), append(args, filter.UserID)
	}
	if filter.UserSubscribeID > 0 {
		where, args = append(where, "user_subscribe_id = ?"), append(args, filter.UserSubscribeID)
	}
	condition := strings.Join(where, " AND ")
	var total uint64
	countQuery := fmt.Sprintf("SELECT count() FROM %s.traffic_events FINAL WHERE %s", s.database, condition)
	if err := s.conn.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return TrafficPage{}, err
	}
	totalRows, err := analyticsInt64("traffic row count", total)
	if err != nil {
		return TrafficPage{}, err
	}
	capacity, ok := trafficPageCapacity(total, limit, offset)
	if !ok {
		return TrafficPage{List: []TrafficEvent{}, Total: totalRows}, nil
	}
	query := fmt.Sprintf(`SELECT event_id, event_time, server_id, user_id, user_subscribe_id, download, upload
		FROM %s.traffic_events FINAL WHERE %s
		ORDER BY event_time DESC, event_id DESC LIMIT ? OFFSET ?`, s.database, condition)
	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := s.conn.Query(ctx, query, queryArgs...)
	if err != nil {
		return TrafficPage{}, err
	}
	defer rows.Close()
	list := make([]TrafficEvent, 0, capacity)
	for rows.Next() {
		var event TrafficEvent
		var serverID, userID, userSubscribeID, download, upload uint64
		if err := rows.Scan(&event.EventID, &event.EventTime, &serverID, &userID, &userSubscribeID, &download, &upload); err != nil {
			return TrafficPage{}, err
		}
		if err := assignTrafficEventValues(&event, serverID, userID, userSubscribeID, download, upload); err != nil {
			return TrafficPage{}, err
		}
		list = append(list, event)
	}
	if err := rows.Err(); err != nil {
		return TrafficPage{}, err
	}
	return TrafficPage{List: list, Total: totalRows}, nil
}

func normalizeTrafficPage(limit, offset int) (int, int) {
	if limit <= 0 || limit > 200 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func trafficPageCapacity(total uint64, limit, offset int) (int, bool) {
	// normalizeTrafficPage guarantees non-negative offset and a limit in [1, 200].
	if total == 0 || uint64(offset) >= total { // #nosec G115 -- non-negative int always fits uint64
		return 0, false
	}
	remaining := total - uint64(offset) // #nosec G115 -- non-negative int always fits uint64
	if remaining < uint64(limit) {      // #nosec G115 -- positive bounded int always fits uint64
		return int(remaining), true // #nosec G115 -- remaining is less than limit, which is at most 200
	}
	return limit, true
}

func (s *clickHouseStore) QueryTrafficSeries(ctx context.Context, filter TrafficSeriesFilter) (TrafficSeries, error) {
	intervalMillis := filter.Interval.Milliseconds()
	if intervalMillis <= 0 {
		return TrafficSeries{}, fmt.Errorf("traffic series interval must be positive")
	}
	table, timestampColumn := trafficSeriesSource(filter.Interval)
	where := []string{
		fmt.Sprintf("%s >= fromUnixTimestamp64Milli(?)", timestampColumn),
		fmt.Sprintf("%s < fromUnixTimestamp64Milli(?)", timestampColumn),
	}
	args := []any{filter.Start.UnixMilli(), filter.End.UnixMilli()}
	if filter.UserID > 0 {
		where, args = append(where, "user_id = ?"), append(args, filter.UserID)
	}
	if filter.UserSubscribeID > 0 {
		where, args = append(where, "user_subscribe_id = ?"), append(args, filter.UserSubscribeID)
	}
	query := fmt.Sprintf(`SELECT
		intDiv(toUnixTimestamp64Milli(%s), ?) * ? AS bucket_millis,
		sum(download), sum(upload)
		FROM %s.%s FINAL
		WHERE %s
		GROUP BY bucket_millis
		ORDER BY bucket_millis`, timestampColumn, s.database, table, strings.Join(where, " AND "))
	queryArgs := append([]any{intervalMillis, intervalMillis}, args...)
	rows, err := s.conn.Query(ctx, query, queryArgs...)
	if err != nil {
		return TrafficSeries{}, err
	}
	defer rows.Close()

	list := make([]TrafficSeriesPoint, 0)
	var totalDownload, totalUpload int64
	for rows.Next() {
		var timestamp int64
		var download, upload uint64
		if err := rows.Scan(&timestamp, &download, &upload); err != nil {
			return TrafficSeries{}, err
		}
		downloadValue, err := analyticsInt64("download", download)
		if err != nil {
			return TrafficSeries{}, err
		}
		uploadValue, err := analyticsInt64("upload", upload)
		if err != nil {
			return TrafficSeries{}, err
		}
		if totalDownload > math.MaxInt64-downloadValue || totalUpload > math.MaxInt64-uploadValue {
			return TrafficSeries{}, fmt.Errorf("traffic series total exceeds int64")
		}
		point := TrafficSeriesPoint{
			Timestamp: time.UnixMilli(timestamp).UTC(),
			Download:  downloadValue,
			Upload:    uploadValue,
		}
		totalDownload += point.Download
		totalUpload += point.Upload
		list = append(list, point)
	}
	if err := rows.Err(); err != nil {
		return TrafficSeries{}, err
	}
	return TrafficSeries{List: list, Download: totalDownload, Upload: totalUpload}, nil
}

func trafficSeriesSource(interval time.Duration) (table string, timestampColumn string) {
	switch {
	case interval >= time.Hour:
		return "traffic_hourly", "bucket_time"
	case interval >= time.Minute:
		return "traffic_minute", "bucket_time"
	default:
		return "traffic_events", "event_time"
	}
}

func analyticsUint64(name string, value int64) (uint64, error) {
	if value < 0 {
		return 0, fmt.Errorf("analytics %s must not be negative", name)
	}
	return uint64(value), nil // #nosec G115 -- the negative range is rejected above.
}

func analyticsInt64(name string, value uint64) (int64, error) {
	if value > math.MaxInt64 {
		return 0, fmt.Errorf("analytics %s exceeds int64", name)
	}
	return int64(value), nil // #nosec G115 -- values above MaxInt64 are rejected above.
}

func assignAccessEventIDs(event *AccessEvent, serverID, userID, userSubscribeID uint64) error {
	var err error
	event.ServerID, err = analyticsInt64("server id", serverID)
	if err != nil {
		return err
	}
	event.UserID, err = analyticsInt64("user id", userID)
	if err != nil {
		return err
	}
	event.UserSubscribeID, err = analyticsInt64("user subscription id", userSubscribeID)
	return err
}

func assignTrafficEventValues(event *TrafficEvent, serverID, userID, userSubscribeID, download, upload uint64) error {
	var err error
	event.ServerID, err = analyticsInt64("server id", serverID)
	if err != nil {
		return err
	}
	event.UserID, err = analyticsInt64("user id", userID)
	if err != nil {
		return err
	}
	event.UserSubscribeID, err = analyticsInt64("user subscription id", userSubscribeID)
	if err != nil {
		return err
	}
	event.Download, err = analyticsInt64("download", download)
	if err != nil {
		return err
	}
	event.Upload, err = analyticsInt64("upload", upload)
	return err
}
