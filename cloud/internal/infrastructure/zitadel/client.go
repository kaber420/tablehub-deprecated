package zitadel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tablehub/cloud/internal/domain/user"
)

// KeyConfig holds the parsed service account JSON structure.
type KeyConfig struct {
	Type   string `json:"type"`
	KeyID  string `json:"keyId"`
	Key    string `json:"key"`
	UserID string `json:"userId"`
}

type client struct {
	apiURL      string
	keyPath     string
	projectID   string
	orgID       string
	httpClient  *http.Client
	tokenMutex  sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

type noopClient struct{}

// NewClient creates a new Zitadel API Client.
func NewClient(apiURL, keyPath, projectID, orgID string) user.IdentityProvider {
	if keyPath == "" {
		return &noopClient{}
	}
	return &client{
		apiURL:     apiURL,
		keyPath:    keyPath,
		projectID:  projectID,
		orgID:      orgID,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (n *noopClient) CreateAndInviteUser(ctx context.Context, email, orgID, role string) (string, error) {
	return "noop_zitadel_usr_" + uuid.NewString(), nil
}

func (n *noopClient) AssignRole(ctx context.Context, userID, orgID, role string) error {
	return nil
}

func (n *noopClient) DeleteUser(ctx context.Context, userID string) error {
	return nil
}

func (n *noopClient) CreateOrganization(ctx context.Context, name string) (string, error) {
	return "noop_zitadel_org_" + uuid.NewString(), nil
}

func (n *noopClient) DeleteOrganization(ctx context.Context, orgID string) error {
	return nil
}

// getAccessToken authenticates using the machine key and caches the token.
func (c *client) getAccessToken(ctx context.Context) (string, error) {
	c.tokenMutex.Lock()
	// Return cached token if still valid for at least 2 minutes
	if c.cachedToken != "" && time.Until(c.tokenExpiry) > 2*time.Minute {
		token := c.cachedToken
		c.tokenMutex.Unlock()
		return token, nil
	}
	c.tokenMutex.Unlock()

	// Perform actual token exchange outside the lock
	token, expiry, err := c.refreshToken(ctx)
	if err != nil {
		return "", err
	}

	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()

	// Double check
	if c.cachedToken != "" && time.Until(c.tokenExpiry) > 2*time.Minute {
		return c.cachedToken, nil
	}

	c.cachedToken = token
	c.tokenExpiry = expiry

	return c.cachedToken, nil
}

func (c *client) refreshToken(ctx context.Context) (string, time.Time, error) {
	if c.keyPath == "" {
		return "", time.Time{}, fmt.Errorf("Zitadel machine key path is not configured")
	}

	// Read and parse service account key file
	data, err := os.ReadFile(c.keyPath)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("read machine key file: %w", err)
	}

	var keyConfig KeyConfig
	if err := json.Unmarshal(data, &keyConfig); err != nil {
		return "", time.Time{}, fmt.Errorf("parse machine key: %w", err)
	}

	// Parse private key
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(keyConfig.Key))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("parse RSA private key: %w", err)
	}

	// Create JWT assertions
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": keyConfig.UserID,
		"sub": keyConfig.UserID,
		"aud": c.apiURL,
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyConfig.KeyID

	signedJWT, err := token.SignedString(privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign JWT: %w", err)
	}

	// Exchange for access token
	tokenEndpoint := fmt.Sprintf("%s/oauth/v2/token", c.apiURL)
	formBody := fmt.Sprintf("grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer&assertion=%s", url.QueryEscape(signedJWT))

	req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, bytes.NewBufferString(formBody))
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", time.Time{}, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", time.Time{}, fmt.Errorf("decode token response: %w", err)
	}

	expiry := now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	return tokenResp.AccessToken, expiry, nil
}

// CreateAndInviteUser creates a human user in Zitadel.
func (c *client) CreateAndInviteUser(ctx context.Context, email, orgID, role string) (string, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("zitadel auth: %w", err)
	}

	endpoint := fmt.Sprintf("%s/management/v1/users/human", c.apiURL)
	userPayload := map[string]interface{}{
		"userName": email,
		"profile": map[string]interface{}{
			"firstName":   "Invited",
			"lastName":    "User",
			"displayName": email,
		},
		"email": map[string]interface{}{
			"email":           email,
			"isEmailVerified": false,
		},
	}

	bodyBytes, err := json.Marshal(userPayload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	activeOrgID := c.orgID
	if orgID != "" {
		activeOrgID = orgID
	}
	if activeOrgID != "" {
		req.Header.Set("x-zitadel-orgid", activeOrgID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create user API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("zitadel create user error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var userResp struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return "", fmt.Errorf("decode create user response: %w", err)
	}

	// Now assign the role
	if err := c.AssignRole(ctx, userResp.UserID, activeOrgID, role); err != nil {
		// Clean up user if role assignment fails (compensating action)
		_ = c.DeleteUser(ctx, userResp.UserID)
		return "", fmt.Errorf("assign user role: %w", err)
	}

	return userResp.UserID, nil
}

// AssignRole grants the specified role to the user on the project.
func (c *client) AssignRole(ctx context.Context, userID, orgID, role string) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("zitadel auth: %w", err)
	}

	endpoint := fmt.Sprintf("%s/management/v1/users/%s/grants", c.apiURL, userID)
	payload := map[string]interface{}{
		"projectId": c.projectID,
		"roleKeys":  []string{role},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	if orgID != "" {
		req.Header.Set("x-zitadel-orgid", orgID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("create grant API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("zitadel grant role error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// DeleteUser deletes the user from Zitadel.
func (c *client) DeleteUser(ctx context.Context, userID string) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("zitadel auth: %w", err)
	}

	endpoint := fmt.Sprintf("%s/management/v1/users/%s", c.apiURL, userID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	if c.orgID != "" {
		req.Header.Set("x-zitadel-orgid", c.orgID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete user API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("zitadel delete user error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (c *client) CreateOrganization(ctx context.Context, name string) (string, error) {
	return "noop_zitadel_org_" + uuid.NewString(), nil
}

func (c *client) DeleteOrganization(ctx context.Context, orgID string) error {
	return nil
}
