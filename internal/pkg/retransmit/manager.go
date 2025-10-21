package retransmit

import (
	"sync/atomic"
	"time"

	"github.com/JrMarcco/jit/xsync"
	"github.com/JrMarcco/synp"
	messagev1 "github.com/JrMarcco/synp-api/api/go/message/v1"
	"github.com/JrMarcco/synp/internal/pkg/message"
	"go.uber.org/zap"
)

const (
	DefaultRetryInterval = 8 * time.Second
	DefaultMaxRetryCnt   = 3
)

// Task 为重传任务。
// 负责对 downstream 消息的失败重传。
// 每个消息需要一个重传任务，一个重传任务只能对应一个消息。
type Task struct {
	// messageID string

	conn synp.Conn
	msg  *messagev1.Message

	timerPtr      atomic.Pointer[time.Timer] // 重传定时器
	retransmitCnt atomic.Int32               // 重传次数

	manager *Manager
}

func (t *Task) run() {
	// 尝试加载任务，如果不存在说明已被停止。
	if _, ok := t.manager.tasks.Load(t.msg.MessageId); !ok {
		return
	}

	t.retransmitCnt.Add(1)

	// 检查重传次数。
	if t.retransmitCnt.Load() >= t.manager.maxRetryCnt {
		t.manager.logger.Warn(
			"[synp-retransmit-manager] retransmit task reach max retry cnt",
			zap.String("connection_id", t.conn.Id()),
			zap.String("message_id", t.msg.MessageId),
			zap.Int32("retransmit_count", t.retransmitCnt.Load()),
		)
		_ = t.manager.stopAndDelete(t.msg.MessageId)
		return
	}

	// 重传。
	err := t.manager.taskFunc(t.conn, t.msg)
	if err != nil {
		t.manager.logger.Error(
			"[synp-retransmit-manager] failed to retransmit message",
			zap.String("connection_id", t.conn.Id()),
			zap.String("message_id", t.msg.MessageId),
			zap.Int32("retransmit_count", t.retransmitCnt.Load()),
			zap.Error(err),
		)

		// 重传失败，直接停止重传任务。
		_ = t.manager.stopAndDelete(t.msg.MessageId)
		return
	}

	t.conn.UpdateActivityTime()
	t.manager.logger.Debug(
		"[synp-retransmit-manager] successfully retransmit message",
		zap.String("connection_id", t.conn.Id()),
		zap.String("message_id", t.msg.MessageId),
		zap.Int32("retransmit_count", t.retransmitCnt.Load()),
	)

	// 更新定时器。
	t.timerPtr.Store(time.AfterFunc(t.manager.retryInterval, t.run))
}

func (t *Task) stop() {
	if timer := t.timerPtr.Load(); timer != nil {
		// Stop() 返回 false 表示 timer 已经过期或被停止。
		// 此时需要排空 channel 防止 goroutine 泄露。
		if !timer.Stop() {
			// 尝试排空 channel。
			select {
			case <-timer.C:
			default:
			}
		}
	}
}

// Manager 为重传管理器，负责管理重传任务。
// 重传使用固定间隔重试，直到成功或达到最大重传次数。
type Manager struct {
	tasks *xsync.Map[string, *Task] // message_id -> retransmit.Task

	totalTaskCnt  atomic.Int64
	retryInterval time.Duration // 重传间隔
	maxRetryCnt   int32         // 最大重传次数

	taskFunc message.PushFunc
	closed   atomic.Bool

	logger *zap.Logger
}

func (m *Manager) Start(conn synp.Conn, msg *messagev1.Message) {
	if m.closed.Load() {
		return
	}

	task := &Task{
		conn:    conn,
		msg:     msg,
		manager: m,
	}

	if _, ok := m.tasks.LoadOrStore(msg.MessageId, task); ok {
		return
	}

	task.timerPtr.Store(time.AfterFunc(m.retryInterval, task.run))
	m.totalTaskCnt.Add(1)

	m.logger.Debug(
		"[synp-retransmit-manager] successfully start retransmit task",
		zap.String("connection_id", conn.Id()),
		zap.String("message_id", msg.MessageId),
		zap.Duration("retry_interval", m.retryInterval),
		zap.Int32("max_retry_cnt", m.maxRetryCnt),
	)
}

// Stop 停止指定消息的重传任务。
func (m *Manager) Stop(messageId string) {
	if task, ok := m.tasks.Load(messageId); ok {
		_ = m.stopAndDelete(messageId)

		m.logger.Debug(
			"[synp-retransmit-manager] successfully stop retransmit task",
			zap.String("message_id", messageId),
			zap.Int32("retransmit_count", task.retransmitCnt.Load()),
		)
	}
}

func (m *Manager) TotalTaskCnt() int64 {
	return m.totalTaskCnt.Load()
}

// ClearByConn 清除指定连接的重传任务。
func (m *Manager) ClearByConn(connId string) {
	var cnt int
	m.tasks.Range(func(messageId string, task *Task) bool {
		if task.conn.Id() == connId {
			if m.stopAndDelete(messageId) {
				cnt++
			}
		}
		return true
	})

	if cnt > 0 {
		m.logger.Info(
			"[synp-retransmit-manager] successfully clear retransmit tasks by connection",
			zap.String("connection_id", connId),
			zap.Int("task_cleared_cnt", cnt),
		)
	}
}

func (m *Manager) stopAndDelete(messageId string) bool {
	if task, ok := m.tasks.LoadAndDelete(messageId); ok {
		task.stop()
		m.totalTaskCnt.Add(-1)
		return true
	}
	return false
}

// Close 关闭重传管理器。
func (m *Manager) Close() {
	if !m.closed.CompareAndSwap(false, true) {
		return
	}

	var cnt int
	m.tasks.Range(func(messageId string, _ *Task) bool {
		if m.stopAndDelete(messageId) {
			cnt++
		}
		return true
	})

	m.logger.Info(
		"[synp-retransmit-manager] retransmit manager closed",
		zap.Int("task_cleared_cnt", cnt),
	)
}

func NewManager(retryInterval time.Duration, maxRetryCnt int32, taskFunc message.PushFunc, logger *zap.Logger) *Manager {
	if retryInterval <= 0 {
		retryInterval = DefaultRetryInterval
	}
	if maxRetryCnt <= 0 {
		maxRetryCnt = DefaultMaxRetryCnt
	}

	if taskFunc == nil {
		taskFunc = func(conn synp.Conn, msg *messagev1.Message) error {
			return nil
		}
	}

	return &Manager{
		tasks:         &xsync.Map[string, *Task]{},
		retryInterval: retryInterval,
		maxRetryCnt:   maxRetryCnt,
		taskFunc:      taskFunc,
		logger:        logger,
	}
}
