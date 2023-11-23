package cmd

import (
	"bytes"
	"fmt"
	"os"
	"regexp"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	entranslations "github.com/go-playground/validator/v10/translations/en"

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
		case "Pipeline":
			err = multierr.Append(err, ValidatePipeline(u))
		case "Component":
			err = multierr.Append(err, ValidateComponent(u))
		default:
			fmt.Println("no validation specified for " + u.GetKind())
		}
	}

	return err
}

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

	return translate(validate.Struct(fields), translator)
}

func ValidatePipeline(u unstructured.Unstructured) error {
	validate, translator, err := getValidator()
	if err != nil {
		return err
	}

	fields := &struct {
		APIVersion string `json:"apiVersion" validate:"required,eq=tekton.dev/v1"`
		Kind       string `json:"kind" validate:"required,eq=Pipeline"`
		Metadata   struct {
			Name string `json:"name" validate:"required,kebab-case"`
		} `json:"metadata"`
	}{}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &fields)
	if err != nil {
		return err
	}

	return translate(validate.Struct(fields), translator)
}

func ValidateComponent(u unstructured.Unstructured) error {
	validate, translator, err := getValidator()
	if err != nil {
		return err
	}

	fields := &struct {
		APIVersion string `json:"apiVersion" validate:"required,eq=supply-chain.apps.tanzu.vmware.com/v1alpha1"`
		Kind       string `json:"kind" validate:"required,eq=Component"`
		Metadata   struct {
			Name   string            `json:"name" validate:"required,kebab-case,contains-semver"`
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

	return translate(validate.Struct(fields), translator)
}

func translate(err error, translator ut.Translator) error {
	if err != nil {
		var translated error

		errs := err.(validator.ValidationErrors)

		for _, e := range errs {
			translated = multierr.Append(translated, fmt.Errorf(e.Translate(translator)))
		}

		return translated
	}
	return err
}

func getValidator() (*validator.Validate, ut.Translator, error) {
	translator := en.New()
	uni := ut.New(translator, translator)

	trans, _ := uni.GetTranslator("en")
	validate := validator.New(validator.WithRequiredStructEnabled())

	err := entranslations.RegisterDefaultTranslations(validate, trans)
	if err != nil {
		return nil, nil, err
	}

	err = multierr.Combine(
		validate.RegisterValidation("kebab-case", ValidateKebabCase),
		validate.RegisterValidation("contains-semver", ValidateContainsSemanticVersion),
		validate.RegisterValidation("contains-catalog-label", ValidateContainsCatalogLabel),
		validate.RegisterValidation("contains-all", ValidateContainsAll),
	)
	if err != nil {
		return nil, nil, fmt.Errorf(`failed to add custom validations": %s`, err)
	}

	err = validate.RegisterTranslation("required", trans, func(ut ut.Translator) error {
		return ut.Add("required", "Key '{0}': is required", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", fe.StructNamespace())
		return t
	})
	if err != nil {
		return nil, nil, err
	}

	err = validate.RegisterTranslation("eq", trans, func(ut ut.Translator) error {
		return ut.Add("eq", "Key '{0}': Expected {1} to equal {2}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("eq", fe.StructNamespace(), fe.Value().(string), fe.Param())
		return t
	})
	if err != nil {
		return nil, nil, err
	}

	err = validate.RegisterTranslation("kebab-case", trans, func(ut ut.Translator) error {
		return ut.Add("kebab-case", "Key '{0}': {1} does not appear to be in kebab-case", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("kebab-case", fe.StructNamespace(), fe.Value().(string))
		return t
	})
	if err != nil {
		return nil, nil, err
	}

	err = validate.RegisterTranslation("contains-semver", trans, func(ut ut.Translator) error {
		return ut.Add("contains-semver", "Key '{0}': {1} Does not end in a semantic version", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("contains-semver", fe.StructNamespace(), fe.Value().(string))
		return t
	})
	if err != nil {
		return nil, nil, err
	}

	err = validate.RegisterTranslation("contains-catalog-label", trans, func(ut ut.Translator) error {
		return ut.Add("contains-catalog-label", "Key '{0}': Does not contain the key/value 'supply-chain.apps.tanzu.vmware.com/catalog: tanzu'", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("contains-catalog-label", fe.StructNamespace())
		return t
	})
	if err != nil {
		return nil, nil, err
	}

	err = validate.RegisterTranslation("contains-all", trans, func(ut ut.Translator) error {
		return ut.Add("contains-all", "Key '{0}': Must only contain the values [ALL]", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("contains-all", fe.StructNamespace())
		return t
	})
	if err != nil {
		return nil, nil, err
	}
	return validate, trans, nil
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

func ValidateContainsCatalogLabel(fl validator.FieldLevel) bool {
	numberOfEntries := len(fl.Field().MapKeys())
	if numberOfEntries == 0 {
		return false
	}

	m := fl.Field().Interface().(map[string]string)
	v, ok := m["supply-chain.apps.tanzu.vmware.com/catalog"]
	if !ok {
		return false
	}

	return v == "tanzu"
}

func ValidateContainsAll(fl validator.FieldLevel) bool {
	m := fl.Field().Interface().([]string)
	return len(m) == 1 && m[0] == "ALL"
}
