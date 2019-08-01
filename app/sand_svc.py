import json
import os
import random
import string

from datetime import datetime
from shutil import which


class SandService:

    def __init__(self, services):
        self.data_svc = services.get('data_svc')
        self.utility_svc = services.get('utility_svc')
        self.log = self.utility_svc.create_logger('sandcat')

    async def compile(self, platform):
        if which('go') is not None:
            key = await self._insert_unique_deployment_key()
            self.log.debug('New agent compiled with key = %s' % key)
            main_module = 'plugins/sandcat/gocat/sandcat.go'
            output = 'plugins/sandcat/payloads/%s' % platform
            os.system('GOOS=%s go build -o %s -ldflags="-s -w" %s' % (platform, output, main_module))

    async def beacon(self, paw, platform, server, group, files):
        agent = await self.data_svc.explode_agents(criteria=dict(paw=paw))
        if agent:
            self.log.debug('Beacon (%s)' % paw)
            last_seen = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            updated = dict(last_seen=last_seen, checks=agent[0]['checks'] + 1, platform=platform, server=server)
            await self.data_svc.update('core_agent', 'paw', paw, data=updated)
            await self.data_svc.create_group(name=group, paws=[paw])
            return agent[0]['id']
        else:
            self.log.debug('New beacon (%s)' % paw)
            queued = dict(last_seen=datetime.now(), paw=paw, checks=1, platform=platform, server=server, files=files)
            agent_id = await self.data_svc.create_agent(agent=queued)
            await self.data_svc.create_group(name=group, paws=[paw])
            return agent_id

    async def instructions(self, agent_id):
        commands = await self.data_svc.explode_chain(criteria=dict(host_id=agent_id))
        instructions = []
        for link in [c for c in commands if not c['collect']]:
            await self.data_svc.update('core_chain', key='id', value=link['id'], data=dict(collect=datetime.now()))
            payload = await self.data_svc.explode_payloads(criteria=dict(ability=link['ability']))
            instructions.append(json.dumps(dict(id=link['id'], sleep=link['jitter'],
                                                command=link['command'], cleanup=link['cleanup'],
                                                payload=payload[0]['payload'] if payload else '')))
        return json.dumps(instructions)

    async def post_results(self, paw, link_id, output, status):
        self.log.debug('[AGENT] posting results (%s)' % paw)
        finished = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        await self.data_svc.create_result(result=dict(link_id=link_id, output=output))
        await self.data_svc.update('core_chain', key='id', value=link_id, data=dict(status=int(status), finish=finished))
        return json.dumps(dict(status=True))

    """ PRIVATE """

    @staticmethod
    async def _insert_unique_deployment_key(size=30):
        key = ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(size))
        distraction = 'plugins/sandcat/gocat/deception/deception.go'
        lines = open(distraction, 'r').readlines()
        lines[-1] = 'var key = "%s"' % key
        out = open(distraction, 'w')
        out.writelines(lines)
        out.close()
        return key
