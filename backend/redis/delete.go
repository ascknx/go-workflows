package redis

import (
	"context"
	"fmt"

	"github.com/cschleiden/go-workflows/internal/core"
	redis "github.com/redis/go-redis/v9"
)

// KEYS[1] - instance key
// KEYS[2] - pending events key
// KEYS[3] - history key
// KEYS[4] - instances-by-creation key
// ARGV[1] - instance segment
var deleteCmd = redis.NewScript(
	`redis.call("DEL", KEYS[1], KEYS[2], KEYS[3])
	return redis.call("ZREM", KEYS[4], ARGV[1])`)

// deleteInstance deletes an instance from Redis. It does not attempt to remove any future events or pending
// workflow tasks. It's assumed that the instance is in the finished state.
//
// Note: might want to revisit this in the future if we want to support removing hung instances.
func deleteInstance(ctx context.Context, rdb redis.UniversalClient, instance *core.WorkflowInstance, keyPrefix string) error {
	if err := deleteCmd.Run(ctx, rdb, []string{
		instanceKey(keyPrefix, instance),
		pendingEventsKey(keyPrefix, instance),
		historyKey(keyPrefix, instance),
		instancesByCreation(keyPrefix),
	}, instanceSegment(instance)).Err(); err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	return nil
}
