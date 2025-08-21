# ns8-nethsecurity-controller

Setup and start an instance of [nethsecurity-controller](https://github.com/NethServer/nethsecurity-controller).

Each node can host multiple controller instances.

The module is composed by the following containers:
- [nethsecurity-api](#api-server): REST API python server to manage nethsec-vpn clients
- [nethsecurity-vpn](#vpn): OpenVPN server, it authenticates the machines and create routes for the proxy
- [nethsecurity-ui](#proxy-and-ui): lighttpd instance serving static UI files
- [nethsecurity-proxy](#proxy-and-ui): traefik forwards requests to the connected machines using the machine name as path prefix
- [promtail](#promtail): log collector for Loki, it listens for syslog messages on the VPN address and forwards them to Loki
- [prometheus](#prometheus): metrics collector, it scrapes metrics from the connected machines
- [loki](#loki): log storage, it stores logs from promtail
- [grafana](#grafana): metrics visualization, it visualizes metrics from prometheus and logs from loki
- [webssh](#webssh): web-based ssh client
- [timescale](#timescale): time-series database for storing metrics


## Install

Instantiate the module with:

    add-module ghcr.io/nethserver/nethsecurity-controller:latest

The output of the command will return the instance name.
Output example:

    {"module_id": "nethsecurity-controller1", "image_name": "nethsecurity-controller", "image_url": "ghcr.io/nethserver/nethsecurity-controller:latest"}

## Configure

You can configure the controller directly from the UI or use the command line.

Let's assume that the nethsecurity-controller instance is named `nethsecurity-controller1`.

Launch `configure-module`, by setting the following parameters:
- `host`: a fully qualified domain name for the controller
- `lets_encrypt`: enable or disable Let's Encrypt certificate
- `ovpn_network`: OpenVPN network
- `ovpn_netmask`: OpenVPN netmask
- `ovpn_cn`: OpenVPN Certificate CN
- `api_user`: controller admin user
- `api_password`: controller admin password, change it after first login
- `loki_retention`: Loki retention period in days (default: ``180`` days)
- `prometheus_retention`: Promtail and Timescale retention period in days (default: 15 days)
- `maxmind_license`: [MaxMind](https://www.maxmind.com/) license key to download the GEO IP database, the database is loaded every time the API server is started;
  this field is optional, omit it if you do not have a valid license key

Example:

    api-cli run  module/nethsecurity-controller1/configure-module --data '{"host": "mycontroller.nethsecurity.org", "lets_encrypt": false, "ovpn_network": "172.19.64.0", "ovpn_netmask": "255.255.255.0", "ovpn_cn": "nethsec", "api_user": "admin", "api_password": "password", "loki_retention": 180, "prometheus_retention": 15, "maxmind_license": "xxx"}'

The above command will:
- start and configure the nethsecurity-controller instance
- setup a the following routes inside traefik:
  - one route to reach the controller based on the `host` parameter like `https://mycontroller.nethsecurity.org/`
  - one route to reach the prometheus, with a random generated URL like `https://myontroller.nethsecurity.org/f0365996-c1b3-4252-9cf3-c2e7e86ed617/`
  - one route to reach the loki, with a random generated URL like `https://mycontroller.nethsecurity.org/3e3e3e3e-3e3e-3e3e-3e3e-3e3e3e3e3e3e/`
  - one route to reach the grafana, with a well-know URL like `https://mycontroller.nethsecurity.org/grafana/`
- setup Promtail syslog receiver
- setup Prometheus for metrics scraping
- setup Loki for receiving logs
- setup Grafana for metrics visualization

Once the controller is configured, you access the controller URL, eg. `mycontroller.nethsecurity.org`, and manage NethSecurity units.

## Module overview

The module is composed by the following systemd units:
- controller.service: runs the container pod, all containers are part of the same pod; it can start and stop all the containers at once
- api.service: runs the nethsecurity-api container
- vpn.service: runs the nethsecurity-vpn container
- ui.service: runs the nethsecurity-ui container
- proxy.service: runs the nethsecurity-proxy container
- promtail.service: runs the promtail container
- prometheus.service: runs the prometheus container
- loki.service: runs the loki container
- grafana.service: runs the grafana container
- webssh.service: runs the webssh container

### API Server

The [api server](https://github.com/NethServer/nethsecurity-controller/tree/master/api) gives NethSecurity the ability to register itself to NS8 (through [`ns-plug`](https://dev.nethsecurity.org/nethsecurity/packages/ns-plug/)) and gives access to the on-demand generated credentials for the VPN.

The API also registers the endpoints for the [Traefik Proxy](#proxy-and-ui) that allows direct interaction with the firewall, even if it's not in the same network.

### VPN

The [OpenVPN container](https://github.com/NethServer/nethsecurity-controller/tree/master/vpn) tunnels connection from the NethSecurity to the NS8 through a VPN tunnel, due to [firewall configuration](https://github.com/NethServer/ns8-nethsecurity-controller/blob/main/imageroot/actions/configure-module/20configure#L87) in NS8, no client can be reached from other clients and only client-server communication is allowed.

The module uses the NS8 [TUN feature](https://dev.nethsecurity.org/ns8-core/core/tun/) to create a new network interface and assign it to the VPN container.

### Proxy and UI

The [UI](https://github.com/NethServer/nethsecurity-controller/tree/master/ui) allows the browse of the interface directly off the NethSecurity installation, this is possible due to the [Traefik Proxy](https://github.com/NethServer/nethsecurity-controller/tree/master/proxy) server that redirects the urls to the correct IP inside the VPN.

### Promtail

[Promtail](https://grafana.com/docs/loki/latest/send-data/promtail/) is a log collector for Loki, it listens for syslog messages on the VPN address and forwards them to Loki. The configuration is available at `/home/nethsecurity-controller1/.config/etc/promtail.yml`.
Promtail sets the following labels:
- `job` fixed to `syslog`
- unit `hostname`
- log `level`
- `application` name
- syslog `facility` name
- `controller_name` the name of the controller as configured in `ovpn_cn`

### Prometheus

[Prometheus](https://prometheus.io/) is a metrics collector, it scrapes metrics from the connected machines. The configuration is available at `/home/nethsecurity-controller1/.config/state/prometheus.yml` and it's generated every time by the `configure-module` action.
It has a the following targets:
- static target with job_name `loki` that scrapes Loki metrics
- dynamic targets with job_name `node` that scrapes metrics from the connected machines from the API server `http://<prometheus_user>:<prometheus_pass>@127.0.0.1:<API_PORT/prometheus/targets`

- `instance` the VPN IP of the connected machine with the netdata port (eg. `172.19.64.3:19999`)
- `job` fixed to `node`
- `node` the VPN IP of the connected machine
- `unit` the unit unique name of the connected machine

Access to Prometheus web interface is protected using a random generated URL, you can find it inside the module configuration file at `/home/nethsecurity-controller1/.config/state/config.json`.

### Loki

[Loki](https://grafana.com/oss/loki/) is a log storage, it stores logs from promtail. The configuration is available at `/home/nethsecurity-controller1/.config/etc/loki.yml`.

It uses TSDB as storage and it's configured to store logs for `loki_retention` days.

You can use `logcli` to query the logs.
First, access the module with `runagent` and source the environment file `loki.env` to set the `LOKI_ADDR` variable:
```
runagent -m nethsecurity-controller1 /bin/bash
. loki.env
```

List labels:
```
LOKI_ADDR=http://127.0.0.1:${LOKI_HTTP_PORT} logcli labels
```

You can do the same with curl: `curl -v http://127.0.0.1:${LOKI_HTTP_PORT}/loki/api/v1/labels`

Query logs:
```
LOKI_ADDR=http://127.0.0.1:${LOKI_HTTP_PORT} logcli series --analyze-labels '{hostname="NethSec"}'
LOKI_ADDR=http://127.0.0.1:${LOKI_HTTP_PORT} logcli query  '{hostname="NethSec"}'
```

Access to Loki web interface is protected using a random generated URL, you can find it inside the module configuration file at `/home/nethsecurity-controller1/.config/state/config.json`.

### Grafana

[Grafana](https://grafana.com/grafana/) is a metrics visualization, it visualizes metrics from prometheus and logs from loki. It's configured via environment variables and the configuration is available at `/home/nethsecurity-controller1/.config/state/grafana.env`, it also has a static configuration file at `/home/nethsecurity-controller1/.config/grafana.yml`.

The modules has already two pre-configured datasources: Loki and Prometheus containers.
It has also some pre-configured dashboards:

- nethsecurity.json: a dashboard with the most important metrics from the connected machines, like CPU, memory, disk, network, and system load
- logs.json: a dashboard where you can visualize the logs from all the connected machines and filter them by hostname, application, and priority
- loki.json: a dashboard with the most important metrics from Loki, like the number of logs ingested, the number of logs dropped, and the status of queries
- network_traffic.json: this dashboard uses data from Timescale database and shows the global network traffic by unit
- network_traffic_by_client.json: this dashboard uses data from Timescale database and shows the network traffic by unit and client (a client is a machine connected to the unit local network)
- network_traffic_by_host.json: this dashboard uses data from Timescale database and shows the network traffic by unit and host (a host is a machine on the internet)
- malware.json: this dashboard uses data from Timescale database and shows the malware blocked by the unit
- vpn.json: this dashboard uses data from Timescale database and shows the VPN connections

Grafana is accessible at `https://<controller-host>/grafana/`, default credentials are the same set for the controller. You should change them on the first login.

### WebSSH

[WebSSH](https://github.com/huashengdun/webssh) is a web-based ssh client. It's configured using parameters in the webssh.service unit.

Access to WebSSH is protected using a random generated URL, you can find it inside the module configuration file at `/home/nethsecurity-controller1/.config/state/config.json`.

### Timescale

[Timescale](https://docs.timescale.com/latest/main) is a time-series database for storing metrics. It's configured via environment variables and the configuration is available at `/home/nethsecurity-controller1/.config/state/db.env`.

You can connect to the database with the following command:
```
runagent -m nethsecurity-controller1
source db.env; podman exec -it timescale psql -U "${POSTGRES_USER}" -p "${POSTGRES_PORT}"
```

## Development mode

You can configure the API server to run in development mode, this will enable the debug mode and allow
access from remote UI clients.

Just set the `GIN_MODE` environment variable to `debug` in the `api.env` file and restart the controller:
```
runagent -m nethsecurity-controller1
sed -i 's/GIN_MODE=release/GIN_MODE=debug/' api.env
systemctl --user restart controller.service
```

## Troubleshooting

### Logs

To see al logs from the controller using the command line, execute this command con the node where the controller is running:
```
journalctl _UID=$(id -u nethsecurity-controller1)
```

To select only the logs about migrations, you can use the `--grep` option to filter the logs:
```
journalctl _UID=$(id -u nethsecurity-controller1) --grep 'MIGRATION'
```

### Database maintenance

The database is used to store configuration and metrics.
See [Database design](https://github.com/NethServer/nethsecurity-controller/tree/main/api#database-design) for more details.

#### DPI stats cleanup

The Timescale database is used to store metrics, it can grow quickly.
The `dpi_stats` can grow of millions of rows per day.
On clean machines after version 2.0.0, the `dpi_stats` has an index on the `uuid` field to speedup cleanup queries.

It's possible to manually create the index on upgraded machines, but it can take a long time to complete.
To create the index, run the following command on the node where the controller is running:
```
runagent -m nethsecurity-controller1
source db.env; podman exec -it timescale psql -U "${POSTGRES_USER}" -p "${POSTGRES_PORT}" -c 'CREATE INDEX IF NOT EXISTS idx_dpi_stats_uuid ON dpi_stats (uuid);'
```
If the dpi_stats table is too big to create an index, you can TRUNCATE it to remove all rows:
```
runagent -m nethsecurity-controller1
source db.env; podman exec -it timescale psql -U "${POSTGRES_USER}" -p "${POSTGRES_PORT}" -c 'TRUNCATE TABLE dpi_stats;'
```

Removing data from the `dpi_stats` table can take a long time, depending on the number of rows.
To speed up the cleanup process, you can run a batch deletion procedure that deletes rows in batches.
This will prevent long locks on the table and allow other queries to run in parallel. It will also reduce the load on the database.
To cleanup the `dpi_stats` table, you can run the following procedure to delete this rows older than 1 day.
```sql
DO $$
DECLARE
  batch_deleted INTEGER;
  total_deleted INTEGER := 0;
  batch_size INTEGER := 50000;
  max_iterations INTEGER := 1000; -- Prevent infinite loops
  iterations INTEGER := 0;
BEGIN
  RAISE NOTICE 'Starting batch deletion process at %', now();
  
  LOOP
    DELETE FROM dpi_stats
    WHERE (uuid, time) IN (
      SELECT uuid, time
      FROM dpi_stats
      WHERE time < (NOW() - INTERVAL '1 days')
      ORDER BY time
      LIMIT batch_size
    );
    
    GET DIAGNOSTICS batch_deleted = ROW_COUNT;
    total_deleted := total_deleted + batch_deleted;
    iterations := iterations + 1;
    
    RAISE NOTICE 'Batch %: Deleted % rows. Total deleted so far: %', 
      iterations, batch_deleted, total_deleted;
    
    EXIT WHEN batch_deleted = 0 OR iterations >= max_iterations;
    COMMIT;
    
  END LOOP;
  
  RAISE NOTICE 'Deletion complete at %. Total rows deleted: %', 
    now(), total_deleted;
END $$;
```

#### Cleanup orphaned unit data

To see if the cleanup job is scheduled, use this query:
```
SELECT * FROM timescaledb_information.jobs WHERE proc_name = 'cleanup_orphaned_unit_data';
```

To see the status of the cleanup job, use this query:
```
SELECT * FROM timescaledb_information.job_stats WHERE job_id = (SELECT job_id FROM timescaledb_information.jobs WHERE proc_name = 'cleanup_orphaned_unit_data');
```

You can run the cleanup job manually with the following query:
```
SELECT cleanup_orphaned_unit_data()'
```

### VPN

If you need to know the VPN current status, execute the following:
```
runagent -m nethsecurity-controller1
podman exec -ti vpn /bin/sh
kill -SIGUSR2 1
```

OpenVPN will dump statistics inside the journalctl, something like:
```
...
Aug 21 15:31:50 node1 vpn[1269361]: 2025-08-21 13:31:50 Virtual Address,Common Name,Real Address,Last Ref
Aug 21 15:31:50 node1 vpn[1269361]: 2025-08-21 13:31:50 172.22.221.26,xxxxxx-xxx-xxx-xx-xxxxx,1.2.3.4:57292,2025-08-21 13:31:48
Aug 21 15:31:50 node1 vpn[1269361]: 2025-08-21 13:31:50 172.22.221.147,xxxxxx-xxx-xxx-xx-xxxxx,1.2.3.4:44210,2025-08-21 13:31:47
...
```

### Prometheus

Prometheus has a useful dashboard to visualize the scrape targets and their status.
Prometheus can automatically find units to scrape by calling the `http://127.0.0.1:<api_port>/prometheus/targets` URL
every 60 seconds.

To see the current status of the targets, you can access the Prometheus UI at `https://<controller_fqdn>/<prometheus_path>/targets`.

You can find the the URL using this command:
```
runagent -m nethsecurity-controller1 cat config.json | jq -r '"https://\(.host)\(.prometheus_path)/targets"'
```

Looking into the target list you can find:

- what targets are up or down
- last scrape time
- last scrape duration

### Unit registration

Sometime the unit registration process may fail or become stale.
In such cases, you can check inside the log what is the reason of the failure.

Execute:
```sh
journalctl | grep -A1 'RegisterUnit'
```

Output will look like this:

```
Aug 21 16:05:29 ns8hks02 api[1269474]: nethsecurity_controller 2025/08/21 14:05:29 unit.go:485: [ERROR][RegisterUnit] unit not allowed, no certificate found for: <UUID>
Aug 21 16:05:29 ns8hks02 proxy[1269525]: <IP_ADDRESS> - - [21/Aug/2025:14:05:29 +0000] "POST /api/units/register HTTP/1.1" 403 91 "-" "-" 8723 "routerapi@file" "http://127.0.0.1:20030/" 0ms
--
Aug 21 16:05:32 ns8hks02 api[1269474]: nethsecurity_controller 2025/08/21 14:05:32 unit.go:369: [ERROR][RegisterUnit] unit subscription is required
Aug 21 16:05:32 ns8hks02 proxy[1269525]: <IP_ADDRESS> - - [21/Aug/2025:14:05:32 +0000] "POST /api/units/register HTTP/1.1" 403 87 "-" "-" 8724 "routerapi@file" "http://127.0.0.1:20030/" 0ms
```

## Uninstall

To uninstall the instance:

    remove-module --no-preserve nethsecurity-controller1
