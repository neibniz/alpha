package accesslog

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/neibniz/alpha/xray-agent/internal/panel"
)

const ParserVersion uint16 = 1

func Parse(line string) (panel.AccessEvent, error) {
	line = strings.TrimSpace(line)
	fromAt := strings.Index(line, " from ")
	if fromAt <= 0 {
		return panel.AccessEvent{}, fmt.Errorf("missing access source")
	}
	eventTime, err := time.ParseInLocation("2006/01/02 15:04:05.999999999", line[:fromAt], time.Local)
	if err != nil {
		return panel.AccessEvent{}, fmt.Errorf("parse access timestamp: %w", err)
	}
	remainder := line[fromAt+len(" from "):]
	status, source, tail, ok := splitStatus(remainder)
	if !ok {
		return panel.AccessEvent{}, fmt.Errorf("missing access status")
	}

	event := panel.AccessEvent{Timestamp: eventTime.UnixMilli(), Status: status, ParserVersion: ParserVersion}
	_, event.SourceIP, event.SourcePort = parseEndpoint(source)
	tail, event.UserSubscribeID = stripUser(tail)
	if !strings.HasPrefix(tail, " ") {
		if destination, rest, found := strings.Cut(tail, " "); found {
			event.Network, event.DestinationHost, event.DestinationPort = parseEndpoint(destination)
			tail = strings.TrimSpace(rest)
		} else if tail != "" {
			event.Network, event.DestinationHost, event.DestinationPort = parseEndpoint(tail)
			tail = ""
		}
	} else {
		tail = strings.TrimSpace(tail)
	}
	if ip := net.ParseIP(event.DestinationHost); ip != nil {
		event.DestinationIP = ip.String()
		event.DestinationHost = ""
	}
	if strings.HasPrefix(tail, "[") {
		if end := strings.IndexByte(tail, ']'); end > 1 {
			event.OutboundTag = tail[1:end]
			tail = strings.TrimSpace(tail[end+1:])
		}
	}
	event.Reason = tail
	return event, nil
}

func splitStatus(value string) (status, source, tail string, ok bool) {
	accepted := strings.Index(value, " accepted ")
	rejected := strings.Index(value, " rejected ")
	index := -1
	marker := ""
	switch {
	case accepted >= 0 && (rejected < 0 || accepted < rejected):
		index, marker, status = accepted, " accepted ", "accepted"
	case rejected >= 0:
		index, marker, status = rejected, " rejected ", "rejected"
	default:
		return "", "", "", false
	}
	return status, strings.TrimSpace(value[:index]), value[index+len(marker):], true
}

func stripUser(value string) (string, int64) {
	marker := " email: "
	index := strings.LastIndex(value, marker)
	if index < 0 {
		return value, 0
	}
	email := strings.TrimSpace(value[index+len(marker):])
	if !strings.HasPrefix(email, "u_") {
		return value[:index], 0
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(email, "u_"), 10, 64)
	if err != nil || id <= 0 {
		return value[:index], 0
	}
	return value[:index], id
}

func parseEndpoint(value string) (network, host string, port uint16) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", 0
	}
	if prefix, rest, ok := strings.Cut(value, ":"); ok && (prefix == "tcp" || prefix == "udp") {
		network, value = prefix, rest
	}
	parsedHost, parsedPort, err := net.SplitHostPort(value)
	if err != nil {
		return network, strings.Trim(value, "[]"), 0
	}
	n, err := strconv.ParseUint(parsedPort, 10, 16)
	if err != nil {
		return network, strings.Trim(parsedHost, "[]"), 0
	}
	return network, strings.Trim(parsedHost, "[]"), uint16(n)
}
