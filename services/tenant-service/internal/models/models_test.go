package models

import (
	"testing"
)

func TestTenantTierString(t *testing.T) {
	tests := []struct {
		tier     TenantTier
		expected string
	}{
		{TierFree, "free"},
		{TierBasic, "basic"},
		{TierProfessional, "professional"},
		{TierEnterprise, "enterprise"},
		{TierUnspecified, "unspecified"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.tier.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestProvisioningStatusString(t *testing.T) {
	tests := []struct {
		status   ProvisioningStatus
		expected string
	}{
		{StatusPending, "pending"},
		{StatusProvisioningDB, "provisioning_database"},
		{StatusCreatingAdmin, "creating_admin"},
		{StatusSettingQuota, "setting_quota"},
		{StatusSendingEmail, "sending_email"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusUnspecified, "unspecified"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.status.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
