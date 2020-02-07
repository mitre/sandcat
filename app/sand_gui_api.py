from aiohttp_jinja2 import template


class SandGuiApi:

    def __init__(self, services):
        self.auth_svc = services.get('auth_svc')
        self.app_svc = services.get('app_svc')
        self.data_svc = services.get('data_svc')

    @template('sandcat.html')
    async def splash(self, request):
        await self.auth_svc.check_permissions(request)
        delivery_cmds = await self.data_svc.locate('abilities', dict(ability_id='2f34977d-9558-4c12-abad-349716777c6b'))
        return dict(delivery_cmds=delivery_cmds)
