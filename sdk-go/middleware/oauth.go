package middleware

import (
	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func OAuthTokenToProto(token *oauth2.Token) *sharedv1beta1.OAuthToken {
	return &sharedv1beta1.OAuthToken{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       timestamppb.New(token.Expiry),
		ExpiresIn:    token.ExpiresIn,
	}
}

func OAuthTokenFromProto(token *sharedv1beta1.OAuthToken) *oauth2.Token {
	if token == nil {
		return nil
	}
	return &oauth2.Token{
		AccessToken:  token.GetAccessToken(),
		TokenType:    token.GetTokenType(),
		RefreshToken: token.GetRefreshToken(),
		Expiry:       token.GetExpiry().AsTime(),
		ExpiresIn:    token.GetExpiresIn(),
	}
}

func OAuthConfigFromProto(proto *sharedv1beta1.OAuthConfig) (*oauth2.Config, []oauth2.AuthCodeOption) {
	var config = &oauth2.Config{
		ClientID:     proto.ClientId,
		ClientSecret: proto.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  proto.Endpoint.AuthUrl,
			TokenURL: proto.Endpoint.TokenUrl,
		},
	}

	if proto.Endpoint.DeviceAuthUrl != nil {
		config.Endpoint.DeviceAuthURL = proto.Endpoint.GetDeviceAuthUrl()
	}

	if len(proto.Scopes) > 0 {
		config.Scopes = proto.Scopes
	}

	var opts []oauth2.AuthCodeOption
	if len(proto.Params) > 0 {
		for key, val := range proto.Params {
			opts = append(opts, oauth2.SetAuthURLParam(key, val))
		}
	}

	return config, opts
}
