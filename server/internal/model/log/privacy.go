package log

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type systemLogPrivateData struct {
	IP         string `json:"ip,omitempty"`
	Identifier string `json:"identifier,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
	Content    string `json:"content,omitempty"`
}

type subscribeAccessPrivateData struct {
	ClientIP  string `json:"client_ip,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

func systemLogPrivacyScope(reference string) string {
	return "system_logs.private_data:" + reference
}

func subscribeAccessPrivacyScope(reference string) string {
	return "subscribe_access_logs.private_data:" + reference
}

func sensitiveSystemLogType(logType Type) bool {
	switch logType {
	case TypeEmailMessage, TypeLogin, TypeRegister, TypeLogout, TypeSubscribe:
		return true
	default:
		return false
	}
}

func (m *defaultLogModel) protectSystemLog(data *SystemLog) error {
	if data == nil || !m.privacyCodec.Enabled() || !sensitiveSystemLogType(Type(data.Type)) {
		return nil
	}
	private := systemLogPrivateData{
		IP:         data.IP,
		Identifier: data.Identifier,
		UserAgent:  data.UserAgent,
	}
	if Type(data.Type) == TypeEmailMessage {
		private.Content = data.Content
	}
	if private.IP == "" && private.Identifier == "" && private.UserAgent == "" && strings.TrimSpace(private.Content) == "" {
		return nil
	}
	payload, err := json.Marshal(private)
	if err != nil {
		return fmt.Errorf("encode system log private data: %w", err)
	}
	reference := uuid.NewString()
	sealed, err := m.privacyCodec.Seal(systemLogPrivacyScope(reference), string(payload))
	if err != nil {
		return fmt.Errorf("encrypt system log private data: %w", err)
	}
	data.PrivacyRef = reference
	data.PrivateData = sealed
	return nil
}

func (m *defaultLogModel) protectSubscribeAccessLog(data *SubscribeAccessLog) error {
	if data == nil || !m.privacyCodec.Enabled() {
		return nil
	}
	private := subscribeAccessPrivateData{
		ClientIP:  data.ClientIP,
		UserAgent: data.UserAgent,
	}
	if private.ClientIP == "" && private.UserAgent == "" {
		return nil
	}
	payload, err := json.Marshal(private)
	if err != nil {
		return fmt.Errorf("encode subscribe access private data: %w", err)
	}
	reference := uuid.NewString()
	sealed, err := m.privacyCodec.Seal(subscribeAccessPrivacyScope(reference), string(payload))
	if err != nil {
		return fmt.Errorf("encrypt subscribe access private data: %w", err)
	}
	data.PrivacyRef = reference
	data.PrivateData = sealed
	return nil
}

func (m *defaultLogModel) openSystemLogPrivate(reference, sealed string) (*systemLogPrivateData, error) {
	if sealed == "" {
		return nil, nil
	}
	if reference == "" {
		return nil, fmt.Errorf("system log private data reference is missing")
	}
	payload, err := m.privacyCodec.Open(systemLogPrivacyScope(reference), sealed)
	if err != nil {
		return nil, fmt.Errorf("decrypt system log private data: %w", err)
	}
	var private systemLogPrivateData
	if err := json.Unmarshal([]byte(payload), &private); err != nil {
		return nil, fmt.Errorf("decode system log private data: %w", err)
	}
	return &private, nil
}

func (m *defaultLogModel) revealLoginRows(rows []LoginLogRow) error {
	for index := range rows {
		private, err := m.openSystemLogPrivate(rows[index].PrivacyRef, rows[index].PrivateData)
		if err != nil {
			return err
		}
		if private == nil {
			continue
		}
		rows[index].LoginIP = private.IP
		rows[index].UserAgent = private.UserAgent
		rows[index].Revealed = true
	}
	return nil
}

func (m *defaultLogModel) revealRegisterRows(rows []RegisterLogRow) error {
	for index := range rows {
		private, err := m.openSystemLogPrivate(rows[index].PrivacyRef, rows[index].PrivateData)
		if err != nil {
			return err
		}
		if private == nil {
			continue
		}
		rows[index].Identifier = private.Identifier
		rows[index].RegisterIP = private.IP
		rows[index].UserAgent = private.UserAgent
		rows[index].Revealed = true
	}
	return nil
}

func (m *defaultLogModel) revealMessageRows(rows []MessageLogRow) error {
	for index := range rows {
		private, err := m.openSystemLogPrivate(rows[index].PrivacyRef, rows[index].PrivateData)
		if err != nil {
			return err
		}
		if private == nil || strings.TrimSpace(private.Content) == "" {
			continue
		}
		var message Message
		if err := message.Unmarshal([]byte(private.Content)); err != nil {
			return fmt.Errorf("decode private message log content: %w", err)
		}
		rows[index].Platform = message.Platform
		rows[index].To = message.To
		rows[index].Subject = message.Subject
		rows[index].Content = string(message.Content)
		rows[index].Status = message.Status
		rows[index].Revealed = true
	}
	return nil
}

func (m *defaultLogModel) revealSubscribeAccessLogs(rows []*SubscribeAccessLog) error {
	for _, row := range rows {
		if row == nil || row.PrivateData == "" {
			continue
		}
		if row.PrivacyRef == "" {
			return fmt.Errorf("subscribe access private data reference is missing")
		}
		payload, err := m.privacyCodec.Open(subscribeAccessPrivacyScope(row.PrivacyRef), row.PrivateData)
		if err != nil {
			return fmt.Errorf("decrypt subscribe access private data: %w", err)
		}
		var private subscribeAccessPrivateData
		if err := json.Unmarshal([]byte(payload), &private); err != nil {
			return fmt.Errorf("decode subscribe access private data: %w", err)
		}
		row.ClientIP = private.ClientIP
		row.UserAgent = private.UserAgent
		row.Revealed = true
	}
	return nil
}
