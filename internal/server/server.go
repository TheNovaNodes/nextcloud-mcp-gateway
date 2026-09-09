package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/caldav"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/deck"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/hitl"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/ocs"
	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/webdav"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Server coordinates MCP tool dispatching, HITL policies, and protocol clients.
type Server struct {
	mcpServer *mcpserver.MCPServer
	davCli    *webdav.Client
	caldavCli *caldav.Client
	deckCli   *deck.Client
	ocsCli    *ocs.Client
	hitlMgr   *hitl.Manager
}

// NewServer registers all 16 Nextcloud MCP tools and returns a configured Server.
func NewServer(
	davCli *webdav.Client,
	caldavCli *caldav.Client,
	deckCli *deck.Client,
	ocsCli *ocs.Client,
	hitlMgr *hitl.Manager,
) *Server {
	mcpSrv := mcpserver.NewMCPServer("nextcloud-mcp-gateway", "2.0.0")

	s := &Server{
		mcpServer: mcpSrv,
		davCli:    davCli,
		caldavCli: caldavCli,
		deckCli:   deckCli,
		ocsCli:    ocsCli,
		hitlMgr:   hitlMgr,
	}

	s.registerTools()
	return s
}

// MCPServer returns the underlying MCP server instance.
func (s *Server) MCPServer() *mcpserver.MCPServer {
	return s.mcpServer
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("JSON serialization error: %v", err)), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

