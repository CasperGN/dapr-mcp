# dapr-mcp
MCP Server for Dapr APIs

## Tool Status

As this project is still very much WIP, below you'll find the relevant status of the tools

| tool-category | tool | status | notes | test |
|---|---|--|------|------|
| actors | invoke_actor_method | ? | | |
| bindings | invoke_output_binding | Functional | | Issue an http call with dapr by sending the content of mcp.json as a post request |
| conversation | converse_with_llm | Functional | | Can you converse with the llm in your tools about dapr? |
| crypto | encrypt_data | Not Functional | This tool can get blocked by some models. | |
| crypto | decrypt_data | Not Functional | This tool gets blocked often by standard agents. | |
| invoke | invoke_service | ? | | |
| lock | acquire_lock | Functional | | Get a lock on <filename> |
| lock | release_lock | Functional | | Release the lock |
| metadata | get_components | Functional | | find which components are available |
| pubsub | publish_event | Not Functional | | |
| pubsub | publish_event_with_metadata | Not Functional | | |
| secrets | get_secret | Functional | Description could be better | get the secret "agent-configuration:log_level" |
| secrets | get_bulk_secrets | Functional | | bulk get all secrets |
| state | save_state | Functional | | Save the content of mcp.json with dapr |
| state | get_state | Functional | | Get the saved file |
| state | delete_state | Functional | | Delete the saved file |
| state | execute_transaction | Functional | | Atomically save all files under .github/workflows with dapr |

## OpenTelemetry

This MCP server implements comprehensive OpenTelemetry tracing and baggage propagation. It extracts `traceparent` and `baggage` headers from incoming requests, creates child spans for each tool operation, and injects baggage back into responses. All spans include relevant Dapr operation attributes for better observability.

### Environment Variables

Configure OTEL export by setting the following environment variables:

#### Required
- `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` or `OTEL_EXPORTER_OTLP_ENDPOINT`: The OTLP endpoint URL for your tracing backend
  - For gRPC: `http://localhost:4317` (note: no `grpc://` prefix)
  - For HTTP: `http://localhost:4318/v1/traces`

#### Optional
- `OTEL_EXPORTER_OTLP_TRACES_PROTOCOL` or `OTEL_EXPORTER_OTLP_PROTOCOL`: Protocol to use (`grpc`, `http/protobuf`, `http/json`)
  - Defaults to `grpc` if not specified
- `OTEL_EXPORTER_OTLP_HEADERS`: Additional headers for authentication (e.g., `authorization=Bearer your-token`)
- `OTEL_SERVICE_NAME`: Override the default service name (`daprmcp`)
- `OTEL_SERVICE_VERSION`: Override the default service version (`v1.0.0`)

### Examples

**Jaeger (gRPC):**
```bash
export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://localhost:4317
export OTEL_EXPORTER_OTLP_TRACES_PROTOCOL=grpc
```

**Jaeger (HTTP):**
```bash
export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://localhost:4318/v1/traces
export OTEL_EXPORTER_OTLP_TRACES_PROTOCOL=http/protobuf
```

**With Authentication:**
```bash
export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=https://localhost:443
export OTEL_EXPORTER_OTLP_HEADERS="authorization=Bearer your-token"
```

### Span Attributes

Each tool span includes relevant attributes:
- `dapr.operation`: The Dapr operation type (e.g., `save_state`, `publish_event`)
- `dapr.store`: State store name (for state operations)
- `dapr.key`: State key (for state operations)
- `dapr.pubsub`: PubSub component name (for pubsub operations)
- `dapr.topic`: Topic name (for pubsub operations)
- `dapr.operations_count`: Number of operations (for transactions)

## Run locally

Setup:
```shell
dapr init
```

You'll need 3 terminals:

Terminal 2:
```shell
dapr run --app-id daprmcp --resources-path components -- go run cmd/daprmcp/main.go --http localhost:8080
```

Terminal 3:

```shell
python3.13 -m venv .venv            
source .venv/bin/activate
pip install -r test/requirements.txt
dapr run --app-id daprmcpagent --resources-path test/components -- .venv/bin/python test/app.py
```

Terminal 4:
```shell
curl -i -X POST http://localhost:8001/run -H "Content-Type: application/json" -d '{"task": "Only save this message in a state with a random key. Do nothing else. Message: Hello Steve!"}'
```

> NB: Currently the instructions and details could use improvements. Same can be said for Steve's instructions. This is purely for demo purpose.

## Add to vscode

Run

```shell
dapr run --app-id daprmcp --resources-path components -- go run cmd/daprmcp/main.go --http localhost:8080
```

Add the following file as `.vscode/mcp.json`

```json
{
  "servers": {
    "dapr-mcp": {
      "type": "http",
      "url": "http://localhost:8080"
    }
  }
}
```