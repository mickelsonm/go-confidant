package confidant

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
)

var (
	svc *kms.KMS
)

//GlobalConfig ... used for setting things globally for AWS/Confidant/etc
type GlobalConfig struct {
	//AWSRegion is the region we use with KMS
	AWSRegion string
}

//Config ...
type Config struct {
	//TokenLife is lifetime in minutes for the token
	TokenLife int
	//AuthKey is the KMS auth key
	AuthKey string
	//FromContext is the IAM role name requesting secrets (our client)
	FromContext string
	//ToContext is the IAM role name of the Confidant server
	ToContext string
	//URL is the url of the confidant server
	URL string
}

//Configure ... configures using global settings
func Configure(cfg *GlobalConfig) {
	svc = kms.New(session.New(
		&aws.Config{
			Region: aws.String(cfg.AWSRegion),
		},
	))
}

//ServiceResult ... service result
type ServiceResult struct {
	Error   error
	Result  bool
	Service []byte
}

//GetService ... gets service information from Confidant
func GetService(cfg *Config) ServiceResult {
	result := ServiceResult{}
	if svc == nil {
		result.Error = errors.New("kms service has not been initialized")
		return result
	}

	now := time.Now()
	tAfter := now.Add(time.Duration(cfg.TokenLife) * time.Minute)

	getDatetimeFormat := func(t time.Time) string {
		return fmt.Sprintf("%d%d%dT%d%d%dZ", t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second())
	}

	//setup the payload to encrypt
	payload, err := json.Marshal(struct {
		NotBefore string `json:"not_before"`
		NotAfter  string `json:"not_after"`
	}{
		NotBefore: getDatetimeFormat(now),
		NotAfter:  getDatetimeFormat(tAfter),
	})
	if err != nil {
		result.Error = fmt.Errorf("failed to setup payload to kms: %s\n", err)
		return result
	}

	//try encrypting via kms encrypt
	data, err := svc.Encrypt(&kms.EncryptInput{
		KeyId:     aws.String(cfg.AuthKey),
		Plaintext: payload,
		EncryptionContext: map[string]*string{
			"from": aws.String(cfg.FromContext),
			"to":   aws.String(cfg.ToContext),
		},
	})

	if err != nil {
		result.Error = fmt.Errorf("encrypt payload via kms error: %s\n", err)
		return result
	}

	//setup the http request to Confidant
	c := &http.Client{}
	url := fmt.Sprintf("%s/v1/services/%s", cfg.URL, cfg.FromContext)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		result.Error = fmt.Errorf(
			"failed to create http request to confidant server (%s): %s\n",
			cfg.URL, err)
		return result
	}

	//gets the token from the AWS response we got back
	token := base64.URLEncoding.EncodeToString(data.CiphertextBlob)

	//setup the authorization for Confidant
	basic := base64.URLEncoding.EncodeToString([]byte(cfg.FromContext + ":" + token))
	req.Header.Set("Authorization", fmt.Sprintf("Basic: %s", basic))

	//make the request to Confidant
	resp, err := c.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to make the request to confidant: %s\n", err)
		return result
	}

	//handles the responses we got back
	switch resp.StatusCode {
	case http.StatusNotFound:
		result.Error = errors.New("service not found in confidant")
		return result
	case http.StatusUnauthorized:
		result.Error = errors.New("authentication or authorization failed")
		return result
	case http.StatusOK:
		buf := bytes.Buffer{}
		err := json.NewDecoder(resp.Body).Decode(buf)
		if err != nil {
			result.Error = errors.New("failed to decode response from confidant")
		} else {
			result.Result = true
			result.Service = buf.Bytes()
		}
		return result
	default:
		result.Error = fmt.Errorf("received unexpected return from confidant (status: %d)\n", resp.StatusCode)
		return result
	}
}
