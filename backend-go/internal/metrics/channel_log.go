package metrics

import (
	"sync"
	"time"
)

// ChannelLog 单次上游请求日志
type ChannelLog struct {
	RequestID     string    `json:"requestId"` // 请求唯一标识
	ChannelIndex  int       `json:"-"`         // 创建时的渠道索引（不序列化，仅用于内部验证）
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
	InterfaceType string    `json:"interfaceType"`           // 接口类型（Messages/Responses/Gemini）
	RequestSource string    `json:"requestSource,omitempty"` // 请求来源（proxy/capability_test）

	// 请求生命周期状态
	Status      string     `json:"status"`                // pending/connecting/first_byte/streaming/completed/failed
	StartTime   time.Time  `json:"startTime"`             // 请求开始时间
	ConnectedAt *time.Time `json:"connectedAt,omitempty"` // 连接建立时间
	FirstByteAt *time.Time `json:"firstByteAt,omitempty"` // 首字节到达时间
	CompletedAt *time.Time `json:"completedAt,omitempty"` // 请求完成时间
}

const (
	RequestSourceProxy          = "proxy"
	RequestSourceCapabilityTest = "capability_test"
	maxChannelLogs              = 50

	// 请求状态常量
	StatusPending    = "pending"
	StatusConnecting = "connecting"
	StatusFirstByte  = "first_byte"
	StatusStreaming  = "streaming"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// ChannelLogStore 渠道日志存储（内存环形缓冲区）
type ChannelLogStore struct {
	mu                sync.RWMutex
	logs              map[int][]*ChannelLog   // key: channelIndex
	deletedRequestIDs map[string]struct{}     // 被删除渠道上的请求，避免终态补写污染其他渠道
}

func NewChannelLogStore() *ChannelLogStore {
	return &ChannelLogStore{
		logs:              make(map[int][]*ChannelLog),
		deletedRequestIDs: make(map[string]struct{}),
	}
}

func (s *ChannelLogStore) Record(channelIndex int, log *ChannelLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if log != nil && log.RequestID != "" {
		delete(s.deletedRequestIDs, log.RequestID)
	}
	s.logs[channelIndex] = append(s.logs[channelIndex], log)
	if len(s.logs[channelIndex]) > maxChannelLogs {
		evicted := s.logs[channelIndex][:len(s.logs[channelIndex])-maxChannelLogs]
		for _, item := range evicted {
			if item != nil && item.RequestID != "" {
				delete(s.deletedRequestIDs, item.RequestID)
			}
		}
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
			for _, log := range logs {
				if log != nil && log.RequestID != "" {
					s.deletedRequestIDs[log.RequestID] = struct{}{}
				}
			}
			continue
		case idx > channelIndex:
			for _, log := range logs {
				if log != nil {
					log.ChannelIndex = idx - 1
				}
			}
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
	for _, logs := range s.logs {
		for _, log := range logs {
			if log != nil && log.RequestID != "" {
				s.deletedRequestIDs[log.RequestID] = struct{}{}
			}
		}
	}
	s.logs = make(map[int][]*ChannelLog)
}

func (s *ChannelLogStore) Get(channelIndex int) []*ChannelLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	src := s.logs[channelIndex]
	if len(src) == 0 {
		return nil
	}
	// 返回深拷贝，按时间倒序（最新在前），避免并发修改问题
	result := make([]*ChannelLog, len(src))
	for i, j := 0, len(src)-1; j >= 0; i, j = i+1, j-1 {
		// 深拷贝每个日志对象
		logCopy := *src[j]
		result[i] = &logCopy
	}
	return result
}

// UpdateStatus 描述 Update 的结果
type UpdateStatus int

const (
	UpdateFound UpdateStatus = iota
	UpdateMissingEvicted
	UpdateMissingDeleted
)

// Update 更新指定请求日志（通过 RequestID 匹配）
// 若渠道删除导致索引漂移，会跨索引查找并更新；返回值用于区分淘汰与删除。
func (s *ChannelLogStore) Update(channelIndex int, requestID string, updateFn func(*ChannelLog)) UpdateStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	if requestID == "" {
		return UpdateMissingEvicted
	}

	if logs, ok := s.logs[channelIndex]; ok {
		for i := range logs {
			if logs[i].RequestID == requestID {
				updateFn(logs[i])
				return UpdateFound
			}
		}
	}

	for _, logs := range s.logs {
		for i := range logs {
			if logs[i].RequestID == requestID {
				updateFn(logs[i])
				return UpdateFound
			}
		}
	}

	if _, deleted := s.deletedRequestIDs[requestID]; deleted {
		return UpdateMissingDeleted
	}

	return UpdateMissingEvicted
}
