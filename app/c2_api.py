import json
from datetime import datetime
from urllib.parse import urlparse

from aiohttp import web

from app.base.c2 import C2


class Api(C2):

    def __init__(self, app, services):
        self.agent_svc = services.get('agent_svc')
        app.router.add_route('POST', '/sand/ping', self._ping)
        app.router.add_route('POST', '/sand/instructions', self._instructions)
        app.router.add_route('POST', '/sand/results', self._results)
        super().__init__(services)
        self.log = self.add_c2('api', self)

    """ PRIVATE """

    async def _ping(self, request):
        return web.Response(text=self.agent_svc.encode_string('pong'))

    async def _instructions(self, request):
        data = json.loads(self.agent_svc.decode_bytes(await request.read()))
        url = urlparse(data['server'])
        port = '443' if url.scheme == 'https' else 80
        data['server'] = '%s://%s:%s' % (url.scheme, url.hostname, url.port if url.port else port)
        agent = await self.agent_svc.handle_heartbeat(**data)
        instructions = await self.agent_svc.get_instructions(data['paw'])
        response = dict(sleep=await agent.calculate_sleep(), instructions=instructions)
        return web.Response(text=self.agent_svc.encode_string(json.dumps(response)))

    async def _results(self, request):
        data = json.loads(self.agent_svc.decode_bytes(await request.read()))
        data['time'] = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        status = await self.agent_svc.save_results(data['id'], data['output'], data['status'], data['pid'])
        return web.Response(text=self.agent_svc.encode_string(status))
