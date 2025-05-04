package connectlog

import (
	"encoding/json"

	"google.golang.org/protobuf/proto"
)

// calculateSize provides a simple size estimation
func calculateSize(payload any) int {
	if payload == nil {
		return 0
	}

	switch v := payload.(type) {
	case proto.Message:
		return proto.Size(v)
	case []byte:
		return len(v)
	case string:
		return len(v)
	case json.RawMessage:
		return len(v)
	case interface{ Size() int }:
		return v.Size()
	default:
		return -1 // Unknown type
	}
}
