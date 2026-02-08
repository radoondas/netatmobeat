[![Build Status](https://travis-ci.org/radoondas/netatmobeat.svg?branch=7.3)](https://travis-ci.org/radoondas/netatmobeat)

# Netatmobeat

Welcome to Netatmobeat. 

This beat will pull data from public [Netatmo](https://www.netatmo.com/) API for weather gathered from weather stations around the world.
You can look at [weather map](https://weathermap.netatmo.com/) provided by Netatmo.

The beat is able to pull data from public API and index them in to Elasticsearch. Once data are indexed, yuo can analyse and visualise on [map](https://www.elastic.co/guide/en/kibana/current/tilemap.html).

To start working with netatmobeat you need to have an account at Netatmo [DEV](https://dev.netatmo.com) to be able to access API. Once you are signed in, configure new [App](https://dev.netatmo.com/myaccount/createanapp) to be able to connect to dev API. 

## Installation
Download and install appropriate package for your system. Check release [page](https://github.com/radoondas/netatmobeat/releases) for latest packages.

For docker image `docker pull radoondas/netatmobeat`

## Authentication

Netatmo removed username/password authentication (password grant) in July 2023. The beat now uses **OAuth2 refresh tokens** exclusively.

### First-time setup

1. Go to [https://dev.netatmo.com/apps/](https://dev.netatmo.com/apps/) and create (or select) your application
2. Note your **Client ID** and **Client Secret**
3. In the **Token Generator** section, select the `read_station` scope, click **Generate Token**, and authorize
4. Copy the **refresh token** into your `netatmobeat.yml`

### Token rotation

Since May 2024, Netatmo rotates refresh tokens on every use -- the old refresh token is invalidated immediately. The beat persists the latest token pair to a file (`netatmobeat-tokens.json` by default) after every refresh. On subsequent restarts, the beat loads tokens from this file automatically.

If the token file is lost or the refresh token expires, you will need to repeat the setup steps above to obtain a new refresh token.

For troubleshooting, recovery procedures, and key log message interpretation, see the [Operational Runbook](docs/RUNBOOK.md).

## Configuration

### Authentication
```yaml
netatmobeat:
  client_id: "abcdefghijklmn"
  client_secret: "mysecretfromapp"

  # Obtain from https://dev.netatmo.com/apps/ Token Generator (scope: read_station)
  refresh_token: "your_refresh_token_here"

  # Path to persist rotated tokens (default: netatmobeat-tokens.json)
  # token_file: "netatmobeat-tokens.json"
```

> **Note:** The `username` and `password` fields are no longer supported and will be ignored if present.

### Public weather data

Define geographic regions to gather data from. Regions are not exact shapes in terms of response as they are provided from Netatmo cache.
```yaml
  public_weather:
    enabled: true
    period: 10m
    regions:
      - region:
        enabled: true
        name: "EMEA"
        description: "Slovakia"
        lat_ne: 49.650266
        lon_ne: 22.780239
        lat_sw: 47.780377
        lon_sw: 16.759731
      - region:
        enabled: true
        name: "Spain"
        description: "Somewhere in EU"
        lat_ne: 43.417618
        lon_ne: 3.569562
        lat_sw: 36.867098
        lon_sw: -9.438251
```

### Station data

Requires at least one station ID. Suggested period is 5m.
```yaml
  weather_stations:
    enabled: false
    period: 5m
    ids: [ "st:at:io:ni:dd" ]
```

Configure your output and/or monitoring options. See `netatmobeat.yml` for the full reference.

### Docker / Kubernetes

When running in containers, the token file must be on a **persistent volume** so rotated tokens survive container restarts:
```yaml
  token_file: "/data/netatmobeat-tokens.json"
```

See `netatmobeat.docker.yml` for a complete Docker example.

## Validating Configuration

Test that the config file is syntactically correct:
```
./netatmobeat test config -c netatmobeat.yml
```

Authentication is validated at startup — the beat exits immediately with a clear error if credentials are invalid:
```
./netatmobeat -c netatmobeat.yml -e
```

## Run

```
./netatmobeat -c netatmobeat.yml -e
```

## Visualisations
This is an example of temperature visualisation

![Map](docs/img/map_vis.png)

## Build
If you want to build Netatmobeat from scratch, follow [build](BUILD.md) documentation.
