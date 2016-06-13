package stormpathweb

import (
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/jarias/stormpath-sdk-go"
)

func isAccessTokenCookieSecure(r *http.Request) bool {
	if Config.AccessTokenCookieSecure == nil {
		return r.URL.Scheme == "https"
	}
	return *Config.AccessTokenCookieSecure
}

func accessTokenCookiePath() string {
	if Config.AccessTokenCookiePath == "" {
		if Config.BasePath == "" {
			return "/"
		}
		return Config.BasePath
	}
	return Config.AccessTokenCookiePath
}

func accesstokenCookieDomain(r *http.Request) string {
	if Config.AccessTokenCookieDomain == "" {
		if r.Host != "" {
			if strings.Contains(r.Host, ":") {
				return r.Host[:strings.Index(r.Host, ":")]
			}
			return r.Host
		}
	}
	return Config.AccessTokenCookieDomain
}

func getAccessTokenCookie(accessToken string, expires time.Time, r *http.Request) *http.Cookie {
	cookie := &http.Cookie{Value: accessToken, Name: Config.AccessTokenCookieName, Expires: expires}

	cookie.HttpOnly = Config.AccessTokenCookieHTTPOnly
	cookie.Secure = isAccessTokenCookieSecure(r)
	cookie.Path = accessTokenCookiePath()
	cookie.Domain = accesstokenCookieDomain(r)

	return cookie
}

func isRefreshTokenCookieSecure(r *http.Request) bool {
	if Config.RefreshTokenCookieSecure == nil {
		return r.URL.Scheme == "https"
	}
	return *Config.RefreshTokenCookieSecure
}

func refreshTokenCookiePath() string {
	if Config.RefreshTokenCookiePath == "" {
		if Config.BasePath == "" {
			return "/"
		}
		return Config.BasePath
	}
	return Config.RefreshTokenCookiePath
}

func refreshTokenCookieDomain(r *http.Request) string {
	if Config.RefreshTokenCookieDomain == "" {
		if r.Host != "" {
			if strings.Contains(r.Host, ":") {
				return r.Host[:strings.Index(r.Host, ":")]
			}
			return r.Host
		}
	}
	return Config.RefreshTokenCookieDomain
}

func getRefreshTokenCookie(refreshToken string, expires time.Time, r *http.Request) *http.Cookie {
	cookie := &http.Cookie{Value: refreshToken, Name: Config.RefreshTokenCookieName, Expires: expires}

	cookie.HttpOnly = Config.RefreshTokenCookieHTTPOnly
	cookie.Secure = isRefreshTokenCookieSecure(r)
	cookie.Path = refreshTokenCookiePath()
	cookie.Domain = refreshTokenCookieDomain(r)

	return cookie
}

func saveAuthenticationResult(w http.ResponseWriter, r *http.Request, authenticationResult stormpath.AuthResult, application *stormpath.Application) error {
	var err error

	oauthAccessTokenResult, ok := authenticationResult.(*stormpath.OAuthAccessTokenResult)
	if !ok {
		account := authenticationResult.GetAccount()

		oauthAccessTokenResult, err = exchangeToken(account, application)
		if err != nil {
			return err
		}
	}

	http.SetCookie(w, getAccessTokenCookie(oauthAccessTokenResult.AccessToken, getJwtExpiration(oauthAccessTokenResult.AccessToken), r))
	http.SetCookie(w, getRefreshTokenCookie(oauthAccessTokenResult.RefreshToken, getJwtExpiration(oauthAccessTokenResult.RefreshToken), r))

	return nil
}

func getJwtExpiration(jwtToken string) time.Time {
	token, _ := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(stormpath.GetClient().ClientConfiguration.APIKeySecret), nil
	})

	exp := time.Duration(token.Claims["exp"].(float64)) * time.Second

	return time.Unix(0, exp.Nanoseconds())
}

func getJwtID(jwtToken string) string {
	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(stormpath.GetClient().ClientConfiguration.APIKeySecret), nil
	})
	if err != nil {
		return ""
	}

	return token.Claims["jti"].(string)
}

func clearAuthentication(w http.ResponseWriter, r *http.Request, application *stormpath.Application) {
	accessTokenCookie, err := r.Cookie(Config.AccessTokenCookieName)
	if err == nil {
		accessToken := &stormpath.OAuthToken{}
		accessToken.Href = stormpath.GetClient().ClientConfiguration.BaseURL + "accessTokens/" + getJwtID(accessTokenCookie.Value)
		accessToken.Delete()
	}

	refreshTokenCookie, err := r.Cookie(Config.RefreshTokenCookieName)
	if err == nil {
		refreshToken := &stormpath.OAuthToken{}
		refreshToken.Href = stormpath.GetClient().ClientConfiguration.BaseURL + "refreshTokens/" + getJwtID(refreshTokenCookie.Value)
		refreshToken.Delete()
	}

	http.SetCookie(w, &http.Cookie{Name: Config.AccessTokenCookieName, Expires: time.Now().Add(-1 * time.Second)})
	http.SetCookie(w, &http.Cookie{Name: Config.RefreshTokenCookieName, Expires: time.Now().Add(-1 * time.Second)})
}
