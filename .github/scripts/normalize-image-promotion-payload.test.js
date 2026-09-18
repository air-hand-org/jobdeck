const assert = require('node:assert/strict');

const normalize = require('./normalize-image-promotion-payload.js');

assert.deepEqual(
  normalize({ services: ['auth', 'auth-proxy'], tag: 'v1.2.3' }),
  {
    services: '["auth","auth-proxy"]',
    serviceNames: 'auth, auth-proxy',
    serviceSlug: 'auth-auth-proxy',
    tag: 'v1.2.3',
    version: '1.2.3',
  },
);
assert.equal(normalize({ service: 'auth', tag: 'v1.2.3-rc.1' }).version, '1.2.3-rc.1');
assert.throws(() => normalize({ services: ['auth', 'auth'], tag: 'v1.2.3' }), /duplicates/);
assert.throws(() => normalize({ services: ['auth'], tag: '1.2.3' }), /semantic version/);
