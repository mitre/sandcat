from aiohttp_jinja2 import template

from app.service.auth_svc import check_authorization


class SandGuiApi:

    def __init__(self, services):
        self.auth_svc = services.get('auth_svc')
        self.app_svc = services.get('app_svc')
        self.data_svc = services.get('data_svc')

    @check_authorization
    @template('sandcat.html')
    async def splash(self, request):
        delivery_cmds = await self.data_svc.locate('abilities', dict(ability_id='2f34977d-9558-4c12-abad-349716777c6b'))
        return dict(delivery_cmds=delivery_cmds)
