*** Settings ***
Library    SSHLibrary

*** Variables ***
${ADMIN_USER}    admin
${ADMIN_PASSWORD}    Nethesis,1234
@{DASHBOARD_UIDS}    ady5vjxjqywowd    liz0yRCZz    MQHVDmtWk    W3S__804z    b14a1181-a2ee-4df4-a732-888e0190037f
...    c8d43e5f-068e-4cac-9283-2318d9e1911b    fe0af3cb-9be0-4b2a-8ccb-86704956cf2e    dd395331-5dc0-4172-b243-8646c0ca3ccd    fdxlb58zxvpj4f

*** Test Cases ***
Check if nethsecurity-controller is installed correctly
    ${output}  ${rc} =    Execute Command    add-module ${IMAGE_URL} 1
    ...    return_rc=True
    Should Be Equal As Integers    ${rc}  0
    &{output} =    Evaluate    ${output}
    Set Global Variable    ${module_id}    ${output.module_id}

Check if nethsecurity-controller can be configured
    ${out}  ${err}  ${rc} =    Execute Command    api-cli run module/${module_id}/configure-module --data '{"host": "controller.dom.test", "lets_encrypt": false, "api_user": "admin", "api_password": "Nethesis,1234", "ovpn_network": "172.19.64.0", "ovpn_netmask": "255.255.255.0", "ovpn_cn": "nethsec", "loki_retention": 180, "prometheus_retention": 15}'
    ...    return_rc=True  return_stdout=True  return_stderr=True
    Should Be Equal As Integers    ${rc}  0

Take screenshots
    [Tags]    ui
    Import Library    Browser
    New Browser    chromium    headless=True
    New Context    ignoreHTTPSErrors=True
    Login to cluster-admin
    Go To    https://${NODE_ADDR}/cluster-admin/#/apps/${module_id}
    Wait For Elements State    iframe >>> h2 >> text="Status"    visible    timeout=10s
    Sleep    5s
    Take Screenshot    filename=${OUTPUT DIR}/browser/screenshot/1._Status.png
    Go To    https://${NODE_ADDR}/cluster-admin/#/apps/${module_id}?page=settings
    Wait For Elements State    iframe >>> h2 >> text="Settings"    visible    timeout=10s
    Sleep    5s
    Take Screenshot    filename=${OUTPUT DIR}/browser/screenshot/2._Settings.png
    Close Browser

Check if admin interface is accessible
    Wait Until Keyword Succeeds    60 times    10 seconds    Access Admin Interface

Check if grafana is accessible
    Wait Until Keyword Succeeds    60 times    10 seconds    Access Grafana

Check if grafana datasources are healthy
    Wait Until Keyword Succeeds    30 times    10 seconds    Grafana Datasource Is Healthy    prometheus
    Wait Until Keyword Succeeds    30 times    10 seconds    Grafana Datasource Is Healthy    loki
    Wait Until Keyword Succeeds    30 times    10 seconds    Grafana Datasource Is Healthy    timescale

Check if grafana dashboards are provisioned
    Wait Until Keyword Succeeds    30 times    10 seconds    Grafana Dashboards Are Provisioned

Check if loki is running
    ${output}  ${rc} =    Execute Command    runagent -m ${module_id} systemctl --user is-active loki.service
    ...    return_rc=True
    Should Be Equal As Integers    ${rc}  0
    Should Contain    ${output}    active

Check if prometheus is running
    ${output}  ${rc} =    Execute Command    runagent -m ${module_id} systemctl --user is-active prometheus.service
    ...    return_rc=True
    Should Be Equal As Integers    ${rc}  0
    Should Contain    ${output}    active

Check if prometheus is ready
    ${port} =    Read Module Env    prometheus.env    PROMETHEUS_PORT
    ${path} =    Read Module Env    prometheus.env    PROMETHEUS_PATH
    ${user} =    Read Module Env    secret.env    PROMETHEUS_AUTH_USERNAME
    ${pass} =    Read Module Env    secret.env    PROMETHEUS_AUTH_PASSWORD
    Set Suite Variable    ${prom_url}    http://127.0.0.1:${port}${path}
    Set Suite Variable    ${prom_auth}    ${user}:${pass}
    Wait Until Keyword Succeeds    30 times    5 seconds    Prometheus Request    /-/ready

Check if prometheus requires authentication
    ${out} =    Execute Command    curl -s -o /dev/null -w '\%{http_code}' '${prom_url}/api/v1/status/buildinfo'
    Should Be Equal    ${out}    401

Check if prometheus configuration is loaded
    ${out} =    Prometheus Request    /api/v1/status/config
    Should Contain    ${out}    "status":"success"
    Should Contain    ${out}    job_name: node
    Should Contain    ${out}    job_name: loki

Check if prometheus scrapes loki
    Wait Until Keyword Succeeds    30 times    10 seconds    Prometheus Target Is Up    loki

Check if promtail is running
    ${output}  ${rc} =    Execute Command    runagent -m ${module_id} systemctl --user is-active promtail.service
    ...    return_rc=True
    Should Be Equal As Integers    ${rc}  0
    Should Contain    ${output}    active

Check if webssh is running
    ${output}  ${rc} =    Execute Command    runagent -m ${module_id} systemctl --user is-active webssh.service
    ...    return_rc=True
    Should Be Equal As Integers    ${rc}  0
    Should Contain    ${output}    active

