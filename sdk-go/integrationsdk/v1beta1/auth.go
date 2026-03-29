package v1beta1

import (
	"time"

	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func OAuthConfigFromProto(proto *sharedv1beta1.OAuthConfig) (*oauth2.Config, []oauth2.AuthCodeOption) {
	config := &oauth2.Config{
		ClientID:     proto.ClientId,
		ClientSecret: proto.ClientSecret,
		RedirectURL:  proto.RedirectUrl,
		Endpoint: oauth2.Endpoint{
			AuthURL:       proto.Endpoint.GetAuthUrl(),
			TokenURL:      proto.Endpoint.GetTokenUrl(),
			DeviceAuthURL: proto.Endpoint.GetDeviceAuthUrl(),
		},
	}

	switch proto.AuthStyle {
	case sharedv1beta1.AuthStyle_AUTH_STYLE_IN_HEADER:
		config.Endpoint.AuthStyle = oauth2.AuthStyleInHeader
	case sharedv1beta1.AuthStyle_AUTH_STYLE_IN_PARAMS:
		config.Endpoint.AuthStyle = oauth2.AuthStyleInParams
	default:
		config.Endpoint.AuthStyle = oauth2.AuthStyleAutoDetect
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

func OAuthTokenToProto(token *oauth2.Token) *sharedv1beta1.OAuthToken {
	return &sharedv1beta1.OAuthToken{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       timestamppb.New(token.Expiry),
		ExpiresIn:    int64(time.Until(token.Expiry)),
	}
}

func OAuthTokenFromProto(proto *sharedv1beta1.OAuthToken) *oauth2.Token {
	if proto == nil {
		return nil
	}

	expiresIn := proto.GetExpiresIn()
	if expiresIn == 0 {
		expiresIn = int64(time.Until(proto.GetExpiry().AsTime()))
	}

	return &oauth2.Token{
		AccessToken:  proto.GetAccessToken(),
		TokenType:    proto.GetTokenType(),
		RefreshToken: proto.GetRefreshToken(),
		Expiry:       proto.GetExpiry().AsTime(),
		ExpiresIn:    proto.GetExpiresIn(),
	}
}
