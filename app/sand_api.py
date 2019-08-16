import json
from datetime import datetime

from aiohttp import web

from plugins.sandcat.app.sand_svc import SandService


class SandApi:

    def __init__(self, services):
        self.sand_svc = SandService(services)
        self.utility_svc = services.get('utility_svc')
        self.file_svc = services.get('file_svc')
        self.agent_svc = services.get('agent_svc')

    async def instructions(self, request):
        paw = request.headers.get('X-PAW')
        data = json.loads(self.utility_svc.decode_bytes(await request.read()))
        data['server'] = '%s://%s' % (request.scheme, request.host)
        await self.agent_svc.handle_heartbeat(paw, **data)
        instructions = await self.sand_svc.instructions(paw)
        return web.Response(text=self.utility_svc.encode_string(instructions))

    async def results(self, request):
        paw = request.headers.get('X-PAW')
        data = json.loads(self.utility_svc.decode_bytes(await request.read()))
        data['time'] = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        status = await self.sand_svc.post_results(paw, data['link_id'], data['output'], data['status'])
        return web.Response(text=self.utility_svc.encode_string(status))