Check if timescale is running
    ${output}  ${rc} =    Execute Command    runagent -m ${module_id} systemctl --user is-active timescale.service
    ...    return_rc=True
    Should Be Equal As Integers    ${rc}  0
    Should Contain    ${output}    active

Check health endpoint
    Wait Until Keyword Succeeds    60 times    10 seconds    Check API Health

Check admin login
    ${token}=    Get Admin JWT Token
    Should Not Be Empty    ${token}
    Set Suite Variable    ${admin_token}    ${token}

Check unit creation
    Create Unit

Check if nethsecurity-controller is removed correctly
    ${rc} =    Execute Command    remove-module --no-preserve ${module_id}
    ...    return_rc=True  return_stdout=False
    Should Be Equal As Integers    ${rc}  0

*** Keywords ***
Access Admin Interface
    ${out}  ${err}  ${rc} =    Execute Command    curl -f -k -H "Host: controller.dom.test" https://127.0.0.1
    ...    return_rc=True  return_stdout=True  return_stderr=True
    Should Be Equal As Integers    ${rc}  0

Access Grafana
    ${out}  ${err}  ${rc} =    Execute Command    curl -s -k -L -u 'admin:Nethesis,1234' -H "Host: controller.dom.test" https://127.0.0.1/grafana/api/org
    ...    return_rc=True  return_stdout=True  return_stderr=True
    Should Be Equal As Integers    ${rc}  0
    Should Contain    ${out}    "name":"Main Org."

Grafana Request
    [Arguments]    ${path}
    ${out}  ${err}  ${rc} =    Execute Command    curl -s -S -f -k -L -u '${ADMIN_USER}:${ADMIN_PASSWORD}' -H "Host: controller.dom.test" 'https://127.0.0.1/grafana${path}'
    ...    return_rc=True  return_stdout=True  return_stderr=True
    Should Be Equal As Integers    ${rc}  0    ${err}
    RETURN    ${out}

Grafana Datasource Is Healthy
    [Arguments]    ${uid}
    ${out} =    Grafana Request    /api/datasources/uid/${uid}/health
    Should Contain    ${out}    "status":"OK"

Grafana Dashboards Are Provisioned
    ${out} =    Grafana Request    /api/search?type=dash-db
    FOR    ${uid}    IN    @{DASHBOARD_UIDS}
        Should Contain    ${out}    "uid":"${uid}"
    END

Read Module Env
    [Arguments]    ${file}    ${key}
    ${out}  ${rc} =    Execute Command    runagent -m ${module_id} sed -n 's/^${key}=//p' ${file}
    ...    return_rc=True
    Should Be Equal As Integers    ${rc}  0
    Should Not Be Empty    ${out}
    RETURN    ${out}

Prometheus Request
    [Arguments]    ${path}
    ${out}  ${err}  ${rc} =    Execute Command    curl -s -S -f -u '${prom_auth}' '${prom_url}${path}'
    ...    return_rc=True  return_stdout=True  return_stderr=True
    Should Be Equal As Integers    ${rc}  0    ${err}
    RETURN    ${out}

Prometheus Target Is Up
    [Arguments]    ${job}
    ${out} =    Prometheus Request    /api/v1/query?query=up%7Bjob%3D%22${job}%22%7D
    Should Contain    ${out}    "job":"${job}"
    Should Match Regexp    ${out}    "value":\\[[0-9.]+,"1"\\]

Check API Health
    ${out}  ${err}  ${rc} =    Execute Command    curl -s -k -H "Host: controller.dom.test" https://127.0.0.1/api/health
    ...    return_rc=True  return_stdout=True  return_stderr=True
    Should Be Equal As Integers    ${rc}  0
    Should Contain    ${out}    "status":"ok"

Get Admin JWT Token
    ${out}  ${err}  ${rc} =    Execute Command    curl -s -k -X POST -H "Host: controller.dom.test" -H "Content-Type: application/json" -d '{"username": "admin", "password": "Nethesis,1234"}' https://127.0.0.1/api/login
    ...    return_rc=True  return_stdout=True  return_stderr=True
    Should Be Equal As Integers    ${rc}  0
    Should Contain    ${out}    "token"
    ${token}=    Get JWT Token From Response    ${out}
    RETURN    ${token}

Get JWT Token From Response
    [Arguments]    ${response}
    ${token}=    Execute Command    echo '${response}' | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['token'])"
    ...    return_rc=False
    RETURN    ${token}

Create Unit
    ${unit_id}=  Set Variable  a141448d-2160-4857-b654-98a9d08843b9
    ${out}  ${err}  ${rc} =    Execute Command    curl -s -k -X POST -H "Host: controller.dom.test" -H "Content-Type: application/json" -H "Authorization: Bearer ${admin_token}" -d '{"unit_id": "${unit_id}"}' https://127.0.0.1/api/units
    ...    return_rc=True  return_stdout=True  return_stderr=True
    Should Be Equal As Integers    ${rc}  0
    Should Contain    ${out}    "code":200
    Should Contain    ${out}    "join_code"

Login to cluster-admin
    New Page    https://${NODE_ADDR}/cluster-admin/
    Fill Text    text="Username"    ${ADMIN_USER}
    Click    button >> text="Continue"
    Fill Text    text="Password"    ${ADMIN_PASSWORD}
    Click    button >> text="Log in"
    Wait For Elements State    css=#main-content    visible    timeout=10s
