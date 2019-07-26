import json
from datetime import datetime

from aiohttp import web

from plugins.sandcat.app.sand_svc import SandService


class SandApi:

    def __init__(self, services):
        self.sand_svc = SandService(services)
        self.utility_svc = services.get('utility_svc')
        self.file_svc = services.get('file_svc')

    async def download(self, request):
        platform = request.headers.get('platform')
        await self.sand_svc.compile(platform)
        file_path, headers = await self.file_svc.find_file(platform)
        if file_path:
            return web.FileResponse(path=file_path, headers=headers)
        return web.HTTPNotFound(body='File not found')

    async def beacon(self, request):
        paw = request.headers.get('X-PAW')
        data = json.loads(self.utility_svc.decode_bytes(await request.read()))
        data['server'] = '%s://%s' % (request.scheme, request.host)
        agent_id = await self.sand_svc.beacon(paw, **data)
        instructions = await self.sand_svc.instructions(agent_id)
        return web.Response(text=self.utility_svc.encode_string(instructions))

    async def results(self, request):
        paw = request.headers.get('X-PAW')
        data = json.loads(self.utility_svc.decode_bytes(await request.read()))
        data['time'] = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        status = await self.sand_svc.post_results(paw, data['link_id'], data['output'], data['status'])
        return web.Response(text=self.utility_svc.encode_string(status))
