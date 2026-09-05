#!/usr/bin/env python3
"""Sub2API quick import v1. Python 3.11+, no third-party dependencies.

Credentials arrive through HTTPS POST or stdin, never command arguments.
Recovery is offline and limited to the selected Agent's managed fields.
"""
import argparse
import copy
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile
import tomllib
import urllib.request
from urllib.parse import urlparse
from contextlib import contextmanager

PROVIDER = 'sub2api_quick'
PATHS = {'claude': '.claude/settings.json', 'codex': '.codex/config.toml', 'opencode': '.config/opencode/opencode.json'}


def secure_dir(path):
    path.mkdir(parents=True, exist_ok=True, mode=0o700)
    if os.name == 'nt':
        # Restrict newly owned credential/journal directories to the current user.
        sid = subprocess.check_output(['whoami', '/user', '/fo', 'csv', '/nh'], text=True).strip().split(',')[-1].strip('"')
        subprocess.run(['icacls', str(path), '/inheritance:r', '/grant:r', f'*{sid}:(OI)(CI)F'], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    else:
        path.chmod(0o700)


def atomic_write(path, text):
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, name = tempfile.mkstemp(prefix='.sub2api-', dir=path.parent)
    try:
        with os.fdopen(fd, 'w', encoding='utf-8', newline='') as stream:
            stream.write(text)
            stream.flush()
            os.fsync(stream.fileno())
        if os.name == 'nt':
            sid = subprocess.check_output(['whoami', '/user', '/fo', 'csv', '/nh'], text=True).strip().split(',')[-1].strip('"')
            subprocess.run(['icacls', name, '/inheritance:r', '/grant:r', f'*{sid}:F'], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        os.replace(name, path)
    finally:
        if os.path.exists(name): os.unlink(name)


def target(root, agent):
    if agent not in PATHS: raise ValueError('Unsupported Agent')
    path = root / PATHS[agent]
    # Do not follow symlinks/junctions into another client's or user's directory.
    for part in [path, *path.parents]:
        if part == root.parent: break
        if part.is_symlink() or (hasattr(part, 'is_junction') and part.is_junction()):
            raise ValueError('Linked configuration paths require manual setup')
    return path


def load(text, agent):
    data = tomllib.loads(text) if agent == 'codex' else json.loads(text or '{}')
    if not isinstance(data, dict): raise ValueError('Configuration must be an object')
    return data


def get(data, parts):
    for part in parts:
        if not isinstance(data, dict) or part not in data: return {'exists': False}
        data = data[part]
    return {'exists': True, 'value': copy.deepcopy(data)}


def put(data, parts, value):
    key = parts[0]
    if len(parts) == 1:
        if value['exists']: data[key] = copy.deepcopy(value['value'])
        else: data.pop(key, None)
        return
    if key not in data:
        if not value['exists']: return
        data[key] = {}
    if not isinstance(data[key], dict): raise ValueError('Configuration field conflicts with an object')
    put(data[key], parts[1:], value)
    if not data[key]: del data[key]


def render(text, agent, changes):
    data = load(text, agent)
    desired = copy.deepcopy(data)
    for change in changes: put(desired, change['path'], change['value'])
    if agent != 'codex': return json.dumps(desired, indent=2, ensure_ascii=False) + '\n'
    # Touch only simple root selectors and our own provider table. Parse before and
    # after; reject unusual layouts rather than risk changing unrelated TOML.
    result = text
    for change in changes:
        parts, value = change['path'], change['value']
        if len(parts) == 1:
            boundary = re.search(r'^\s*\[', result, re.M)
            end = boundary.start() if boundary else len(result)
            head, tail = result[:end], result[end:]
            pattern = rf'(?m)^[ \t]*{re.escape(parts[0])}[ \t]*=.*(?:\n|$)'
            head = re.sub(pattern, '', head)
            if value['exists']: head = f'{parts[0]} = {json.dumps(value["value"])}\n' + head
            result = head + tail
        else:
            if parts != ['model_providers', PROVIDER]: raise ValueError('Unsupported TOML change')
            pattern = rf'(?ms)^\[model_providers\.{PROVIDER}\][ \t]*\r?\n.*?(?=^\[|\Z)'
            result = re.sub(pattern, '', result)
            if value['exists']:
                result = result.rstrip() + f'\n\n[model_providers.{PROVIDER}]\n'
                for key, item in value['value'].items():
                    result += f'{key} = {json.dumps(item)}\n'
    if load(result, agent) != desired: raise ValueError('Complex TOML layout requires manual configuration')
    return result


def configuration(payload):
    if payload.get('version') != 1: raise ValueError('Unsupported configuration version')
    agent = payload['agent']; key = payload['api_key']; base = payload['base_url'].rstrip('/'); model = payload['model']
    if agent not in PATHS or not key or not model or len(model) > 200: raise ValueError('Invalid configuration')
    parsed = urlparse(base)
    if parsed.scheme != 'https' and not (parsed.scheme == 'http' and parsed.hostname in ('127.0.0.1', 'localhost')):
        raise ValueError('HTTPS gateway required')
    if parsed.username or parsed.password or parsed.query or parsed.fragment: raise ValueError('Invalid gateway URL')
    if agent == 'claude':
        fields = [(['env', 'ANTHROPIC_BASE_URL'], base), (['env', 'ANTHROPIC_AUTH_TOKEN'], key), (['env', 'ANTHROPIC_MODEL'], model)]
    elif agent == 'codex':
        fields = [(['model'], model), (['model_provider'], PROVIDER), (['model_providers', PROVIDER], dict(name='Sub2API', base_url=base, wire_api='responses', experimental_bearer_token=key, requires_openai_auth=False))]
    else:
        protocol = payload.get('protocol', 'openai')
        npm = {'openai': '@ai-sdk/openai', 'anthropic': '@ai-sdk/anthropic', 'compatible': '@ai-sdk/openai-compatible', 'gemini': '@ai-sdk/google'}.get(protocol)
        if not npm: raise ValueError('Unsupported protocol')
        provider = dict(npm=npm, name='Sub2API', options=dict(baseURL=base, apiKey=key), models={model: {'name': model}})
        fields = [(['provider', PROVIDER], provider), (['model'], f'{PROVIDER}/{model}')]
    return [dict(path=path, value={'exists': True, 'value': value}) for path, value in fields]


@contextmanager
def locked(root, agent):
    target(root, agent)
    folder = root / '.sub2api-quick-import' / agent
    for parent in [folder, folder.parent]:
        if parent.is_symlink(): raise ValueError('Linked recovery directory')
    secure_dir(folder)
    lock = folder / 'lock'
    try: fd = os.open(lock, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
    except FileExistsError: raise ValueError('Another import/cleanup is active; inspect the recovery directory before retrying')
    os.close(fd)
    try: yield folder
    finally: lock.unlink(missing_ok=True)


def read_journal(folder):
    path = folder / 'journal.json'
    return json.loads(path.read_text(encoding='utf-8')) if path.exists() else []


def write_journal(folder, records):
    atomic_write(folder / 'journal.json', json.dumps(records, ensure_ascii=False))


def check_pending(records):
    if records and (records[-1].get('pending') or records[-1].get('cleanup_pending')):
        raise ValueError('Interrupted operation: inspect journal and restore its before_text before retrying')


def install(root, payload):
    agent = payload['agent']; changes = configuration(payload); path = target(root, agent)
    if agent == 'opencode' and path.with_suffix('.jsonc').exists():
        raise ValueError('Existing OpenCode JSONC configuration: use manual configuration to avoid precedence conflicts')
    with locked(root, agent) as folder:
        records = read_journal(folder); check_pending(records)
        before = path.read_text(encoding='utf-8-sig') if path.exists() else ''
        original = load(before, agent)
        after = render(before, agent, changes)
        if after == before: return
        record = dict(agent=agent, existed=path.exists(), before_text=before, after_text=after,
                      changes=changes, inverse=[dict(path=c['path'], value=get(original, c['path'])) for c in changes], pending=True)
        records.append(record); write_journal(folder, records)
        try:
            atomic_write(path, after)
            # Save the complete standard-library runner for offline recovery.
            atomic_write(folder / 'restore.py', Path(__file__).read_text(encoding='utf-8'))
            record['pending'] = False; write_journal(folder, records)
        except Exception:
            if record['existed']: atomic_write(path, before)
            else: path.unlink(missing_ok=True)
            records.pop(); write_journal(folder, records)
            raise


def clean(root, agent):
    path = target(root, agent)
    with locked(root, agent) as folder:
        records = read_journal(folder)
        if not records: raise ValueError('No configuration to clean for this Agent')
        record = records[-1]
        if record['agent'] != agent: raise ValueError('Agent mismatch')
        current = path.read_text(encoding='utf-8-sig') if path.exists() else ''
        if record.get('cleanup_pending'):
            if current == record['cleanup_result_text'] or (not path.exists() and record['cleanup_delete']):
                write_journal(folder, records[:-1])
                return
            if current != record['cleanup_before_text']:
                raise ValueError('Interrupted cleanup conflict: subsequent edits preserved')
            record.pop('cleanup_pending')
            write_journal(folder, records)
        check_pending(records)
        data = load(current, agent)
        for change in record['changes']:
            if get(data, change['path']) != change['value']: raise ValueError('Configuration conflict: later edits preserved. Restore the changed fields manually or revert them and retry.')
        result = record['before_text'] if current == record['after_text'] else render(current, agent, record['inverse'])
        # Persist intent before touching the file; roll back if journal commit fails.
        record['cleanup_pending'] = True
        record['cleanup_before_text'] = current
        record['cleanup_result_text'] = result
        record['cleanup_delete'] = not record['existed'] and not load(result, agent)
        write_journal(folder, records)
        try:
            if not record['existed'] and not load(result, agent): path.unlink(missing_ok=True)
            else: atomic_write(path, result)
            write_journal(folder, records[:-1])
        except Exception:
            atomic_write(path, current)
            record.pop('cleanup_pending', None)
            write_journal(folder, records)
            raise


def require_client(agent):
    if shutil.which(agent): return
    if agent == 'opencode':
        candidates = [Path('/Applications/OpenCode.app'), Path.home() / 'Applications/OpenCode.app']
        if os.name == 'nt':
            local = Path(os.environ.get('LOCALAPPDATA', ''))
            candidates += [local / 'Programs/@opencode-aidesktop/OpenCode.exe', local / 'OpenCode/OpenCode.exe']
        if any(path.exists() for path in candidates): return
    instructions = {'claude': 'Install Claude Code: https://code.claude.com/docs/en/setup', 'codex': 'Install Codex using its official instructions: https://developers.openai.com/codex/cli', 'opencode': 'Install OpenCode: https://opencode.ai/download'}
    raise ValueError(instructions[agent] + '. Reopen the terminal and generate a new command afterwards.')


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs): raise ValueError('Redirect refused')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['install', 'clean'])
    parser.add_argument('--agent', required=True, choices=list(PATHS))
    parser.add_argument('--server')
    parser.add_argument('--ticket')
    parser.add_argument('--stdin', action='store_true', help='Read configuration from stdin (isolated tests)')
    parser.add_argument('--home', type=Path, default=Path.home(), help='Explicit isolated home for testing')
    args = parser.parse_args()
    try:
        root = args.home.resolve()
        if args.action == 'clean':
            print(f'Restore the most recent Sub2API import for {args.agent} only. Other Agents are preserved.')
            if input('Continue? [y/N] ').strip().lower() != 'y': return
            clean(root, args.agent)
            print('Restored. Restart the client. Run again to undo an earlier import, if needed.')
        else:
            require_client(args.agent)
            overrides = {'opencode': ['OPENCODE_CONFIG', 'OPENCODE_CONFIG_CONTENT', 'XDG_CONFIG_HOME'], 'claude': ['CLAUDE_CONFIG_DIR', 'ANTHROPIC_BASE_URL', 'ANTHROPIC_AUTH_TOKEN', 'ANTHROPIC_API_KEY'], 'codex': ['CODEX_HOME']}[args.agent]
            if any(os.environ.get(name) for name in overrides): raise ValueError('Custom configuration environment detected. Use manual setup or clear overrides first.')
            if args.stdin: payload = json.load(sys.stdin)
            else:
                if not args.server or not args.ticket: raise ValueError('Missing server or ticket')
                parsed = urlparse(args.server)
                if parsed.scheme != 'https' or parsed.username or parsed.password or parsed.query or parsed.fragment: raise ValueError('HTTPS server required')
                request = urllib.request.Request(args.server.rstrip('/') + '/api/v1/quick-import/exchange', data=json.dumps({'ticket': args.ticket, 'agent': args.agent}).encode(), headers={'Content-Type': 'application/json'}, method='POST')
                with urllib.request.build_opener(NoRedirect).open(request, timeout=30) as response:
                    payload = json.load(response)['data']
            if payload['agent'] != args.agent: raise ValueError('Agent mismatch')
            install(root, payload)
            print(f'Configured {args.agent}. Restart the client. A project configuration may override user settings.')
            print(f'Offline recovery: python "{root / ".sub2api-quick-import" / args.agent / "restore.py"}" clean --agent {args.agent}')
    except Exception as error:
        # Network errors and parser exceptions may embed a credential-bearing body.
        if isinstance(error, ValueError) and not isinstance(error, (json.JSONDecodeError, tomllib.TOMLDecodeError)):
            print(str(error), file=sys.stderr)
        else: print('Operation failed. Original configuration was preserved or recovery information is available. Check network, permissions and configuration syntax.', file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__': main()
