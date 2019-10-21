from plugins.sandcat.app.sand_api import SandApi
from plugins.sandcat.app.sand_gui_api import SandGuiApi
from plugins.sandcat.app.sand_svc import SandService

name = 'Sandcat'
description = 'A custom multi-platform RAT'
address = '/plugin/sandcat/gui'


async def initialize(app, services):
    file_svc = services.get('file_svc')
    await file_svc.add_special_payload('sandcat.go', SandService(file_svc).dynamically_compile)

    cat_api = SandApi(services=services)
    cat_gui_api = SandGuiApi(services=services)
    app.router.add_static('/sandcat', 'plugins/sandcat/static/', append_version=True)
    app.router.add_static('/malicious', 'plugins/sandcat/static/malicious', append_version=True)
    # cat
    app.router.add_route('POST', '/sand/ping', cat_api.ping)
    app.router.add_route('POST', '/sand/instructions', cat_api.instructions)
    app.router.add_route('POST', '/sand/results', cat_api.results)
    # gui
    app.router.add_route('GET', '/plugin/sandcat/gui', cat_gui_api.splash)
    app.router.add_route('GET', '/plugin/sandcat/clone', cat_gui_api.clone_new_site)
    app.router.add_route('GET', '/plugin/sandcat/malicious', cat_gui_api.malicious)
