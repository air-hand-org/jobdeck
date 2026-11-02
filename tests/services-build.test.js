const { test } = require('node:test');
const assert = require('node:assert/strict');
const { mkdtempSync, mkdirSync, writeFileSync, readFileSync, rmSync } = require('node:fs');
const { tmpdir } = require('node:os');
const { join, resolve } = require('node:path');
const { spawnSync } = require('node:child_process');

const servicesMakefile = resolve(__dirname, '../mkfiles/services.mk');
const services = ['auth', 'job-api', 'job-worker', 'ui', 'demo'];

function fixture(t, { failing, missing } = {}) {
  const root = mkdtempSync(join(tmpdir(), 'services-build-'));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  for (const service of services) {
    if (service === missing) continue;
    const directory = join(root, 'services', service);
    mkdirSync(directory, { recursive: true });
    writeFileSync(join(directory, 'Makefile'), `.PHONY: build\nbuild:\n\t@echo ${service} >> ../../built\n\t@${service === failing ? 'false' : 'true'}\n`);
  }
  return root;
}

function build(root) {
  return spawnSync('make', ['--no-print-directory', '-f', servicesMakefile, 'services-build'], {
    cwd: root, encoding: 'utf8',
  });
}

for (const service of services) {
  test(`propagates ${service} build failure`, (t) => {
    const root = fixture(t, { failing: service });
    const result = build(root);
    assert.notEqual(result.status, 0);
    assert.deepEqual(readFileSync(join(root, 'built'), 'utf8').trim().split('\n'),
      services.slice(0, services.indexOf(service) + 1));
  });

  test(`fails when ${service} checkout is missing`, (t) => {
    const result = build(fixture(t, { missing: service }));
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, new RegExp(`services/${service}`));
  });
}

test('builds every checked-out service through its build target', (t) => {
  const root = fixture(t);
  const result = build(root);
  assert.equal(result.status, 0, result.stderr);
  assert.deepEqual(readFileSync(join(root, 'built'), 'utf8').trim().split('\n'), services);
});
