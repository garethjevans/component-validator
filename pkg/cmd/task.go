package cmd

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func ValidateTask(u unstructured.Unstructured) error {
	validate, translator, err := getValidator()
	if err != nil {
		return err
	}

	fields := &struct {
		APIVersion string `json:"apiVersion" validate:"required,eq=tekton.dev/v1"`
		Kind       string `json:"kind" validate:"required,eq=Task"`
		Metadata   struct {
			Name string `json:"name" validate:"required,kebab-case"`
		} `json:"metadata" validate:"required"`
		Spec struct {
			Params []struct {
				Name  string `json:"name" validate:"required,kebab-case"`
				Value string `json:"value"`
			} `json:"params" validate:"dive"`
			Results []struct {
				Name string `json:"name" validate:"required,kebab-case"`
				Type string `json:"type"`
			} `json:"results" validate:"dive"`
			StepTemplate struct {
				SecurityContext struct {
					AllowPrivilegeEscalation bool `json:"allowPrivilegeEscalation" validate:"eq=false"`
					Capabilities             struct {
						Drop []string `json:"drop" validate:"contains-all"`
					} `json:"capabilities" validate:"required"`
					RunAsNonRoot   bool `json:"runAsNonRoot" validate:"required,eq=true"`
					RunAsUser      int  `json:"runAsUser" validate:"required,ne=0"`
					SeccompProfile struct {
						Type string `json:"type" validate:"required,eq=RuntimeDefault"`
					} `json:"seccompProfile" validate:"required"`
				} `json:"securityContext" validate:"required"`
			} `json:"stepTemplate" validate:"required"`
		} `json:"spec" validate:"required"`
	}{}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &fields)
	if err != nil {
		return err
	}

	return translate(fields.Kind, fields.Metadata.Name, validate.Struct(fields), translator)
}
