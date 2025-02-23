package autoprovision

import (
	"testing"
	"time"

	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
	"github.com/bitrise-steplib/steps-ios-auto-provision-appstoreconnect/appstoreconnect"
	"github.com/stretchr/testify/require"
)

func Test_profileName(t *testing.T) {
	tests := []struct {
		profileType appstoreconnect.ProfileType
		bundleID    string
		want        string
		wantErr     bool
	}{
		{
			profileType: appstoreconnect.IOSAppDevelopment,
			bundleID:    "io.bitrise.app",
			want:        "Bitrise iOS development - (io.bitrise.app)",
			wantErr:     false,
		},
		{
			profileType: appstoreconnect.IOSAppStore,
			bundleID:    "io.bitrise.app",
			want:        "Bitrise iOS app-store - (io.bitrise.app)",
			wantErr:     false,
		},
		{
			profileType: appstoreconnect.IOSAppAdHoc,
			bundleID:    "io.bitrise.app",
			want:        "Bitrise iOS ad-hoc - (io.bitrise.app)",
			wantErr:     false,
		},
		{
			profileType: appstoreconnect.IOSAppInHouse,
			bundleID:    "io.bitrise.app",
			want:        "Bitrise iOS enterprise - (io.bitrise.app)",
			wantErr:     false,
		},

		{
			profileType: appstoreconnect.TvOSAppDevelopment,
			bundleID:    "io.bitrise.app",
			want:        "Bitrise tvOS development - (io.bitrise.app)",
			wantErr:     false,
		},
		{
			profileType: appstoreconnect.TvOSAppStore,
			bundleID:    "io.bitrise.app",
			want:        "Bitrise tvOS app-store - (io.bitrise.app)",
			wantErr:     false,
		},
		{
			profileType: appstoreconnect.TvOSAppAdHoc,
			bundleID:    "io.bitrise.app",
			want:        "Bitrise tvOS ad-hoc - (io.bitrise.app)",
			wantErr:     false,
		},
		{
			profileType: appstoreconnect.TvOSAppInHouse,
			bundleID:    "io.bitrise.app",
			want:        "Bitrise tvOS enterprise - (io.bitrise.app)",
			wantErr:     false,
		},
		{
			profileType: appstoreconnect.ProfileType("unknown"),
			bundleID:    "io.bitrise.app",
			want:        "",
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.profileType), func(t *testing.T) {
			got, err := ProfileName(tt.profileType, tt.bundleID)
			if (err != nil) != tt.wantErr {
				t.Errorf("profileName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("profileName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_findMissingContainers(t *testing.T) {
	tests := []struct {
		name        string
		projectEnts serialized.Object
		profileEnts serialized.Object
		want        []string
		wantErr     bool
	}{
		{
			name: "equal without container",
			projectEnts: serialized.Object(map[string]interface{}{
				"com.apple.developer.icloud-container-identifiers": []interface{}{},
			}),
			profileEnts: serialized.Object(map[string]interface{}{
				"com.apple.developer.icloud-container-identifiers": []interface{}{},
			}),

			want:    nil,
			wantErr: false,
		},
		{
			name: "equal with container",
			projectEnts: serialized.Object(map[string]interface{}{
				"com.apple.developer.icloud-container-identifiers": []interface{}{"container1"},
			}),
			profileEnts: serialized.Object(map[string]interface{}{
				"com.apple.developer.icloud-container-identifiers": []interface{}{"container1"},
			}),

			want:    nil,
			wantErr: false,
		},
		{
			name: "profile has more containers than project",
			projectEnts: serialized.Object(map[string]interface{}{
				"com.apple.developer.icloud-container-identifiers": []interface{}{},
			}),
			profileEnts: serialized.Object(map[string]interface{}{
				"com.apple.developer.icloud-container-identifiers": []interface{}{"container1"},
			}),

			want:    nil,
			wantErr: false,
		},
		{
			name: "project has more containers than profile",
			projectEnts: serialized.Object(map[string]interface{}{
				"com.apple.developer.icloud-container-identifiers": []interface{}{"container1"},
			}),
			profileEnts: serialized.Object(map[string]interface{}{
				"com.apple.developer.icloud-container-identifiers": []interface{}{},
			}),

			want:    []string{"container1"},
			wantErr: false,
		},
		{
			name: "project has containers but profile doesn't",
			projectEnts: serialized.Object(map[string]interface{}{
				"com.apple.developer.icloud-container-identifiers": []interface{}{"container1"},
			}),
			profileEnts: serialized.Object(map[string]interface{}{
				"otherentitlement": "",
			}),

			want:    []string{"container1"},
			wantErr: false,
		},
		{
			name: "error check",
			projectEnts: serialized.Object(map[string]interface{}{
				"com.apple.developer.icloud-container-identifiers": "break",
			}),

			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findMissingContainers(tt.projectEnts, tt.profileEnts)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, got, tt.want)
		})
	}
}

type MockProfile struct {
	attributes appstoreconnect.ProfileAttributes
}

func (m MockProfile) ID() string {
	return ""
}

func (m MockProfile) Attributes() appstoreconnect.ProfileAttributes {
	return m.attributes
}

func (m MockProfile) CertificateIDs() (map[string]bool, error) {
	return nil, nil
}

func (m MockProfile) DeviceIDs() (map[string]bool, error) {
	return nil, nil
}

func (m MockProfile) BundleID() (appstoreconnect.BundleID, error) {
	return appstoreconnect.BundleID{}, nil
}

func Test_IsProfileExpired(t *testing.T) {
	tests := []struct {
		prof                Profile
		minProfileDaysValid int
		name                string
		want                bool
	}{
		{
			name:                "no days set - profile expiry date after current time",
			minProfileDaysValid: 0,
			prof:                MockProfile{attributes: appstoreconnect.ProfileAttributes{ExpirationDate: appstoreconnect.Time(time.Now().Add(5 * time.Hour))}},
			want:                false,
		},
		{
			name:                "no days set - profile expiry date before current time",
			minProfileDaysValid: 0,
			prof:                MockProfile{attributes: appstoreconnect.ProfileAttributes{ExpirationDate: appstoreconnect.Time(time.Now().Add(-5 * time.Hour))}},
			want:                true,
		},
		{
			name:                "days set - profile expiry date after current time + days set",
			minProfileDaysValid: 2,
			prof:                MockProfile{attributes: appstoreconnect.ProfileAttributes{ExpirationDate: appstoreconnect.Time(time.Now().Add(5 * 24 * time.Hour))}},
			want:                false,
		},
		{
			name:                "days set - profile expiry date before current time + days set",
			minProfileDaysValid: 2,
			prof:                MockProfile{attributes: appstoreconnect.ProfileAttributes{ExpirationDate: appstoreconnect.Time(time.Now().Add(1 * 24 * time.Hour))}},
			want:                true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsProfileExpired(tt.prof, tt.minProfileDaysValid); got != tt.want {
				t.Errorf("checkProfileExpiry() = %v, want %v", got, tt.want)
			}
		})
	}
}
