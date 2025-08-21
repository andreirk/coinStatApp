package queue

import (
	"coinStatApp/internal/app/dto"
	"coinStatApp/internal/domain/model"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaConfig holds Kafka connection configuration
type KafkaConfig struct {
	Brokers       []string
	Topic         string
	ConsumerGroup string
	BatchSize     int
	BatchTimeout  int
}

// SwapProducer defines interface for producing swap events
type SwapProducer interface {
	PublishSwap(ctx context.Context, swap *model.Swap) error
	Close() error
}

// SwapConsumer defines interface for consuming swap events
type SwapConsumer interface {
	Subscribe(ctx context.Context) (<-chan *model.Swap, error)
	Commit(ctx context.Context, swap *model.Swap) error
	Close() error
}

// KafkaProducer implements SwapProducer using Kafka
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer creates a new Kafka producer
func NewKafkaProducer(config KafkaConfig) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Topic:        config.Topic,
		Balancer:     &kafka.Hash{}, // Use hash-based partitioning for token-based ordering
		RequiredAcks: kafka.RequireAll,
		// For exactly-once semantics:
		// Async:       false,
	}

	return &KafkaProducer{writer: writer}
}

// PublishSwap sends a swap event to Kafka
func (p *KafkaProducer) PublishSwap(ctx context.Context, swap *model.Swap) error {
	data, err := json.Marshal(swap)
	if err != nil {
		return err
	}

	// Use token as key for partitioning (events for same token go to same partition)
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(swap.Token),
		Value: data,
		Time:  time.Now(),
	})
}

// PublishSwapBatch sends batch swap events to Kafka
func (p *KafkaProducer) PublishSwapBatch(ctx context.Context, swaps []*dto.SwapDTO) error {
	msgSlice := make([]kafka.Message, len(swaps))
	for i, swap := range swaps {
		data, err := json.Marshal(swap)
		if err != nil {
			return err
		}
		msgSlice[i] = kafka.Message{
			Key:   []byte(swap.Token),
			Value: data,
			Time:  time.Now(),
		}
	}
	// Use token as key for partitioning (events for same token go to same partition)
	return p.writer.WriteMessages(ctx, msgSlice...)
}

// Close closes the producer
func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}

// KafkaConsumer implements SwapConsumer using Kafka
type KafkaConsumer struct {
	reader        *kafka.Reader
	topic         string
	pendingMsgs   map[string]kafka.Message // Map of swap ID to Kafka message
	pendingMsgsMu sync.RWMutex             // Mutex to protect the pendingMsgs map
	batchSize     int                      // Number of messages to accumulate before batch commit
	batchTimeout  time.Duration            // Max time to wait before committing a batch
}

// NewKafkaConsumer creates a new Kafka consumer
func NewKafkaConsumer(config KafkaConfig) *KafkaConsumer {
	// Disable auto-commit to allow explicit commits
	reader := kafka.NewReader(kafka.ReaderConfig{

		Brokers:        config.Brokers,
		Topic:          config.Topic,
		GroupID:        config.ConsumerGroup,
		MinBytes:       10e3,              // 10KB
		MaxBytes:       10e6,              // 10MB
		CommitInterval: 0,                 // Disable auto commit - we'll handle this manually
		StartOffset:    kafka.FirstOffset, // Start from oldest message if no offset is stored
	})

	return &KafkaConsumer{
		reader:        reader,
		topic:         config.Topic,
		pendingMsgs:   make(map[string]kafka.Message),
		pendingMsgsMu: sync.RWMutex{},
		batchSize:     config.BatchSize,                                      // Commit after batchSize messages
		batchTimeout:  time.Duration(config.BatchTimeout) * time.Millisecond, // Or after tiemout, whichever comes first
	}
}

