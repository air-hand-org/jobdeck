const { test } = require('node:test');
const assert = require('node:assert/strict');
const prepare = require('./eligibility');

const fixture = (eventName = 'pull_request') => {
  const pr = { number: 293, state: 'open', merged: false, draft: false, auto_merge: null, head: { sha: 'tested', repo: { full_name: 'air-hand-org/jobdeck' } } };
  const context = { eventName, repo: { owner: 'air-hand-org', repo: 'jobdeck' }, serverUrl: 'https://github.com', runId: 123, payload: { pull_request: structuredClone(pr), action: 'created', issue: { number: 293, pull_request: {} }, comment: { body: '/review', user: { login: 'writer' } } } };
  let permission = { permission: 'write' };
  const github = { rest: { pulls: { get: async () => ({ data: pr }) }, repos: { getCollaboratorPermissionLevel: async () => ({ data: permission }) } } };
  const run = () => prepare({ github, context });
  return { pr, context, run, setPermission: (value) => { permission = value; } };
};

test('automatic requests require a current eligible tested head', async () => {
  assert.ok(await fixture().run());
});

for (const change of [
  (f) => { f.pr.draft = true; },
  (f) => { f.pr.state = 'closed'; },
  (f) => { f.pr.merged = true; },
  (f) => { f.pr.auto_merge = {}; },
  (f) => { f.pr.head.sha = 'newer'; },
  (f) => { f.pr.head.repo.full_name = 'fork/jobdeck'; },
  (f) => { f.context.payload.pull_request.head.repo.full_name = 'fork/jobdeck'; },
]) test(`automatic rejects ineligible state: ${change}`, async () => {
  const f = fixture(); change(f); assert.equal(await f.run(), null);
});

for (const permission of ['read', 'triage', 'none']) test(`manual rejects ${permission}`, async () => {
  const f = fixture('issue_comment'); f.setPermission({ permission }); assert.equal(await f.run(), null);
});
for (const permission of ['write', 'maintain', 'admin']) test(`manual accepts ${permission}`, async () => {
  const f = fixture('issue_comment'); f.setPermission({ permission }); assert.ok(await f.run());
});
test('custom role with push permission is accepted', async () => {
  const f = fixture('issue_comment'); f.setPermission({ permission: 'custom', user: { permissions: { push: true } } }); assert.ok(await f.run());
});
for (const command of ['/review ', '/review\n', '/review extra', 'please /review']) test(`rejects non-exact command ${JSON.stringify(command)}`, async () => {
  const f = fixture('issue_comment'); f.context.payload.comment.body = command; assert.equal(await f.run(), null);
});
test('rejects issue comments and edited comments', async () => {
  const f = fixture('issue_comment'); delete f.context.payload.issue.pull_request; assert.equal(await f.run(), null);
  f.context.payload.issue.pull_request = {}; f.context.payload.action = 'edited'; assert.equal(await f.run(), null);
});
test('API failures fail closed', async () => {
  const f = fixture('issue_comment'); f.setPermission(null); await assert.rejects(f.run);
});

test('manual command permits draft, closed, merged and fork PRs with current write permission', async () => {
  const f = fixture('issue_comment');
  Object.assign(f.pr, { state: 'closed', merged: true, draft: true, auto_merge: {} });
  f.pr.head.repo.full_name = 'fork/jobdeck';
  assert.equal(await f.run(), true);
});
