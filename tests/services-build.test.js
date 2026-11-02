const { test } = require('node:test');
const assert = require('node:assert/strict');
const { mkdtempSync, mkdirSync, writeFileSync, readFileSync, rmSync, existsSync } = require('node:fs');
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
    writeFileSync(join(directory, 'Makefile'), `.PHONY: setup build\nsetup:\n\t@true\nbuild:\n\t@echo ${service} >> ../../built\n\t@${service === failing ? 'false' : 'true'}\n`);
  }
  return root;
}

test('prepares missing service tools before building with the service environment', (t) => {
  const root = fixture(t);
  for (const service of services) {
    const directory = join(root, 'services', service);
    writeFileSync(join(directory, 'mise.toml'), '[env]\n_.path = ["./bin"]\n');
    writeFileSync(join(directory, 'Makefile'), `.PHONY: setup build
setup:
\t@mkdir -p bin
\t@printf '#!/bin/sh\\necho ${service} >> ../../built\\n' > bin/service-build-tool
\t@chmod +x bin/service-build-tool
build:
\t@service-build-tool
`);
  }
  const result = build(root);
  assert.equal(result.status, 0, result.stderr);
  assert.deepEqual(readFileSync(join(root, 'built'), 'utf8').trim().split('\n'), services);
});

function build(root) {
  return spawnSync('make', ['--no-print-directory', '-f', servicesMakefile, 'services-build'], {
    cwd: root, encoding: 'utf8',
    env: { ...process.env, MISE_TRUSTED_CONFIG_PATHS: root,
      MISE_GLOBAL_CONFIG_FILE: join(root, 'global.toml'),
      MISE_SYSTEM_CONFIG_FILE: join(root, 'system.toml') },
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

  test(`stops when ${service} tool setup fails`, (t) => {
    const root = fixture(t);
    writeFileSync(join(root, 'services', service, 'Makefile'),
      '.PHONY: setup build\nsetup:\n\t@false\nbuild:\n\t@touch ../../unexpected-build\n');
    const result = build(root);
    assert.notEqual(result.status, 0);
    const previous = services.slice(0, services.indexOf(service));
    if (previous.length) {
      assert.deepEqual(readFileSync(join(root, 'built'), 'utf8').trim().split('\n'), previous);
    }
    assert.equal(existsSync(join(root, 'unexpected-build')), false);
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
