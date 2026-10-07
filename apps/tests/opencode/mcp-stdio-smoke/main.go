// Exercise unmodified upstream MCP client and server over native child streams.
package main

import (
	"context"
	"fmt"
	"kos"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		s := server.NewMCPServer("KolibriOS native stdio", "1.0.0")
		s.AddTool(mcp.NewTool("echo", mcp.WithString("text", mcp.Required())), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			text, ok := request.Params.Arguments["text"].(string)
			if !ok {
				return nil, fmt.Errorf("text is required")
			}
			if os.Getenv("MCP_TEST_ENV") != "Привет" {
				return nil, fmt.Errorf("child environment lost")
			}
			return mcp.NewToolResultText(text), nil
		})
		if err := server.NewStdioServer(s).Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
			panic(err)
		}
		return
	}
	console, ok := kos.OpenConsole("Original MCP stdio on KolibriOS")
	if !ok {
		panic("console initialization")
	}
	defer console.Exit(false)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	c, err := client.NewStdioMCPClient(os.Args[0], []string{"MCP_TEST_ENV=Привет"}, "serve")
	if err != nil {
		panic(err)
	}
	request := mcp.InitializeRequest{}
	request.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	request.Params.ClientInfo = mcp.Implementation{Name: "KolibriOS test", Version: "1.0.0"}
	initialized, err := c.Initialize(ctx, request)
	if err != nil || initialized.ServerInfo.Name != "KolibriOS native stdio" {
		fmt.Println(err)
		panic("MCP initialize")
	}
	listed, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil || len(listed.Tools) != 1 || listed.Tools[0].Name != "echo" {
		panic("MCP list tools")
	}
	text := "Привет 😀\n" + strings.Repeat("native byte stream ", 6000)
	call := mcp.CallToolRequest{}
	call.Params.Name = "echo"
	call.Params.Arguments = map[string]any{"text": text}
	result, err := c.CallTool(ctx, call)
	if err != nil || result.IsError || len(result.Content) != 1 {
		fmt.Println(err)
		panic("MCP tool call")
	}
	content, ok := result.Content[0].(mcp.TextContent)
	if !ok || content.Text != text {
		panic("MCP exact Unicode response")
	}
	if err := c.Close(); err != nil {
		fmt.Println(err)
		panic("MCP child shutdown")
	}
	fmt.Println("Original MCP initialize, listTools, 108KB Unicode call and shutdown PASS")
	kos.DebugString("OPENCODE_MCP_STDIO_NATIVE_PASS\n")
}
