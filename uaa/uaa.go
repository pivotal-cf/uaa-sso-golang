package uaa

import (
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
)

var InvalidRefreshToken = errors.New("UAA Invalid Refresh Token")

type Failure struct {
    code    int
    message string
}

func NewFailure(code int, message []byte) Failure {
    return Failure{
        code:    code,
        message: string(message),
    }
}

func (failure Failure) Error() string {
    return fmt.Sprintf("UAA Failure: %d %s", failure.code, failure.message)
}

type UAAInterface interface {
    Exchange(string) (Token, error)
    Refresh(string) (Token, error)
    LoginURL() string
}

type UAA struct {
    loginURL       string
    uaaURL         string
    ClientID       string
    ClientSecret   string
    RedirectURL    string
    Scope          string
    State          string
    AccessType     string
    ApprovalPrompt string
}

func NewUAA(loginURL, uaaURL, clientID, clientSecret string) UAA {
    return UAA{
        loginURL:     loginURL,
        uaaURL:       uaaURL,
        ClientID:     clientID,
        ClientSecret: clientSecret,
    }
}

func (u UAA) AuthorizeURL() string {
    return fmt.Sprintf("%s/oauth/authorize", u.loginURL)
}

func (u UAA) LoginURL() string {
    fmt.Printf("%+v\n", u)
    v := url.Values{}
    v.Set("access_type", u.AccessType)
    v.Set("approval_prompt", u.ApprovalPrompt)
    v.Set("client_id", u.ClientID)
    v.Set("redirect_uri", u.RedirectURL)
    v.Set("response_type", "code")
    v.Set("scope", u.Scope)
    v.Set("state", u.State)

    return u.AuthorizeURL() + "?" + v.Encode()
}

func (u UAA) tokenURL() string {
    return fmt.Sprintf("%s/oauth/token", u.uaaURL)
}

func (u UAA) Exchange(authCode string) (Token, error) {
    token := NewToken()

    params := url.Values{
        "grant_type":   {"authorization_code"},
        "redirect_uri": {u.RedirectURL},
        "scope":        {u.Scope},
        "code":         {authCode},
    }

    code, body, err := u.makeRequest("POST", u.tokenURL(), strings.NewReader(params.Encode()))
    if err != nil {
        return token, err
    }

    if code > 399 {
        return token, NewFailure(code, body)
    }

    json.Unmarshal(body, &token)
    return token, nil
}

func (u UAA) Refresh(refreshToken string) (Token, error) {
    token := NewToken()
    params := url.Values{
        "grant_type":    {"refresh_token"},
        "redirect_uri":  {u.RedirectURL},
        "refresh_token": {refreshToken},
    }
    code, body, err := u.makeRequest("POST", u.tokenURL(), strings.NewReader(params.Encode()))
    if err != nil {
        return token, err
    }
    switch {
    case code == http.StatusUnauthorized:
        return token, InvalidRefreshToken
    case code > 399:
        return token, NewFailure(code, body)
    }

    json.Unmarshal(body, &token)
    return token, nil
}

func (u UAA) makeRequest(method, fullURL string, requestBody io.Reader) (int, []byte, error) {
    uri, err := url.Parse(fullURL)
    if err != nil {
        return 0, []byte{}, err
    }

    host := uri.Scheme + "://" + uri.Host
    client := NewClient(host, u.ClientID, u.ClientSecret)
    return client.MakeRequest(method, uri.RequestURI(), requestBody)
}

func (u UAA) GetClientToken() (Token, error) {
    token := NewToken()
    params := url.Values{
        "grant_type":   {"client_credentials"},
        "redirect_uri": {u.RedirectURL},
    }
    code, body, err := u.makeRequest("POST", u.tokenURL(), strings.NewReader(params.Encode()))
    if err != nil {
        return token, err
    }

    if code > 399 {
        return token, NewFailure(code, body)
    }

    json.Unmarshal(body, &token)
    return token, nil
}
