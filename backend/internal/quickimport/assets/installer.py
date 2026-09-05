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
import uuid
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
    catalog = normalize_models(payload.get('models', [{'id': model}]))
    if model not in {item['id'] for item in catalog}: raise ValueError('Selected model is not in the gateway model list. Choose an available model and retry.')
    if agent == 'claude':
        fields = [(['env', 'ANTHROPIC_BASE_URL'], base), (['env', 'ANTHROPIC_AUTH_TOKEN'], key), (['env', 'ANTHROPIC_MODEL'], model)]
        fields += [(['env', 'ANTHROPIC_CUSTOM_MODEL_OPTION'], model), (['env', 'ANTHROPIC_CUSTOM_MODEL_OPTION_NAME'], 'lianjieai · ' + model), (['env', 'ANTHROPIC_CUSTOM_MODEL_OPTION_DESCRIPTION'], 'lianjieai gateway')]
        if payload.get('claude_model_picker_supported', True):
            fields.append((['modelPicker'], {'options': [{'model': item['id'], 'label': 'lianjieai · ' + item['name']} for item in catalog], 'replaceBuiltInOptions': True}))
    elif agent == 'codex':
        fields = [(['model'], model), (['model_provider'], PROVIDER), (['model_providers', PROVIDER], dict(name='lianjieai', base_url=base, wire_api='responses', experimental_bearer_token=key, requires_openai_auth=False))]
        if payload.get('catalog_path'): fields.append((['model_catalog_json'], payload['catalog_path']))
    else:
        protocol = payload.get('protocol', 'openai')
        npm = {'openai': '@ai-sdk/openai', 'anthropic': '@ai-sdk/anthropic', 'compatible': '@ai-sdk/openai-compatible', 'gemini': '@ai-sdk/google'}.get(protocol)
        if not npm: raise ValueError('Unsupported protocol')
        models = {item['id']: {'name': item['name']} for item in catalog}
        if model not in models: raise ValueError('Selected model is not in the gateway model list. Choose an available model and retry.')
        provider = dict(npm=npm, name='lianjieai', options=dict(baseURL=base, apiKey=key), models=models)
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
    payload = copy.deepcopy(payload)
    agent = payload['agent']; path = target(root, agent)
    if agent == 'opencode' and path.with_suffix('.jsonc').exists():
        # Desktop may create a schema-only JSONC file. Leave it intact when it
        # cannot override our provider/model; complex JSONC requires manual setup.
        try:
            overlay = json.loads(path.with_suffix('.jsonc').read_text(encoding='utf-8-sig'))
            safe = isinstance(overlay, dict) and not set(overlay).difference({'$schema'})
        except (ValueError, OSError): safe = False
        if not safe:
            raise ValueError('Existing OpenCode JSONC configuration: use manual configuration to avoid precedence conflicts')
    with locked(root, agent) as folder:
        records = read_journal(folder); check_pending(records)
        owned = []
        if agent == 'codex' and payload.get('codex_manifest') is not None:
            manifest = payload['codex_manifest']
            validate_codex_manifest(manifest, payload['model'])
            catalog_file = folder / ('models-' + uuid.uuid4().hex + '.json')
            owned.append({'path': str(catalog_file.relative_to(root)), 'text': json.dumps(manifest, ensure_ascii=False, indent=2) + '\n'})
            payload['catalog_path'] = str(catalog_file)
        changes = configuration(payload)
        before = path.read_text(encoding='utf-8-sig') if path.exists() else ''
        original = load(before, agent)
        after = render(before, agent, changes)
        if after == before: return
        record = dict(agent=agent, existed=path.exists(), before_text=before, after_text=after,
                      changes=changes, inverse=[dict(path=c['path'], value=get(original, c['path'])) for c in changes], pending=True, owned_files=owned)
        records.append(record); write_journal(folder, records)
        try:
            for item in owned: atomic_write(owned_path(root, folder, item), item['text'])
            atomic_write(path, after)
            # Save the complete standard-library runner for offline recovery.
            atomic_write(folder / 'restore.py', Path(__file__).read_text(encoding='utf-8'))
            record['pending'] = False; write_journal(folder, records)
        except Exception:
            if record['existed']: atomic_write(path, before)
            else: path.unlink(missing_ok=True)
            for item in owned: owned_path(root, folder, item).unlink(missing_ok=True)
            records.pop(); write_journal(folder, records)
            raise


