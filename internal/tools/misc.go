// internal/tools/misc.go
package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type miscClient interface {
	ServerTimeMillis() int64
}

type GetServerTimeArgs struct{}

type ServerTimeData struct {
	ServerTime int64 `json:"serverTime"`
}

type GetServerTimeResult struct {
	Message string         `json:"message"`
	Data    ServerTimeData `json:"data"`
}

func getServerTimeHandler(client miscClient) mcp.ToolHandlerFor[GetServerTimeArgs, GetServerTimeResult] {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ GetServerTimeArgs) (*mcp.CallToolResult, GetServerTimeResult, error) {
		return nil, GetServerTimeResult{
			Message: "Server time retrieved successfully",
			Data:    ServerTimeData{ServerTime: client.ServerTimeMillis()},
		}, nil
	}
}

// RegisterMiscTools registers the utility MCP tools on server.
func RegisterMiscTools(server *mcp.Server, client miscClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_get_server_time",
		Description: "Get estimated server time",
	}, getServerTimeHandler(client))
}
