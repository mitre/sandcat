from aiohttp_jinja2 import template

from app.service.auth_svc import check_authorization
from app.utility.base_world import BaseWorld


class SandGuiApi(BaseWorld):

    def __init__(self, services):
        self.auth_svc = services.get('auth_svc')
        self.app_svc = services.get('app_svc')
        self.data_svc = services.get('data_svc')

    @check_authorization
    @template('sandcat.html')
    async def splash(self, request):
        red_ability = '2f34977d-9558-4c12-abad-349716777c6b'
        blue_ability = 'b02f97e2-6be3-4583-9080-7c3b1e7b573e'
        access = dict(access=tuple(await self.auth_svc.get_permissions(request)))
        delivery_cmds = await self.data_svc.locate('abilities', dict(
            ability_id=blue_ability if self.Access.BLUE in access['access'] else red_ability)
        )
        return dict(delivery_cmds=delivery_cmds)
