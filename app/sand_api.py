import json
from datetime import datetime

import aiohttp_jinja2
from aiohttp import web
from aiohttp_jinja2 import template

from plugins.sandcat.app.sand_svc import SandService


class SandApi:

    def __init__(self, services):
        self.sand_svc = SandService(services)
        self.utility_svc = services.get('utility_svc')

    @template('sandcat.html')
    async def splash(self, request):
        return dict()

    async def registration(self, request):
        data = json.loads(self.utility_svc.decode_bytes(await request.read()))
        data['server'] = '%s://%s' % (request.scheme, request.host)
        registration = await self.sand_svc.registration(**data)
        return web.Response(text=self.utility_svc.encode_string(registration))

    async def instructions(self, request):
        data = json.loads(self.utility_svc.decode_bytes(await request.read()))
        agent = await self.sand_svc.check_in(data['paw'], data['executor'])
        instructions = await self.sand_svc.instructions(agent)
        return web.Response(text=self.utility_svc.encode_string(instructions))

    async def results(self, request):
        data = json.loads(self.utility_svc.decode_bytes(await request.read()))
        data['time'] = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        status = await self.sand_svc.post_results(data['paw'], data['link_id'], data['output'], data['status'])
        return web.Response(text=self.utility_svc.encode_string(status))

    async def render(self, request):
        name = request.headers.get('file')
        group = request.rel_url.query.get('group')
        environment = request.app[aiohttp_jinja2.APP_KEY]
        url_root = '{scheme}://{host}'.format(scheme=request.scheme, host=request.host)
        headers = dict([('CONTENT-DISPOSITION', 'attachment; filename="%s"' % name)])
        rendered = await self.sand_svc.render_file(name, group, environment, url_root)
        if rendered:
            return web.HTTPOk(body=rendered, headers=headers)
        return web.HTTPNotFound(body=rendered)

    async def download(self, request):
        name = request.headers.get('file')
        file_path, headers = await self.sand_svc.download_file(name)
        if file_path:
            return web.FileResponse(path=file_path, headers=headers)
        return web.HTTPNotFound(body='File not found')

