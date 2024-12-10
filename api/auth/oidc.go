package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"git.torden.tech/jonasbg/fit/types"
	"golang.org/x/oauth2"
)

var (
	Config          oauth2.Config
	GlobalOIDConfig *types.OpenIDConfiguration
)

func InitOIDC() error {
	issuerURL := os.Getenv("OIDC_ISSUER_URL")
	if issuerURL == "" {
		return fmt.Errorf("OIDC_ISSUER_URL is not set")
	}

	var err error
	GlobalOIDConfig, err = fetchOpenIDConfiguration(issuerURL)
	if err != nil {
		return fmt.Errorf("failed to fetch OpenID configuration: %v", err)
	}

	Config = oauth2.Config{
		ClientID:     os.Getenv("OIDC_CLIENT_ID"),
		ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("OIDC_REDIRECT_URL"),
		Endpoint: oauth2.Endpoint{
			AuthURL:  GlobalOIDConfig.AuthorizationEndpoint,
			TokenURL: GlobalOIDConfig.TokenEndpoint,
		},
		Scopes: []string{"openid", "profile", "email"},
	}

	return nil
}

func fetchOpenIDConfiguration(issuerURL string) (*types.OpenIDConfiguration, error) {
	resp, err := http.Get(issuerURL + "/.well-known/openid-configuration")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var config types.OpenIDConfiguration
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func GetUserInfo(accessToken string) (*types.UserInfo, error) {
	if GlobalOIDConfig == nil || GlobalOIDConfig.UserinfoEndpoint == "" {
		return nil, fmt.Errorf("UserInfo endpoint is not available")
	}

	req, err := http.NewRequest("GET", GlobalOIDConfig.UserinfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed with status code: %d", resp.StatusCode)
	}

	var userInfo types.UserInfo
	err = json.Unmarshal(body, &userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user info: %v", err)
	}

	return &userInfo, nil
}
