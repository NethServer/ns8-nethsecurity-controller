*** Settings ***
Library           SSHLibrary
Library           Browser
Suite Setup       Connect to the node

*** Variables ***
${SSH_KEYFILE}    %{HOME}/.ssh/id_ecdsa

*** Keywords ***
Connect to the node
    Open Connection    ${NODE_ADDR}
    Login With Public Key    root    ${SSH_KEYFILE}
    ${output} =    Execute Command    systemctl is-system-running --wait
    Should Be True    '${output}' == 'running' or '${output}' == 'degraded'

*** Test Cases ***
Install nethsecurity-controller module
    ${output}    ${rc} =    Execute Command    add-module ${IMAGE_URL} 1
    ...    return_rc=True
    Should Be Equal As Integers    ${rc}    0
    &{output} =    Evaluate    ${output}
    Set Suite Variable    ${module_id}    ${output.module_id}

Take screenshot of Status page
    New Browser    chromium    headless=True
    New Page    https://${NODE_ADDR}/cluster-admin/#/apps/${module_id}
    Take Screenshot    filename=${OUTPUT DIR}/nethsecurity-controller-status.png
    Close Browser

Take screenshot of Settings page
    New Browser    chromium    headless=True
    New Page    https://${NODE_ADDR}/cluster-admin/#/apps/${module_id}?page=settings
    Take Screenshot    filename=${OUTPUT DIR}/nethsecurity-controller-settings.png
    Close Browser

Remove nethsecurity-controller module
    ${rc} =    Execute Command    remove-module --no-preserve ${module_id}
    ...    return_rc=True    return_stdout=False
    Should Be Equal As Integers    ${rc}    0
