#!/usr/bin/env python3
import asyncio
import logging

from dapr_agents import DurableAgent
from dapr_agents.agents.configs import (
    AgentExecutionConfig,
    AgentMemoryConfig,
    AgentPubSubConfig,
    AgentRegistryConfig,
    AgentStateConfig,
)
from dapr_agents.memory import ConversationDaprStateMemory
from dapr_agents.storage.daprstores.stateservice import StateStoreService
from dapr_agents.tool.mcp import MCPClient
from dapr_agents.workflow.runners import AgentRunner
from dapr_agents.llm import DaprChatClient

async def _load_mcp_tools() -> list:
    client = MCPClient()
    await client.connect_sse("local", url="http://localhost:8080")
    return client.get_all_tools()


def main() -> None:
    logging.basicConfig(level=logging.INFO)

    try:
        tools = asyncio.run(_load_mcp_tools())
    except Exception:
        logging.exception("Failed to load MCP tools via SSE")
        return

    asyncio.set_event_loop(asyncio.new_event_loop())

    agent = DurableAgent(
        name="Steve",
        role="Dapr Client",
        goal="Help humans interact with Dapr through your MCP tool.",
        instructions=[
            "Answer clearly and helpfully.",
            "Call MCP tools to satisfy the users asks.",
        ],
        llm=DaprChatClient(component_name='ollama'),
        tools=tools,
        pubsub = AgentPubSubConfig(
            pubsub_name="messagepubsub",
            agent_topic="agent.requests",
            broadcast_topic="agents.broadcast",
        ),
        state = AgentStateConfig(
            store=StateStoreService(store_name="agentstatestore"),
        ),
        registry = AgentRegistryConfig(
            store=StateStoreService(store_name="agentstatestore"),
            team_name="agent-team",
        ),
        execution = AgentExecutionConfig(max_iterations=4),
        memory = AgentMemoryConfig(
            store=ConversationDaprStateMemory(
                store_name="agentstatestore",
                session_id="agent-session",
            )
        ),
    )

    agent.start()
    runner = AgentRunner()
    try:
        runner.serve(agent, port=8001)
    finally:
        runner.shutdown(agent)

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        pass
