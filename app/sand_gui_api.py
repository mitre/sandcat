import os
import shutil

from aiohttp import web
from aiohttp_jinja2 import template


class SandGuiApi:

    def __init__(self, services):
        self.auth_svc = services.get('auth_svc')

    @template('sandcat.html')
    async def splash(self, request):
        await self.auth_svc.check_permissions(request)
        return dict(site_status='up' if os.path.isfile('plugins/sandcat/static/malicious/index.html') else 'down')

    @staticmethod
    async def malicious(request):
        return web.HTTPFound('/malicious/index.html')

    async def clone_new_site(self, request):
        await self.auth_svc.check_permissions(request)
        url = request.rel_url.query['url']
        location = 'plugins/sandcat/static/malicious/'
        shutil.rmtree(location, ignore_errors=True)
        os.system(f"""wget -E -H -k -K -p -q -nH --cut-dirs=1 %s --directory %s --no-check-certificate""" % (url, location))
        self.auth_svc.prepend_to_file('%s/index.html' % location, '<script src="/sandcat/js/malicious.js"></script>')
        self.auth_svc.prepend_to_file('%s/index.html' % location, '<meta http-equiv="Expires" content="0">')
        self.auth_svc.prepend_to_file('%s/index.html' % location, '<meta http-equiv="Pragma" content="no-cache">')
        self.auth_svc.prepend_to_file('%s/index.html' % location, '<meta http-equiv="Cache-Control" content="no-cache, no-store, must-revalidate">')
        return web.Response()
