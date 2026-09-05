import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('installer', Path(__file__).parents[1] / 'assets' / 'installer.py')
installer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(installer)

class InstallerTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
    def tearDown(self):
        self.tmp.cleanup()
    def payload(self, agent):
        return dict(version=1, agent=agent, api_key='mock-test-key', base_url='https://example.com/v1', model='test-model', protocol='openai')
    def test_all_agents_round_trip_and_isolation(self):
        for agent in ['opencode', 'claude', 'codex']:
            installer.install(self.root, self.payload(agent))
        installer.clean(self.root, 'opencode')
        self.assertFalse((self.root / '.config/opencode/opencode.json').exists())
        self.assertTrue((self.root / '.claude/settings.json').exists())
        self.assertTrue((self.root / '.codex/config.toml').exists())
        installer.clean(self.root, 'claude')
        installer.clean(self.root, 'codex')
        self.assertFalse((self.root / '.codex/config.toml').exists())
    def test_preserves_later_edits_and_original_provider(self):
        path = self.root / '.config/opencode/opencode.json'
        path.parent.mkdir(parents=True)
        path.write_text(json.dumps({'model': 'original/model', 'provider': {'other': {'name': 'keep'}}}))
        installer.install(self.root, self.payload('opencode'))
        data = json.loads(path.read_text(encoding='utf-8')); data['theme'] = 'later'; path.write_text(json.dumps(data))
        installer.clean(self.root, 'opencode')
        self.assertEqual(json.loads(path.read_text(encoding='utf-8')), {'model': 'original/model', 'provider': {'other': {'name': 'keep'}}, 'theme': 'later'})
    def test_conflict_does_not_partially_clean(self):
        installer.install(self.root, self.payload('opencode'))
        path = self.root / '.config/opencode/opencode.json'
        data = json.loads(path.read_text(encoding='utf-8')); data['model'] = 'user/change'; path.write_text(json.dumps(data))
        with self.assertRaisesRegex(ValueError, 'conflict'):
            installer.clean(self.root, 'opencode')
        self.assertIn('sub2api_quick', json.loads(path.read_text(encoding='utf-8'))['provider'])
    def test_repeated_install_restores_in_reverse_order(self):
        p = self.payload('claude'); installer.install(self.root, p)
        p['api_key'] = 'second-mock-key'; installer.install(self.root, p)
        installer.clean(self.root, 'claude')
        path = self.root / '.claude/settings.json'
        self.assertEqual(json.loads(path.read_text(encoding='utf-8'))['env']['ANTHROPIC_AUTH_TOKEN'], 'mock-test-key')
        installer.clean(self.root, 'claude'); self.assertFalse(path.exists())
    def test_codex_preserves_unrelated_tables(self):
        path = self.root / '.codex/config.toml'; path.parent.mkdir(parents=True)
        original = '# keep comment\nmodel = "old"\n[features]\nshell_tool = true\n'
        path.write_text(original)
        installer.install(self.root, self.payload('codex'))
        self.assertIn('[features]', path.read_text(encoding='utf-8'))
        installer.clean(self.root, 'codex')
        self.assertEqual(path.read_text(encoding='utf-8'), original)
    def test_invalid_original_is_not_overwritten(self):
        path = self.root / '.claude/settings.json'; path.parent.mkdir(parents=True); path.write_text('{bad')
        with self.assertRaises(ValueError): installer.install(self.root, self.payload('claude'))
        self.assertEqual(path.read_text(encoding='utf-8'), '{bad')
    def test_cleanup_journal_failure_rolls_back_and_can_retry(self):
        installer.install(self.root, self.payload('claude'))
        p = self.payload('claude'); p['api_key'] = 'second-mock-key'; installer.install(self.root, p)
        original_write = installer.write_journal
        count = 0
        def fail_once(folder, records):
            nonlocal count
            count += 1
            if count == 2: raise OSError('simulated disk failure')
            original_write(folder, records)
        with patch.object(installer, 'write_journal', fail_once):
            with self.assertRaises(OSError): installer.clean(self.root, 'claude')
        installer.clean(self.root, 'claude')
        installer.clean(self.root, 'claude')
        self.assertFalse((self.root / '.claude/settings.json').exists())
    def test_missing_client_has_install_instructions(self):
        with patch.object(installer.shutil, 'which', return_value=None):
            with self.assertRaisesRegex(ValueError, 'Install Codex'):
                installer.require_client('codex')
    def test_empty_desktop_jsonc_does_not_block_import(self):
        path = self.root / '.config/opencode/opencode.jsonc'; path.parent.mkdir(parents=True)
        path.write_text('{"$schema":"https://opencode.ai/config.json"}')
        installer.install(self.root, self.payload('opencode'))
        installer.clean(self.root, 'opencode')
        self.assertTrue(path.exists())
    def test_conflicting_jsonc_is_not_overwritten(self):
        path = self.root / '.config/opencode/opencode.jsonc'; path.parent.mkdir(parents=True)
        path.write_text('{"model":"existing/model"}')
        with self.assertRaisesRegex(ValueError, 'JSONC'):
            installer.install(self.root, self.payload('opencode'))
    def test_connectivity_probe_uses_gateway_models_and_bearer_auth(self):
        opener = unittest.mock.MagicMock()
        response = opener.open.return_value.__enter__.return_value
        response.read.return_value = b'{"data":[]}'
        p = self.payload('opencode'); p['probe_url'] = 'https://example.com/v1/models'
        with patch.object(installer.urllib.request, 'build_opener', return_value=opener):
            installer.verify_connection(p)
        request = opener.open.call_args.args[0]
        self.assertEqual(request.full_url, 'https://example.com/v1/models')
        self.assertEqual(request.get_header('Authorization'), 'Bearer mock-test-key')
    def test_opencode_imports_named_provider_with_all_gateway_models(self):
        p = self.payload('opencode')
        p['models'] = [{'id': 'test-model'}, {'id': 'another-model', 'name': 'Another model'}]
        installer.install(self.root, p)
        data = json.loads((self.root / '.config/opencode/opencode.json').read_text(encoding='utf-8'))
        provider = data['provider']['sub2api_quick']
        self.assertEqual(provider['name'], 'lianjieai')
        self.assertEqual(set(provider['models']), {'test-model', 'another-model'})
        self.assertEqual(data['model'], 'sub2api_quick/test-model')
        installer.clean(self.root, 'opencode')
        self.assertFalse((self.root / '.config/opencode/opencode.json').exists())
    def test_probe_returns_catalog_and_ignores_malformed_entries(self):
        opener = unittest.mock.MagicMock()
        response = opener.open.return_value.__enter__.return_value
        response.read.return_value = b'{"data":[{"id":"test-model"},{"id":"other","name":"Other"},{"id":"other"},{"bad":"entry"}]}'
        p = self.payload('opencode'); p['probe_url'] = 'https://example.com/v1/models'
        with patch.object(installer.urllib.request, 'build_opener', return_value=opener):
            models = installer.verify_connection(p)
        self.assertEqual(models, [{'id':'test-model','name':'test-model'}, {'id':'other','name':'Other'}])
    def test_claude_model_picker_and_legacy_fallback(self):
        p = self.payload('claude'); p['models'] = [{'id':'test-model'}, {'id':'other'}]
        installer.install(self.root, p)
        path = self.root / '.claude/settings.json'; data = json.loads(path.read_text(encoding='utf-8'))
        self.assertEqual([row['model'] for row in data['modelPicker']['options']], ['test-model','other'])
        self.assertEqual(data['env']['ANTHROPIC_CUSTOM_MODEL_OPTION_NAME'], 'lianjieai · test-model')
        installer.clean(self.root,'claude')
        p['claude_model_picker_supported'] = False
        installer.install(self.root,p)
        data = json.loads(path.read_text(encoding='utf-8'))
        self.assertNotIn('modelPicker',data)
        self.assertEqual(data['env']['ANTHROPIC_CUSTOM_MODEL_OPTION'],'test-model')
    def test_codex_catalog_is_restored_with_configuration(self):
        p=self.payload('codex'); p['codex_manifest']={'models':[{'slug':'test-model','display_name':'Test model'}]}
        installer.install(self.root,p)
        import tomllib
        data=tomllib.loads((self.root/'.codex/config.toml').read_text(encoding='utf-8'))
        self.assertEqual(data['model_providers']['sub2api_quick']['name'],'lianjieai')
        catalog=Path(data['model_catalog_json'])
        self.assertEqual(json.loads(catalog.read_text(encoding='utf-8')),p['codex_manifest'])
        installer.clean(self.root,'codex')
        self.assertFalse(catalog.exists())
        self.assertFalse((self.root/'.codex/config.toml').exists())
    def test_codex_legacy_fields_preserve_gateway_instructions(self):
        source={'models':[{'model_messages':{'instructions_template':'gateway instructions'},'supports_reasoning_summary_parameter':True}]}
        result=installer.compatible_codex_manifest(source)
        self.assertEqual(result['models'][0]['base_instructions'],'gateway instructions')
        self.assertTrue(result['models'][0]['supports_reasoning_summaries'])
        self.assertNotIn('base_instructions',source['models'][0])
        source['models'][0]['base_instructions']='existing'
        self.assertEqual(installer.compatible_codex_manifest(source)['models'][0]['base_instructions'],'existing')
    def test_codex_sync_requests_versioned_catalog(self):
        p=self.payload('codex'); p['probe_url']='https://example.com/v1/models'
        opener=unittest.mock.MagicMock()
        opener.open.return_value.__enter__.return_value.read.return_value=b'{"models":[{"slug":"test-model"}]}'
        with patch.object(installer,'verify_connection',return_value=[{'id':'test-model'}]), patch.object(installer,'client_version',return_value=(0,142,5)), patch.object(installer.urllib.request,'build_opener',return_value=opener):
            installer.synchronize_models(p)
        request=opener.open.call_args.args[0]
        self.assertEqual(request.full_url,'https://example.com/v1/models?client_version=0.142.5')
        self.assertEqual(request.get_header('Authorization'),'Bearer mock-test-key')
        self.assertEqual(p['codex_manifest']['models'][0]['slug'],'test-model')
    def test_codex_catalog_cleanup_failure_rolls_back(self):
        p=self.payload('codex'); p['codex_manifest']={'models':[{'slug':'test-model'}]}
        installer.install(self.root,p)
        import tomllib
        config=self.root/'.codex/config.toml'; original=config.read_text(encoding='utf-8')
        catalog=Path(tomllib.loads(original)['model_catalog_json'])
        original_write=installer.write_journal
        count=0
        def fail_once(folder,records):
            nonlocal count
            count+=1
            if count==2: raise OSError('simulated failure')
            original_write(folder,records)
        with patch.object(installer,'write_journal',fail_once):
            with self.assertRaises(OSError): installer.clean(self.root,'codex')
        self.assertEqual(config.read_text(encoding='utf-8'),original)
        self.assertTrue(catalog.exists())
        installer.clean(self.root,'codex')
        self.assertFalse(catalog.exists())
    def test_modified_codex_catalog_blocks_cleanup_without_partial_changes(self):
        p=self.payload('codex'); p['codex_manifest']={'models':[{'slug':'test-model'}]}
        installer.install(self.root,p)
        import tomllib
        config=self.root/'.codex/config.toml'; original=config.read_text(encoding='utf-8')
        catalog=Path(tomllib.loads(original)['model_catalog_json']);catalog.write_text('{"models":[]}')
        with self.assertRaisesRegex(ValueError,'conflict'): installer.clean(self.root,'codex')
        self.assertEqual(config.read_text(encoding='utf-8'),original)

class ExchangeTests(unittest.TestCase):
    def test_exchange_identifies_client_and_reports_safe_http_status(self):
        import io
        import urllib.error
        opener=unittest.mock.MagicMock()
        opener.open.side_effect=urllib.error.HTTPError('https://example.com',403,'secret-body',{},None)
        output=io.StringIO()
        with patch.object(installer.sys,'argv',['installer.py','install','--agent','opencode','--server','https://example.com','--ticket','a'*64]), patch.object(installer,'require_client'), patch.dict(installer.os.environ,{'USERPROFILE':str(Path.home()),'HOME':str(Path.home())},clear=True), patch.object(installer.urllib.request,'build_opener',return_value=opener), patch.object(installer.sys,'stderr',output):
            with self.assertRaises(SystemExit): installer.main()
        request=opener.open.call_args.args[0]
        self.assertEqual(request.get_header('User-agent'),'lianjieai-quick-import/1.0')
        self.assertIn('configuration exchange: HTTP 403',output.getvalue())
        self.assertNotIn('secret-body',output.getvalue())

if __name__ == '__main__': unittest.main()
