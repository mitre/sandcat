import json
from datetime import datetime

from aiohttp import web

from plugins.sandcat.app.sand_svc import SandService


class SandApi:

    def __init__(self, services):
        self.sand_svc = SandService(services)
        self.utility_svc = services.get('utility_svc')

    async def registration(self, request):
        paw = request.headers.get('X-PAW')
        data = json.loads(self.utility_svc.decode_bytes(await request.read()))
        data['server'] = '%s://%s' % (request.scheme, request.host)
        registration = await self.sand_svc.registration(paw, **data)
        return web.Response(text=self.utility_svc.encode_string(registration))

    async def instructions(self, request):
        paw = request.headers.get('X-PAW')
        agent = await self.sand_svc.check_in(paw)
        if not agent:
            return web.Response(text=json.dumps(dict(status=False)))
        instructions = await self.sand_svc.instructions(agent)
        return web.Response(text=self.utility_svc.encode_string(instructions))

    async def results(self, request):
        paw = request.headers.get('X-PAW')
        data = json.loads(self.utility_svc.decode_bytes(await request.read()))
        data['time'] = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        status = await self.sand_svc.post_results(paw, data['link_id'], data['output'], data['status'])
        return web.Response(text=self.utility_svc.encode_string(status))
