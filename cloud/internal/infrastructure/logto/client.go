package logto

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/tablehub/cloud/internal/domain/user"
)

type client struct {
	apiURL      string
	appID       string
	appSecret   string
	orgID       string
	httpClient  *http.Client
	tokenMutex  sync.Mutex
	cachedToken string
	tokenExpiry time.Time
	logger      zerolog.Logger
}

// NewClient creates a new Logto Management API client.
func NewClient(apiURL, appID, appSecret, orgID string, logger zerolog.Logger) user.IdentityProvider {
	return &client{
		apiURL:     apiURL,
		appID:      appID,
		appSecret:  appSecret,
		orgID:      orgID,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger.With().Str("component", "logto-client").Logger(),
	}
}

// getAccessToken fetches and caches the Management API access token using client credentials.
func (c *client) getAccessToken(ctx context.Context) (string, error) {
	c.tokenMutex.Lock()
	if c.cachedToken != "" && time.Until(c.tokenExpiry) > 2*time.Minute {
		token := c.cachedToken
		c.tokenMutex.Unlock()
		c.logger.Debug().Msg("Using cached access token")
		return token, nil
	}
	c.tokenMutex.Unlock()

	c.logger.Debug().Msg("Refreshing access token")
	token, expiry, err := c.refreshToken(ctx)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to refresh access token")
		return "", err
	}

	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()

	c.cachedToken = token
	c.tokenExpiry = expiry

	c.logger.Debug().Time("expiry", expiry).Msg("Access token refreshed successfully")
	return c.cachedToken, nil
}

func (c *client) refreshToken(ctx context.Context) (string, time.Time, error) {
	tokenEndpoint := fmt.Sprintf("%s/oidc/token", c.apiURL)
	resource := "https://default.logto.app/api"

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.appID)
	data.Set("client_secret", c.appSecret)
	data.Set("resource", resource)
	data.Set("scope", "all")

	req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("token request failed: %w", err)
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

	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	return tokenResp.AccessToken, expiry, nil
}

func generateRandomPassword() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes) + "aA1!"
}

func (c *client) CreateAndInviteUser(ctx context.Context, email, orgID, role string) (string, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("logto auth: %w", err)
	}

	// 1. Create the user in Logto
	endpoint := fmt.Sprintf("%s/api/users", c.apiURL)
	userPayload := map[string]interface{}{
		"primaryEmail": email,
		"username":     email,
		"password":     generateRandomPassword(),
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("logto create user api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("logto create user error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var userResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return "", fmt.Errorf("decode create user response: %w", err)
	}

	// Determine orgID
	activeOrgID := c.orgID
	if orgID != "" {
		activeOrgID = orgID
	}

	// 2. Assign organization and role
	if err := c.AssignRole(ctx, userResp.ID, activeOrgID, role); err != nil {
		_ = c.DeleteUser(ctx, userResp.ID)
		return "", fmt.Errorf("assign user role to org: %w", err)
	}

	return userResp.ID, nil
}