func (s *Server) registerTools() {
	// 1. execute_pending_action
	s.mcpServer.AddTool(
		mcp.NewTool("execute_pending_action",
			mcp.WithDescription("Execute a destructive action that was previously blocked by HITL."),
			mcp.WithString("token", mcp.Required(), mcp.Description("One-time token from the blocked action")),
		),
		s.handleExecutePendingAction,
	)

	// 2. nextcloud_health
	s.mcpServer.AddTool(
		mcp.NewTool("nextcloud_health",
			mcp.WithDescription("Check Nextcloud instance health, version, status.php and WebDAV availability."),
		),
		s.handleNextcloudHealth,
	)

	// 3. list_files
	s.mcpServer.AddTool(
		mcp.NewTool("list_files",
			mcp.WithDescription("List files and folders in a Nextcloud directory via WebDAV PROPFIND."),
			mcp.WithString("path", mcp.Description("Directory path in Nextcloud storage (default: '/')")),
			mcp.WithNumber("offset", mcp.Description("Item offset for pagination (default: 0)")),
			mcp.WithNumber("limit", mcp.Description("Maximum items to return (default: 50, max: 100)")),
		),
		s.handleListFiles,
	)

	// 4. read_file
	s.mcpServer.AddTool(
		mcp.NewTool("read_file",
			mcp.WithDescription("Read textual content of a file from Nextcloud storage."),
			mcp.WithString("path", mcp.Required(), mcp.Description("Path to file in Nextcloud storage")),
		),
		s.handleReadFile,
	)

	// 5. write_file (HITL)
	s.mcpServer.AddTool(
		mcp.NewTool("write_file",
			mcp.WithDescription("Create or overwrite a file in Nextcloud storage. (HITL protected)"),
			mcp.WithString("path", mcp.Required(), mcp.Description("Target file path in Nextcloud")),
			mcp.WithString("content", mcp.Required(), mcp.Description("Text content to write")),
		),
		s.handleWriteFile,
	)

	// 6. delete_file (HITL)
	s.mcpServer.AddTool(
		mcp.NewTool("delete_file",
			mcp.WithDescription("Delete a file or folder from Nextcloud storage. (HITL protected)"),
			mcp.WithString("path", mcp.Required(), mcp.Description("Target path to delete in Nextcloud")),
		),
		s.handleDeleteFile,
	)

	// 7. create_folder (HITL)
	s.mcpServer.AddTool(
		mcp.NewTool("create_folder",
			mcp.WithDescription("Create a new folder in Nextcloud storage. (HITL protected)"),
			mcp.WithString("path", mcp.Required(), mcp.Description("Target folder path to create")),
		),
		s.handleCreateFolder,
	)

	// 8. get_user_info
	s.mcpServer.AddTool(
		mcp.NewTool("get_user_info",
			mcp.WithDescription("Retrieve user storage quota, display name, and details via Nextcloud OCS API."),
		),
		s.handleGetUserInfo,
	)

	// 9. list_deck_boards
	s.mcpServer.AddTool(
		mcp.NewTool("list_deck_boards",
			mcp.WithDescription("List all Nextcloud Deck Kanban boards available to the user."),
		),
		s.handleListDeckBoards,
	)

	// 10. list_deck_stacks
	s.mcpServer.AddTool(
		mcp.NewTool("list_deck_stacks",
			mcp.WithDescription("List all stacks (columns) in a Nextcloud Deck board."),
			mcp.WithNumber("board_id", mcp.Required(), mcp.Description("Board ID")),
		),
		s.handleListDeckStacks,
	)

	// 11. create_deck_card
	s.mcpServer.AddTool(
		mcp.NewTool("create_deck_card",
			mcp.WithDescription("Create a new Kanban card in Nextcloud Deck."),
			mcp.WithNumber("board_id", mcp.Required(), mcp.Description("Board ID")),
			mcp.WithNumber("stack_id", mcp.Required(), mcp.Description("Stack ID")),
			mcp.WithString("title", mcp.Required(), mcp.Description("Card title")),
			mcp.WithString("description", mcp.Description("Card description")),
		),
		s.handleCreateDeckCard,
	)

	// 12. update_deck_card
	s.mcpServer.AddTool(
		mcp.NewTool("update_deck_card",
			mcp.WithDescription("Update an existing Kanban card in Nextcloud Deck (e.g. to move it to another stack/column)."),
			mcp.WithNumber("board_id", mcp.Required(), mcp.Description("Board ID")),
			mcp.WithNumber("stack_id", mcp.Required(), mcp.Description("Stack ID")),
			mcp.WithNumber("card_id", mcp.Required(), mcp.Description("Card ID")),
			mcp.WithString("title", mcp.Required(), mcp.Description("Card title")),
			mcp.WithString("description", mcp.Description("Card description")),
			mcp.WithNumber("order", mcp.Description("Card order within stack")),
		),
		s.handleUpdateDeckCard,
	)

	// 13. delete_deck_card
	s.mcpServer.AddTool(
		mcp.NewTool("delete_deck_card",
			mcp.WithDescription("Delete a Kanban card in Nextcloud Deck."),
			mcp.WithNumber("board_id", mcp.Required(), mcp.Description("Board ID")),
			mcp.WithNumber("stack_id", mcp.Required(), mcp.Description("Stack ID")),
			mcp.WithNumber("card_id", mcp.Required(), mcp.Description("Card ID")),
		),
		s.handleDeleteDeckCard,
	)

	// 14. list_calendar_events
	s.mcpServer.AddTool(
		mcp.NewTool("list_calendar_events",
			mcp.WithDescription("List events from a Nextcloud CalDAV calendar."),
			mcp.WithString("calendar_name", mcp.Description("Calendar name (default: 'personal')")),
		),
		s.handleListCalendarEvents,
	)

	// 15. create_calendar_event
	s.mcpServer.AddTool(
		mcp.NewTool("create_calendar_event",
			mcp.WithDescription("Create a new event in Nextcloud CalDAV calendar using iCalendar (.ics)."),
			mcp.WithString("event_uid", mcp.Required(), mcp.Description("Unique event identifier")),
			mcp.WithString("summary", mcp.Required(), mcp.Description("Event title or summary")),
			mcp.WithString("dtstart", mcp.Required(), mcp.Description("Start timestamp (e.g. 20260909T140000Z)")),
			mcp.WithString("dtend", mcp.Required(), mcp.Description("End timestamp (e.g. 20260909T150000Z)")),
			mcp.WithString("calendar_name", mcp.Description("Calendar name (default: 'personal')")),
		),
		s.handleCreateCalendarEvent,
	)

	// 16. delete_calendar_event
	s.mcpServer.AddTool(
		mcp.NewTool("delete_calendar_event",
			mcp.WithDescription("Delete an event from a Nextcloud CalDAV calendar."),
			mcp.WithString("event_uid", mcp.Required(), mcp.Description("Unique event identifier")),
			mcp.WithString("calendar_name", mcp.Description("Calendar name (default: 'personal')")),
		),
		s.handleDeleteCalendarEvent,
	)
}

