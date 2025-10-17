<template>
  <cv-grid fullWidth>
    <cv-row>
      <cv-column class="page-title">
        <h2>{{ $t("settings.title") }}</h2>
      </cv-column>
    </cv-row>
    <cv-row v-if="error.getConfiguration">
      <cv-column>
        <NsInlineNotification kind="error" :title="$t('action.get-configuration')" :description="error.getConfiguration"
          :showCloseButton="false" />
      </cv-column>
    </cv-row>
    <cv-row>
      <cv-column>
        <cv-tile light>
          <NsInlineNotification v-if="!loading.getConfiguration" class="mg-bottom-xlg maxwidth" kind="info"
            :description="$t('settings.vpn_port_description', { port: vpn_port })
              " :showCloseButton="false">
            <NsButton></NsButton>
          </NsInlineNotification>
          <cv-form @submit.prevent="configureModule">
            <NsTextInput :label="$t('settings.network')" v-model="network" :placeholder="$t('settings.network')"
              :disabled="stillLoading || !firstConfig" :invalid-message="error.network" ref="network" :helper-text="$t('settings.network_helper')">
              <template #tooltip>{{
                $t("settings.network_tooltip")
              }}</template>
            </NsTextInput>
            <NsTextInput :label="$t('settings.netmask')" v-model="netmask" :placeholder="$t('settings.netmask')"
              :helper-text="$t('settings.netmask_helper')" :disabled="stillLoading || !firstConfig"
              :invalid-message="error.netmask" ref="netmask" class="mg-bottom-xlg">
              <template #tooltip>{{
                $t("settings.netmask_tooltip")
              }}</template>
            </NsTextInput>
            <div class="mg-top-xxlg">
              <NsTextInput :label="$t('settings.user')" v-model="user" :placeholder="$t('settings.user')"
                :disabled="stillLoading || !firstConfig"
                :invalid-message="error.user" ref="user" :helper-text="$t('settings.user_helper')">
                <template #tooltip>{{ $t("settings.user_tooltip") }}</template>
              </NsTextInput>
              <cv-row v-if="firstConfig">
                <cv-column>
                  <NsInlineNotification class="mg-bottom-xlg maxwidth" kind="info"
                    :title="$t('settings.password_information_title')" :description="$t('settings.password_information_description')
                      " :showCloseButton="false">
                    <NsButton></NsButton>
                  </NsInlineNotification>
                </cv-column>
              </cv-row>
              <NsTextInput :label="$t('settings.password')" v-model="password"
                :disabled="stillLoading || !firstConfig"
                :invalid-message="error.password" :placeholder="passwordPlaceholder" ref="password"
                class="mg-bottom-xlg" :helper-text="$t('settings.password_helper')">
                <template #tooltip>{{ $t("settings.password_tooltip") }}</template>
              </NsTextInput>
            </div>
            <div class="mg-top-xxlg">
              <NsTextInput :label="$t('settings.host')" v-model="host" placeholder="controller.mydomain.org"
                :disabled="stillLoading" :invalid-message="error.host" ref="host"
                :helper-text="$t('settings.host_helper')"></NsTextInput>
              <NsToggle
                value="letsEncrypt"
                :label="core.$t('apps_lets_encrypt.request_https_certificate')"
                v-model="lets_encrypt"
                :disabled="stillLoading"
                class="mg-bottom"
              >
                <template #tooltip>
                  <div class="mg-bottom-sm">
                    {{ core.$t("apps_lets_encrypt.lets_encrypt_tips") }}
                  </div>
                  <div class="mg-bottom-sm">
                    <cv-link @click="goToCertificates">
                      {{ core.$t("apps_lets_encrypt.go_to_tls_certificates") }}
                    </cv-link>
                  </div>
                </template>
                <template slot="text-left">{{
                  $t("settings.disabled")
                }}</template>
                <template slot="text-right">{{
                  $t("settings.enabled")
                }}</template>
              </NsToggle>
              <cv-row v-if="isLetsEncryptCurrentlyEnabled && !lets_encrypt">
                <cv-column>
                  <NsInlineNotification
                    kind="warning"
                    :title="
                      core.$t('apps_lets_encrypt.lets_encrypt_disabled_warning')
                    "
                    :description="
                      core.$t(
                        'apps_lets_encrypt.lets_encrypt_disabled_warning_description',
                        {
                          node: this.status.node_ui_name
                            ? this.status.node_ui_name
                            : this.status.node,
                        }
                      )
                    "
                    :showCloseButton="false"
                  />
                </cv-column>
              </cv-row>
            </div>
            <!-- advanced options -->
            <cv-accordion ref="accordion" class="maxwidth mg-bottom">
              <cv-accordion-item :open="toggleAccordion[0]">
                <template slot="title">{{ $t("settings.advanced") }}</template>
                <template slot="content">
                  <div class="mg-top-xxlg">
                    <NsTextInput :label="$t('settings.cn')" v-model="cn" :placeholder="$t('settings.cn')" :disabled="stillLoading || !firstConfig" :helper-text="$t('settings.cn_helper')" tooltipAlignment="end" 
                      tooltipDirection="right" ref="cn" :invalid-message="$t(error.cn)">
                      <template #tooltip>{{
                        $t("settings.cn_tooltip")
                      }}</template>
                    </NsTextInput>
                    <NsTextInput v-model.trim="loki_retention" ref="loki_retention"
                      :invalid-message="$t(error.loki_retention)" type="number" :label="$t('settings.loki_retention')"
                      :helper-text="$t('settings.loki_retention_helper')" :disabled="stillLoading">
                    </NsTextInput>
                    <NsTextInput v-model.trim="prometheus_retention" ref="prometheus_retention"
                      :invalid-message="$t(error.prometheus_retention)" type="number"
                      :label="$t('settings.prometheus_retention')"
                      :helper-text="$t('settings.prometheus_retention_helper')" :disabled="stillLoading">
                      <template #tooltip>{{
                          $t("settings.prometheus_retention_tooltip")
                        }}</template>
                    </NsTextInput>
                    <NsTextInput v-model.trim="maxmind_license"
                      ref="maxmind_license"
                      :invalid-message="$t(error.maxmind_license)"
                      type="password"
                      :label="$t('settings.maxmind_license')"
                      :helper-text="$t('settings.maxmind_license_helper')"
                      :disabled="stillLoading"
                      :passwordHideLabel="$t('password.hide_password')"
                      :passwordShowLabel="$t('password.show_password')">
                    </NsTextInput>
                  </div>
                  <div class="mg-top-xxlg">
                    <label class="bx--label">
                      <div class="label-and-tooltip">
                        <span>
                          {{ $t('settings.allowed_ips') }}
                        </span>
                        <!-- tooltip -->
                        <cv-interactive-tooltip
                          alignment="center"
                          direction="right"
                          class="info"
                        >
                          <template slot="content">
                            {{ $t("settings.allowed_ips_tooltip") }}
                          </template>
                        </cv-interactive-tooltip>
                      </div>
                    </label>
                    <cv-text-area
                      v-model="allowed_ips" 
                      ref="allowed_ips"
                      :invalid-message="$t(error.allowed_ips)"
                      :helper-text="$t('settings.allowed_ips_helper')"
                      :disabled="loading.configureModule"
                      :rows="4">
                    ></cv-text-area>
                  </div>
                </template>
              </cv-accordion-item>
            </cv-accordion>
            <!-- end advanced options -->

            <cv-row v-if="error.configureModule">
              <cv-column>
                <NsInlineNotification kind="error" :title="$t('action.configure-module')"
                  :description="error.configureModule" :showCloseButton="false" />
              </cv-column>
            </cv-row>
            <cv-row v-if="error.getStatus">
              <cv-column>
                <NsInlineNotification
                  kind="error"
                  :title="$t('action.get-status')"
                  :description="error.getStatus"
                  :showCloseButton="false"
                />
              </cv-column>
            </cv-row>
            <cv-row v-if="validationErrorDetails.length">
              <cv-column>
                <NsInlineNotification
                  kind="error"
                  :title="core.$t('apps_lets_encrypt.cannot_obtain_certificate')"
                  :description="formattedValidationErrorDetails"
                  :showCloseButton="false"
                />
              </cv-column>
            </cv-row>
            <NsButton kind="primary" :icon="Save20" :loading="loading.configureModule"
              :disabled="stillLoading">{{ $t("settings.save") }}</NsButton>
          </cv-form>
        </cv-tile>
      </cv-column>
    </cv-row>
  </cv-grid>
