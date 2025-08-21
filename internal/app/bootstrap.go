package app

import (
	"coinStatApp/config"
	"coinStatApp/internal/app/dto"
	"coinStatApp/internal/domain/model"
	"coinStatApp/internal/domain/repository"
	"coinStatApp/internal/domain/service"
	ws "coinStatApp/internal/handlers/websocket"
	redisrepo "coinStatApp/internal/infrastructure/cache"
	"coinStatApp/internal/infrastructure/queue"
	chrepo "coinStatApp/internal/infrastructure/storage"
	"context"
	"log"
)

// Processor defines the common interface for both standard and Kafka event processors
type Processor interface {
	Run(ctx context.Context) error
}

// AppContext holds all app dependencies
type AppContext struct {
	Config         *config.Config
	StatsService   *service.TimeWindowedStatisticsService
	Broadcaster    *ws.WebSocketBroadcaster
	EventProcessor Processor // Changed from concrete type to interface
	KafkaConsumer  *queue.KafkaConsumer
	KafkaProducer  *queue.KafkaProducer
	SwapCh         chan *dto.SwapDTO
}

// NewApp initializes the app context with all dependencies
func NewApp(ctx context.Context, config *config.Config) (*AppContext, error) {
	app := &AppContext{}
	// Load configuration
	app.Config = config
	log.Println("Configuration loaded")
	// Initialize cache implementation (Redis)
	var statsCache repository.StatisticsCache
	redisRepo := redisrepo.NewRedisRepository(app.Config.RedisAddr, app.Config.RedisPassword, app.Config.RedisDB)
	statsCache = redisRepo
	log.Println("Redis cache initialized")
	// Try to initialize persistent storage implementation (ClickHouse)
	var statsPersistence repository.StatisticsPersistence
	chConfig := chrepo.ClickHouseConfig{
		Addr:     app.Config.ClickhouseAddr,
		Username: app.Config.ClickhouseUsername,
		Password: app.Config.ClickhousePassword,
		Timeout:  app.Config.ClickhouseTimeout,
	}
	clickhouseRepo, err := chrepo.NewClickHouseRepository(chConfig)
	if err != nil {
		log.Printf("Warning: Failed to connect to ClickHouse: %v. Continuing with Redis only.", err)
	} else {
		statsPersistence = clickhouseRepo
		log.Println("ClickHouse persistent storage initialized")
	}
	// Create statistics service with proper time window management
	app.StatsService = service.NewTimeWindowedStatisticsService(statsCache, statsPersistence)
	log.Println("Statistics service initialized with appropriate storage backends")
	// Setup broadcaster
	app.Broadcaster = ws.NewWebSocketBroadcaster()

	// Setup direct channel for event processing
	app.setupDirectChannel()

	// Setup Kafka configuration
	kafkaConfig := queue.KafkaConfig{
		Brokers:       config.KafkaBrokers,
		Topic:         config.KafkaTopic,
		ConsumerGroup: config.KafkaConsumerGroup,
		BatchSize:     config.KafkaBatchSize,
		BatchTimeout:  config.KafkaBatchTimeout,
	}
	// Try to setup Kafka consumer
	app.KafkaConsumer = queue.NewKafkaConsumer(kafkaConfig)

	if app.KafkaConsumer != nil {
		log.Println("Using Kafka for event consumption...")

		// Subscribe to Kafka topic
		swapChannel, err := app.KafkaConsumer.Subscribe(ctx)
		if err != nil {
			log.Fatalf("Failed to subscribe to Kafka: %v", err)
		}

		app.SwapCh = convertSwapChannel(swapChannel)
		log.Println("Kafka consumer subscribed to topic")

		app.EventProcessor = NewEventProcessor(app.SwapCh, app.StatsService, app.Broadcaster)

		// Start Kafka producer for demo/testing purposes
		app.KafkaProducer = queue.NewKafkaProducer(kafkaConfig)
		log.Println("Kafka consumer and producer initialized")
	} else {
		log.Println("Kafka not configured or unavailable, using direct channel...")
		// Fallback to direct channel
		app.SwapCh = make(chan *dto.SwapDTO, app.Config.EventBufferSize)
		app.EventProcessor = NewEventProcessor(app.SwapCh, app.StatsService, app.Broadcaster)
	}
	//app.KafkaProducer = queue.NewKafkaProducer(kafkaConfig)

	return app, nil
}

// setupDirectChannel creates a direct channel for swap events
func (a *AppContext) setupDirectChannel() {
	// Create direct channel for event processing
	a.SwapCh = make(chan *dto.SwapDTO, a.Config.EventBufferSize)
	processor := NewEventProcessor(a.SwapCh, a.StatsService, a.Broadcaster)
	a.EventProcessor = processor
	log.Println("Direct channel setup completed")
}

// convertSwapChannel converts a channel of domain models to a channel of DTOs
func convertSwapChannel(modelCh <-chan *model.Swap) chan *dto.SwapDTO {
	dtoCh := make(chan *dto.SwapDTO)

	go func() {
		for swap := range modelCh {
			if swap != nil {
				dtoCh <- dto.FromModel(swap)
			}
		}
		close(dtoCh)
	}()

	return dtoCh
}

// Cleanup performs graceful shutdown of all components
func (a *AppContext) Cleanup(ctx context.Context) {
	if a.KafkaConsumer != nil {
		log.Println("Closing Kafka consumer...")
		if err := a.KafkaConsumer.Close(); err != nil {
			log.Printf("Error closing Kafka consumer: %v", err)
		}
	}

	if a.KafkaProducer != nil {
		log.Println("Closing Kafka producer...")
		if err := a.KafkaProducer.Close(); err != nil {
			log.Printf("Error closing Kafka producer: %v", err)
		}
	}

	// Close any remaining channels
	if a.SwapCh != nil {
		log.Println("Closing direct channel...")
		close(a.SwapCh)
	}

	log.Println("All resources cleaned up")
}
