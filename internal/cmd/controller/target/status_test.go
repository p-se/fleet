package target

import (
	"testing"

	fleet "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Test_limit(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		val     []*intstr.IntOrString
		want    int
		wantErr bool
	}{
		{
			name:  "fixed value below count",
			count: 10,
			val: []*intstr.IntOrString{
				{IntVal: 5},
			},
			want: 5,
		},
		{
			name:  "fixed value above count",
			count: 10,
			val: []*intstr.IntOrString{
				{IntVal: 15},
			},
			want: 15,
		},
		{
			name:  "with value with zero count",
			count: 0,
			val: []*intstr.IntOrString{
				{IntVal: 15},
			},
			want: 1,
		},
		{
			name:  "fixed value with negative count",
			count: -15,
			val: []*intstr.IntOrString{
				{IntVal: 15},
			},
			want: 1,
		},
		{
			name:  "two fixed values should take the first one",
			count: 10,
			val: []*intstr.IntOrString{
				{IntVal: 5},
				{IntVal: 15},
			},
			want: 5,
		},
		{
			name:  "two fixed values should ignore nil",
			count: 10,
			val: []*intstr.IntOrString{
				nil,
				{IntVal: 15},
			},
			want: 15,
		},
		{
			name:  "percent value 50",
			count: 10,
			val: []*intstr.IntOrString{
				{Type: intstr.String, StrVal: "50%"},
			},
			want: 5,
		},
		{
			name:  "percent value 10",
			count: 10,
			val: []*intstr.IntOrString{
				{Type: intstr.String, StrVal: "10%"},
			},
			want: 1,
		},
		{
			name:  "negative percent value",
			count: 10,
			val: []*intstr.IntOrString{
				{Type: intstr.String, StrVal: "-10%"},
			},
			want: 1,
		},
		{
			name:  "percent value 10 with count 5",
			count: 5,
			val: []*intstr.IntOrString{
				{Type: intstr.String, StrVal: "10%"},
			},
			want: 1,
		},
		{
			name:  "no value should match count",
			count: 50,
			val:   []*intstr.IntOrString{},
			want:  50,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := limit(tt.count, tt.val...)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("limit() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("limit() succeeded unexpectedly")
			}
			if got != tt.want {
				t.Errorf("limit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isAvailable(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		target *fleet.BundleDeployment
		want   bool
	}{
		{
			name:   "empty target should be not be unavailable",
			target: nil,
			want:   false,
		},
		{
			name: "ready but AppliedDeploymentID does not match DeploymentID",
			target: &fleet.BundleDeployment{
				Spec: fleet.BundleDeploymentSpec{
					DeploymentID: "123",
				},
				Status: fleet.BundleDeploymentStatus{
					AppliedDeploymentID: "456",
					Ready:               true,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUnavailable(tt.target)
			if got != tt.want {
				t.Errorf("isAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_upToDate(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		target *Target
		want   bool
	}{
		{
			name:   "is not up-to-date if deployment is nil",
			target: &Target{},
			want:   false,
		},
		{
			name: "is not up-to-date if .Spec.StagedDeploymentID does not match target.DeploymentID",
			target: &Target{
				Deployment: &fleet.BundleDeployment{
					Spec: fleet.BundleDeploymentSpec{
						DeploymentID:       "id",
						StagedDeploymentID: "off-id",
					},
					Status: fleet.BundleDeploymentStatus{
						AppliedDeploymentID: "id",
					},
				},
				DeploymentID: "id",
			},
			want: false,
		},
		{
			name: "is not up-to-date if .Spec.DeploymentID does not match target.DeploymentID",
			target: &Target{
				Deployment: &fleet.BundleDeployment{
					Spec: fleet.BundleDeploymentSpec{
						DeploymentID:       "off-id",
						StagedDeploymentID: "id",
					},
					Status: fleet.BundleDeploymentStatus{
						AppliedDeploymentID: "id",
					},
				},
				DeploymentID: "id",
			},
			want: false,
		},
		{
			name: "is not up-to-date if .Status.AppliedDeploymentID does not match target.DeploymentID",
			target: &Target{
				Deployment: &fleet.BundleDeployment{
					Spec: fleet.BundleDeploymentSpec{
						DeploymentID:       "id",
						StagedDeploymentID: "id",
					},
					Status: fleet.BundleDeploymentStatus{
						AppliedDeploymentID: "off-id",
					},
				},
				DeploymentID: "id",
			},
			want: false,
		},
		{
			name: "is up-to-date",
			target: &Target{
				Deployment: &fleet.BundleDeployment{
					Spec: fleet.BundleDeploymentSpec{
						DeploymentID:       "id",
						StagedDeploymentID: "id",
					},
					Status: fleet.BundleDeploymentStatus{
						AppliedDeploymentID: "id",
					},
				},
				DeploymentID: "id",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := upToDate(tt.target)
			if got != tt.want {
				t.Errorf("upToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_updateStatusAndCheckUnavailable(t *testing.T) {
	availableTarget := func() *Target {
		return &Target{
			Deployment: &fleet.BundleDeployment{
				Spec: fleet.BundleDeploymentSpec{
					DeploymentID:       "id",
					StagedDeploymentID: "id",
				},
				Status: fleet.BundleDeploymentStatus{
					AppliedDeploymentID: "id",
				},
			},
			DeploymentID: "id",
		}
	}
	unavailableTarget := func() *Target {
		return &Target{
			Deployment: &fleet.BundleDeployment{
				Spec: fleet.BundleDeploymentSpec{
					DeploymentID:       "id",
					StagedDeploymentID: "id",
				},
				Status: fleet.BundleDeploymentStatus{
					AppliedDeploymentID: "id",
				},
			},
			DeploymentID: "off-id",
		}
	}

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		status  *fleet.PartitionStatus
		targets []*Target
		want    bool
	}{
		{
			name: "should be available if all targest are available",
			status: &fleet.PartitionStatus{
				MaxUnavailable: 0,
			},
			targets: []*Target{
				availableTarget(),
			},
			want: false,
		},
		{
			name: "should be unavailable if one targest are unavailable but 0 can be",
			status: &fleet.PartitionStatus{
				MaxUnavailable: 0,
			},
			targets: []*Target{
				availableTarget(),
				unavailableTarget(),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateStatusAndCheckUnavailable(tt.status, tt.targets)
			if got != tt.want {
				t.Errorf("updateStatusUnavailable() = %v, want %v", got, tt.want)
			}
		})
	}
}
