package idgen

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/monzo/slog"

	"github.com/bwmarrin/snowflake"
)

var node *snowflake.Node

func Init(ctx context.Context) error {
	var err error

	h := fnv.New64()
	_, err = h.Write([]byte(time.Now().Format(time.RFC3339Nano)))
	if err != nil {
		return err
	}

	nodeID := int64(h.Sum64() % 1023)

	slog.Info(ctx, "Creating Snowflake hash node with nodeid %v", nodeID)

	node, err = snowflake.NewNode(nodeID)
	return err
}

func New(prefix string) string {
	id := node.Generate()
	return fmt.Sprintf("%s_%s", prefix, id.Base58())
}

func Parse(id string) (snowflake.ID, error) {
	parts := strings.Split(id, "_")
	if len(parts) != 2 {
		return -1, fmt.Errorf("expected ID formatted as prefix_id, got %s", id)
	}
	return snowflake.ParseBase58([]byte(parts[1]))
}
