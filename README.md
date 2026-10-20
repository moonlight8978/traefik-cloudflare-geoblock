# Traefik Cloudflare GeoBlock

Simple traefik plugin to use cloudflare CF-IPCountry header for geoblock

### Usage

1. Enable the plugin in your Traefik static configuration:

```yaml
experimental:
  plugins:
    cfgeoblock:
      moduleName: github.com/moonlight8978/traefik-cloudflare-geoblock
      version: vX.X.X  # Check latest release
```

2. Configure the middleware in your dynamic configuration:

```yaml
http:
  middlewares:
    my-geoblock:
      plugin:
        cfgeoblock:
          # Mode: "include" to allow only specified countries, "exclude" to block specified countries
          # Default: "include"
          mode: "include"

          # List of country codes to include/exclude (based on mode)
          # Uses ISO 3166-1 alpha-2 country codes (e.g., US, JP, VN)
          countries:
            - US
            - JP

          # Whether to allow requests without CF-IPCountry header
          # Default: true
          allowEmpty: true

          # Exceptions: List of Public IP addresses or ranges that are allowed despite not being in the whitelisted countries (include mode only)
          # Supports IPv4, IPv6, and ranges in the form "start-end"
          # Client IP is resolved from the first valid value, in priority order:
          #   1. CF-Connecting-IP header
          #   2. X-Real-IP header
          #   3. X-Forwarded-For header (first entry)
          #   4. Request remote address
          # Headers can be spoofed if the origin is reachable without going through Cloudflare
          exceptions:
            - 2a0d:4cc0:80:5bf4:5498:31ff:fe82:5521
            - 49.36.45.120
            - 49.36.45.120-49.36.45.130
```

3. Apply the middleware to your routers:

```yaml
http:
  routers:
    my-router:
      rule: Host(`example.com`)
      middlewares:
        - my-geoblock
      service: my-service
```

### Notes

- This plugin requires your site to be proxied through Cloudflare
- The plugin uses the `CF-IPCountry` header provided by Cloudflare
- Country codes should be in ISO 3166-1 alpha-2 format (e.g., US, JP, VN)
- If `allowEmpty` is false, requests without the `CF-IPCountry` header will be blocked
- Mode options:
  - `include`: Only allow traffic from specified countries
  - `exclude`: Block traffic from specified countries
- In `include` mode, `exceptions` can be used to allow specific IPs or ranges even when their country is not in `countries`
- In `exclude` mode, `exceptions` are ignored and excluded countries remain blocked
