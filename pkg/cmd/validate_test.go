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
apiVersion: tekton.dev/v1beta1
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
		},
		{
			name: "valid component",
			doc: `---
apiVersion: supply-chain.apps.tanzu.vmware.com/v1alpha1
kind: Component
metadata:
  name: my-pipeline
spec:
  description: a valid description
  pipelineRun:
    pipelineRef:
      name: a-pipeline
`,
			expectedErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := cmd.Parse([]byte(tc.doc))

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
