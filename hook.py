from plugins.sandcat.app.sand_svc import SandService

name = 'Sandcat'
description = 'A custom multi-platform RAT'
address = None


async def enable(services):
    file_svc = services.get('file_svc')
    sand_svc = SandService(services)
    await file_svc.add_special_payload('sandcat.go', sand_svc.dynamically_compile_executable)
    await file_svc.add_special_payload('shared.go', sand_svc.dynamically_compile_library)
