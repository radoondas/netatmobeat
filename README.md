# Netatmobeat

Welcome to Netatmobeat. 

This beat will pull data from public [Netatmo](https://www.netatmo.com/) API for weather gathered from weather stations around the world.
You can look at [weather map](https://weathermap.netatmo.com/) provided by Netatmo.

The beat is able to pull data from public API and index them in to Elasticsearch. Once data are indexed, yuo can analyse and visualise on [map](https://www.elastic.co/guide/en/kibana/current/tilemap.html).

To start working with netatmobeat you need to have an account at Netatmo [DEV](https://dev.netatmo.com) to be able to access API. Once you are signed in, configure new [App](https://dev.netatmo.com/myaccount/createanapp) to be able to connect to dev API. 

## Installation
Download and install appropriate package for your system. Check release [page](https://github.com/radoondas/netatmobeat/releases) for latest packages.

For docker image `docker pull radoondas/netatmobeat`

## Configuration

Configure authentication after you create application in https://dev.netatmo.com and paste values for your application.
```
  client_id: "abcdefghijklmn"
  client_secret: "mysecretfromapp"
```

 Username/password to your Netatmo dev account
```
  username: "user@email"
  password: "password"
```

Public weather configuration. Define regions you want to gather data from. Regions are not exact shapes in terms of a response as they are provided from Netatmo cache.
```
  public_weather:
    enabled: true
    regions:
      - region:
        enabled: true
        name: "EMEA"
        description: "Slovakia"
        lat_ne: 49.650266
        lon_ne: 22.780239
        lat_sw: 47.780377
      - region:
        enabled: true
        name: "Spain"
        description: "Somewhere in EU"
        lat_ne: 43.417618
        lon_ne: 3.569562
        lat_sw: 36.867098
        lon_sw: -9.438251
```

Configure your output and/or monitoring options

## Run

```
./netatmobeat -c netatmobeat.yml -e 
```

## Visualisations
This is an example of temperature visualisation

![Map](docs/img/map_vis.png)
