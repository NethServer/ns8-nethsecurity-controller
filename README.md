# ns8-nextsec-controller

Setup and start an instance of [nextsecurity-controller](https://github.com/NethServer/nextsecurity-controller).

Each node can host multiple controller instances.

## Install

Instantiate the module with:

    add-module ghcr.io/nethserver/nextsec-controller:latest 1

The output of the command will return the instance name.
Output example:

    {"module_id": "nextsec-controller1", "image_name": "nextsec-controller", "image_url": "ghcr.io/nethserver/nextsec-controller:latest"}

## Configure

Let's assume that the nextsec-controller instance is named `nextsec-controller1`.

Launch `configure-module`, by setting the following parameters:
- `host`: a fully qualified domain name for the controller
- `lets_encrypt`: enable or disable Let's Encrypt certificate

Example:

    api-cli run module/nextsec-controller1/configure-module --data '{"host": "nscontroller.nethserver.org", "lets_encrypt": false}'

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
