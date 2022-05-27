<template>
  <cv-grid fullWidth>
    <cv-row>
      <cv-column class="page-title">
        <h2>{{ $t("settings.title") }}</h2>
      </cv-column>
    </cv-row>
    <cv-row v-if="error.getConfiguration">
      <cv-column>
        <NsInlineNotification
          kind="error"
          :title="$t('action.get-configuration')"
          :description="error.getConfiguration"
          :showCloseButton="false"
        />
      </cv-column>
    </cv-row>
    <cv-row>
      <cv-column>
        <cv-tile light>
          <cv-form @submit.prevent="configureModule">
            <cv-text-input
              :label="$t('settings.host')"
              v-model="host"
              placeholder="controller.mydomain.org"
              :disabled="loading.getConfiguration || loading.configureModule"
              :invalid-message="error.host"
              ref="host"
            ></cv-text-input>
            <cv-toggle
              value="letsEncrypt"
              :label="$t('settings.lets_encrypt')"
              v-model="lets_encrypt"
              :disabled="loading.getConfiguration || loading.configureModule"
              class="mg-bottom"
            >
              <template slot="text-left">{{
                $t("settings.disabled")
              }}</template>
              <template slot="text-right">{{
                $t("settings.enabled")
              }}</template>
            </cv-toggle>
            <cv-text-input
              :label="$t('settings.cn')"
              v-model="cn"
              :placeholder="$t('settings.cn')"
              :disabled="loading.getConfiguration || loading.configureModule"
              :invalid-message="error.cn"
              ref="cn"
            ></cv-text-input>
            <cv-text-input
              :label="$t('settings.user')"
              v-model="user"
              :placeholder="$t('settings.user')"
              :disabled="loading.getConfiguration || loading.configureModule"
              :invalid-message="error.user"
              ref="user"
            ></cv-text-input>
            <cv-text-input
              :label="$t('settings.password')"
              v-model="password"
              :placeholder="'***********'"
              :disabled="loading.getConfiguration || loading.configureModule"
              :invalid-message="error.password"
              ref="password"
            ></cv-text-input>
            <cv-text-input
              :label="$t('settings.network')"
              v-model="network"
              :placeholder="$t('settings.network')"
              :disabled="
                loading.getConfiguration ||
                loading.configureModule ||
                !firstConfig
              "
              :invalid-message="error.network"
              ref="network"
            ></cv-text-input>
            <cv-text-input
              :label="$t('settings.netmask')"
              v-model="netmask"
              :placeholder="$t('settings.netmask')"
              :disabled="
                loading.getConfiguration ||
                loading.configureModule ||
                !firstConfig
              "
              :invalid-message="error.netmask"
              ref="netmask"
            ></cv-text-input>
            <cv-row v-if="error.configureModule">
              <cv-column>
                <NsInlineNotification
                  kind="error"
                  :title="$t('action.configure-module')"
                  :description="error.configureModule"
                  :showCloseButton="false"
                />
              </cv-column>
            </cv-row>
            <NsButton
              kind="primary"
              :icon="Save20"
              :loading="loading.configureModule"
              :disabled="loading.getConfiguration || loading.configureModule"
              >{{ $t("settings.save") }}</NsButton
            >
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
      urlCheckInterval: null,
      host: "",
      lets_encrypt: false,
      user: "",
      password: "",
      network: "",
      netmask: "",
      cn: "",
      firstConfig: true,
      loading: {
        getConfiguration: false,
        configureModule: false,
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
      },
    };
  },
  computed: {
    ...mapState(["instanceName", "core", "appName"]),
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
  },
  methods: {
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
      }
      this.cn = config.ovpn_cn;
      this.lets_encrypt = config.lets_encrypt;
      this.network = config.ovpn_network;
      this.netmask = config.ovpn_netmask;
      this.user = config.api_user;
      this.password = config.api_password;

      this.focusElement("host");
    },
    validateConfigureModule() {
      this.clearErrors(this);
      let isValidationOk = true;
      let fields = ["host", "cn", "network", "netmask", "user"];

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

      return isValidationOk;
    },
    configureModuleValidationFailed(validationErrors) {
      this.loading.configureModule = false;

      for (const validationError of validationErrors) {
        const param = validationError.parameter;

        // set i18n error message
        this.error[param] = this.$t("settings." + validationError.error);
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
</style>