def owned_path(root, folder, item):
    path = root / item['path']
    if path.parent != folder or not re.fullmatch(r'models-[a-f0-9]{32}\.json', path.name) or path.is_symlink():
        raise ValueError('Invalid recovery catalog path')
    return path


def clean(root, agent):
    path = target(root, agent)
    with locked(root, agent) as folder:
        records = read_journal(folder)
        if not records: raise ValueError('No configuration to clean for this Agent')
        record = records[-1]
        if record['agent'] != agent: raise ValueError('Agent mismatch')
        current = path.read_text(encoding='utf-8-sig') if path.exists() else ''
        owned = record.get('owned_files', [])
        for item in owned:
            auxiliary = owned_path(root, folder, item)
            if not auxiliary.exists() and record.get('cleanup_pending'): continue
            if not auxiliary.exists() or auxiliary.read_text(encoding='utf-8') != item['text']:
                raise ValueError('Model catalog conflict: later edits preserved')
        if record.get('cleanup_pending'):
            if current == record['cleanup_result_text'] or (not path.exists() and record['cleanup_delete']):
                for item in owned: owned_path(root, folder, item).unlink(missing_ok=True)
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
            for item in owned: owned_path(root, folder, item).unlink(missing_ok=True)
            write_journal(folder, records[:-1])
        except Exception:
            for item in owned: atomic_write(owned_path(root, folder, item), item['text'])
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


def normalize_models(items):
    if not isinstance(items, list): raise ValueError('Invalid model list')
    result = []
    seen = set()
    for item in items:
        if not isinstance(item, dict): continue
        model_id = item.get('id')
        if not isinstance(model_id, str) or not model_id.strip() or len(model_id) > 200 or any(ord(c) < 32 for c in model_id) or model_id in seen: continue
        seen.add(model_id)
        name = item.get('name') or item.get('display_name')
        result.append({'id': model_id, 'name': name if isinstance(name, str) and name.strip() else model_id})
    return result


def verify_connection(payload):
    base = urlparse(payload['base_url'])
    probe = urlparse(payload['probe_url'])
    if probe.scheme != 'https' or (probe.scheme, probe.netloc) != (base.scheme, base.netloc) or probe.username or probe.password or probe.query or probe.fragment:
        raise ValueError('Invalid connectivity probe URL')
    request = urllib.request.Request(payload['probe_url'], headers={'Authorization': 'Bearer ' + payload['api_key'], 'User-Agent': 'lianjieai-quick-import/1.0', 'Accept': 'application/json'})
    with urllib.request.build_opener(NoRedirect).open(request, timeout=20) as response:
        result = json.loads(response.read(1024 * 1024))
    if not isinstance(result, dict) or not isinstance(result.get('data'), list):
        raise ValueError('Gateway did not return a model list. Check the API base URL.')
    return normalize_models(result['data'])


def validate_codex_manifest(manifest, selected):
    if not isinstance(manifest, dict) or not isinstance(manifest.get('models'), list):
        raise ValueError('Gateway did not return a Codex model catalog')
    if not any(isinstance(item, dict) and item.get('slug') == selected for item in manifest['models']):
        raise ValueError('Selected model is not in the Codex model catalog')


def compatible_codex_manifest(manifest):
    manifest = copy.deepcopy(manifest)
    for item in manifest['models']:
        if not isinstance(item, dict): raise ValueError('Invalid Codex model descriptor')
        messages = item.get('model_messages') or {}
        if 'base_instructions' not in item and isinstance(messages.get('instructions_template'), str):
            item['base_instructions'] = messages['instructions_template']
        if 'supports_reasoning_summaries' not in item and isinstance(item.get('supports_reasoning_summary_parameter'), bool):
            item['supports_reasoning_summaries'] = item['supports_reasoning_summary_parameter']
    return manifest


