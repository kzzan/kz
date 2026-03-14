package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"example/pkg/config"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

type Message struct {
	ID         string    `json:"id"`
	Topic      string    `json:"topic"`
	Payload    string    `json:"payload"`
	CreatedAt  time.Time `json:"created_at"`
	RetryCount int       `json:"retry_count"`
}

type MessageHandler func(ctx context.Context, msg *Message) error

type Queue interface {
	Publish(ctx context.Context, topic string, payload interface{}) error
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error
	GetLength(ctx context.Context, topic string) (int64, error)
}

type redisQueue struct {
	client *redis.Client
	logger *zerolog.Logger
}

func NewQueue(i do.Injector) (Queue, error) {
	cfg    := do.MustInvoke[*config.Config](i)
	logger := do.MustInvoke[*zerolog.Logger](i)

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接 Redis 失败: %w", err)
	}

	logger.Info().
		Str("host", cfg.Redis.Host).
		Int("port", cfg.Redis.Port).
		Msg("Redis queue connected")

	return &redisQueue{client: client, logger: logger}, nil
}

func (q *redisQueue) Publish(ctx context.Context, topic string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := &Message{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Topic:     topic,
		Payload:   string(data),
		CreatedAt: time.Now(),
	}
	msgData, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: fmt.Sprintf("queue:%s", topic),
		Values: map[string]interface{}{"data": string(msgData)},
	}).Result()
	if err != nil {
		q.logger.Error().Err(err).Str("topic", topic).Msg("publish failed")
	}
	return err
}

func (q *redisQueue) Subscribe(ctx context.Context, topic string, handler MessageHandler) error {
	streamKey := fmt.Sprintf("queue:%s", topic)
	group     := fmt.Sprintf("group:%s", topic)
	consumer  := fmt.Sprintf("consumer:%d", time.Now().UnixNano())

	_ = q.client.XGroupCreateMkStream(ctx, streamKey, group, "0").Err()

	q.logger.Info().Str("topic", topic).Str("group", group).Msg("queue subscription started")

	for {
		select {
		case <-ctx.Done():
			q.logger.Info().Str("topic", topic).Msg("queue subscription cancelled")
			return nil
		default:
			messages, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    group,
				Consumer: consumer,
				Streams:  []string{streamKey, ">"},
				Count:    1,
				Block:    time.Second,
			}).Result()
			if err != nil && err != redis.Nil {
				q.logger.Error().Err(err).Str("topic", topic).Msg("read failed")
				time.Sleep(time.Second)
				continue
			}
			for _, stream := range messages {
				for _, m := range stream.Messages {
					if raw, ok := m.Values["data"].(string); ok {
						var qMsg Message
						if err := json.Unmarshal([]byte(raw), &qMsg); err != nil {
							q.logger.Error().Err(err).Str("id", m.ID).Msg("unmarshal failed")
						} else if err := handler(ctx, &qMsg); err != nil {
							q.logger.Warn().Err(err).Str("id", m.ID).Msg("handler failed")
						}
					}
					_ = q.client.XAck(ctx, streamKey, group, m.ID).Err()
				}
			}
		}
	}
}

func (q *redisQueue) GetLength(ctx context.Context, topic string) (int64, error) {
	return q.client.XLen(ctx, fmt.Sprintf("queue:%s", topic)).Result()
}

func (q *redisQueue) Shutdown() error {
	return q.client.Close()
}
