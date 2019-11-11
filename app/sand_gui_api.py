from aiohttp_jinja2 import template


class SandGuiApi:

    def __init__(self, services):
        self.auth_svc = services.get('auth_svc')
        self.app_svc = services.get('app_svc')
        self.data_svc = services.get('data_svc')

    @template('sandcat.html')
    async def splash(self, request):
        await self.auth_svc.check_permissions(request)
        plugins = [dict(name=getattr(p, 'name'), address=getattr(p, 'address')) for p in self.app_svc.get_plugins()]
        return dict(plugins=plugins)
