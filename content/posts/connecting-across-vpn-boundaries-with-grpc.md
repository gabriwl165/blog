---
title: "Bypassing VPNs: Connecting Across the VPN Boundary with a Long-Lived Bidirectional gRPC Stream"
date: 2026-09-16T00:00:00-03:00
draft: false
description: "How a long-lived bidirectional gRPC stream connects integration pipelines with systems inside customer VPNs."
tags: ["Go", "gRPC", "Distributed Systems"]
categories: ["Engineering"]
---

# Connecting Across the VPN Boundary with a Long-Lived Bidirectional gRPC Stream

Integrations often need to reach systems that customers deliberately keep off the public internet. That can mean connecting an integration pipeline to a customer’s CMMS, ERP, or industrial IoT systems while those systems remain inside a private network protected by a VPN.

The approach presented in this talk is to give the customer environment an Agent and let that Agent establish a long-lived, bidirectional gRPC connection to a Proxy. The Agent connects outward, registers its identity, and keeps the stream open. When the Integration Pipeline needs to perform work, it sends a request containing an Agent ID. The Proxy uses that identity to select the corresponding live session and sends the request back through the same connection.

This is not a slide-by-slide implementation guide. It is a compact explanation of the design shown in the 11-slide talk: the connectivity constraint, the proxy bridge, the Agent session lifecycle, an SAP example, and an extension to industrial telemetry. The code and payloads below are simplified illustrations of those ideas, not production-ready code.

## The connectivity constraint

The starting point is straightforward: an integration platform needs access to customer systems, but those systems are not public. The integration runs in external infrastructure, while the customer’s CMMS, ERP, and IoT sensors sit in a private network behind a VPN. A direct call from the public side cannot simply cross that boundary.

A proxy provides the missing bridge. Instead of exposing the customer’s internal systems to the public internet, a customer-side Agent can make a connection to a public endpoint owned by the proxy. The proxy then carries integration calls into the customer network through the Agent.

The important shift is the direction of the connection. The Agent initiates the channel to the Proxy. Once that channel exists, the Proxy can use it in both directions: the Agent can send its registration and remain available, while the Integration Pipeline can send requests back to the Agent over the established stream.

## One public listener, many customer sessions

The talk’s central mechanism is a long-lived bidirectional gRPC connection. The Proxy exposes a gRPC listener on `:50051`, and each customer Agent opens a stream to that listener. The stream is not created for one request and immediately discarded; it represents the Agent’s active session.

Conceptually, the Proxy has two responsibilities:

1. Accept and keep Agent streams alive.
2. Route an integration request to the right stream.

The first responsibility is represented by a normal gRPC server listener. In simplified Go, the setup looks like this:

```go
// Illustrative only: the talk's simplified proxy listener.
lis, err := net.Listen("tcp", ":50051")
if err != nil {
    log.Fatalf("listen for gRPC: %v", err)
}

grpcServer := grpc.NewServer()
go func() {
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("serve gRPC: %v", err)
    }
}()
```

This snippet shows the public entry point, not a complete server. The meaningful architectural detail is that the listener is ready for the Agent’s long-lived connection. The actual work happens when the Agent connects, identifies itself, and the Proxy associates that stream with an Agent ID.

## The Agent registers and stays connected

On the customer side, the Agent opens a gRPC client channel to the Proxy, creates the control client, and calls a bidirectional method represented in the talk as `ConnectChannel`. The first message carries the Agent’s identity. After registration, the Agent remains connected rather than treating the channel as a one-shot request.

An intentionally simplified version of that flow is:

```go
// Illustrative only; credentials, retries, and shutdown handling are omitted.
conn, err := grpc.NewClient(
    "localhost:50051",
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
if err != nil {
    log.Fatalf("connection failed: %v", err)
}
defer conn.Close()

client := pb.NewAgentControlClient(conn)
stream, err := client.ConnectChannel(context.Background())
if err != nil {
    log.Fatalf("stream creation failed: %v", err)
}

// First message: register this Agent with the Proxy.
err = stream.Send(&pb.AgentPayload{AgentId: agentID})
```

The talk uses this registration message to establish the relationship between an identity and a live connection. For example, one customer environment can register as `Agent A-102`, while another can register as `Agent B-204`. Both sessions can be connected to the same Proxy, but requests addressed to one must not be delivered to the other.

The example uses `insecure.NewCredentials()` to keep the mechanics visible in a small code sample. That is only an illustration of the API shape in the talk. It is not a statement about deployment transport security, and it should not be read as a production configuration.

## Agent-ID routing and session lifecycle

After receiving the initial registration message, the Proxy stores a session under the supplied Agent ID. A simplified server-side handler looks like this:

```go
// Illustrative only: synchronization, validation, and request handling are omitted.
func (s *ProxyServer) ConnectChannel(
    stream pb.AgentControl_ConnectChannelServer,
) error {
    initMsg, err := stream.Recv()
    if err != nil {
        return err
    }

    agentID := initMsg.AgentId
    session := &AgentSession{stream: stream}
    s.agents.Store(agentID, session)
    fmt.Printf("[gRPC] Agent '%s' connected and registered.\n", agentID)

    // Keep the session registered until the Agent disconnects.
    <-stream.Context().Done()

    s.agents.Delete(agentID)
    fmt.Printf("[gRPC] Agent '%s' disconnected.\n", agentID)
    return stream.Context().Err()
}
```

