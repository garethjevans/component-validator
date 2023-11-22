package cmd

import (
	"bytes"
	"fmt"
	"os"
	"regexp"

	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stoewer/go-strcase"
	"go.uber.org/multierr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	Path string
)

// NewValidateCmd creates a new token command.
func NewValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "validate",
		Short:        "Validates all components with the path supplied",
		Long:         "",
		Example:      "component-validator validate --path config/carvel.yaml",
		Aliases:      []string{"v"},
		RunE:         validate,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
	}

	cmd.Flags().StringVarP(&Path, "path", "p", "config/carvel.yaml", "The path to the component config to validate")

	return cmd
}

func Parse(source []byte) error {
	dec := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(source), 1024)

	var err error
	for {
		var u unstructured.Unstructured
		if dec.Decode(&u) != nil {
			break
		}

		switch u.GetKind() {
		case "Task":
			err = multierr.Append(err, ValidateTask(u))
			break
		case "Pipeline":
			err = multierr.Append(err, ValidatePipeline(u))
			break
		case "Component":
			err = multierr.Append(err, ValidateComponent(u))
			break
		default:
			fmt.Println("no validation specified for " + u.GetKind())
			break
		}
	}

	return err
}

func ValidateTask(u unstructured.Unstructured) error {
	validate := validator.New()

	fields := &struct {
		APIVersion string `json:"apiVersion" validate:"required,eq=tekton.dev/v1"`
		Kind       string `json:"kind" validate:"required,eq=Task"`
		Metadata   struct {
			Name string `json:"name" validate:"required,kebab-case"`
		} `json:"metadata"`
	}{}

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &fields)
	if err != nil {
		return err
	}

	err = validate.RegisterValidation("kebab-case", ValidateKebabCase)
	if err != nil {
		return fmt.Errorf(`failed to add custom validation for "{kebab-case}": %s`, err)
	}

	return validate.Struct(fields)
}

func ValidatePipeline(u unstructured.Unstructured) error {
	validate := validator.New()

	fields := &struct {
		APIVersion string `json:"apiVersion" validate:"required,eq=tekton.dev/v1"`
		Kind       string `json:"kind" validate:"required,eq=Pipeline"`
		Metadata   struct {
			Name string `json:"name" validate:"required,kebab-case"`
		} `json:"metadata"`
	}{}

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &fields)
	if err != nil {
		return err
	}

	err = validate.RegisterValidation("kebab-case", ValidateKebabCase)
	if err != nil {
		return fmt.Errorf(`failed to add custom validation for "{kebab-case}": %s`, err)
	}

	return validate.Struct(fields)
}

func ValidateComponent(u unstructured.Unstructured) error {
	validate := validator.New()

	fields := &struct {
		APIVersion string `json:"apiVersion" validate:"required,eq=supply-chain.apps.tanzu.vmware.com/v1alpha1"`
		Kind       string `json:"kind" validate:"required,eq=Component"`
		Metadata   struct {
			Name string `json:"name" validate:"required,kebab-case,contains-semver"`
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

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &fields)
	if err != nil {
		return err
	}

	err = validate.RegisterValidation("kebab-case", ValidateKebabCase)
	if err != nil {
		return fmt.Errorf(`failed to add custom validation for "{kebab-case}": %s`, err)
	}

	err = validate.RegisterValidation("contains-semver", ValidateContainsSemanticVersion)
	if err != nil {
		return fmt.Errorf(`failed to add custom validation for "{contains-semver}": %s`, err)
	}

	return validate.Struct(fields)
}

func validate(cmd *cobra.Command, args []string) error {
	b, err := os.ReadFile(Path)
	if err != nil {
		return err
	}

	err = Parse(b)

	errors := multierr.Errors(err)
	if len(errors) > 0 {
		for _, e := range errors {
			logrus.Errorf("%s", e)
		}
	}

	return err
}

func ValidateKebabCase(fl validator.FieldLevel) bool {
	name := fl.Field().String()
	return name == strcase.KebabCase(name)
}

func ValidateContainsSemanticVersion(fl validator.FieldLevel) bool {
	re := regexp.MustCompile(`(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
	name := fl.Field().String()
	return re.MatchString(name)
}
