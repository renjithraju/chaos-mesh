// Copyright 2020 Chaos Mesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var jvmchaoslog = logf.Log.WithName("jvmchaos-resource")

// +kubebuilder:webhook:path=/mutate-chaos-mesh-org-v1alpha1-jvmchaos,mutating=true,failurePolicy=fail,groups=chaos-mesh.org,resources=jvmchaos,verbs=create;update,versions=v1alpha1,name=mjvmchaos.kb.io

var _ webhook.Defaulter = &JVMChaos{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *JVMChaos) Default() {
	jvmchaoslog.Info("default", "name", in.Name)

	in.Spec.Selector.DefaultNamespace(in.GetNamespace())
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-chaos-mesh-org-v1alpha1-jvmchaos,mutating=false,failurePolicy=fail,groups=chaos-mesh.org,resources=jvmchaos,versions=v1alpha1,name=vjvmchaos.kb.io

var _ ChaosValidator = &JVMChaos{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *JVMChaos) ValidateCreate() error {
	jvmchaoslog.Info("validate create", "name", in.Name)

	return in.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *JVMChaos) ValidateUpdate(old runtime.Object) error {
	jvmchaoslog.Info("validate update", "name", in.Name)

	return in.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *JVMChaos) ValidateDelete() error {
	jvmchaoslog.Info("validate delete", "name", in.Name)

	// Nothing to do?
	return nil
}

// Validate validates chaos object
func (in *JVMChaos) Validate() error {
	specField := field.NewPath("spec")
	allErrs := in.ValidateScheduler(specField)
	allErrs = append(allErrs, in.ValidatePodMode(specField)...)
	allErrs = append(allErrs, in.validateJvmChaos(specField)...)
	if len(allErrs) > 0 {
		return fmt.Errorf(allErrs.ToAggregate().Error())
	}

	return nil
}

// ValidateScheduler validates the scheduler and duration
func (in *JVMChaos) ValidateScheduler(spec *field.Path) field.ErrorList {
	return ValidateScheduler(in, spec)
}

// ValidatePodMode validates the value with podmode
func (in *JVMChaos) ValidatePodMode(spec *field.Path) field.ErrorList {
	return ValidatePodMode(in.Spec.Value, in.Spec.Mode, spec.Child("value"))
}

func (in *JVMChaos) validateJvmChaos(spec *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	targetField := spec.Child("target")
	actionField := spec.Child("action")
	flagsField := spec.Child("flags")
	matcherField := spec.Child("matcher")
	if actions, ok := JvmSpec[in.Spec.Target]; ok {

		if actionPR, actionOK := actions[in.Spec.Action]; actionOK {
			if actionPR.Flags != nil {
				allErrs = append(allErrs, in.validateParameterRules(in.Spec.Flags, actionPR.Flags, flagsField, targetField, actionField)...)
			}

			if actionPR.Matcher != nil {
				allErrs = append(allErrs, in.validateParameterRules(in.Spec.Matchers, actionPR.Matcher, matcherField, targetField, actionField)...)
			}

		} else {
			supportActions := make([]JVMChaosAction, 0)
			for k := range actions {
				supportActions = append(supportActions, k)
			}

			notSupportedError := field.NotSupported(actionField, in.Spec.Action, toString(supportActions))
			errorMsg := fmt.Sprintf("target: %s does not match action: %s, action detail error: %s",
				in.Spec.Target, in.Spec.Action, notSupportedError)
			allErrs = append(allErrs, field.Invalid(targetField, in.Spec.Target, errorMsg))
		}
	} else {
		allErrs = append(allErrs, field.Invalid(targetField, in.Spec.Target, "unknown JVM chaos target"))
	}

	return allErrs
}

func toString(actions []JVMChaosAction) []string {
	ret := make([]string, 0)
	for _, act := range actions {
		ret = append(ret, string(act))
	}
	return ret
}

func (in *JVMChaos) validateParameterRules(parameters map[string]string, rules []ParameterRules, parent *field.Path, target *field.Path, action *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	for _, rule := range rules {
		innerField := parent.Child(rule.Name)

		var value = ""
		var exist = false
		if parameters != nil {
			value, exist = parameters[rule.Name]
		}
		if rule.Required && !exist {
			errorMsg := fmt.Sprintf("with %s: %s, %s: %s", target, in.Spec.Target, action, in.Spec.Action)
			allErrs = append(allErrs, field.Required(innerField, errorMsg))
		}

		if exist && rule.Required && rule.ParameterType == StringType {
			if len(value) == 0 {
				errorMsg := fmt.Sprintf("%s:%s cannot be empty", innerField, value)
				allErrs = append(allErrs, field.Invalid(innerField, value, errorMsg))
			}
		}

		if exist && rule.ParameterType == IntType {
			_, err := strconv.Atoi(value)
			if err != nil {
				errorMsg := fmt.Sprintf("%s:%s cannot parse as Int", innerField, value)
				allErrs = append(allErrs, field.Invalid(innerField, value, errorMsg))
			}
		}

		if exist && rule.ParameterType == BoolType {
			_, err := strconv.ParseBool(value)
			if err != nil {
				errorMsg := fmt.Sprintf("%s:%s cannot parse as boolean", innerField, value)
				allErrs = append(allErrs, field.Invalid(innerField, value, errorMsg))
			}
		}
	}
	return allErrs
}

// +kubebuilder:object:generate=false

// JvmSpec from chaosblade-jvm-spec.yaml
// chaosblade-jvm-spec.yaml file generated by https://github.com/chaosblade-io/chaosblade-exec-jvm/blob/master/chaosblade-exec-service/src/main/java/com/alibaba/chaosblade/exec/service/build/SpecMain.java
var JvmSpec = map[JVMChaosTarget]map[JVMChaosAction]ActionParameterRules{
	SERVLET: {
		JVMDelayAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "time", ParameterType: IntType, Required: true},
				{Name: "offset", ParameterType: IntType},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "method"},
				{Name: "querystring"},
				{Name: "requestpath"},
			},
		},
		JVMExceptionAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "exception", Required: true},
				{Name: "exception-message"},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "method"},
				{Name: "querystring"},
				{Name: "requestpath"},
			},
		},
	},
	PSQL: {
		JVMDelayAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "time", ParameterType: IntType, Required: true},
				{Name: "offset", ParameterType: IntType},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "sqltype"},
				{Name: "database"},
				{Name: "port", ParameterType: IntType},
				{Name: "host"},
				{Name: "table"},
			},
		},
		JVMExceptionAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "exception", Required: true},
				{Name: "exception-message"},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "sqltype"},
				{Name: "database"},
				{Name: "port", ParameterType: IntType},
				{Name: "host"},
				{Name: "table"},
			},
		},
	},
	MYSQL: {
		JVMDelayAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "time", ParameterType: IntType, Required: true},
				{Name: "offset", ParameterType: IntType},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "sqltype"},
				{Name: "database"},
				{Name: "port", ParameterType: IntType},
				{Name: "host"},
				{Name: "table"},
			},
		},
		JVMExceptionAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "exception", Required: true},
				{Name: "exception-message"},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "sqltype"},
				{Name: "database"},
				{Name: "port", ParameterType: IntType},
				{Name: "host"},
				{Name: "table"},
			},
		},
	},
	JEDIS: {
		JVMDelayAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "time", ParameterType: IntType, Required: true},
				{Name: "offset", ParameterType: IntType},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "cmd"},
				{Name: "key"},
			},
		},
		JVMExceptionAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "exception", Required: true},
				{Name: "exception-message"},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "cmd"},
				{Name: "key"},
			},
		},
	},
	HTTP: {
		JVMDelayAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "time", ParameterType: IntType, Required: true},
				{Name: "offset", ParameterType: IntType},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "httpclient4", ParameterType: BoolType},
				{Name: "rest", ParameterType: BoolType},
				{Name: "httpclient3", ParameterType: BoolType},
				{Name: "uri", Required: true},
			},
		},
		JVMExceptionAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "exception", Required: true},
				{Name: "exception-message"},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "httpclient4", ParameterType: BoolType},
				{Name: "rest", ParameterType: BoolType},
				{Name: "httpclient3", ParameterType: BoolType},
				{Name: "uri", Required: true},
			},
		},
	},
	ROCKETMQ: {
		JVMDelayAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "time", ParameterType: IntType, Required: true},
				{Name: "offset", ParameterType: IntType},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "producerGroup"},
				{Name: "topic"},
				{Name: "consumerGroup"},
			},
		},
		JVMExceptionAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "exception", Required: true},
				{Name: "exception-message"},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "producerGroup"},
				{Name: "topic"},
				{Name: "consumerGroup"},
			},
		},
	},
	TARS: {
		JVMDelayAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "time", ParameterType: IntType, Required: true},
				{Name: "offset", ParameterType: IntType},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "servant", ParameterType: BoolType},
				{Name: "functionname"},
				{Name: "client", ParameterType: BoolType},
				{Name: "servantname", Required: true},
			},
		},
		JVMExceptionAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "exception", Required: true},
				{Name: "exception-message"},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "servant", ParameterType: BoolType},
				{Name: "functionname"},
				{Name: "client", ParameterType: BoolType},
				{Name: "servantname", Required: true},
			},
		},
	},
	DUBBO: {
		JVMDelayAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "time", ParameterType: IntType, Required: true},
				{Name: "offset", ParameterType: IntType},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "appname"},
				{Name: "provider", ParameterType: BoolType},
				{Name: "service"},
				{Name: "version"},
				{Name: "consumer", ParameterType: BoolType},
				{Name: "group"},
			},
		},
		JVMExceptionAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "exception", Required: true},
				{Name: "exception-message"},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "appname"},
				{Name: "provider", ParameterType: BoolType},
				{Name: "service"},
				{Name: "version"},
				{Name: "consumer", ParameterType: BoolType},
				{Name: "group"},
			},
		},
		JVMThreadPoolFullAction: ActionParameterRules{
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "provider", ParameterType: BoolType},
			},
		},
	},
	JVM: {
		JVMDelayAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "time", ParameterType: IntType, Required: true},
				{Name: "offset", ParameterType: IntType},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "classname", Required: true},
				{Name: "after", ParameterType: BoolType},
				{Name: "methodname", Required: true},
			},
		},
		JVMExceptionAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "exception", Required: true},
				{Name: "exception-message"},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "classname", Required: true},
				{Name: "after", ParameterType: BoolType},
				{Name: "methodname", Required: true},
			},
		},
		JVMCodeCacheFillingAction: ActionParameterRules{},
		JVMCpuFullloadAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "cpu-count", ParameterType: IntType},
			},
		},
		JVMThrowDeclaredExceptionAction: ActionParameterRules{
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "classname", Required: true},
				{Name: "after", ParameterType: BoolType},
				{Name: "methodname", Required: true},
			},
		},
		JVMReturnAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "value", Required: true},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "classname", Required: true},
				{Name: "after", ParameterType: BoolType},
				{Name: "methodname", Required: true},
			},
		},
		JVMScriptAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "script-file"},
				{Name: "script-type"},
				{Name: "script-content"},
				{Name: "script-name"},
			},
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
				{Name: "classname", Required: true},
				{Name: "after", ParameterType: BoolType},
				{Name: "methodname", Required: true},
			},
		},
		JVMOOMAction: ActionParameterRules{
			Flags: []ParameterRules{
				{Name: "area", Required: true},
				{Name: "wild-mode", ParameterType: BoolType},
				{Name: "interval", ParameterType: IntType},
				{Name: "block", ParameterType: IntType},
			},
		},
	},
	DRUID: {
		JVMConnectionPoolFullAction: ActionParameterRules{
			Matcher: []ParameterRules{
				{Name: "effect-count", ParameterType: IntType},
				{Name: "effect-percent", ParameterType: IntType},
			},
		},
	},
}

type ActionParameterRules struct {
	Flags   []ParameterRules
	Matcher []ParameterRules
}

type ParameterType string

const (
	IntType    ParameterType = "int"
	BoolType   ParameterType = "bool"
	StringType ParameterType = "string"
)

type ParameterRules struct {
	Name          string
	ParameterType ParameterType
	Required      bool
}
