# dapr-mcp
MCP Server for Dapr APIs

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