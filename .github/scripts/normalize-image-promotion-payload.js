const serviceNamePattern = /^[a-z0-9][a-z0-9-]*$/;
const versionTagPattern = /^v[0-9]+(\.[0-9]+){2}([-.][0-9A-Za-z.-]+)?$/;

module.exports = (payload) => {
  const services = Array.isArray(payload.services)
    ? payload.services
    : typeof payload.service === 'string'
      ? [payload.service]
      : null;

  if (!services || services.length === 0 || !services.every((service) => typeof service === 'string' && serviceNamePattern.test(service))) {
    throw new Error('Expected a non-empty services array or a valid legacy service string.');
  }
  if (new Set(services).size !== services.length) {
    throw new Error('Services must not contain duplicates.');
  }
  if (typeof payload.tag !== 'string' || !versionTagPattern.test(payload.tag)) {
    throw new Error(`Image tag must be a v-prefixed semantic version: ${payload.tag}`);
  }

  return {
    services: JSON.stringify(services),
    serviceNames: services.join(', '),
    serviceSlug: services.join('-'),
    tag: payload.tag,
    version: payload.tag.slice(1),
  };
};