There are two lifecycle events in this example. Registration adds the Agent to the connected-agent registry. Disconnection removes it. The Proxy waits on the stream context so that the mapping remains available for the lifetime of the connection; it is not removed immediately after the first message.

That lifecycle is what makes routing possible. A request from the Integration Pipeline includes an Agent ID, the Proxy looks up the active session, and the selected Agent receives the request on its existing channel. If `Agent A-102` is selected, `Agent B-204` remains connected but is not involved in that dispatch. In the talk’s diagram, A-102 is the selected session and B-204 is standing by.

The Agent ID is therefore more than descriptive metadata. It is the routing key that turns a shared public listener into separate paths to customer environments. The pipeline does not need to address the customer’s private host directly; it asks the Proxy to reach the Agent associated with the identity in the request.

## A concrete use case: SAP through the Agent

The connection becomes more useful when the Agent can reach systems locally inside the customer environment. The talk uses SAP as the example. The Agent connects to the customer’s internally hosted SAP system through SAP’s proprietary RFC protocol. It can read data, transform it, and write data back. The Integration Pipeline initiates the operation, but the Agent performs the local SAP interaction.

The path is:

```text
Integration Pipeline
        │ request + Agent ID
        ▼
Proxy Server
        │ route to the matching live session
        ▼
Agent A-102
        │ SAP RFC through gorfc
        ▼
Customer's internal SAP ERP
```

The talk identifies `gorfc` as an open-source Go library used by the Agent for the SAP RFC interaction. The important separation is between routing and execution. The Proxy does not need to be inside the customer’s SAP network or understand the local ERP connection. It selects the Agent; the Agent invokes SAP locally.

The illustrated request in the talk is:

```json
{
  "agent_id": "Agent A-102",
  "function": "CRIAR_ORDEM_SERVICO",
  "host": "10.1.2.4",
  "destination": "SAP"
}
```

This JSON is an illustrative dispatch payload, not a declared wire contract. It shows the information relevant to the routing story: the target Agent, the SAP function, the destination, and the example internal host. `CRIAR_ORDEM_SERVICO` represents the SAP function the selected Agent should invoke to create a service order.

The dispatch sequence is deliberately simple:

1. The Integration Pipeline creates a request containing `agent_id` and the function to invoke.
2. The Proxy resolves `Agent A-102` against its active sessions.
3. The Proxy delivers the request to that Agent over the long-lived gRPC stream.
4. The Agent invokes `CRIAR_ORDEM_SERVICO` locally through SAP RFC.

The stream is bidirectional, so it can carry messages in both directions. However, the talk does not define a response contract or a result-handling path for the SAP operation. Its point is the established bidirectional channel and the Agent’s local invocation, not a specific way to return or process the outcome.

## Extending the path to industrial IoT

The final technical slide applies the same architecture to telemetry. The sensors remain on-premise. Temperature and vibration signals enter a broker or gateway through industrial protocols such as MQTT and OPC UA. The Agent reads those signals and streams telemetry through the existing long-lived bidirectional gRPC connection.

The conceptual path is:

```text
Temperature and vibration sensors
        │ MQTT / OPC UA
        ▼
On-premise broker or gateway
        ▼
On-premise Agent
        │ long-lived bidirectional gRPC
        ▼
Proxy
        ▼
Machine Learning pipeline
```

This extension preserves the boundary from the original problem. Sensors and the industrial protocol ingress stay in the customer environment; they do not need to become publicly reachable. The Agent is the local reader and stream producer, while the Proxy provides the path to the Machine Learning pipeline.

It also shows why a persistent channel is useful beyond request/response integrations. The SAP example sends a targeted operation to one Agent. The IoT example uses the same connection to carry a continuing flow of signals. One architecture can therefore support both commands toward an on-premise system and telemetry moving toward the integration platform, while Agent-ID routing keeps customer sessions distinct.

## Conclusion

The design presented in the talk addresses a common integration constraint without requiring customer systems to be exposed directly to the public internet. A customer-side Agent opens a connection to a Proxy, registers its Agent ID, and keeps a bidirectional gRPC stream alive. The Proxy holds the session, resolves incoming requests by Agent ID, and delivers each request to the matching environment.

For SAP, that means the Integration Pipeline can route a function such as `CRIAR_ORDEM_SERVICO` to `Agent A-102`, where the Agent uses SAP’s RFC protocol locally. For industrial IoT, the same path lets an on-premise Agent read MQTT and OPC UA telemetry and stream it toward a Machine Learning pipeline.

The core idea is compact: establish the path from inside the customer network, keep it available, and route through the identity of the connected Agent. A single public gRPC listener becomes a set of logical, customer-specific paths without turning the private systems themselves into public endpoints.
