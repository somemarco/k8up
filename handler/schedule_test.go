package handler

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vshn/k8up/api/v1alpha1"
	"github.com/vshn/k8up/cfg"
	"github.com/vshn/k8up/job"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"
)

func TestScheduleHandler_mergeResourcesWithDefaults(t *testing.T) {
	tests := []struct {
		name                        string
		globalCPUResourceLimit      string
		globalCPUResourceRequest    string
		globalMemoryResourceLimit   string
		globalMemoryResourceRequest string
		template                    v1.ResourceRequirements
		resources                   v1.ResourceRequirements
		expected                    v1.ResourceRequirements
	}{
		{
			name:     "Given_NoGlobalDefaults_And_NoScheduleDefaults_When_NoSpec_Then_LeaveEmpty",
			expected: v1.ResourceRequirements{},
		},
		{
			name: "Given_NoGlobalDefaults_And_NoScheduleDefaults_When_Spec_Then_UseSpec",
			resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("50m"),
				},
			},
			expected: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("50m"),
				},
			},
		},
		{
			name: "Given_NoGlobalDefaults_And_ScheduleDefaults_When_NoSpec_Then_ApplyScheduleDefaults",
			template: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
			},
			expected: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
			},
		},
		{
			name: "Given_NoGlobalDefaults_And_ScheduleDefaults_When_Spec_Then_UseSpec",
			template: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
			},
			resources: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("50m"),
				},
			},
			expected: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("50m"),
				},
			},
		},
		{
			name:                        "Given_GlobalDefaults_And_NoScheduleDefaults_When_NoSpec_Then_UseGlobalDefaults",
			globalMemoryResourceRequest: "10Mi",
			template: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
			},
			expected: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
				Requests: v1.ResourceList{
					v1.ResourceMemory: resource.MustParse("10Mi"),
				},
			},
		},
		{
			name:                        "Given_GlobalDefaults_And_NoScheduleDefaults_When_Spec_Then_UseSpec",
			globalMemoryResourceRequest: "10Mi",
			resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceMemory: resource.MustParse("20Mi"),
				},
			},
			expected: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceMemory: resource.MustParse("20Mi"),
				},
			},
		},
		{
			name:                   "Given_GlobalDefaults_And_ScheduleDefaults_When_NoSpec_Then_UseSchedule",
			globalCPUResourceLimit: "10m",
			template: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
			},
			expected: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
			},
		},
		{
			name:                   "Given_GlobalDefaults_And_ScheduleDefaults_When_Spec_Then_UseSpec",
			globalCPUResourceLimit: "10m",
			template: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("100m"),
				},
			},
			resources: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
			},
			expected: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
			},
		},
	}
	cfg.Config = cfg.NewDefaultConfig()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.Config.GlobalCPUResourceLimit = tt.globalCPUResourceLimit
			cfg.Config.GlobalCPUResourceRequest = tt.globalCPUResourceRequest
			cfg.Config.GlobalMemoryResourceLimit = tt.globalMemoryResourceLimit
			cfg.Config.GlobalMemoryResourceRequest = tt.globalMemoryResourceRequest
			require.NoError(t, cfg.Config.ValidateSyntax())
			s := ScheduleHandler{schedule: &v1alpha1.Schedule{Spec: v1alpha1.ScheduleSpec{
				ResourceRequirementsTemplate: tt.template,
			}}}
			result := s.mergeResourcesWithDefaults(tt.resources)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScheduleHandler_generateSchedule(t *testing.T) {
	name := "k8up-system/my-scheduled-backup"
	tests := []struct {
		name             string
		schedule         string
		expectedSchedule string
	}{
		{
			name:             "WhenScheduleRandomHourlyGiven_ThenReturnStableRandomizedSchedule",
			schedule:         "@hourly-random",
			expectedSchedule: "2 * * * *",
		},
		{
			name:             "WhenScheduleRandomHourlyGiven_ThenReturnStableRandomizedSchedule",
			schedule:         "@daily-random",
			expectedSchedule: "2 14 * * *",
		},
		{
			name:             "WhenScheduleRandomHourlyGiven_ThenReturnStableRandomizedSchedule",
			schedule:         "@weekly-random",
			expectedSchedule: "2 14 0 * *",
		},
		{
			name:             "WhenScheduleRandomHourlyGiven_ThenReturnStableRandomizedSchedule",
			schedule:         "@monthly-random",
			expectedSchedule: "2 14 0 2 *",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ScheduleHandler{
				Config: job.Config{Log: zap.New(zap.UseDevMode(true))},
			}
			result := s.generateSchedule(name, tt.schedule)
			assert.Equal(t, tt.expectedSchedule, result)
		})
	}
}
