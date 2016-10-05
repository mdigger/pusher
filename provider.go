package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http2"
)

type ProviderToken struct {
	teamID     [10]byte          // 10 character Team ID
	keyID      [10]byte          // 10 character Key ID
	privateKey *ecdsa.PrivateKey // private key for sign
	jwt        string            // cached JWT
	created    time.Time         // cache creation time
	mu         sync.RWMutex
}

// Error parsing token provider.
var (
	ErrPTBad           = errors.New("bad provider token")
	ErrPTBadKeyID      = errors.New("bad provider token key id")
	ErrPTBadTeamID     = errors.New("bad provider token team ID")
	ErrPTBadPrivateKey = errors.New("bad provider token private key")
	ErrPTNotSet        = errors.New("token not set")
)

func NewProviderToken(teamID, keyID string, privateKeyData []byte) (*ProviderToken, error) {
	jwt := new(ProviderToken)
	if len(teamID) != 10 {
		return nil, ErrPTBadTeamID
	}
	copy(jwt.teamID[:], teamID)
	if len(keyID) != 10 {
		return nil, ErrPTBadKeyID
	}
	copy(jwt.keyID[:], keyID)

	block, data := pem.Decode(privateKeyData)
	if block != nil {
		data = block.Bytes
	}
	private, err := x509.ParsePKCS8PrivateKey(data)
	if err != nil {
		return nil, err
	}
	privateKey, ok := private.(*ecdsa.PrivateKey)
	if !ok {
		return nil, ErrPTBadPrivateKey
	}
	jwt.privateKey = privateKey
	return jwt, nil
}

// httpAPNSClient http.Client для отправки push-уведомлений.
var httpAPNSClient = &http.Client{Timeout: 15 * time.Second}

// Push отправляет push-уведомление на сервер APNS.
func (pt *ProviderToken) Push(notification Notification) (id string, err error) {
	// формируем запрос на отсылку push-уведомления
	req, err := notification.Request()
	if err != nil {
		return "", err
	}
	// запрашиваем и устанавливаем токен для авторизации APNS
	token, err := pt.JWT()
	if err != nil {
		return "", err
	}
	req.Header.Set("authorization", token)
	// отсылаем запрос
	resp, err := httpAPNSClient.Do(req)
	if resp.Body != nil {
		defer func() {
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		}()
	}
	// смотрим на ошибки обработки запроса
	if err, ok := err.(*url.Error); ok {
		if err, ok := err.Err.(http2.GoAwayError); ok {
			return "", APNSError(0, strings.NewReader(err.DebugData))
		}
	}
	if err != nil {
		return "", err
	}
	// разбираем ответ от APNS-сервиса
	id = resp.Header.Get("apns-id")
	if resp.StatusCode == http.StatusOK {
		return id, nil
	}
	// возвращаем описание ошибки
	return id, APNSError(resp.StatusCode, resp.Body)
}

var JWTLifeTime = time.Minute * 55

func (pt *ProviderToken) JWT() (string, error) {
	if pt == nil {
		return "", ErrPTNotSet
	}
	pt.mu.RLock()
	jwt := pt.jwt
	created := pt.created
	pt.mu.RUnlock()
	if jwt == "" || time.Since(created) > JWTLifeTime {
		return pt.createJWT()
	}
	return jwt, nil
}

func (pt *ProviderToken) createJWT() (string, error) {
	if pt.privateKey == nil {
		return "", ErrPTBadPrivateKey
	}
	buf := []byte(`************` +
		`{"alg":"ES256","kid":"0000000000"}.` + // header
		`*************` +
		`{"iss":"0000000000","iat":0000000000}.` + // claims
		`*******************************************` +
		`*******************************************`) // sign
	// header
	copy(buf[34:44], pt.keyID[:10])
	base64.RawURLEncoding.Encode(buf[:46], buf[12:46])
	// claims
	copy(buf[68:78], pt.teamID[:10])
	created := time.Now()
	copy(buf[86:96], []byte(strconv.FormatInt(created.Unix(), 10)))
	base64.RawURLEncoding.Encode(buf[47:97], buf[60:97])
	// sign
	sum := sha256.Sum256(buf[:97])
	r, s, err := ecdsa.Sign(rand.Reader, pt.privateKey, sum[:])
	if err != nil {
		panic(err)
	}
	copy(buf[120:152], r.Bytes())
	copy(buf[152:186], s.Bytes())
	base64.RawURLEncoding.Encode(buf[98:186], buf[120:186])
	jwt := fmt.Sprintf("bearer %s", buf)
	pt.mu.Lock()
	pt.jwt = jwt
	pt.created = created
	pt.mu.Unlock()
	return jwt, nil
}

type jsonProviderToken struct {
	TeamID     string `json:"teamId"`
	KeyID      string `json:"keyId"`
	PrivateKey []byte `json:"privateKey"`
}

// MarshalJSON returns the description of the ProviderToken using the JSON
// format.
func (pt *ProviderToken) MarshalJSON() ([]byte, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	privateKey, err := x509.MarshalECPrivateKey(pt.privateKey)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&jsonProviderToken{
		TeamID:     string(pt.teamID[:]),
		KeyID:      string(pt.keyID[:]),
		PrivateKey: privateKey,
	})
}

// UnmarshalJSON restores the ProviderToken from a JSON format.
func (pt *ProviderToken) UnmarshalJSON(data []byte) error {
	var jsonPT = new(jsonProviderToken)
	if err := json.Unmarshal(data, jsonPT); err != nil {
		return err
	}
	if len(jsonPT.TeamID) != 10 {
		return ErrPTBadTeamID
	}
	if len(jsonPT.KeyID) != 10 {
		return ErrPTBadKeyID
	}
	key, err := x509.ParseECPrivateKey(jsonPT.PrivateKey)
	if err != nil {
		return err
	}
	copy(pt.teamID[:], jsonPT.TeamID)
	copy(pt.keyID[:], jsonPT.KeyID)
	pt.privateKey = key
	return nil
}
