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
			errMessage:  "APIVersion is not equal to tekton.dev/v1",
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
			errMessage:  "Key 'Metadata.Labels': Does not contain the key/value 'supply-chain.apps.tanzu.vmware.com/catalog: tanzu'",
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
			errMessage:  "Key 'Metadata.Name': MY_COMPONENT-1.0.0 does not appear to be in kebab-case",
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
			errMessage:  "Key 'Metadata.Name': my-component Does not end in a semantic version",
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