func (s *Server) handleExecutePendingAction(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	token := req.GetString("token", "")
	if token == "" {
		return jsonResult(map[string]any{"status": "error", "error": "Token is required"})
	}

	act, err := s.hitlMgr.Pop(token)
	if err != nil {
		return jsonResult(map[string]any{"status": "error", "error": err.Error()})
	}

	switch act.Type {
	case "delete_file":
		p, _ := act.Details["path"].(string)
		res, _ := s.davCli.DeleteResource(ctx, p)
		return jsonResult(res)
	case "write_file":
		p, _ := act.Details["path"].(string)
		c, _ := act.Details["content"].(string)
		res, _ := s.davCli.WriteFile(ctx, p, c)
		return jsonResult(res)
	case "create_folder":
		p, _ := act.Details["path"].(string)
		res, _ := s.davCli.CreateFolder(ctx, p)
		return jsonResult(res)
	default:
		return jsonResult(map[string]any{"status": "error", "error": fmt.Sprintf("Unknown action type: %s", act.Type)})
	}
}

func (s *Server) handleNextcloudHealth(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	res, _ := s.ocsCli.HealthCheck(ctx)
	return jsonResult(res)
}

func (s *Server) handleListFiles(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := req.GetString("path", "/")
	offset := req.GetInt("offset", 0)
	limit := req.GetInt("limit", 50)
	res, _ := s.davCli.ListFiles(ctx, p, offset, limit)
	return jsonResult(res)
}

func (s *Server) handleReadFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := req.GetString("path", "")
	res, _ := s.davCli.ReadFile(ctx, p)
	return jsonResult(res)
}

func (s *Server) handleWriteFile(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := req.GetString("path", "")
	c := req.GetString("content", "")
	res := s.hitlMgr.Request("write_file", map[string]any{"path": p, "content": c})
	return jsonResult(res)
}

func (s *Server) handleDeleteFile(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := req.GetString("path", "")
	res := s.hitlMgr.Request("delete_file", map[string]any{"path": p})
	return jsonResult(res)
}

func (s *Server) handleCreateFolder(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := req.GetString("path", "")
	res := s.hitlMgr.Request("create_folder", map[string]any{"path": p})
	return jsonResult(res)
}

func (s *Server) handleGetUserInfo(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	res, _ := s.ocsCli.GetUserInfo(ctx)
	return jsonResult(res)
}

func (s *Server) handleListDeckBoards(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	res, _ := s.deckCli.ListBoards(ctx)
	return jsonResult(res)
}

func (s *Server) handleListDeckStacks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	boardID := req.GetInt("board_id", 0)
	res, _ := s.deckCli.ListStacks(ctx, boardID)
	return jsonResult(res)
}

func (s *Server) handleCreateDeckCard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	boardID := req.GetInt("board_id", 0)
	stackID := req.GetInt("stack_id", 0)
	title := req.GetString("title", "")
	desc := req.GetString("description", "")
	res, _ := s.deckCli.CreateCard(ctx, boardID, stackID, title, desc)
	return jsonResult(res)
}

func (s *Server) handleUpdateDeckCard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	boardID := req.GetInt("board_id", 0)
	stackID := req.GetInt("stack_id", 0)
	cardID := req.GetInt("card_id", 0)
	title := req.GetString("title", "")
	desc := req.GetString("description", "")
	order := req.GetInt("order", 0)
	res, _ := s.deckCli.UpdateCard(ctx, boardID, stackID, cardID, title, desc, order)
	return jsonResult(res)
}

func (s *Server) handleDeleteDeckCard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	boardID := req.GetInt("board_id", 0)
	stackID := req.GetInt("stack_id", 0)
	cardID := req.GetInt("card_id", 0)
	res, _ := s.deckCli.DeleteCard(ctx, boardID, stackID, cardID)
	return jsonResult(res)
}

func (s *Server) handleListCalendarEvents(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cal := req.GetString("calendar_name", "personal")
	res, _ := s.caldavCli.ListEvents(ctx, cal)
	return jsonResult(res)
}

func (s *Server) handleCreateCalendarEvent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uid := req.GetString("event_uid", "")
	summary := req.GetString("summary", "")
	dtstart := req.GetString("dtstart", "")
	dtend := req.GetString("dtend", "")
	cal := req.GetString("calendar_name", "personal")
	res, _ := s.caldavCli.CreateEvent(ctx, uid, summary, dtstart, dtend, cal)
	return jsonResult(res)
}

func (s *Server) handleDeleteCalendarEvent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uid := req.GetString("event_uid", "")
	cal := req.GetString("calendar_name", "personal")
	res, _ := s.caldavCli.DeleteEvent(ctx, uid, cal)
	return jsonResult(res)
}
