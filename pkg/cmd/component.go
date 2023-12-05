package cmd

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func ValidateComponent(u unstructured.Unstructured) error {
	validate, translator, err := getValidator()
	if err != nil {
		return err
	}

	fields := &struct {
		APIVersion string `json:"apiVersion" validate:"required,eq=supply-chain.apps.tanzu.vmware.com/v1alpha1"`
		Kind       string `json:"kind" validate:"required,eq=Component"`
		Metadata   struct {
			Name   string            `json:"name" validate:"required,kebab-case,contains-semver,not-contains-component"`
			Labels map[string]string `json:"labels" validate:"contains-catalog-label"`
		} `json:"metadata"`
		Spec struct {
			Description string `json:"description" validate:"required"`
			PipelineRun struct {
				Params []struct {
					Name string `json:"name" validate:"required,kebab-case"`
				}
				PipelineRef struct {
					Name string `json:"name" validate:"required,kebab-case"`
				} `json:"pipelineRef" validate:"required"`
			} `json:"pipelineRun" validate:"required"`
		} `json:"spec" validate:"required"`
	}{}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &fields)
	if err != nil {
		return err
	}

	return translate(fields.Kind, fields.Metadata.Name, validate.Struct(fields), translator)
}
