package redis_tile_storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/raphoester/clickplanet.lol-backend/internal/domain"
	"github.com/redis/go-redis/v9"
)

func New(
	redis *redis.Client,
	setAndPublishSha1 string,
) *Storage {
	return &Storage{
		redis:             redis,
		setAndPublishSha1: setAndPublishSha1,
	}
}

type Storage struct {
	redis             *redis.Client
	setAndPublishSha1 string
}

const channel = "tileUpdates"

func (s *Storage) Set(ctx context.Context, tile uint32, value string) error {
	_, err := s.redis.EvalSha(
		ctx,
		s.setAndPublishSha1,
		[]string{strconv.FormatUint(uint64(tile), 10)},
		value, channel,
	).Result()

	if err != nil {
		return fmt.Errorf("failed to set tile: %w", err)
	}

	return nil
}

func (s *Storage) GetFullState(ctx context.Context) (map[uint32]string, error) {
	iter := s.redis.Scan(ctx, 0, "*", 0).Iterator()

	keys := make([]string, 0, 100_000)
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan keys: %w", err)
	}

	retMap := make(map[uint32]string, len(keys))
	batchSize := 100
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batchKeys := keys[i:end]
		values, err := s.redis.MGet(ctx, batchKeys...).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get tile values: %w", err)
		}

		for j, key := range batchKeys {
			if values[j] == nil {
				continue
			}

			tile, err := strconv.ParseUint(key, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse tileId to int: %w", err)
			}

			retMap[uint32(tile)] = values[j].(string)
		}
	}

	return retMap, nil
}

func (s *Storage) Subscribe(ctx context.Context) <-chan domain.TileUpdate {
	ch := make(chan domain.TileUpdate)

	go func() {
		pubSub := s.redis.Subscribe(ctx, channel)
		defer func() { _ = pubSub.Close() }()

		for msg := range pubSub.Channel() {
			payload := make(map[string]struct {
				OldValue string `json:"o"`
				NewValue string `json:"n"`
			}, 1)
			if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
				continue
			}

			for tile, value := range payload { // there should be only one key
				tileId, err := strconv.ParseUint(tile, 10, 32)
				if err != nil {
					continue
				}

				ch <- domain.TileUpdate{
					Tile:     uint32(tileId),
					Value:    value.NewValue,
					Previous: value.OldValue,
				}
			}
		}
	}()

	return ch
}
