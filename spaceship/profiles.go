package spaceship

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-ios-auto-provision-appstoreconnect/appstoreconnect"
	"github.com/bitrise-steplib/steps-ios-auto-provision-appstoreconnect/autoprovision"
)

const (
	profileNameArgKey   = "--profile-name"
	profileTypeArgKey   = "--profile-type"
	certificateIDArgKey = "--certificate-id"

	bundleIDIdentifierArgKey = "--bundle-id"
	bundleIDNameArgKey       = "--bundle-id-name"
	entitlementsArgKey       = "--entitlements"
)

// Profile ...
type Profile struct {
	attributes     appstoreconnect.ProfileAttributes
	id             string
	bundleID       string
	deviceIDs      []string
	certificateIDs []string
}

// ID ...
func (p Profile) ID() string {
	return p.id
}

// Attributes ...
func (p Profile) Attributes() appstoreconnect.ProfileAttributes {
	return p.attributes
}

// CertificateIDs ...
func (p Profile) CertificateIDs() (map[string]bool, error) {
	ids := make(map[string]bool)
	for _, id := range p.certificateIDs {
		ids[id] = true
	}

	return ids, nil
}

// DeviceIDs ...
func (p Profile) DeviceIDs() (map[string]bool, error) {
	ids := make(map[string]bool)
	for _, id := range p.deviceIDs {
		ids[id] = true
	}

	return ids, nil
}

// BundleID ...
func (p Profile) BundleID() (appstoreconnect.BundleID, error) {
	return appstoreconnect.BundleID{
		ID: p.id,
		Attributes: appstoreconnect.BundleIDAttributes{
			Identifier: p.bundleID,
			Name:       p.attributes.Name,
		},
	}, nil
}

// ProfileClient ...
type ProfileClient struct {
	client *Client
}

// NewSpaceshipProfileClient ...
func NewSpaceshipProfileClient(client *Client) *ProfileClient {
	return &ProfileClient{client: client}
}

// ProfileInfo ...
type ProfileInfo struct {
	ID           string                           `json:"id"`
	UUID         string                           `json:"uuid"`
	Name         string                           `json:"name"`
	Status       appstoreconnect.ProfileState     `json:"status"`
	Expiry       time.Time                        `json:"expiry"`
	Platform     appstoreconnect.BundleIDPlatform `json:"platform"`
	Content      string                           `json:"content"`
	AppID        string                           `json:"app_id"`
	BundleID     string                           `json:"bundle_id"`
	Certificates []string                         `json:"certificates"`
	Devices      []string                         `json:"devices"`
}

func newProfile(p ProfileInfo) (Profile, error) {
	contents, err := base64.StdEncoding.DecodeString(p.Content)
	if err != nil {
		return Profile{}, fmt.Errorf("failed to decode profile contents: %v", err)
	}

	return Profile{
		attributes: appstoreconnect.ProfileAttributes{
			Name:           p.Name,
			UUID:           p.UUID,
			ProfileState:   appstoreconnect.ProfileState(p.Status),
			ProfileContent: contents,
			Platform:       p.Platform,
			ExpirationDate: appstoreconnect.Time(p.Expiry),
		},
		id:             p.ID,
		bundleID:       p.BundleID,
		certificateIDs: p.Certificates,
		deviceIDs:      p.Devices,
	}, nil
}

// AppInfo ...
type AppInfo struct {
	ID       string `json:"id"`
	BundleID string `json:"bundleID"`
	Name     string `json:"name"`
}