func (c *client) AssignRole(ctx context.Context, userID, orgID, role string) error {
	logger := c.logger.With().Str("method", "AssignRole").Str("user_id", userID).Str("org_id", orgID).Str("role", role).Logger()

	activeOrgID := c.orgID
	if orgID != "" {
		activeOrgID = orgID
	}
	if activeOrgID == "" {
		logger.Debug().Msg("No organization ID provided, skipping role assignment")
		return nil
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get access token")
		return fmt.Errorf("logto auth: %w", err)
	}

	// 1. Add user to the organization
	logger.Info().Msg("Adding user to organization")
	orgUserEndpoint := fmt.Sprintf("%s/api/organizations/%s/users", c.apiURL, activeOrgID)
	orgPayload := map[string]interface{}{
		"userIds": []string{userID},
	}
	orgBytes, err := json.Marshal(orgPayload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", orgUserEndpoint, bytes.NewBuffer(orgBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to add user to organization")
		return fmt.Errorf("logto add user to org api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		logger.Error().Int("status", resp.StatusCode).Str("body", string(respBody)).Msg("Failed to add user to organization")
		return fmt.Errorf("logto add user to org error (status %d): %s", resp.StatusCode, string(respBody))
	}
	logger.Info().Msg("User added to organization successfully")

	// 2. Resolve Logto role ID from role name
	logger.Info().Msg("Resolving organization role ID")
	roleID, err := c.resolveOrganizationRoleID(ctx, token, role)
	if err != nil {
		// Log the error but don't fail - role assignment is best-effort
		logger.Warn().Err(err).Str("role", role).Msg("Failed to resolve organization role ID, continuing without role assignment")
		return nil
	}
	logger.Info().Str("role_id", roleID).Msg("Organization role ID resolved successfully")

	// 3. Assign organization role
	logger.Info().Msg("Assigning organization role")
	roleEndpoint := fmt.Sprintf("%s/api/organizations/%s/users/%s/roles", c.apiURL, activeOrgID, userID)
	rolePayload := map[string]interface{}{
		"roleIds": []string{roleID},
	}
	roleBytes, err := json.Marshal(rolePayload)
	if err != nil {
		return err
	}

	reqRole, err := http.NewRequestWithContext(ctx, "POST", roleEndpoint, bytes.NewBuffer(roleBytes))
	if err != nil {
		return err
	}
	reqRole.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	reqRole.Header.Set("Content-Type", "application/json")

	respRole, err := c.httpClient.Do(reqRole)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to assign organization role")
		return fmt.Errorf("logto assign role api: %w", err)
	}
	defer respRole.Body.Close()

	if respRole.StatusCode != http.StatusOK && respRole.StatusCode != http.StatusCreated && respRole.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(respRole.Body)
		logger.Error().Int("status", respRole.StatusCode).Str("body", string(respBody)).Msg("Failed to assign organization role")
		return fmt.Errorf("logto assign role error (status %d): %s", respRole.StatusCode, string(respBody))
	}

	logger.Info().Msg("Organization role assigned successfully")
	return nil
}

// resolveOrganizationRoleID fetches Logto organization roles and returns the ID
// matching the given internal role name.
// Role name mapping: owner/admin → "admin", viewer → "collaborator".
func (c *client) resolveOrganizationRoleID(ctx context.Context, token, role string) (string, error) {
	logger := c.logger.With().Str("method", "resolveOrganizationRoleID").Str("role", role).Logger()

	// Map internal role names to Logto organization role names
	logtoRoleName := role
	if role == "owner" || role == "admin" {
		logtoRoleName = "admin"
	} else if role == "viewer" {
		logtoRoleName = "collaborator"
	}
	logger.Info().Str("logto_role_name", logtoRoleName).Msg("Resolving organization role ID")

	endpoint := fmt.Sprintf("%s/api/organization-roles", c.apiURL)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create request for organization roles")
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to call organization roles API")
		return "", fmt.Errorf("logto list organization roles api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		logger.Error().Int("status", resp.StatusCode).Str("body", string(respBody)).Msg("Organization roles API returned error")
		return "", fmt.Errorf("logto list organization roles error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var roles []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Key  string `json:"key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		logger.Error().Err(err).Msg("Failed to decode organization roles response")
		return "", fmt.Errorf("decode organization roles: %w", err)
	}

	logger.Info().Int("total_roles", len(roles)).Msg("Retrieved organization roles from Logto")
	for _, r := range roles {
		logger.Debug().Str("id", r.ID).Str("name", r.Name).Str("key", r.Key).Msg("Found organization role")
		if r.Name == logtoRoleName || r.Key == logtoRoleName {
			logger.Info().Str("role_id", r.ID).Msg("Found matching organization role")
			return r.ID, nil
		}
	}

	// Log available roles for debugging
	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = fmt.Sprintf("%s (key: %s)", r.Name, r.Key)
	}
	logger.Error().Str("requested_role", logtoRoleName).Strs("available_roles", roleNames).Msg("Organization role not found in Logto")
	return "", fmt.Errorf("organization role %q not found in Logto (available: %d roles)", logtoRoleName, len(roles))
}

func (c *client) DeleteUser(ctx context.Context, userID string) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("logto auth: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/users/%s", c.apiURL, userID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("logto delete user api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("logto delete user error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (c *client) CreateOrganization(ctx context.Context, name string) (string, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("logto auth: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/organizations", c.apiURL)
	payload := map[string]interface{}{
		"name": name,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("logto create org api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("logto create org error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var orgResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&orgResp); err != nil {
		return "", fmt.Errorf("decode create org response: %w", err)
	}

	return orgResp.ID, nil
}

func (c *client) DeleteOrganization(ctx context.Context, orgID string) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("logto auth: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/organizations/%s", c.apiURL, orgID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("logto delete org api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("logto delete org error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}
