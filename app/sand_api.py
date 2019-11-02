import json
from datetime import datetime

from aiohttp import web
from urllib.parse import urlparse


class SandApi:

    def __init__(self, services):
        self.file_svc = services.get('file_svc')
        self.agent_svc = services.get('agent_svc')

    async def ping(self, request):
        return web.Response(text=self.agent_svc.encode_string('pong'))

    async def instructions(self, request):
        data = json.loads(self.agent_svc.decode_bytes(await request.read()))
        url = urlparse(data['server'])
        port = '443' if url.scheme == 'https' else 80
        data['server'] = '%s://%s:%s' % (url.scheme, url.hostname, url.port if url.port else port)
        agent = await self.agent_svc.handle_heartbeat(**data)
        instructions = await self.agent_svc.get_instructions(data['paw'])
        response = dict(sleep=await agent.calculate_sleep(), instructions=instructions)
        return web.Response(text=self.agent_svc.encode_string(json.dumps(response)))

    async def results(self, request):
        data = json.loads(self.agent_svc.decode_bytes(await request.read()))
        data['time'] = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        status = await self.agent_svc.save_results(data['id'], data['output'], data['status'], data['pid'])
        return web.Response(text=self.agent_svc.encode_string(status))
