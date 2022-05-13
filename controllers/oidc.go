package controllers

type KeycloakClient struct {
	// TODO: implement
}

func (k *KeycloakClient) GetOIDCConfig(clientName string, redirectURL string) (*OidcConfig, error) {
	// TODO: we currently just fake it, but this would connect to
	// Keycloak, create an OIDC Client and return its information.
	return &OidcConfig{
		ClientID:     "",
		ClientSecret: "",
		RedirectURL:  redirectURL,
	}, nil
}