def client_version(agent):
    executable = shutil.which(agent)
    if not executable: return None
    try:
        result = subprocess.run([executable, '--version'], capture_output=True, text=True, timeout=10)
        match = re.search(r'\b(\d+)\.(\d+)\.(\d+)\b', result.stdout)
        return tuple(map(int, match.groups())) if result.returncode == 0 and match else None
    except (OSError, subprocess.TimeoutExpired): return None


def synchronize_models(payload):
    payload['models'] = verify_connection(payload)
    if payload['agent'] == 'claude':
        payload['claude_model_picker_supported'] = (client_version('claude') or (0, 0, 0)) >= (2, 1, 242)
        if not payload['claude_model_picker_supported']:
            print('Claude Code < 2.1.242: selected model is configured. Upgrade for the full lianjieai /model menu, or rerun setup with another model.')
    if payload['agent'] == 'codex':
        version = '.'.join(map(str, client_version('codex') or (0, 147, 0)))
        request = urllib.request.Request(payload['probe_url'] + '?client_version=' + version, headers={'Authorization': 'Bearer ' + payload['api_key'], 'User-Agent': 'lianjieai-quick-import/1.0', 'Accept': 'application/json'})
        with urllib.request.build_opener(NoRedirect).open(request, timeout=30) as response:
            manifest = json.loads(response.read(8 * 1024 * 1024))
        validate_codex_manifest(manifest, payload['model'])
        payload['codex_manifest'] = compatible_codex_manifest(manifest)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['install', 'clean'])
    parser.add_argument('--agent', required=True, choices=list(PATHS))
    parser.add_argument('--server')
    parser.add_argument('--ticket')
    parser.add_argument('--stdin', action='store_true', help='Read configuration from stdin (isolated tests)')
    parser.add_argument('--home', type=Path, default=Path.home(), help='Explicit isolated home for testing')
    args = parser.parse_args()
    stage = 'preflight'
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
                stage = 'configuration exchange'
                request = urllib.request.Request(args.server.rstrip('/') + '/api/v1/quick-import/exchange', data=json.dumps({'ticket': args.ticket, 'agent': args.agent}).encode(), headers={'Content-Type': 'application/json', 'User-Agent': 'lianjieai-quick-import/1.0', 'Accept': 'application/json'}, method='POST')
                with urllib.request.build_opener(NoRedirect).open(request, timeout=30) as response:
                    payload = json.load(response)['data']
            if payload['agent'] != args.agent: raise ValueError('Agent mismatch')
            configuration(payload)
            if not args.stdin:
                stage = 'model catalog'
                synchronize_models(payload)
            stage = 'configuration write'
            install(root, payload)
            print(f'Configured {args.agent}. Restart the client. A project configuration may override user settings.')
            print(f'Offline recovery: python "{root / ".sub2api-quick-import" / args.agent / "restore.py"}" clean --agent {args.agent}')
    except Exception as error:
        # Network errors and parser exceptions may embed a credential-bearing body.
        if isinstance(error, urllib.error.HTTPError):
            print(f'Failed during {stage}: HTTP {error.code}. Generate a new import command and retry; if it persists, report this stage and status.', file=sys.stderr)
        elif isinstance(error, urllib.error.URLError):
            print(f'Failed during {stage}: network or TLS connection error. Check connectivity and retry with a new command.', file=sys.stderr)
        elif isinstance(error, ValueError) and not isinstance(error, (json.JSONDecodeError, tomllib.TOMLDecodeError)):
            print(str(error), file=sys.stderr)
        else: print('Operation failed. Original configuration was preserved or recovery information is available. Check network, permissions and configuration syntax.', file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__': main()
