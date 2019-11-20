from plugins.sandcat.app.sand_gui_api import SandGuiApi
from plugins.sandcat.app.sand_svc import SandService

name = 'Sandcat'
description = 'A custom multi-platform RAT'
address = '/plugin/sandcat/gui'


async def initialize(app, services):
    file_svc = services.get('file_svc')
    await file_svc.add_special_payload('sandcat.go', SandService(services).dynamically_compile)

    cat_gui_api = SandGuiApi(services=services)
    app.router.add_static('/sandcat', 'plugins/sandcat/static/', append_version=True)

    # gui
    app.router.add_route('GET', '/plugin/sandcat/gui', cat_gui_api.splash)
