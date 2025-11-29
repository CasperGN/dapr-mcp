# dapr-mcp
MCP Server for Dapr APIs

## Status

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

## Run locally

Setup:
```shell
dapr init
```

You'll need 4 terminals:

Terminal 1:
```shell
python3 -m http.server 8000
```

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