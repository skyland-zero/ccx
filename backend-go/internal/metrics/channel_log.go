package metrics

import (
	"sync"
	"time"
)

// ChannelLog 单次上游请求日志
type ChannelLog struct {
	Timestamp     time.Time `json:"timestamp"`
	Model         string    `json:"model"`                   // 实际使用的模型（重定向后）
	OriginalModel string    `json:"originalModel,omitempty"` // 原始请求模型（仅当重定向时有值）
	StatusCode    int       `json:"statusCode"`
	DurationMs    int64     `json:"durationMs"`
	Success       bool      `json:"success"`
	KeyMask       string    `json:"keyMask"`
	BaseURL       string    `json:"baseUrl"`
	ErrorInfo     string    `json:"errorInfo"`
	IsRetry       bool      `json:"isRetry"`
	InterfaceType string    `json:"interfaceType"` // 接口类型（Messages/Responses/Gemini）
}

const maxChannelLogs = 50

// ChannelLogStore 渠道日志存储（内存环形缓冲区）
type ChannelLogStore struct {
	mu   sync.RWMutex
	logs map[int][]*ChannelLog // key: channelIndex
}

func NewChannelLogStore() *ChannelLogStore {
	return &ChannelLogStore{logs: make(map[int][]*ChannelLog)}
}

func (s *ChannelLogStore) Record(channelIndex int, log *ChannelLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs[channelIndex] = append(s.logs[channelIndex], log)
	if len(s.logs[channelIndex]) > maxChannelLogs {
		s.logs[channelIndex] = s.logs[channelIndex][len(s.logs[channelIndex])-maxChannelLogs:]
	}
}

// RemoveAndShift 删除指定渠道日志，并将其后的渠道日志索引前移一位，保持与删除后的渠道切片索引一致。
func (s *ChannelLogStore) RemoveAndShift(channelIndex int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.logs) == 0 {
		return
	}

	shifted := make(map[int][]*ChannelLog, len(s.logs))
	for idx, logs := range s.logs {
		switch {
		case idx == channelIndex:
			continue
		case idx > channelIndex:
			shifted[idx-1] = logs
		default:
			shifted[idx] = logs
		}
	}

	s.logs = shifted
}

// ClearAll 清除所有渠道日志，仅用于需要整体重置日志缓存的场景。
func (s *ChannelLogStore) ClearAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = make(map[int][]*ChannelLog)
}

func (s *ChannelLogStore) Get(channelIndex int) []*ChannelLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	src := s.logs[channelIndex]
	if len(src) == 0 {
		return nil
	}
	// 返回副本，按时间倒序（最新在前）
	result := make([]*ChannelLog, len(src))
	for i, j := 0, len(src)-1; j >= 0; i, j = i+1, j-1 {
		result[i] = src[j]
	}
	return result
}
