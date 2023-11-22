package cmd_test

import (
	"testing"

	"github.com/garethjevans/component-validator/pkg/cmd"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		doc         string
		expectedErr bool
		errMessage  string
	}{
		{
			name: "v1 pipelines and tasks",
			doc: `---
apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: my-pipeline
---
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: my-pipeline
`,
			expectedErr: false,
		},
		{
			name: "v1beta1 pipelines and tasks",
			doc: `---
apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: my-pipeline
---
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: my-pipeline
`,
			expectedErr: true,
			errMessage:  "Key: 'APIVersion' Error:Field validation for 'APIVersion' failed on the 'eq' tag",
		},
		{
			name: "valid component",
			doc: `---
apiVersion: supply-chain.apps.tanzu.vmware.com/v1alpha1
kind: Component
metadata:
  name: my-pipeline-1.0.0
  labels:
    supply-chain.apps.tanzu.vmware.com/catalog: tanzu
spec:
  description: a valid description
  pipelineRun:
    pipelineRef:
      name: a-pipeline
`,
			expectedErr: false,
		},
		{
			name: "invalid catalog",
			doc: `---
apiVersion: supply-chain.apps.tanzu.vmware.com/v1alpha1
kind: Component
metadata:
  name: my-pipeline-1.0.0
  labels:
    supply-chain.apps.tanzu.vmware.com/catalog: other
spec:
  description: a valid description
  pipelineRun:
    pipelineRef:
      name: a-pipeline
`,
			expectedErr: true,
			errMessage:  "Key: 'Metadata.Labels' Error:Field validation for 'Labels' failed on the 'contains-catalog-label' tag",
		},
		{
			name: "invalid component - name",
			doc: `---
apiVersion: supply-chain.apps.tanzu.vmware.com/v1alpha1
kind: Component
metadata:
  name: MY_COMPONENT-1.0.0
  labels:
    supply-chain.apps.tanzu.vmware.com/catalog: tanzu
spec:
  description: a valid description
  pipelineRun:
    pipelineRef:
      name: a-pipeline
`,
			expectedErr: true,
			errMessage:  "Key: 'Metadata.Name' Error:Field validation for 'Name' failed on the 'kebab-case' tag",
		},
		{
			name: "invalid component - no semver",
			doc: `---
apiVersion: supply-chain.apps.tanzu.vmware.com/v1alpha1
kind: Component
metadata:
  name: my-component
  labels:
    supply-chain.apps.tanzu.vmware.com/catalog: tanzu
spec:
  description: a valid description
  pipelineRun:
    pipelineRef:
      name: a-pipeline
`,
			expectedErr: true,
			errMessage:  "Key: 'Metadata.Name' Error:Field validation for 'Name' failed on the 'contains-semver' tag",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := cmd.Parse([]byte(tc.doc))

			if tc.expectedErr {
				assert.Error(t, err)
				assert.Equal(t, tc.errMessage, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