// FindProfile ...
func (c *ProfileClient) FindProfile(name string, profileType appstoreconnect.ProfileType) (autoprovision.Profile, error) {
	cmd, err := c.client.createRequestCommand("list_profiles",
		profileNameArgKey, name,
		profileTypeArgKey, string(profileType),
	)
	if err != nil {
		return nil, err
	}

	output, err := runSpaceshipCommand(cmd)
	if err != nil {
		return nil, err
	}

	var profileResponse struct {
		Data []ProfileInfo `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &profileResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if len(profileResponse.Data) == 0 {
		return nil, nil
	}
	if len(profileResponse.Data) > 1 {
		log.Warnf("More than one matching profile found, using the first one: %+v", profileResponse.Data)
	}

	profile, err := newProfile(profileResponse.Data[0])
	if err != nil {
		return nil, err
	}

	return profile, nil
}

// DeleteProfile ...
func (c *ProfileClient) DeleteProfile(id string) error {
	cmd, err := c.client.createRequestCommand("delete_profile", "--id", id)
	if err != nil {
		return err
	}

	_, err = runSpaceshipCommand(cmd)
	if err != nil {
		return err
	}

	return nil
}

// CreateProfile ...
func (c *ProfileClient) CreateProfile(name string, profileType appstoreconnect.ProfileType, bundleID appstoreconnect.BundleID, certificateIDs []string, deviceIDs []string) (autoprovision.Profile, error) {
	cmd, err := c.client.createRequestCommand("create_profile",
		bundleIDIdentifierArgKey, bundleID.Attributes.Identifier,
		certificateIDArgKey, certificateIDs[0],
		profileNameArgKey, name,
		profileTypeArgKey, string(profileType),
	)
	if err != nil {
		return nil, err
	}

	output, err := runSpaceshipCommand(cmd)
	if err != nil {
		return nil, err
	}

	var profileResponse struct {
		Data ProfileInfo `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &profileResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v (%s)", err, output)
	}

	profile, err := newProfile(profileResponse.Data)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

// FindBundleID ...
func (c *ProfileClient) FindBundleID(bundleIDIdentifier string) (*appstoreconnect.BundleID, error) {
	cmd, err := c.client.createRequestCommand("get_app",
		bundleIDIdentifierArgKey, bundleIDIdentifier,
	)
	if err != nil {
		return nil, err
	}

	output, err := runSpaceshipCommand(cmd)
	if err != nil {
		return nil, err
	}

	var appResponse struct {
		Data []AppInfo `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &appResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if len(appResponse.Data) == 0 {
		return nil, nil
	}

	bundleID := appResponse.Data[0]
	return &appstoreconnect.BundleID{
		ID: bundleID.ID,
		Attributes: appstoreconnect.BundleIDAttributes{
			Identifier: bundleID.BundleID,
			Name:       bundleID.Name,
		},
	}, nil
}

// CreateBundleID ...
func (c *ProfileClient) CreateBundleID(bundleIDIdentifier string) (*appstoreconnect.BundleID, error) {
	cmd, err := c.client.createRequestCommand("create_app",
		bundleIDIdentifierArgKey, bundleIDIdentifier,
		bundleIDNameArgKey, autoprovision.AppIDName(bundleIDIdentifier),
	)
	if err != nil {
		return nil, err
	}

	output, err := runSpaceshipCommand(cmd)
	if err != nil {
		return nil, err
	}

	var appResponse struct {
		Data AppInfo `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &appResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &appstoreconnect.BundleID{
		ID: appResponse.Data.ID,
		Attributes: appstoreconnect.BundleIDAttributes{
			Identifier: appResponse.Data.BundleID,
			Name:       appResponse.Data.Name,
		},
	}, nil
}

// CheckBundleIDEntitlements ...
func (c *ProfileClient) CheckBundleIDEntitlements(bundleID appstoreconnect.BundleID, projectEntitlements autoprovision.Entitlement) error {
	entitlementsBytes, err := json.Marshal(projectEntitlements)
	if err != nil {
		return err
	}
	entitlementsBase64 := base64.StdEncoding.EncodeToString(entitlementsBytes)

	cmd, err := c.client.createRequestCommand("check_bundleid",
		bundleIDIdentifierArgKey, bundleID.Attributes.Identifier,
		entitlementsArgKey, entitlementsBase64,
	)
	if err != nil {
		return err
	}

	_, err = runSpaceshipCommand(cmd)
	if err != nil {
		return err
	}

	return nil
}

// SyncBundleID ...
func (c *ProfileClient) SyncBundleID(bundleID appstoreconnect.BundleID, projectEntitlements autoprovision.Entitlement) error {
	entitlementsBytes, err := json.Marshal(projectEntitlements)
	if err != nil {
		return err
	}
	entitlementsBase64 := base64.StdEncoding.EncodeToString(entitlementsBytes)

	cmd, err := c.client.createRequestCommand("sync_bundleid",
		bundleIDIdentifierArgKey, bundleID.Attributes.Identifier,
		entitlementsArgKey, entitlementsBase64,
	)
	if err != nil {
		return err
	}

	_, err = runSpaceshipCommand(cmd)
	if err != nil {
		return err
	}

	return nil
}
