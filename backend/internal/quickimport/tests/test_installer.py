import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

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
        data = json.loads(path.read_text()); data['theme'] = 'later'; path.write_text(json.dumps(data))
        installer.clean(self.root, 'opencode')
        self.assertEqual(json.loads(path.read_text()), {'model': 'original/model', 'provider': {'other': {'name': 'keep'}}, 'theme': 'later'})
    def test_conflict_does_not_partially_clean(self):
        installer.install(self.root, self.payload('opencode'))
        path = self.root / '.config/opencode/opencode.json'
        data = json.loads(path.read_text()); data['model'] = 'user/change'; path.write_text(json.dumps(data))
        with self.assertRaisesRegex(ValueError, 'conflict'):
            installer.clean(self.root, 'opencode')
        self.assertIn('sub2api_quick', json.loads(path.read_text())['provider'])
    def test_repeated_install_restores_in_reverse_order(self):
        p = self.payload('claude'); installer.install(self.root, p)
        p['api_key'] = 'second-mock-key'; installer.install(self.root, p)
        installer.clean(self.root, 'claude')
        path = self.root / '.claude/settings.json'
        self.assertEqual(json.loads(path.read_text())['env']['ANTHROPIC_AUTH_TOKEN'], 'mock-test-key')
        installer.clean(self.root, 'claude'); self.assertFalse(path.exists())
    def test_codex_preserves_unrelated_tables(self):
        path = self.root / '.codex/config.toml'; path.parent.mkdir(parents=True)
        original = '# keep comment\nmodel = "old"\n[features]\nshell_tool = true\n'
        path.write_text(original)
        installer.install(self.root, self.payload('codex'))
        self.assertIn('[features]', path.read_text())
        installer.clean(self.root, 'codex')
        self.assertEqual(path.read_text(), original)
    def test_invalid_original_is_not_overwritten(self):
        path = self.root / '.claude/settings.json'; path.parent.mkdir(parents=True); path.write_text('{bad')
        with self.assertRaises(ValueError): installer.install(self.root, self.payload('claude'))
        self.assertEqual(path.read_text(), '{bad')

if __name__ == '__main__': unittest.main()
