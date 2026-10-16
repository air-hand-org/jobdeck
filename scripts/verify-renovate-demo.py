import json
import sys
from pathlib import Path
import hcl2

root = Path(__file__).resolve().parents[1]
demo = Path(sys.argv[1]) if len(sys.argv) > 1 else root.parent / 'jobdeck-demo'
with (root / 'iac/bootstrap/github-demo.tf').open() as f:
    doc = hcl2.load(f)
resources = {name: value for r in doc['resource'] for kind, entries in r.items()
             if kind == 'github_repository_ruleset' for name, value in entries.items()}
base = resources['demo_main']
assert not base.get('bypass_actors'), 'CI ruleset must never have bypass actors'
rules = base['rules'][0]
assert rules['deletion'] and rules['non_fast_forward']
assert rules['pull_request'][0]['required_approving_review_count'] == 0
checks = rules['required_status_checks'][0]
assert checks['strict_required_status_checks_policy']
assert checks['required_check'][0]['context'] == 'check-result'
assert checks['required_check'][0]['integration_id'] == '${data.github_app.github_actions.id}'
assert not checks['do_not_enforce_on_create']
review = resources.get('demo_review_approval')
assert review is not None, 'separate approval ruleset is missing'
assert review['repository'] == '${github_repository.demo.name}'
assert review['conditions'] == base['conditions']
assert review['enforcement'] == 'active'
assert set(review['rules'][0]) == {'pull_request'}
assert review['rules'][0]['pull_request'][0]['required_approving_review_count'] == 1
actors = review['bypass_actors']
assert len(actors) == 2
assert {(a['actor_type'], str(a['actor_id'])) for a in actors} == {
    ('User', '45233435'), ('Integration', '${data.github_app.demo_renovate.id}')}
assert all(a['bypass_mode'] == 'pull_request' for a in actors)
print('PASS: demo rulesets preserve mandatory CI; approval bypass is only air-hand + Renovate')
config = json.loads((demo / 'renovate.json').read_text())
assert config['automerge'] is False
assert config['platformAutomerge'] is True
assert config['automergeType'] == 'pr'
assert config['automergeStrategy'] == 'squash'
assert not config.get('autoApprove', False)
assert config.get('vulnerabilityAlerts', {}).get('automerge') is False
assert config.get('lockFileMaintenance', {}).get('automerge') is False
rules = config['packageRules']
assert len(rules) == 1
assert rules[0]['matchUpdateTypes'] == ['minor', 'patch'] and rules[0]['automerge'] is True
print('PASS: only minor/patch enable platform PR automerge; major and other types remain manual')
