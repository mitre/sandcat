import json

from datetime import datetime


class SandService:

    def __init__(self, services):
        self.data_svc = services.get('data_svc')
        self.file_svc = services.get('file_svc')
        self.utility_svc = services.get('utility_svc')
        self.log = self.utility_svc.create_logger('sandcat')

    async def instructions(self, paw):
        self.log.debug('%s checking for instructions' % paw)
        commands = await self.data_svc.explode_chain(criteria=dict(paw=paw))
        instructions = []
        for link in [c for c in commands if not c['collect']]:
            await self.data_svc.update('core_chain', key='id', value=link['id'], data=dict(collect=datetime.now()))
            payload = await self._gather_payload(link['ability'])
            instructions.append(json.dumps(dict(id=link['id'],
                                                sleep=link['jitter'],
                                                command=link['command'],
                                                cleanup=link['cleanup'],
                                                payload=payload)))
        return json.dumps(instructions)

    async def post_results(self, paw, link_id, output, status):
        self.log.debug('%s posting results' % paw)
        finished = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        await self.data_svc.create_result(result=dict(link_id=link_id, output=output))
        await self.data_svc.update('core_chain', key='id', value=link_id,
                                   data=dict(status=int(status), finish=finished))
        return json.dumps(dict(status=True))

    """ PRIVATE """

    async def _gather_payload(self, ability_id):
        payload = await self.data_svc.explode_payloads(criteria=dict(ability=ability_id))
        return payload[0]['payload'] if payload else ''
