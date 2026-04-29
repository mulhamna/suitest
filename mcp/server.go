package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mulhamna/suitest/internal/config"
)

// Server is the MCP SSE server.
type Server struct {
	cfg      *config.Config
	port     int
	mux      *http.ServeMux
	handlers *Handlers
}

// NewServer creates a new MCP server.
func NewServer(cfg *config.Config, port int) *Server {
	s := &Server{
		cfg:  cfg,
		port: port,
		mux:  http.NewServeMux(),
	}
	s.handlers = NewHandlers(cfg)
	s.registerRoutes()
	return s
}

// Start starts the MCP HTTP server.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	fmt.Printf("MCP server listening on http://localhost%s\n", addr)
	return srv.ListenAndServe()
}

func (s *Server) registerRoutes() {
	// MCP JSON-RPC endpoint
	s.mux.HandleFunc("/mcp", s.handleMCP)

	// SSE endpoint for streaming
	s.mux.HandleFunc("/sse", s.handleSSE)

	// Tool listing
	s.mux.HandleFunc("/tools", s.handleTools)

	// Health check
	s.mux.HandleFunc("/health", s.handleHealth)

	// OpenAPI spec for Claude plugin
	s.mux.HandleFunc("/openapi.yaml", s.handleOpenAPI)
}

// MCPRequest is the JSON-RPC 2.0 request format for MCP.
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse is the JSON-RPC 2.0 response format for MCP.
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError is a JSON-RPC error object.
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONRPCError(w, nil, -32700, "parse error: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var result interface{}
	var err error

	switch req.Method {
	case "initialize":
		result = s.handleInitialize(req.Params)
	case "tools/list":
		result = map[string]interface{}{"tools": GetToolDefinitions()}
	case "tools/call":
		result, err = s.handlers.HandleToolCall(r.Context(), req.Params)
	default:
		writeJSONRPCError(w, req.ID, -32601, "method not found: "+req.Method)
		return
	}

	if err != nil {
		writeJSONRPCError(w, req.ID, -32000, err.Error())
		return
	}

	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleInitialize(params json.RawMessage) interface{} {
	return map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "suitest",
			"version": "0.1.0",
		},
	}
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Send endpoint event for MCP
	fmt.Fprintf(w, "event: endpoint\ndata: /mcp\n\n")
	flusher.Flush()

	// Keep connection alive
	<-r.Context().Done()
}

func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": GetToolDefinitions(),
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "suitest-mcp",
		"version": "0.1.0",
	})
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	// Serve the openapi.yaml content inline
	fmt.Fprint(w, openAPISpec)
}

const openAPISpec = `openapi: "3.0.0"
info:
  title: suitest API
  version: "0.1.0"
  description: AI-powered testing agent
servers:
  - url: http://localhost:3100
paths:
  /run:
    post:
      operationId: runTests
      summary: Run AI-powered tests
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [path]
              properties:
                path:
                  type: string
                mode:
                  type: string
                  enum: [auto, browser, api, unit]
                provider:
                  type: string
                fix:
                  type: boolean
      responses:
        "200":
          description: Test results
  /plan:
    post:
      operationId: planTests
      summary: Generate test plan (dry run)
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [path]
              properties:
                path:
                  type: string
                mode:
                  type: string
      responses:
        "200":
          description: Test plan
  /report:
    get:
      operationId: getReport
      summary: Get the latest test report
      responses:
        "200":
          description: Latest report
  /fix:
    post:
      operationId: fixTest
      summary: Apply AI fix to a test file
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [file, error]
              properties:
                file:
                  type: string
                error:
                  type: string
      responses:
        "200":
          description: Fixed test code
`

func writeJSONRPCError(w http.ResponseWriter, id interface{}, code int, message string) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &MCPError{Code: code, Message: message},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
