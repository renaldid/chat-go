package websocket

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const presenceTTL = 90 * time.Second

type RedisPresence struct {
	client     *redis.Client
	instanceID string
}

func NewRedisPresence(
	client *redis.Client,
	instanceID string,
) *RedisPresence {
	return &RedisPresence{
		client:     client,
		instanceID: instanceID,
	}
}

func (p *RedisPresence) key(conversationID int64) string {
	return fmt.Sprintf(
		"chat:presence:%d:%s",
		conversationID,
		p.instanceID,
	)
}

func (p *RedisPresence) Register(
	ctx context.Context,
	conversationID int64,
	userID int64,
) (bool, error) {
	key := p.key(conversationID)
	field := strconv.FormatInt(userID, 10)

	_, err := p.client.TxPipelined(
		ctx,
		func(pipe redis.Pipeliner) error {
			pipe.HIncrBy(ctx, key, field, 1)
			pipe.Expire(ctx, key, presenceTTL)
			return nil
		},
	)
	if err != nil {
		return false, fmt.Errorf("register redis presence: %w", err)
	}

	count, err := p.userConnectionCount(
		ctx,
		conversationID,
		userID,
	)
	if err != nil {
		return false, err
	}

	return count == 1, nil
}

func (p *RedisPresence) Unregister(
	ctx context.Context,
	conversationID int64,
	userID int64,
) (bool, error) {
	key := p.key(conversationID)
	field := strconv.FormatInt(userID, 10)

	script := redis.NewScript(`
		local value = redis.call("HINCRBY", KEYS[1], ARGV[1], -1)

		if value <= 0 then
			redis.call("HDEL", KEYS[1], ARGV[1])
		end

		redis.call("EXPIRE", KEYS[1], ARGV[2])

		return value
	`)

	if err := script.Run(
		ctx,
		p.client,
		[]string{key},
		field,
		int64(presenceTTL.Seconds()),
	).Err(); err != nil {
		return false, fmt.Errorf("unregister redis presence: %w", err)
	}

	count, err := p.userConnectionCount(
		ctx,
		conversationID,
		userID,
	)
	if err != nil {
		return false, err
	}

	return count == 0, nil
}

func (p *RedisPresence) Refresh(
	ctx context.Context,
	conversationID int64,
) error {
	return p.client.Expire(
		ctx,
		p.key(conversationID),
		presenceTTL,
	).Err()
}

func (p *RedisPresence) OnlineUserIDs(
	ctx context.Context,
	conversationID int64,
) ([]int64, error) {
	keys, err := p.instanceKeys(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	users := make(map[int64]struct{})

	for _, key := range keys {
		values, err := p.client.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("get redis presence: %w", err)
		}

		for rawUserID, rawCount := range values {
			count, err := strconv.ParseInt(rawCount, 10, 64)
			if err != nil || count <= 0 {
				continue
			}

			userID, err := strconv.ParseInt(rawUserID, 10, 64)
			if err != nil {
				continue
			}

			users[userID] = struct{}{}
		}
	}

	result := make([]int64, 0, len(users))

	for userID := range users {
		result = append(result, userID)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	return result, nil
}

func (p *RedisPresence) userConnectionCount(
	ctx context.Context,
	conversationID int64,
	userID int64,
) (int64, error) {
	keys, err := p.instanceKeys(ctx, conversationID)
	if err != nil {
		return 0, err
	}

	field := strconv.FormatInt(userID, 10)

	var total int64

	for _, key := range keys {
		value, err := p.client.HGet(ctx, key, field).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}

			return 0, fmt.Errorf("get redis presence count: %w", err)
		}

		count, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			continue
		}

		total += count
	}

	return total, nil
}

func (p *RedisPresence) instanceKeys(
	ctx context.Context,
	conversationID int64,
) ([]string, error) {
	pattern := fmt.Sprintf(
		"chat:presence:%d:*",
		conversationID,
	)

	var (
		cursor uint64
		keys   []string
	)

	for {
		result, nextCursor, err := p.client.Scan(
			ctx,
			cursor,
			pattern,
			100,
		).Result()
		if err != nil {
			return nil, fmt.Errorf("scan redis presence: %w", err)
		}

		keys = append(keys, result...)
		cursor = nextCursor

		if cursor == 0 {
			break
		}
	}

	return keys, nil
}
