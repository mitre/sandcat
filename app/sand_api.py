import json
from datetime import datetime

from aiohttp import web


class SandApi:

    def __init__(self, services):
        self.file_svc = services.get('file_svc')
        self.agent_svc = services.get('agent_svc')

    async def instructions(self, request):
        data = json.loads(self.agent_svc.decode_bytes(await request.read()))
        data['server'] = '%s://%s' % (request.scheme, request.host)
        data['pid'] = data['ppid'] = 1
        await self.agent_svc.handle_heartbeat(**data)
        instructions = await self.agent_svc.get_instructions(data['paw'])
        return web.Response(text=self.agent_svc.encode_string(instructions))

    async def results(self, request):
        data = json.loads(self.agent_svc.decode_bytes(await request.read()))
        data['time'] = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        status = await self.agent_svc.save_results(data['link_id'], data['output'], data['status'])
        return web.Response(text=self.agent_svc.encode_string(status))
