package hub

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/podscope/podscope/pkg/protocol"
	"google.golang.org/grpc"
)

// GRPCServer handles agent connections
type GRPCServer struct {
	server    *Server
	agents    map[string]*AgentConnection
	agentsMux sync.RWMutex
}

// AgentConnection represents a connected agent
type AgentConnection struct {
	ID            string
	PodName       string
	Namespace     string
	PodIP         string
	ConnectedAt   time.Time
	LastHeartbeat time.Time
	Stats         AgentStats
}

// AgentStats holds agent statistics
type AgentStats struct {
	PacketsCaptured uint64
	BytesCaptured   uint64
	FlowsDetected   uint64
	Errors          uint64
}

// startGRPCServer starts the gRPC server
func (s *Server) startGRPCServer() (*grpc.Server, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.grpcPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()

	// Register our service
	gs := &GRPCServer{
		server: s,
		agents: make(map[string]*AgentConnection),
	}
	registerAgentService(grpcServer, gs)

	go func() {
		log.Printf("gRPC server listening on port %d", s.grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	return grpcServer, nil
}

// Message types for gRPC (simplified without protobuf generation)
type FlowEventMsg struct {
	AgentID string
	Flow    *protocol.Flow
}

type PCAPChunkMsg struct {
	AgentID   string
	Timestamp int64
	Data      []byte
}

type AgentInfoMsg struct {
	ID        string
	PodName   string
	Namespace string
	PodIP     string
	NodeName  string
	SessionID string
}

type RegisterResponseMsg struct {
	Success bool
	Message string
}

type HeartbeatRequestMsg struct {
	AgentID   string
	Timestamp int64
}

type HeartbeatResponseMsg struct {
	ContinueCapture bool
	Message         string
	BPFFilter       string // If set, agent should update its BPF filter
}

// registerAgentService registers the gRPC service with proper handler signatures
func registerAgentService(s *grpc.Server, gs *GRPCServer) {
	s.RegisterService(&grpc.ServiceDesc{
		ServiceName: "podscope.AgentService",
		HandlerType: (*interface{})(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "RegisterAgent",
				Handler:    gs.registerAgentHandler,
			},
			{
				MethodName: "Heartbeat",
				Handler:    gs.heartbeatHandler,
			},
		},
		Streams: []grpc.StreamDesc{
			{
				StreamName:    "StreamFlows",
				Handler:       gs.streamFlowsHandler,
				ClientStreams: true,
			},
			{
				StreamName:    "StreamPCAP",
				Handler:       gs.streamPCAPHandler,
				ClientStreams: true,
			},
		},
		Metadata: "podscope.proto",
	}, gs)
}

func (gs *GRPCServer) registerAgentHandler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	var info AgentInfoMsg
	if err := dec(&info); err != nil {
		return nil, err
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		i := req.(*AgentInfoMsg)
		gs.agentsMux.Lock()
		gs.agents[i.ID] = &AgentConnection{
			ID:            i.ID,
			PodName:       i.PodName,
			Namespace:     i.Namespace,
			PodIP:         i.PodIP,
			ConnectedAt:   time.Now(),
			LastHeartbeat: time.Now(),
		}
		gs.agentsMux.Unlock()

		log.Printf("Agent registered: %s (%s/%s)", i.ID, i.Namespace, i.PodName)

		return &RegisterResponseMsg{
			Success: true,
			Message: "Agent registered successfully",
		}, nil
	}

	if interceptor == nil {
		return handler(ctx, &info)
	}
	return interceptor(ctx, &info, &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/podscope.AgentService/RegisterAgent",
	}, handler)
}

func (gs *GRPCServer) heartbeatHandler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	var req HeartbeatRequestMsg
	if err := dec(&req); err != nil {
		return nil, err
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		r := req.(*HeartbeatRequestMsg)
		gs.agentsMux.Lock()
		if agent, ok := gs.agents[r.AgentID]; ok {
			agent.LastHeartbeat = time.Now()
		}
		gs.agentsMux.Unlock()

		// Get current BPF filter from server
		gs.server.bpfFilterMutex.RLock()
		currentFilter := gs.server.bpfFilter
		gs.server.bpfFilterMutex.RUnlock()

		return &HeartbeatResponseMsg{
			ContinueCapture: true,
			Message:         "OK",
			BPFFilter:       currentFilter,
		}, nil
	}

	if interceptor == nil {
		return handler(ctx, &req)
	}
	return interceptor(ctx, &req, &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/podscope.AgentService/Heartbeat",
	}, handler)
}

func (gs *GRPCServer) streamFlowsHandler(srv interface{}, stream grpc.ServerStream) error {
	for {
		var event FlowEventMsg
		if err := stream.RecvMsg(&event); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		// Add flow to server
		if event.Flow != nil {
			gs.server.AddFlow(event.Flow)
		}

		// Update agent stats
		gs.agentsMux.Lock()
		if agent, ok := gs.agents[event.AgentID]; ok {
			agent.Stats.FlowsDetected++
		}
		gs.agentsMux.Unlock()
	}
}

func (gs *GRPCServer) streamPCAPHandler(srv interface{}, stream grpc.ServerStream) error {
	for {
		var chunk PCAPChunkMsg
		if err := stream.RecvMsg(&chunk); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		// Write PCAP data
		if err := gs.server.AddPCAPData(chunk.AgentID, chunk.Data); err != nil {
			log.Printf("Failed to write PCAP data: %v", err)
		}

		// Update agent stats
		gs.agentsMux.Lock()
		if agent, ok := gs.agents[chunk.AgentID]; ok {
			agent.Stats.BytesCaptured += uint64(len(chunk.Data))
		}
		gs.agentsMux.Unlock()
	}
}

// GetConnectedAgents returns a list of connected agents
func (gs *GRPCServer) GetConnectedAgents() []AgentConnection {
	gs.agentsMux.RLock()
	defer gs.agentsMux.RUnlock()

	agents := make([]AgentConnection, 0, len(gs.agents))
	for _, agent := range gs.agents {
		agents = append(agents, *agent)
	}
	return agents
}
