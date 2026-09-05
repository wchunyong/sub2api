"""Run the real OpenCode binary against a local mock using installer output.

Usage: python opencode_smoke.py <path-to-opencode-executable>
No production credentials; all OpenCode data/config/cache lives in a temporary home.
"""
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

spec = importlib.util.spec_from_file_location('installer', Path(__file__).parents[1] / 'assets/installer.py')
installer = importlib.util.module_from_spec(spec); spec.loader.exec_module(installer)
calls = []

class Mock(BaseHTTPRequestHandler):
    def log_message(self, *_): pass
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers['Content-Length'])))
        calls.append((self.path, self.headers.get('Authorization'), body.get('model')))
        self.send_response(200)
        self.send_header('Content-Type', 'text/event-stream'); self.end_headers()
        chunks = [dict(id='mock', object='chat.completion.chunk', model='mock-model', choices=[dict(index=0, delta={'role': 'assistant', 'content': 'IMPORT_OK'}, finish_reason=None)]), dict(id='mock', object='chat.completion.chunk', model='mock-model', choices=[dict(index=0, delta={}, finish_reason='stop')], usage=dict(prompt_tokens=1, completion_tokens=1, total_tokens=2))]
        for chunk in chunks: self.wfile.write(('data: '+json.dumps(chunk)+'\n\n').encode())
        self.wfile.write(b'data: [DONE]\n\n')

with tempfile.TemporaryDirectory(prefix='sub2api-opencode-') as directory:
    root = Path(directory)
    server = ThreadingHTTPServer(('127.0.0.1', 0), Mock)
    thread = threading.Thread(target=server.serve_forever, daemon=True); thread.start()
    try:
        installer.install(root, dict(version=1, agent='opencode', api_key='mock-smoke-key', base_url=f'http://127.0.0.1:{server.server_port}/v1', model='mock-model', protocol='compatible'))
        env = {k: v for k, v in os.environ.items() if not k.startswith(('OPENCODE_', 'ANTHROPIC_', 'OPENAI_'))}
        env.update(HOME=str(root), USERPROFILE=str(root), XDG_CONFIG_HOME=str(root / '.config'), XDG_DATA_HOME=str(root / '.data'), XDG_STATE_HOME=str(root / '.state'), XDG_CACHE_HOME=str(root / '.cache'), OPENCODE_DISABLE_PROJECT_CONFIG='true', OPENCODE_DISABLE_AUTOUPDATE='true', OPENCODE_CONFIG_CONTENT=json.dumps({'permission': {'*': 'deny'}, 'share': 'disabled', 'autoupdate': False, 'enabled_providers': ['sub2api_quick']}))
        result = subprocess.run([sys.argv[1], 'run', '--format', 'json', 'Reply IMPORT_OK without using tools.'], cwd=root, env=env, capture_output=True, text=True, encoding='utf-8', errors='replace', timeout=150)
        if result.returncode or 'IMPORT_OK' not in result.stdout or not calls:
            print(result.stdout[-1500:]); print(result.stderr[-1500:]); raise SystemExit('OpenCode mock smoke failed')
        assert all(path == '/v1/chat/completions' and auth == 'Bearer mock-smoke-key' and model == 'mock-model' for path, auth, model in calls), 'Wrong endpoint/auth/model'
        installer.clean(root, 'opencode')
        remaining = root / '.config/opencode/opencode.json'
        if remaining.exists():
            data = json.loads(remaining.read_text(encoding='utf-8'))
            assert 'sub2api_quick' not in data.get('provider', {})
            assert 'model' not in data
            print('Preserved fields added by OpenCode:', ', '.join(data))
        print(f'PASS: OpenCode called mock gateway ({len(calls)} request(s)); imported provider and cleanup verified.')
    finally: server.shutdown(); server.server_close()
