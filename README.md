# ns8-nextsec-controller

Setup and start an instance of [nextsecurity-controller](https://github.com/NethServer/nextsecurity-controller).

Each node can host multiple controller instances.

**Note**
This module is not yes present inside NS8 repository, so please install it manually following below instructions.

## Install

Instantiate the module with:

    add-module ghcr.io/nethserver/nextsec-controller:latest

The output of the command will return the instance name.
Output example:

    {"module_id": "nextsec-controller1", "image_name": "nextsec-controller", "image_url": "ghcr.io/nethserver/nextsec-controller:latest"}

## Configure

You can configure the controller directly from the UI or use the command line.

Let's assume that the nextsec-controller instance is named `nextsec-controller1`.

Launch `configure-module`, by setting the following parameters:
- `host`: a fully qualified domain name for the controller
- `lets_encrypt`: enable or disable Let's Encrypt certificate
- `ovpn_network`: OpenVPN network
- `ovpn_netmask`: OpenVPN netmasj
- `ovpn_cn`: OpenVPN Certificate CN
- `api_user`: controller admin user
- `api_password`: controller admin password

Example:

    api-cli run  module/nextsec-controller1/configure-module --data '{"host": "nscontroller.nethserver.org", "lets_encrypt": false, "ovpn_network": "172.19.64.0", "ovpn_netmask": "255.255.255.0", "ovpn_cn": "nextsec", "api_user": "admin", "api_password": "password"}'

The above command will:
- start and configure the nextsec-controller instance
- setup a route inside traefik to reach the controller

Send a test HTTP request to the nextsec-controller backend service:

    curl https://nscontroller.nethserver.org/

## Uninstall

To uninstall the instance:

    remove-module --no-preserve nextsec-controller1

## Testing

Test the module using the `test-module.sh` script:


    ./test-module.sh <NODE_ADDR> ghcr.io/nethserver/nextsec-controller:latest

The tests are made using [Robot Framework](https://robotframework.org/)
