import asyncio
import json


async def handle(reader, writer):
    req = json.loads(await reader.readline())
    result = {
        "init": {"pluginTemplateTypes": ["hello"]},
        "executeNode": {"phase": "Succeeded", "message": "hi"},
        "reconcileNode": {}
    }[req['method']]
    resp = {"id": 0, "jsonrpc": "2.0", "result": result}

    writer.write(json.dumps(resp).encode())
    await writer.drain()

    writer.close()


async def main():
    server = await asyncio.start_server(handle, '127.0.0.1', 12345)
    async with server:
        await server.serve_forever()


asyncio.run(main())
