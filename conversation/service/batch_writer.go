package service

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// BatchConversationService 批量对话写入服务
// 使用缓冲队列和批量写入提高性能
type BatchConversationService struct {
	service   ConversationService
	logger    *zap.Logger
	buffer    chan *ConversationDTO
	batchSize int
	flushInt  time.Duration
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// BatchWriterOption 配置选项
type BatchWriterOption func(*BatchConversationService)

// WithBatchSize 设置批量大小
func WithBatchSize(size int) BatchWriterOption {
	return func(b *BatchConversationService) {
		if size > 0 {
			b.batchSize = size
		}
	}
}

// WithFlushInterval 设置刷新间隔
func WithFlushInterval(interval time.Duration) BatchWriterOption {
	return func(b *BatchConversationService) {
		if interval > 0 {
			b.flushInt = interval
		}
	}
}

// WithBufferSize 设置缓冲队列大小
func WithBufferSize(size int) BatchWriterOption {
	return func(b *BatchConversationService) {
		if size > 0 {
			b.buffer = make(chan *ConversationDTO, size)
		}
	}
}

// NewBatchConversationService 创建批量写入服务
func NewBatchConversationService(
	service ConversationService,
	logger *zap.Logger,
	opts ...BatchWriterOption,
) *BatchConversationService {
	ctx, cancel := context.WithCancel(context.Background())

	bw := &BatchConversationService{
		service:   service,
		logger:    logger,
		buffer:    make(chan *ConversationDTO, 1000), // 默认缓冲 1000 条
		batchSize: 50,                                 // 默认每 50 条批量写入
		flushInt:  100 * time.Millisecond,             // 默认 100ms 刷新一次
		ctx:       ctx,
		cancel:    cancel,
	}

	for _, opt := range opts {
		opt(bw)
	}

	bw.wg.Add(1)
	go bw.run()

	return bw
}

// Create 异步写入单条记录
func (b *BatchConversationService) Create(ctx context.Context, dto *ConversationDTO) error {
	select {
	case b.buffer <- dto:
		return nil
	default:
		// 缓冲队列满，直接同步写入避免丢失
		b.logger.Warn("缓冲队列已满，直接同步写入", zap.String("trace_id", dto.TraceID))
		return b.service.Create(ctx, dto)
	}
}

// CreateBatch 批量写入（直接透传）
func (b *BatchConversationService) CreateBatch(ctx context.Context, dtos []ConversationDTO) error {
	return b.service.CreateBatch(ctx, dtos)
}

// Flush 立即刷新缓冲队列
func (b *BatchConversationService) Flush() {
	b.flush()
}

// Stop 停止服务并刷新所有缓冲数据
func (b *BatchConversationService) Stop() {
	b.cancel()
	b.wg.Wait()
}

// run 后台批量写入循环
func (b *BatchConversationService) run() {
	defer b.wg.Done()

	ticker := time.NewTicker(b.flushInt)
	defer ticker.Stop()

	batch := make([]ConversationDTO, 0, b.batchSize)

	for {
		select {
		case <-b.ctx.Done():
			// 服务停止，刷新剩余数据
			b.flushRemaining()
			return

		case dto := <-b.buffer:
			batch = append(batch, *dto)
			if len(batch) >= b.batchSize {
				b.writeBatch(batch)
				batch = batch[:0] // 重置切片但保留容量
			}

		case <-ticker.C:
			if len(batch) > 0 {
				b.writeBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

// flush 立即刷新当前缓冲
func (b *BatchConversationService) flush() {
	// 创建一个临时批次
	batch := make([]ConversationDTO, 0, b.batchSize)

	// 非阻塞读取所有可用数据
	for {
		select {
		case dto := <-b.buffer:
			batch = append(batch, *dto)
			if len(batch) >= b.batchSize {
				b.writeBatch(batch)
				batch = batch[:0]
			}
		default:
			// 没有更多数据
			if len(batch) > 0 {
				b.writeBatch(batch)
			}
			return
		}
	}
}

// flushRemaining 刷新所有剩余数据（阻塞）
func (b *BatchConversationService) flushRemaining() {
	batch := make([]ConversationDTO, 0, b.batchSize)

	// 读取缓冲队列中所有剩余数据
	for dto := range b.buffer {
		batch = append(batch, *dto)
		if len(batch) >= b.batchSize {
			b.writeBatch(batch)
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		b.writeBatch(batch)
	}
}

// writeBatch 执行批量写入
func (b *BatchConversationService) writeBatch(batch []ConversationDTO) {
	if len(batch) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := b.service.CreateBatch(ctx, batch); err != nil {
		b.logger.Error("批量写入对话记录失败",
			zap.Error(err),
			zap.Int("batch_size", len(batch)),
		)

		// 批量写入失败，尝试单条写入避免全部丢失
		for _, dto := range batch {
			if err := b.service.Create(ctx, &dto); err != nil {
				b.logger.Error("单条写入对话记录失败",
					zap.Error(err),
					zap.String("trace_id", dto.TraceID),
				)
			}
		}
	}
}

// 确保 BatchConversationService 实现了 ConversationService 接口
var _ ConversationService = (*BatchConversationService)(nil)

// GetByTraceID 透传到底层服务
func (b *BatchConversationService) GetByTraceID(ctx context.Context, traceID string) ([]ConversationDTO, error) {
	return b.service.GetByTraceID(ctx, traceID)
}

// ListBySessionKey 透传到底层服务
func (b *BatchConversationService) ListBySessionKey(ctx context.Context, sessionKey string, page, pageSize int) (*ConversationListResult, error) {
	return b.service.ListBySessionKey(ctx, sessionKey, page, pageSize)
}

// ListByTimeRange 透传到底层服务
func (b *BatchConversationService) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, page, pageSize int) (*ConversationListResult, error) {
	return b.service.ListByTimeRange(ctx, startTime, endTime, page, pageSize)
}

// ListRecent 透传到底层服务
func (b *BatchConversationService) ListRecent(ctx context.Context, page, pageSize int) (*ConversationListResult, error) {
	return b.service.ListRecent(ctx, page, pageSize)
}
