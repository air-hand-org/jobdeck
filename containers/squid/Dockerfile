FROM ubuntu/squid:7.2-26.04_edge

COPY squid.conf.template /etc/squid/squid.conf.template
COPY --chmod=755 entrypoint.sh /usr/local/bin/jobdeck-entrypoint.sh

# The Squid Rock image starts its service through Pebble. Replacing its entrypoint
# bypasses that service definition, so preserve Squid's foreground arguments here.
# https://github.com/canonical/squid-rock
ENTRYPOINT ["/usr/local/bin/jobdeck-entrypoint.sh"]
CMD ["-NYC"]