// Subscribe returns a channel of swap events from Kafka
func (c *KafkaConsumer) Subscribe(ctx context.Context) (<-chan *model.Swap, error) {
	swapCh := make(chan *model.Swap, 1000) // Buffer to handle bursts

	// Start a background goroutine for batch commits
	go c.startBatchCommitter(ctx)

	// Start the main consumer goroutine
	go func() {
		defer close(swapCh)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := c.reader.FetchMessage(ctx)
				if err != nil {
					if ctx.Err() == nil { // Only log if not due to context cancellation
						log.Printf("Error fetching message: %v", err)
					}
					return
				}

				var swap model.Swap
				if err := json.Unmarshal(msg.Value, &swap); err != nil {
					log.Printf("Error unmarshalling swap: %v", err)
					// Commit bad messages to avoid getting stuck
					_ = c.reader.CommitMessages(ctx, msg)
					continue
				}

				// Make sure we have an ID for tracking
				if swap.ID == "" {
					swap.ID = fmt.Sprintf("%s-%d-%d", swap.Token, msg.Partition, msg.Offset)
				}

				// Store message for later commit (before sending to channel to ensure we don't miss commits)
				c.pendingMsgsMu.Lock()
				c.pendingMsgs[swap.ID] = msg
				pendingCount := len(c.pendingMsgs)
				c.pendingMsgsMu.Unlock()

				// Log when we're accumulating too many messages
				if pendingCount > c.batchSize*10 {
					log.Printf("Warning: Large number of uncommitted messages: %d, batchSize is %d", pendingCount, c.batchSize)
				}

				// Send to channel (blocking if buffer is full)
				select {
				case <-ctx.Done():
					return
				case swapCh <- &swap:
					// Message is now in the channel to be processed
					// Actual commit will happen in Commit() or batch committer
				}
			}
		}
	}()

	return swapCh, nil
}

// startBatchCommitter runs a background process that periodically commits messages in batches
func (c *KafkaConsumer) startBatchCommitter(ctx context.Context) {
	ticker := time.NewTicker(c.batchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Final commit before shutting down
			c.commitAllPending(context.Background()) // Use a new context since the original is canceled
			return
		case <-ticker.C:
			c.commitAllPending(ctx)
		}
	}
}

// commitAllPending commits all pending messages
func (c *KafkaConsumer) commitAllPending(ctx context.Context) {
	c.pendingMsgsMu.Lock()
	defer c.pendingMsgsMu.Unlock()

	if len(c.pendingMsgs) == 0 {
		return // Nothing to commit
	}

	// Convert map to slice for committing
	msgs := make([]kafka.Message, 0, len(c.pendingMsgs))
	for _, msg := range c.pendingMsgs {
		msgs = append(msgs, msg)
	}

	// Try to commit all messages
	if err := c.reader.CommitMessages(ctx, msgs...); err != nil {
		log.Printf("Error committing batch of %d messages: %v", len(msgs), err)
		return
	}

	// Clear the map after successful commit
	log.Printf("Successfully committed batch of %d messages", len(msgs))
	c.pendingMsgs = make(map[string]kafka.Message)
}

// Commit acknowledges that a swap has been processed
func (c *KafkaConsumer) Commit(ctx context.Context, swap *model.Swap) error {
	if swap == nil || swap.ID == "" {
		return fmt.Errorf("cannot commit null swap or swap with empty ID")
	}

	c.pendingMsgsMu.Lock()
	msg, exists := c.pendingMsgs[swap.ID]
	if !exists {
		c.pendingMsgsMu.Unlock()
		return fmt.Errorf("message for swap %s not found in pending messages", swap.ID)
	}

	// If we have enough messages, commit them all as a batch
	pendingCount := len(c.pendingMsgs)
	shouldBatchCommit := pendingCount >= c.batchSize

	// If not batch committing, just commit this one message
	if !shouldBatchCommit {
		delete(c.pendingMsgs, swap.ID) // Remove from pending before unlocking
		c.pendingMsgsMu.Unlock()

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			return fmt.Errorf("failed to commit message for swap %s: %w", swap.ID, err)
		}
		return nil
	}

	// For batch commit, unlock and call the batch commit function
	c.pendingMsgsMu.Unlock()
	c.commitAllPending(ctx)
	return nil
}

// Close closes the consumer
func (c *KafkaConsumer) Close() error {
	// Final commit of any pending messages
	c.commitAllPending(context.Background())
	return c.reader.Close()
}
