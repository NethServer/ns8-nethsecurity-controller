*** Settings ***
Library    SSHLibrary

*** Test Cases ***
Check if nethsecurity-controller is installed correctly
    ${output}  ${rc} =    Execute Command    add-module ${IMAGE_URL} 1
    ...    return_rc=True
    Should Be Equal As Integers    ${rc}  0
    &{output} =    Evaluate    ${output}
    Set Suite Variable    ${module_id}    ${output.module_id}

Check if nethsecurity-controller can be configured
    ${out}  ${err}  ${rc} =    Execute Command    api-cli run module/${module_id}/configure-module --data '{"host": "controller.dom.test", "lets_encrypt": false, "api_user": "admin", "api_password": "Nethesis,1234", "ovpn_network": "172.19.64.0", "ovpn_netmask": "255.255.255.0", "ovpn_cn": "nethsec", "loki_retention": 180, "prometheus_retention": 15}'
    ...    return_rc=True  return_stdout=True  return_stderr=True
    Should Be Equal As Integers    ${rc}  0

Check if admin interface is accessible
    Wait Until Keyword Succeeds    60 times    10 seconds    Access Admin Interface

Check if grafana is accessible
    Wait Until Keyword Succeeds    60 times    10 seconds    Access Grafana

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