</template>

<script>
import to from "await-to-js";
import { mapState } from "vuex";
import {
  QueryParamService,
  UtilService,
  TaskService,
  IconService,
} from "@nethserver/ns8-ui-lib";

export default {
  name: "Settings",
  mixins: [TaskService, IconService, UtilService, QueryParamService],
  pageTitle() {
    return this.$t("settings.title") + " - " + this.appName;
  },
  data() {
    return {
      q: {
        page: "settings",
      },
      status: {},
      validationErrorDetails: [],
      urlCheckInterval: null,
      host: "",
      lets_encrypt: false,
      isLetsEncryptCurrentlyEnabled: false,
      user: "",
      password: "",
      network: "",
      netmask: "",
      vpn_port: "",
      cn: "",
      firstConfig: true,
      loki_retention: "180",
      prometheus_retention: "15",
      maxmind_license: "",
      passwordPlaceholder: "",
      allowed_ips: "",
      loading: {
        getConfiguration: false,
        configureModule: false,
        getStatus: false,
      },
      error: {
        getConfiguration: "",
        configureModule: "",
        host: "",
        lets_encrypt: "",
        user: "",
        password: "",
        network: "",
        netmask: "",
        cn: "",
        loki_retention: "",
        prometheus_retention: "",
        maxmind_license: "",
        allowed_ips: "",
        getStatus: "",
      },
    };
  },
  computed: {
    ...mapState(["instanceName", "core", "appName"]),
      stillLoading() {
      return (
        this.loading.getConfiguration ||
        this.loading.configureModule ||
        this.loading.getStatus
      );
    },
    formattedValidationErrorDetails() {
      return this.validationErrorDetails.join("\n");
    },
  },
  beforeRouteEnter(to, from, next) {
    next((vm) => {
      vm.watchQueryData(vm);
      vm.urlCheckInterval = vm.initUrlBindingForApp(vm, vm.q.page);
    });
  },
  beforeRouteLeave(to, from, next) {
    clearInterval(this.urlCheckInterval);
    next();
  },
  created() {
    this.getConfiguration();
    this.getStatus();
  },
  methods: {
    goToCertificates() {
      this.core.$router.push("/settings/tls-certificates");
    },
    async getStatus() {
      this.loading.getStatus = true;
      this.error.getStatus = "";
      const taskAction = "get-status";
      const eventId = this.getUuid();
      // register to task error
      this.core.$root.$once(
        `${taskAction}-aborted-${eventId}`,
        this.getStatusAborted
      );
      // register to task completion
      this.core.$root.$once(
        `${taskAction}-completed-${eventId}`,
        this.getStatusCompleted
      );
      const res = await to(
        this.createModuleTaskForApp(this.instanceName, {
          action: taskAction,
          extra: {
            title: this.$t("action." + taskAction),
            isNotificationHidden: true,
            eventId,
          },
        })
      );
      const err = res[0];
      if (err) {
        console.error(`error creating task ${taskAction}`, err);
        this.error.getStatus = this.getErrorMessage(err);
        this.loading.getStatus = false;
        return;
      }
    },
    getStatusAborted(taskResult, taskContext) {
      console.error(`${taskContext.action} aborted`, taskResult);
      this.error.getStatus = this.$t("error.generic_error");
      this.loading.getStatus = false;
    },
    getStatusCompleted(taskContext, taskResult) {
      this.status = taskResult.output;
      this.loading.getStatus = false;
    },
    async getConfiguration() {
      this.loading.getConfiguration = true;
      this.error.getConfiguration = "";
      const taskAction = "get-configuration";
      const eventId = this.getUuid();

      // register to task error
      this.core.$root.$once(
        `${taskAction}-aborted-${eventId}`,
        this.getConfigurationAborted
      );

      // register to task completion
      this.core.$root.$once(
        `${taskAction}-completed-${eventId}`,
        this.getConfigurationCompleted
      );

      const res = await to(
        this.createModuleTaskForApp(this.instanceName, {
          action: taskAction,
          extra: {
            title: this.$t("action." + taskAction),
            isNotificationHidden: true,
            eventId,
          },
        })
      );
      const err = res[0];

      if (err) {
        console.error(`error creating task ${taskAction}`, err);
        this.error.getConfiguration = this.getErrorMessage(err);
        this.loading.getConfiguration = false;
        return;
      }
    },
    getConfigurationAborted(taskResult, taskContext) {
      console.error(`${taskContext.action} aborted`, taskResult);
      this.error.getConfiguration = this.$t("error.generic_error");
      this.loading.getConfiguration = false;
    },
    getConfigurationCompleted(taskContext, taskResult) {
      this.loading.getConfiguration = false;
      const config = taskResult.output;

      this.host = config.host;
      if (config.host == "") {
        this.firstConfig = true;
      } else {
        this.firstConfig = false;
        this.passwordPlaceholder = this.$t("settings.password_placeholder");
      }
      this.cn = config.ovpn_cn;
      this.lets_encrypt = config.lets_encrypt;
      this.isLetsEncryptCurrentlyEnabled = config.lets_encrypt;
      this.network = config.ovpn_network;
      this.netmask = config.ovpn_netmask;
      this.user = config.api_user;
      this.password = config.api_password;
      this.loki_retention = config.loki_retention.toString();
      this.prometheus_retention = config.prometheus_retention.toString();
      this.maxmind_license = config.maxmind_license;
      this.vpn_port = config.vpn_port;
      this.allowed_ips = config.allowed_ips.join("\n");
      this.focusElement("host");
    },
    validateConfigureModule() {
      this.clearErrors(this);
      this.validationErrorDetails = [];
      let isValidationOk = true;
      let fields = ["host", "cn", "network", "netmask", "user", "loki_retention", "prometheus_retention"];

      // On first config the password must be non-empty
      if (this.firstConfig) {
        fields.push("password");
      }

      for (const v of fields) {
        if (!this[v]) {
          isValidationOk = false;
          this.error[v] = this.$t("common.required");
          this.focusElement(v);
        }
      }

      const user_re = new RegExp(/^[a-zA-Z0-9]+$/);
      if (!user_re.test(this.user)) {
        this.error.user = this.$t("error.invalid_user");
        this.focusElement("user");
        isValidationOk = false;
      }
      if (!user_re.test(this.cn)) {
        this.error.cn = this.$t("error.invalid_cn");
        this.focusElement("cn");
        isValidationOk = false;
      }

      const fqdn_re = new RegExp(
        /(?=^.{1,254}$)(^(?:(?!\d+\.|-)[a-zA-Z0-9_-]{1,63}(?<!-)\.?)+(?:[a-zA-Z]{2,})$)/,
        "g"
      );
      if (!fqdn_re.test(this.host)) {
        this.error.host = this.$t("error.invalid_host");
        this.focusElement("host");
        isValidationOk = false;
      }

      // validate loki_retention: minumum 1 day, maximum 365 days
      if (parseInt(this.loki_retention) > 365) {
        this.error.loki_retention = this.$t("error.loki_retention_max");
        this.focusElement("loki_retention");
        isValidationOk = false;
      }
      if (parseInt(this.loki_retention) < 1) {
        this.error.loki_retention = this.$t("error.loki_retention_min");
        this.focusElement("loki_retention");
        isValidationOk = false;
      }

      // validate prometheus_retention: minumum 1 day, maximum 365 days
      if (parseInt(this.prometheus_retention) > 365) {
        this.error.prometheus_retention = this.$t("error.prometheus_retention_max");
        this.focusElement("prometheus_retention");
        isValidationOk = false;
      }
      if (parseInt(this.prometheus_retention) < 1) {
        this.error.prometheus_retention = this.$t("error.prometheus_retention_min");
        this.focusElement("prometheus_retention");
        isValidationOk = false;
      }

      // validate network
      const network_re = new RegExp(
        /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.0$/
      );
      if (!network_re.test(this.network)) {
        this.error.network = this.$t("error.invalid_network");
        this.focusElement("network");
        isValidationOk = false;
      }

      // validate netmask
      const netmask_re = new RegExp(
        /^(255|254|252|248|240|224|192|128|0)\.(255|254|252|248|240|224|192|128|0)\.(255|254|252|248|240|224|192|128|0)\.(0|128|192|224|240|248|252|254|255)$/
      );
      if (!netmask_re.test(this.netmask)) {
        this.error.netmask = this.$t("error.invalid_netmask");
        this.focusElement("netmask");
        isValidationOk = false;
      }

      // validate allowed_ips: each line must be a valid IPv4 or IPv4/CIDR
      const allowed_ips_arr = this.allowed_ips
        .split("\n")
        .map((ip) => ip.trim())
        .filter((ip) => ip !== "");
      if (allowed_ips_arr && allowed_ips_arr.length > 0) {
        const ipv4CidrRe = /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\/([0-9]|[1-2][0-9]|3[0-2]))?$/;
        for (const ip of allowed_ips_arr) {
          if (ip.trim() === "") continue;
          const trimmedIp = ip.trim();
          if (!ipv4CidrRe.test(trimmedIp)) {
            this.error.allowed_ips = this.$t("error.invalid_allowed_ips");
            this.focusElement("allowed_ips");
            isValidationOk = false;
            break;
          }
        }
      }

      return isValidationOk;
    },
    configureModuleValidationFailed(validationErrors) {
      this.loading.configureModule = false;
      let focusAlreadySet = false;
      for (const validationError of validationErrors) {
        const param = validationError.parameter;
        if (validationError.details) {
          // show inline error notification with details
          this.validationErrorDetails = validationError.details
            .split("\n")
            .filter((detail) => detail.trim() !== "");
        } else {
          // set i18n error message
          this.error[param] = this.$t("settings." + validationError.error);
          if (!focusAlreadySet) {
            this.focusElement(param);
            focusAlreadySet = true;
          }
        }
      }
    },
    async configureModule() {
      const isValidationOk = this.validateConfigureModule();
      if (!isValidationOk) {
        return;
      }

      this.loading.configureModule = true;
      const taskAction = "configure-module";
      const eventId = this.getUuid();

      // register to task error
      this.core.$root.$once(
        `${taskAction}-aborted-${eventId}`,
        this.configureModuleAborted
      );

      // register to task validation
      this.core.$root.$once(
        `${taskAction}-validation-failed-${eventId}`,
        this.configureModuleValidationFailed
      );

      // register to task completion
      this.core.$root.$once(
        `${taskAction}-completed-${eventId}`,
        this.configureModuleCompleted
      );

      let params = {
        host: this.host,
        lets_encrypt: this.lets_encrypt,
        ovpn_network: this.network,
        ovpn_netmask: this.netmask,
        ovpn_cn: this.cn,
        api_user: this.user,
        loki_retention: parseInt(this.loki_retention),
        prometheus_retention: parseInt(this.prometheus_retention),
        maxmind_license: this.maxmind_license,
        allowed_ips: this.allowed_ips
          .split("\n")
          .map((ip) => ip.trim())
          .filter((ip) => ip !== "")
      };
      if (this.password) {
        params.api_password = this.password;
      }
      const res = await to(
        this.createModuleTaskForApp(this.instanceName, {
          action: taskAction,
          data: params,
          extra: {
            title: this.$t("settings.configure_instance", {
              instance: this.instanceName,
            }),
            description: this.$t("common.processing"),
            eventId,
          },
        })
      );
      const err = res[0];

      if (err) {
        console.error(`error creating task ${taskAction}`, err);
        this.error.configureModule = this.getErrorMessage(err);
        this.loading.configureModule = false;
        return;
      }
    },
    configureModuleAborted(taskResult, taskContext) {
      console.error(`${taskContext.action} aborted`, taskResult);
      this.error.configureModule = this.$t("error.generic_error");
      this.loading.configureModule = false;
    },
    configureModuleCompleted() {
      this.loading.configureModule = false;

      // reload configuration
      this.getConfiguration();
    },
  },
};
</script>

<style scoped lang="scss">
@import "../styles/carbon-utils";

.mg-bottom {
  margin-bottom: 2rem;
}

.maxwidth {
  max-width: 38rem;
}

.label-and-tooltip {
  display: flex;
  gap: 0.25rem;
  align-items: baseline;
  margin-bottom: -8px;
}
</style>