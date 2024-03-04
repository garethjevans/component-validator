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
			name: "v1 pipeline",
			doc: `---
apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: my-pipeline
`,
			expectedErr: false,
		},
		{
			name: "v1 task",
			doc: `
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: my-pipeline
spec:
  params: []
  results: []
  stepTemplate:
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
      runAsNonRoot: true
      runAsUser: 1001
      seccompProfile:
        type: RuntimeDefault
`,
			expectedErr: false,
		},
		{
			name: "v1 task - run as root",
			doc: `
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: my-pipeline
spec:
  params: []
  results: []
  stepTemplate:
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop: []
      runAsUser: 0
      runAsNonRoot: false
      seccompProfile:
        type: RuntimeDefault
`,
			expectedErr: false,
		},
		{
			name: "v1 task - run as root with non root user",
			doc: `
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: my-pipeline
spec:
  params: []
  results: []
  stepTemplate:
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
      runAsNonRoot: false
      runAsUser: 1001
      seccompProfile:
        type: RuntimeDefault
`,
			expectedErr: true,
			errMessage:  "Task/my-pipeline Key: 'Spec.StepTemplate.SecurityContext.RunAsNonRoot' Error:Field validation for 'RunAsNonRoot' failed on the 'compatible-nonroot' tag",
		},
		{
			name: "v1 task - run as non root with root user",
			doc: `
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: my-pipeline
spec:
  params: []
  results: []
  stepTemplate:
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
      runAsNonRoot: true
      runAsUser: 0
      seccompProfile:
        type: RuntimeDefault
`,
			expectedErr: true,
			errMessage:  "Task/my-pipeline Key: 'Spec.StepTemplate.SecurityContext.RunAsNonRoot' Error:Field validation for 'RunAsNonRoot' failed on the 'compatible-nonroot' tag",
		},
		{
			name: "v1 task - invalid capabilities",
			doc: `
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: my-pipeline
spec:
  params: []
  results: []
  stepTemplate:
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - all
      runAsNonRoot: true
      runAsUser: 1001
      seccompProfile:
        type: RuntimeDefault
`,
			expectedErr: true,
			errMessage:  "Task/my-pipeline Key 'Spec.StepTemplate.SecurityContext.Capabilities.Drop': Must only contain the values [ALL]",
		},
		{
			name: "v1 task - invalid seccomp profile",
			doc: `
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: my-pipeline
spec:
  params: []
  results: []
  stepTemplate:
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
      runAsNonRoot: true
      runAsUser: 1001
      seccompProfile:
        type: Localhost
`,
			expectedErr: true,
			errMessage:  "Task/my-pipeline Key 'Spec.StepTemplate.SecurityContext.SeccompProfile.Type': Expected Localhost to equal RuntimeDefault",
		},
		{
			name: "v1 task - missing spec",
			doc: `
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: my-pipeline
`,
			expectedErr: true,
			errMessage:  "Task/my-pipeline Key 'Spec': is required",
		},
		{
			name: "v1 task - missing step template",
			doc: `
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: my-pipeline
spec:
  params: []
`,
			expectedErr: true,
			errMessage:  "Task/my-pipeline Key 'Spec.StepTemplate': is required",
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
			errMessage:  "Task/my-pipeline Key 'APIVersion': Expected tekton.dev/v1beta1 to equal tekton.dev/v1; Task/my-pipeline Key 'Spec': is required",
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
			errMessage:  "Component/my-pipeline-1.0.0 Key 'Metadata.Labels': Does not contain the key/value 'supply-chain.apps.tanzu.vmware.com/catalog: tanzu'",
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
			errMessage:  "Component/MY_COMPONENT-1.0.0 Key 'Metadata.Name': MY_COMPONENT-1.0.0 does not appear to be in kebab-case",
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
			errMessage:  "Component/my-component Key 'Metadata.Name': my-component Does not end in a semantic version",
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
